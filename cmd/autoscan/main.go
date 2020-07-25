package main

import (
	"fmt"
	"github.com/cloudbox/autoscan/targets/plex"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alecthomas/kong"
	"github.com/cloudbox/autoscan"
	"github.com/cloudbox/autoscan/processor"
	"github.com/cloudbox/autoscan/triggers/radarr"
	"github.com/cloudbox/autoscan/triggers/sonarr"
	"github.com/natefinch/lumberjack"
)

type config struct {
	Triggers struct {
		Radarr []radarr.Config `yaml:"radarr"`
		Sonarr []sonarr.Config `yaml:"sonarr"`
	} `yaml:"triggers"`
	Targets struct {
		Plex []plex.Config `yaml:"plex"`
	} `yaml:"targets"`
}

var (
	// Release variables
	Version   string
	Timestamp string
	GitCommit string

	// CLI
	cli struct {
		Globals

		// flags
		Config    string `type:"path" default:"${config_file}" env:"AUTOSCAN_CONFIG" help:"Config file path"`
		Database  string `type:"path" default:"${database_file}" env:"AUTOSCAN_DATABASE" help:"Database file path"`
		Log       string `type:"path" default:"${log_file}" env:"AUTOSCAN_LOG" help:"Log file path"`
		Verbosity int    `type:"counter" default:"0" short:"v" env:"AUTOSCAN_VERBOSITY" help:"Log level verbosity"`
	}
)

type Globals struct {
	Version VersionFlag `name:"version" help:"Print version information and quit"`
}

type VersionFlag string

func (v VersionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                         { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Println(vars["version"])
	app.Exit(0)
	return nil
}

/* Version */

func main() {
	// parse cli
	ctx := kong.Parse(&cli,
		kong.Name("autoscan"),
		kong.Description("Scan media into target media servers"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Summary: true,
			Compact: true,
		}),
		kong.Vars{
			"version":       fmt.Sprintf("%s (%s@%s)", Version, GitCommit, Timestamp),
			"config_file":   filepath.Join(defaultConfigPath(), "config.yml"),
			"log_file":      filepath.Join(defaultConfigPath(), "activity.log"),
			"database_file": filepath.Join(defaultConfigPath(), "autoscan.db"),
		},
	)

	if err := ctx.Validate(); err != nil {
		fmt.Println("Failed parsing cli:", err)
		os.Exit(1)
	}

	// logging
	switch {
	case cli.Verbosity == 1:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case cli.Verbosity > 1:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Logger = log.Output(io.MultiWriter(zerolog.ConsoleWriter{
		Out: os.Stderr,
	}, zerolog.ConsoleWriter{
		Out: &lumberjack.Logger{
			Filename:   cli.Log,
			MaxSize:    5,
			MaxAge:     14,
			MaxBackups: 5,
		},
		NoColor: true,
	}))

	// run
	mux := http.NewServeMux()

	proc, err := processor.New(cli.Database)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed initialising processor")
	}

	file, err := os.Open(cli.Config)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed opening config")
	}
	defer file.Close()

	c := new(config)
	decoder := yaml.NewDecoder(file)
	decoder.SetStrict(true)
	err = decoder.Decode(c)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed decoding config")
	}

	// triggers
	for _, t := range c.Triggers.Radarr {
		trigger, err := radarr.New(t)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("trigger", t.Name).
				Msg("Failed initialising trigger")
		}
		mux.Handle("/triggers/"+t.Name, trigger(proc.Add))
	}

	for _, t := range c.Triggers.Sonarr {
		trigger, err := sonarr.New(t)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("trigger", t.Name).
				Msg("Failed initialising trigger")
		}
		mux.Handle("/triggers/"+t.Name, trigger(proc.Add))
	}

	go func() {
		if err := http.ListenAndServe(":3000", mux); err != nil {
			log.Fatal().
				Err(err).
				Msg("Failed starting web server")
		}
	}()

	log.Info().
		Int("sonarr", len(c.Triggers.Sonarr)).
		Int("radarr", len(c.Triggers.Radarr)).
		Msg("Initialised triggers")

	// targets
	targets := make([]autoscan.Target, 0)

	if len(c.Targets.Plex) > 0 {
		for _, t := range c.Targets.Plex {
			tp, err := plex.New(t)
			if err != nil {
				log.Fatal().
					Err(err).
					Str("target", "plex").
					Str("url", t.URL).
					Msg("Failed initialising target")
			}

			targets = append(targets, tp)
		}
	}

	log.Info().
		Int("plex", len(c.Targets.Plex)).
		Msg("Initialised targets")

	// processor
	log.Info().
		Msg("Processor started")

	for {
		err = proc.Process(targets)
		if err != nil {
			log.Fatal().
				Err(err).
				Msg("Failed processing targets")
		}

		time.Sleep(1 * time.Second)
	}
}
