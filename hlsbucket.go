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
	// "strconv"
)

var CFGPATH = "hlsbucket.json"

type Config struct {
	SaveDir string
	ExpireCommand string
	HlsReceivePort int
	HlsRelayPort int
}

func handlePacket(buffer []byte, n int) {
	// log.Printf("handlePacket\n")
}

func main() {
	var err error
	var cfgText []byte
	var cfg Config
	var receiver net.PacketConn
	var relay net.PacketConn

	setRelay := make(chan net.Conn)
	relayPacket := make(chan []byte)

	if (len(os.Args) != 1) {
		log.Printf("%s: specify options in %s\n", path.Base(os.Args[0]), CFGPATH);
		os.Exit(1)
	}

	cfgText, err = ioutil.ReadFile(CFGPATH)
	if err != nil {
		log.Printf("%s: could not load config file %s\n", path.Base(os.Args[0]), CFGPATH);
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
		log.Printf("relay port listen error\n")
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
			handlePacket(buffer, n)
			relayPacket <- buffer[0:n]
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

	var data []byte
	var rconn net.Conn
	var rconnSet bool = false
	for {
		select {
		case data = <- relayPacket:
			if rconnSet {
				log.Printf("%d\n", len(data))
				rconn.Write(data)
			}
		case rconn = <- setRelay:
			log.Printf("setting relay to %v\n", rconn.RemoteAddr().String())
			rconnSet = true
		}
	}
}
