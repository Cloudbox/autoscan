package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudbox/autoscan/targets/emby"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v2"

	"github.com/alecthomas/kong"
	"github.com/cloudbox/autoscan"
	"github.com/cloudbox/autoscan/processor"
	"github.com/cloudbox/autoscan/targets/plex"
	"github.com/cloudbox/autoscan/triggers"
	"github.com/cloudbox/autoscan/triggers/bernard"
	"github.com/cloudbox/autoscan/triggers/lidarr"
	"github.com/cloudbox/autoscan/triggers/radarr"
	"github.com/cloudbox/autoscan/triggers/sonarr"
	"github.com/natefinch/lumberjack"
)

type config struct {
	// General configuration
	Port       int           `yaml:"port"`
	MaxRetries int           `yaml:"retries"`
	MinimumAge time.Duration `yaml:"minimum-age"`
	Anchors    []string      `yaml:"anchors"`

	// Authentication for autoscan.HTTPTrigger
	Auth struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"authentication"`

	// autoscan.HTTPTrigger
	Triggers struct {
		Bernard []bernard.Config `yaml:"bernard"`
		Lidarr  []lidarr.Config  `yaml:"lidarr"`
		Radarr  []radarr.Config  `yaml:"radarr"`
		Sonarr  []sonarr.Config  `yaml:"sonarr"`
	} `yaml:"triggers"`

	// autoscan.Target
	Targets struct {
		Plex []plex.Config `yaml:"plex"`
		Emby []emby.Config `yaml:"emby"`
	} `yaml:"targets"`
}

var (
	// Release variables
	Version   string
	Timestamp string
	GitCommit string

	// CLI
	cli struct {
		globals

		// flags
		Config    string `type:"path" default:"${config_file}" env:"AUTOSCAN_CONFIG" help:"Config file path"`
		Database  string `type:"path" default:"${database_file}" env:"AUTOSCAN_DATABASE" help:"Database file path"`
		Log       string `type:"path" default:"${log_file}" env:"AUTOSCAN_LOG" help:"Log file path"`
		Verbosity int    `type:"counter" default:"0" short:"v" env:"AUTOSCAN_VERBOSITY" help:"Log level verbosity"`
	}
)

type globals struct {
	Version versionFlag `name:"version" help:"Print version information and quit"`
}

type versionFlag string

func (v versionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v versionFlag) IsBool() bool                         { return true }
func (v versionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
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

	logger := log.Output(io.MultiWriter(zerolog.ConsoleWriter{
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

	switch {
	case cli.Verbosity == 1:
		log.Logger = logger.Level(zerolog.DebugLevel)
	case cli.Verbosity > 1:
		log.Logger = logger.Level(zerolog.TraceLevel)
	default:
		log.Logger = logger.Level(zerolog.InfoLevel)
	}

	// run
	mux := http.NewServeMux()

	file, err := os.Open(cli.Config)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed opening config")
	}
	defer file.Close()

	// set default values
	c := config{
		MaxRetries: 5,
		MinimumAge: 300 * time.Second,
		Port:       3030,
	}

	decoder := yaml.NewDecoder(file)
	decoder.SetStrict(true)
	err = decoder.Decode(&c)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed decoding config")
	}

	proc, err := processor.New(processor.Config{
		Anchors:       c.Anchors,
		DatastorePath: cli.Database,
		MaxRetries:    c.MaxRetries,
		MinimumAge:    c.MinimumAge,
	})

	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed initialising processor")
	}

	log.Info().
		Stringer("min_age", c.MinimumAge).
		Int("max_retries", c.MaxRetries).
		Strs("anchors", c.Anchors).
		Msg("Initialised processor")

	// Set authentication. If none and running at least one webhook -> warn user.
	authHandler := triggers.WithAuth(c.Auth.Username, c.Auth.Password)
	if (c.Auth.Username == "" || c.Auth.Password == "") &&
		len(c.Triggers.Radarr)+len(c.Triggers.Sonarr) > 0 {
		log.Warn().Msg("Webhooks running without authentication")
	}

	// Daemon Triggers
	for _, t := range c.Triggers.Bernard {
		trigger, err := bernard.New(t)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("trigger", "bernard").
				Msg("Failed initialising trigger")
		}

		go trigger(proc.Add)
	}

	// HTTP Triggers
	for _, t := range c.Triggers.Lidarr {
		trigger, err := lidarr.New(t)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("trigger", t.Name).
				Msg("Failed initialising trigger")
		}

		logHandler := triggers.WithLogger(autoscan.GetLogger(t.Verbosity))
		mux.Handle("/triggers/"+t.Name, logHandler(authHandler(trigger(proc.Add))))
	}

	for _, t := range c.Triggers.Radarr {
		trigger, err := radarr.New(t)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("trigger", t.Name).
				Msg("Failed initialising trigger")
		}

		logHandler := triggers.WithLogger(autoscan.GetLogger(t.Verbosity))
		mux.Handle("/triggers/"+t.Name, logHandler(authHandler(trigger(proc.Add))))
	}

	for _, t := range c.Triggers.Sonarr {
		trigger, err := sonarr.New(t)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("trigger", t.Name).
				Msg("Failed initialising trigger")
		}

		logHandler := triggers.WithLogger(autoscan.GetLogger(t.Verbosity))
		mux.Handle("/triggers/"+t.Name, logHandler(authHandler(trigger(proc.Add))))
	}

	go func() {
		log.Info().Msgf("Starting server on port %d", c.Port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", c.Port), mux); err != nil {
			log.Fatal().
				Err(err).
				Msg("Failed starting web server")
		}
	}()

	log.Info().
		Int("bernard", len(c.Triggers.Bernard)).
		Int("lidarr", len(c.Triggers.Lidarr)).
		Int("sonarr", len(c.Triggers.Sonarr)).
		Int("radarr", len(c.Triggers.Radarr)).
		Msg("Initialised triggers")

	// targets
	targets := make([]autoscan.Target, 0)

	for _, t := range c.Targets.Plex {
		tp, err := plex.New(t)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("target", "plex").
				Str("target_url", t.URL).
				Msg("Failed initialising target")
		}

		targets = append(targets, tp)
	}

	for _, t := range c.Targets.Emby {
		tp, err := emby.New(t)
		if err != nil {
			log.Fatal().
				Err(err).
				Str("target", "emby").
				Str("target_url", t.URL).
				Msg("Failed initialising target")
		}

		targets = append(targets, tp)
	}

	log.Info().
		Int("plex", len(c.Targets.Plex)).
		Int("emby", len(c.Targets.Emby)).
		Msg("Initialised targets")

	// processor
	log.Info().Msg("Processor started")

	targetsAvailable := false

	for {
		if !targetsAvailable {
			err = proc.CheckAvailability(targets)
			switch {
			case err == nil:
				targetsAvailable = true
			case errors.Is(err, autoscan.ErrFatal):
				log.Error().
					Err(err).
					Msg("Fatal error occurred while checking target availability, processor stopped, triggers will continue...")

				// sleep indefinitely
				select {}
			default:
				log.Error().
					Err(err).
					Msg("Not all targets are available, retrying in 5 seconds...")

				time.Sleep(5 * time.Second)
				continue
			}
		}

		err = proc.Process(targets)
		if err != nil {
			switch {
			case errors.Is(err, autoscan.ErrNoScans):
				// No scans currently available, let's wait a couple of seconds
				log.Trace().Msg("Waiting 5 seconds as no scans are available")
				time.Sleep(5 * time.Second)

			case errors.Is(err, autoscan.ErrAnchorUnavailable):
				log.Error().
					Err(err).
					Msg("Not all anchor files are available, retrying in 5 seconds...")

				time.Sleep(5 * time.Second)

			case errors.Is(err, autoscan.ErrFatal):
				// fatal error occurred, processor must stop (however, triggers must not)
				log.Error().
					Err(err).
					Msg("Fatal error occurred while processing targets, processor stopped, triggers will continue...")

				// sleep indefinitely
				select {}

			case errors.Is(err, autoscan.ErrTargetUnavailable):
				targetsAvailable = false

			default:
				// unexpected error
				log.Fatal().
					Err(err).
					Msg("Failed processing targets")
			}
		}
	}
}
