package main

import (
	"net/http"

	"github.com/cloudbox/autoscan"
	"github.com/cloudbox/autoscan/processor"
	"github.com/cloudbox/autoscan/triggers/radarr"
)

func main() {
	scans := make(chan autoscan.Scan, 100)
	mux := http.NewServeMux()

	proc, err := processor.New("autoscan.db")
	if err != nil {
		panic(err)
	}

	radarrTrigger := radarr.New()

	mux.Handle("/triggers/radarr", radarrTrigger(scans))
	go proc.ProcessTriggers(scans)

	http.ListenAndServe(":3000", mux)
}
