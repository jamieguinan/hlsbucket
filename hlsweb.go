// Web interface functions.
package main

import (
	"net/http"
	"fmt"
	"log"
	"os"
	"time"
	"path/filepath"
)


func live_index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/x-mpegurl")

	duration := 3

	fmt.Fprintf(w,"#EXTM3U\n")
	fmt.Fprintf(w,"#EXT-X-TARGETDURATION:%d\n", duration)
	fmt.Fprintf(w,"#EXT-X-VERSION:3\n")
	fmt.Fprintf(w,"#EXT-X-MEDIA-SEQUENCE:%d\n", g.mediaSequence)

	g.recentLock.Lock()
	for e:=g.recent.Front(); e != nil; e = e.Next() {
		// Return everything up to but not including the
		// most recent file, which is still being written.
		if e != g.recent.Back() {
			fmt.Fprintf(w,"#EXTINF:%d, no desc\n", duration)
			// Replace savedir with "ts/".
			segment := fmt.Sprintf("%s", e.Value)
			segment = segment[len(g.cfg.SaveDir)+1:]
			fmt.Fprintf(w, "ts/%s\n", segment)
		}
	}

	g.recentLock.Unlock()
}

func live_meta(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/x-mpegurl")
	fmt.Fprintf(w, "#EXTM3U\n")
	fmt.Fprintf(w, "#EXT-X-STREAM-INF:PROGRAM-ID=1, BANDWIDTH=200000\n")
	fmt.Fprintf(w, "live_index.m3u8\n")
}


func http_server() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hlsbucket default handler %s\n", r.URL.Path)
	})

	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		// Either pick apart the url, or set up a few different
		// handlers.
		log.Printf("play request %s\n", r.URL.Path)
		values := r.URL.Query()
		log.Printf("%d values %s\n", len(values), values)
		if len(values) == 0 {
			log.Printf("live stream!\n")
			live_meta(w, r)
		}

	})

	http.HandleFunc("/live_index.m3u8", func(w http.ResponseWriter, r *http.Request) {
		live_index(w, r)
	})

        http.HandleFunc("/ts/", func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Clean(r.URL.Path)
		// Remove "/ts" prefix.
		p = p[len("/ts/"):]
		log.Printf("ts subpath handler %s\n", p)
		// Add back savedir
		tspath := g.cfg.SaveDir + "/" + p
		f, err := os.Open(tspath)
		if err != nil {
			// FIXME: 404
			log.Printf("%v\n", err)
			return
		}

		s, _ := os.Stat(tspath)
		if s != nil {
			log.Printf("age: %v\n", time.Now().Sub(s.ModTime()))
		}

		w.Header().Set("Content-Type", "video/MP2T")
		block := make([]byte, 32000)
		for {
			n, _ := f.Read(block)
			// log.Printf("%d bytes\n", n)
			if (n == 0) {
				break
			}
			w.Write(block[:n])
		}
	})

	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html>")
		fmt.Fprintf(w, `<meta name="viewport" content="width=device-width, initial-scale=1.0">`)
		fmt.Fprintf(w, `<body>live stream<br><video src="/play"><body></html>`)
	})


	if g.cfg.HlsListenPort == 0 {
		g.cfg.HlsListenPort = 8004
	}
	listenAddr := fmt.Sprintf(":%d", g.cfg.HlsListenPort)
	err := http.ListenAndServe(listenAddr, nil)
	log.Printf(err.Error())
}
