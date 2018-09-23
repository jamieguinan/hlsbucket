package main

import (
	"os"
	"fmt"
	"path"
	"encoding/json"
	"io/ioutil"
	"net"
)

var CFGPATH = "hlsbucket.json"

type Config struct {
	SaveDir string
	ExpireCommand string
	HlsReceivePort int
	HlsRelayPort int
}

func main() {
	var err error
	var cfgText []byte
	var cfg Config
	var receiver net.PacketConn
	var relay net.PacketConn

	if (len(os.Args) != 1) {
		fmt.Fprintf(os.Stderr, "%s: specify options in %s\n", path.Base(os.Args[0]), CFGPATH);
		os.Exit(1)
	}

	cfgText, err = ioutil.ReadFile(CFGPATH)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: could not load config file %s\n", path.Base(os.Args[0]), CFGPATH);
		os.Exit(1)
	}

	err = json.Unmarshal(cfgText, &cfg)
	if err != nil {
		fmt.Printf("unmarshal error!\n")
		os.Exit(1)
	}

	fmt.Printf("saveDir=%s\nexpireCommand=%s\nhlsReceivePort=%d\nhlsRelayPort=%d\n",
		cfg.SaveDir,
		cfg.ExpireCommand,
		cfg.HlsReceivePort,
		cfg.HlsRelayPort)

	receiver, err = net.ListenPacket("udp", fmt.Sprintf(":%d", cfg.HlsReceivePort))
	if err != nil {
		fmt.Printf("receive port listen error\n")
		os.Exit(1)
	}

	relay, err = net.ListenPacket("udp", fmt.Sprintf(":%d", cfg.HlsRelayPort))
	if err != nil {
		fmt.Printf("relay port listen error\n")
		os.Exit(1)
	}

	go func () {
		buffer := make([]byte, 1500)
		for {
			// Q: is "err" here the same as the one above,
			// or does the closure make a new copy?
			var n int
			// int, Addr, error
			n, _, err = receiver.ReadFrom(buffer)
			// Q: if n is <= 0, will err also be set?
			fmt.Fprintf(os.Stderr, "%d bytes received\n", n)
			if err != nil {
				continue
			}
		}
	}()

	go func () {
		buffer := make([]byte, 1500)
		for {
			// Q: is "err" here the same as the one above,
			// or does the closure make a new copy?
			var n int
			// int, Addr, error
			n, _, err = relay.ReadFrom(buffer)
			// Q: if n is <= 0, will err also be set?
			fmt.Fprintf(os.Stderr, "%d bytes received\n", n)
			if err != nil {
				continue
			}
		}
	}()

	select {
		// How do I do select() on the receive and relay ports?
	}

}
