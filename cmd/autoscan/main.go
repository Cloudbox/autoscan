package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/cloudbox/autoscan"
	"github.com/cloudbox/autoscan/processor"
	"github.com/cloudbox/autoscan/targets/test"
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

func main() {
	// TODO: show this via a version command instead?
	fmt.Printf("Version: %s (%s@%s)\n", Version, GitCommit, Timestamp)

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
		trigger, err := radarr.New(t)
		if err != nil {
			panic(err)
		}
		mux.Handle("/triggers/"+t.Name, trigger(proc.Add))
	}

	for _, t := range c.Triggers.Sonarr {
		trigger, err := sonarr.New(t)
		if err != nil {
			panic(err)
		}
		mux.Handle("/triggers/"+t.Name, trigger(proc.Add))
	}

	go func() {
		if err := http.ListenAndServe(":3000", mux); err != nil {
			panic(err)
		}
	}()

	targets := []autoscan.Target{
		test.New(),
	}

	for {
		proc.Process(targets)
		time.Sleep(1 * time.Second)
	}
}
