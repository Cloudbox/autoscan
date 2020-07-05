package main

import (
	"net/http"

	"github.com/cloudbox/autoscan"
	"github.com/cloudbox/autoscan/processor"
	"github.com/cloudbox/autoscan/triggers/radarr"
	"github.com/cloudbox/autoscan/triggers/sonarr"
)

func main() {
	scans := make(chan autoscan.Scan, 100)
	mux := http.NewServeMux()

	proc, err := processor.New("autoscan.db")
	if err != nil {
		panic(err)
	}

	radarrTrigger := radarr.New()
	sonarrTrigger := sonarr.New()

	mux.Handle("/triggers/radarr", radarrTrigger(scans))
	mux.Handle("/triggers/sonarr", sonarrTrigger(scans))
	go proc.ProcessTriggers(scans)

	http.ListenAndServe(":3000", mux)
}
