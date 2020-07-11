// Golang port of hlsbucket.c
//
// * Catch Mpeg2TS packets over UDP
// * Save as .ts files
// * Generate urls for HLS including,
//    /play
//    /live_stream.m3u8
//    /ts/YYYY/MM/DD/HH/*.ts
//
// Also allow relay to second host over UDP and/or TCP.
//
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
	"time"
	"hash/crc32"
	"os/exec"
	"container/list"
	"sync"
	"path/filepath"
	"strings"
	"strconv"
)

var CFGPATH = "hlsbucket.json"
var PKT_SIZE int = 188

type Config struct {
	// exported fields, can be set via json
	SaveDir string
	HlsReceivePort int
	HlsRelayPort int
	HlsListenPort int
	DebugInOut bool
	ExpireCommand string
	ExpireTime string
	StartCode string

	// regular fields
	startCodeBytes []byte
	expireDuration time.Duration
}

type Global struct {
	recent *list.List	// recent .ts files
	mediaSequence int
	recentLock sync.Mutex	// lock for recent and mediaSequence

	cfg Config		// config
	fout *os.File		// current MpegTS file being written
}

var g Global			// global context


func MpegTS_PID(packet []byte) int {
	return ((int(packet[1]) & 0x1f)) << 8 | int(packet[2])
}

func handlePacket(buffer []byte, saveDir string) {
	//pid := MpegTS_PID(buffer)
	//log.Printf("%02x %02x %02x pid %d\n", buffer[0], buffer[1], buffer[2], pid)
	startCodeIndex := bytes.Index(buffer, g.cfg.startCodeBytes)
	if startCodeIndex >= 0 {
		log.Printf("start code @ %d\n", startCodeIndex)
		if g.fout != nil {
			g.fout.Close()
			g.fout = nil
		}

		t := time.Now().UTC()
		outdir := fmt.Sprintf("%s/%04d/%02d/%02d/%02d",
			g.cfg.SaveDir,
			t.Year(),
			t.Month(),
			t.Day(),
			t.Hour())
		// log.Printf("%v\n", outdir)

		if os.MkdirAll(outdir, 0755) != nil {
			log.Printf("error creating %s\n", outdir)
			return
		}

		// Note: I experimented with expiring `outdir`, but it is complicated
		// because the intermediate dirs might contain preserved
		// files. Better solution is to periodically flush the tree of
		// empty folders.

		dt := float64(t.Unix()) + float64(t.Nanosecond())/1000000000.0
		outname := fmt.Sprintf("%s/%.3f.ts", outdir, dt)

		var err error
		g.fout, err = os.Create(outname)
		if (err == nil) {
			// Start new output file.
			g.fout.Chmod(0644)
			if g.cfg.ExpireCommand != "" {
				cmd := exec.Command(g.cfg.ExpireCommand, outname, g.cfg.ExpireTime)
				if cmd.Start() == nil {
					// defer cmd.Wait()
					go cmd.Wait()
				}
			}

			// Add outname to recent list.
			g.recentLock.Lock()
			g.recent.PushBack(outname)
			if g.recent.Len() == 5 {
				// Trim back to 4 names, will only
				// list 3 of those in the m3u8.
				g.recent.Remove(g.recent.Front())
			}
			g.mediaSequence += 1
			g.recentLock.Unlock()
		}
	}

	if g.fout != nil {
		n, err := g.fout.Write(buffer)
		if (err != nil || n != len(buffer) ) {
			log.Printf("handlePacket write error")
		}
	}
}

func vacuum() {
	// Periodically walk SaveDir and remove expired files and
	// empty folders. filepath.Walk is breadth-first, so it takes
	// up to N passes for N-deep empty trees to clear out, but
	// that's Ok.
	for {
		_, sterr := os.Stat(g.cfg.SaveDir)
		if sterr!= nil {
			log.Printf("no savedir\n")
			time.Sleep(time.Second)
			continue
		}
		filepath.Walk(g.cfg.SaveDir, func(path string, info os.FileInfo, err error) error {
			if path == g.cfg.SaveDir {
				// skip
			} else if info.IsDir() {
				files, _ := ioutil.ReadDir(path)
				if len(files) == 0 {
					os.Remove(path)
				}
				// Note: I can imagine that a large tree full of empty
				// folders could consume a lot of IO, starving the writer
				// threads. A 10 millisecond sleep here could
				// alleviate that.
			} else if g.cfg.expireDuration != 0 {
				if info.ModTime().Add(g.cfg.expireDuration).Before(time.Now()) {
					os.Remove(path)
				}
			}
			return nil
		})
		time.Sleep(time.Minute)
	}
}


// Go slices share data with underlying arrays, but sometimes I want
// a new object.
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

	log.SetFlags(log.LstdFlags|log.Lmicroseconds)

	g.recent = list.New()

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

	err = json.Unmarshal(cfgText, &g.cfg)
	if err != nil {
		log.Printf("unmarshal error!\n")
		os.Exit(1)
	}

	g.cfg.expireDuration, err = time.ParseDuration(g.cfg.ExpireTime)
	if err != nil {
		log.Printf("ParseDuration error!\n")
		os.Exit(1)
	}

	scb := []byte("")
	if g.cfg.StartCode != "" {
		parts := strings.Split(g.cfg.StartCode, ".")
		for _, bstr := range parts {
			bval, err := strconv.ParseUint(bstr, 16, 32)
			if err == nil && bval <= 255 {
				scb = append(scb, byte(bval))
			} else {
				scb = []byte("")
				log.Printf("Invalid StartCode in json: %s\n", g.cfg.StartCode)
				break
			}
		}
	}

	if len(scb) > 0 {
		// json configured value
		g.cfg.startCodeBytes = scb
	} else {
		// Default. This is what my CTI program on RaspberryPI sends.
		g.cfg.startCodeBytes = []byte("\x00\x00\x00\x01\x27")
	}

	log.Printf("saveDir=%s\nHlsReceivePort=%d\nHlsRelayPort=%d\n"+
		"ExpireCommand=%s\nExpireTime=%s\nexpireDuration=%d\n",
		g.cfg.SaveDir,
		g.cfg.HlsReceivePort,
		g.cfg.HlsRelayPort,
		g.cfg.ExpireCommand,
		g.cfg.ExpireTime,
		g.cfg.expireDuration)

	receiver, err = net.ListenPacket("udp", fmt.Sprintf(":%d", g.cfg.HlsReceivePort))
	if err != nil {
		log.Printf("receive port listen error\n")
		os.Exit(1)
	}

	relay, err = net.ListenPacket("udp", fmt.Sprintf(":%d", g.cfg.HlsRelayPort))
	if err != nil {
		log.Printf("udp relay port listen error\n")
		os.Exit(1)
	}

	relayListen, err = net.Listen("tcp", fmt.Sprintf(":%d", g.cfg.HlsRelayPort))
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
				handlePacket(buffer[i:i+PKT_SIZE], g.cfg.SaveDir)
			}
			if (g.cfg.DebugInOut) {
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

	go http_server()

	go vacuum()

	// Main loop relays packets between goroutines.
	var data []byte
	var rconn net.Conn
	var rconnSet bool = false
	var tconn net.Conn
	var tconnSet bool = false
	for {
		select {
		case data = <- relayPacket:
			if rconnSet {
				if (g.cfg.DebugInOut) {
					log.Printf("%d bytes out, %#x\n", len(data),
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
