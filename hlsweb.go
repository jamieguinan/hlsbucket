// Web interface functions.
package main

import (
	"net/http"
	"fmt"
	"log"
//	"net/url"
)


func live_m3u8(w http.ResponseWriter, r *http.Request) {
	//  "#EXTM3U\n"
	//  "#EXT-X-TARGETDURATION:%d\n", priv->duration
	//  "#EXT-X-VERSION:3\n"
	//  "#EXT-X-MEDIA-SEQUENCE:%d\n", priv->media_sequence
	//  "#EXTINF:%d,\n"
	//  "file1.ts\n"
	//  "#EXTINF:%d,\n"
	//  "file2.ts\n"
	//  "#EXTINF:%d,\n"
	//  "file3.ts\n"

	g.recentLock.Lock()
	for e:=g.recent.Front(); e != nil; e = e.Next() {
		// Return everything up to but not including the
		// most recent file, which is still being written.
		if e != g.recent.Back() {
			fmt.Fprintf(w, "%s\n", e.Value)
		}
	}

	g.recentLock.Unlock()

}


func http_server() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hlsbucket default handler %s\n", r.URL.Path)
	})

	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		// Either pick apart the url, or set up a few different
		// handlers.
		fmt.Fprintf(w, "play request %s\n", r.URL.Path)
		values := r.URL.Query()
		fmt.Fprintf(w, "%d values %s\n", len(values), values)
		if len(values) == 0 {
			fmt.Fprintf(w, "live stream!\n")
			live_m3u8(w, r)
		}

	})

	err := http.ListenAndServe(":8004", nil)
	log.Printf(err.Error())
}
