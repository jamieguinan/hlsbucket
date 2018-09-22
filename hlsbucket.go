package main

import (
	"os"
	"fmt"
	"path"
	"encoding/json"
	"io/ioutil"
)

var CFGPATH = "hlsbucket.json"

type Config struct {
	SaveDir string
	ExpireCommand string
	HlsReceivePort int
	HlsRelayPort int
}

func main() {
	//var saveDir = ""
	var err error
	var cfgText []byte
	var cfg Config

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
		return
	}

	fmt.Printf("saveDir=%s\nexpireCommand=%s\nhlsReceivePort=%d\nhlsRelayPort=%d\n",
		cfg.SaveDir,
		cfg.ExpireCommand,
		cfg.HlsReceivePort,
		cfg.HlsRelayPort)

}
