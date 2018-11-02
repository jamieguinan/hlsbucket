// Golang port of hlsbucket.c
// Catch Mpeg2TS packets over UDP, save as .ts, generate .m3u8 for HLS
// Also relay to second host.
package main

import (
	"os"
	"fmt"
	"path"
	"encoding/json"
	"io/ioutil"
	"net"
	"log"
	"bytes"
	// "encoding/hex"
	// "strconv"
	"time"
	"hash/crc32"
)

var CFGPATH = "hlsbucket.json"
var PKT_SIZE int = 188

type Config struct {
	SaveDir string
	ExpireCommand string
	HlsReceivePort int
	HlsRelayPort int
	DebugInOut bool
}

var cfg Config			// global config
var fout *os.File		// current MpegTS file being written

func MpegTS_PID(packet []byte) int {
	return ((int(packet[1]) & 0x1f)) << 8 | int(packet[2])
}

func handlePacket(buffer []byte, saveDir string) {
	// pid := MpegTS_PID(buffer)
	// log.Printf("%02x %02x %02x pid %d\n", buffer[0], buffer[1], buffer[2], pid)

	nal7index := bytes.Index(buffer, []byte("\x00\x00\x00\x01\x27"))
	if nal7index >= 0 {
		log.Printf("NAL7 @ %d\n", nal7index)
		if fout != nil {
			fout.Close()
			fout = nil
		}

		t := time.Now().UTC()
		outdir := fmt.Sprintf("%s/%04d/%02d/%02d/%02d",
			cfg.SaveDir,
			t.Year(),
			t.Month(),
			t.Day(),
			t.Hour())
		fmt.Printf("%v\n", outdir)

		if os.MkdirAll(outdir, 0755) != nil {
			log.Printf("error creating %s\n", outdir)
			return
		}

		dt := float64(t.Unix()) + float64(t.Nanosecond())/1000000000.0
		outname := fmt.Sprintf("%s/%.3f.ts", outdir, dt)

		var err error
		fout, err = os.Create(outname)
		if (err == nil) {
			fout.Chmod(0644)
			// expire
		}
	}

	if fout != nil {
		n, err := fout.Write(buffer)
		if (err != nil || n != len(buffer) ) {
			log.Printf("handlePacket write error")
		}
	}
}

// I wrote this tiny function because I'm spoiled by other
// languages that allow making a new array-type object in-line.
func clone(input []byte, start int, end int) []byte {
	cloned := make([]byte, end-start)
	copy(cloned, input[start:end])
	return cloned
}

func main() {
	var err error
	var cfgText []byte
	var receiver net.PacketConn
	var relay net.PacketConn
	var relayListen net.Listener

	setRelay := make(chan net.Conn)
	setRelayTCP := make(chan net.Conn)
	relayPacket := make(chan []byte)

	if (len(os.Args) != 1) {
		log.Printf("%s: specify options in %s\n", path.Base(os.Args[0]), CFGPATH)
		os.Exit(1)
	}

	cfgText, err = ioutil.ReadFile(CFGPATH)
	if err != nil {
		log.Printf("%s: could not load config file %s\n", path.Base(os.Args[0]), CFGPATH)
		os.Exit(1)
	}

	err = json.Unmarshal(cfgText, &cfg)
	if err != nil {
		log.Printf("unmarshal error!\n")
		os.Exit(1)
	}

	fmt.Printf("saveDir=%s\nexpireCommand=%s\nhlsReceivePort=%d\nhlsRelayPort=%d\n",
		cfg.SaveDir,
		cfg.ExpireCommand,
		cfg.HlsReceivePort,
		cfg.HlsRelayPort)

	receiver, err = net.ListenPacket("udp", fmt.Sprintf(":%d", cfg.HlsReceivePort))
	if err != nil {
		log.Printf("receive port listen error\n")
		os.Exit(1)
	}

	relay, err = net.ListenPacket("udp", fmt.Sprintf(":%d", cfg.HlsRelayPort))
	if err != nil {
		log.Printf("udp relay port listen error\n")
		os.Exit(1)
	}

	relayListen, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.HlsRelayPort))
	if err != nil {
		log.Printf("tcp relay port listen error\n")
		os.Exit(1)
	}

	go func () {
		// Receive data, store with expiration.
		buffer := make([]byte, 1500)
		for {
			n, _, err := receiver.ReadFrom(buffer)
			// Q: if n is <= 0, will err also be set?
			if err != nil {
				log.Printf("%v\n", err)
				continue
			}
			for i:=0; i < n; i += PKT_SIZE {
				handlePacket(buffer[i:i+PKT_SIZE], cfg.SaveDir)
			}
			if (cfg.DebugInOut) {
				log.Printf("%d bytes in, %#x\n", n,
					crc32.ChecksumIEEE(buffer[0:n]))
			}
			relayPacket <- clone(buffer, 0, n)
		}
	}()

	go func () {
		// Listen for request to relay.
		buffer := make([]byte, 1500)
		for {
			_, addr, err := relay.ReadFrom(buffer)
			if err != nil {
				continue
			}
			var c net.Conn
			c, err = net.Dial("udp", addr.String())
			if err == nil {
				// Inform main loop.
				setRelay <- c
			} else {
				log.Printf("%v\n", err)
			}
		}
	}()

	go func() {
		// Listen for TCP relay
		for {
			c, err := relayListen.Accept()
			if err == nil {
				setRelayTCP <- c
			} else {
				log.Printf("%v\n", err)
			}

		}
	}()

	var data []byte
	var rconn net.Conn
	var rconnSet bool = false
	var tconn net.Conn
	var tconnSet bool = false
	for {
		select {
		case data = <- relayPacket:
			if rconnSet {
				if (cfg.DebugInOut) {
					fmt.Printf("%d bytes out, %#x\n", len(data),
						crc32.ChecksumIEEE(data))
				}
				rconn.Write(data)
			}
			if tconnSet {
				//log.Printf("%d\n", len(data))
				tconn.Write(data)
			}
		case rconn = <- setRelay:
			log.Printf("setting udp relay to %v\n", rconn.RemoteAddr().String())
			rconnSet = true
		case tconn = <- setRelayTCP:
			log.Printf("setting tcp relay to %v\n", tconn.RemoteAddr().String())
			tconnSet = true
		}
	}
}
