package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/cloudbox/autoscan"
	"github.com/cloudbox/autoscan/processor"
	"github.com/cloudbox/autoscan/triggers/radarr"
	"github.com/cloudbox/autoscan/triggers/sonarr"
	"gopkg.in/yaml.v2"
)

type config struct {
	Triggers struct {
		Radarr []radarr.Config `yaml:"radarr"`
		Sonarr []sonarr.Config `yaml:"sonarr"`
	} `yaml:"triggers"`
}

var (
	// Release variables
	Version   string
	Timestamp string
	GitCommit string
)

func test() (string, error) {
	return "test", errors.New("hmmm")
}

func main() {
	// TODO: show this via a version command instead?
	fmt.Printf("Version: %s (%s@%s)\n", Version, GitCommit, Timestamp)

	s, err := test()
	fmt.Println(s)

	scans := make(chan autoscan.Scan, 100)
	mux := http.NewServeMux()

	proc, err := processor.New("autoscan.db")
	if err != nil {
		panic(err)
	}

	file, err := os.Open("./config.yml")
	if err != nil {
		panic(err)
	}

	c := new(config)
	decoder := yaml.NewDecoder(file)
	decoder.SetStrict(true)
	err = decoder.Decode(c)
	if err != nil {
		panic(err)
	}

	for _, t := range c.Triggers.Radarr {
		trigger := radarr.New(t)
		mux.Handle("/triggers/"+t.Name, trigger(scans))
	}

	for _, t := range c.Triggers.Sonarr {
		trigger := sonarr.New(t)
		mux.Handle("/triggers/"+t.Name, trigger(scans))
	}

	go proc.ProcessTriggers(scans)

	http.ListenAndServe(":3000", mux)
}
