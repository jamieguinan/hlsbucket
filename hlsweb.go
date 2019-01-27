// Web interface functions.
package main

import (
	"net/http"
	"fmt"
	"log"
)

func http_server() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Either pick apart the url, or set up a few different
		// handlers.
		fmt.Fprintf(w, "hlsbucket request %s\n", r.URL.Path)
	})

	err := http.ListenAndServe(":8004", nil)
	log.Printf(err.Error())
}
