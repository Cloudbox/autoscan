package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/cloudbox/autoscan/processor"
	bernard_rs "github.com/cloudbox/autoscan/triggers/bernard-rs"
	"github.com/cloudbox/autoscan/triggers/lidarr"
	"github.com/cloudbox/autoscan/triggers/manual"
	"github.com/cloudbox/autoscan/triggers/radarr"
	"github.com/cloudbox/autoscan/triggers/sonarr"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func pattern(name string) string {
	return fmt.Sprintf("/%s", name)
}

func createCredentials(c config) map[string]string {
	creds := make(map[string]string)
	creds[c.Auth.Username] = c.Auth.Password
	return creds
}

func getRouter(c config, proc *processor.Processor) chi.Router {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Recoverer)

	// Logging-related middleware
	r.Use(hlog.NewHandler(log.Logger))
	r.Use(hlog.RequestIDHandler("id", "request-id"))
	r.Use(hlog.URLHandler("url"))
	r.Use(hlog.MethodHandler("method"))
	r.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Debug().
			Int("status", status).
			Dur("duration", duration).
			Msg("Request processed")
	}))

	// Health check
	r.Get("/health", healthHandler)

	// HTTP-Triggers
	r.Route("/triggers", func(r chi.Router) {
		// Use Basic Auth middleware if username and password are set.
		if c.Auth.Username != "" && c.Auth.Password != "" {
			r.Use(middleware.BasicAuth("Autoscan 1.x", createCredentials(c)))
		}

		// 2.0-style Bernard (Rust Edition) HTTP-trigger
		r.Route("/bernard", func(r chi.Router) {
			trigger, err := bernard_rs.New(c.Triggers.BernardRust)
			if err != nil {
				log.Fatal().Err(err).Str("trigger", "bernard-rs").Msg("Failed initialising trigger")
			}

			r.Post("/{drive}", trigger(proc.Add).ServeHTTP)
		})

		// Mixed-style Manual HTTP-trigger
		r.Route("/manual", func(r chi.Router) {
			trigger, err := manual.New(c.Triggers.Manual)
			if err != nil {
				log.Fatal().Err(err).Str("trigger", "manual").Msg("Failed initialising trigger")
			}

			r.HandleFunc("/", trigger(proc.Add).ServeHTTP)
		})

		// OLD-style HTTP-triggers. Can be converted to the /{trigger}/{id} format in a 2.0 release.
		for _, t := range c.Triggers.Lidarr {
			trigger, err := lidarr.New(t)
			if err != nil {
				log.Fatal().Err(err).Str("trigger", t.Name).Msg("Failed initialising trigger")
			}

			r.Post(pattern(t.Name), trigger(proc.Add).ServeHTTP)
		}

		for _, t := range c.Triggers.Radarr {
			trigger, err := radarr.New(t)
			if err != nil {
				log.Fatal().Err(err).Str("trigger", t.Name).Msg("Failed initialising trigger")
			}

			r.Post(pattern(t.Name), trigger(proc.Add).ServeHTTP)
		}

		for _, t := range c.Triggers.Sonarr {
			trigger, err := sonarr.New(t)
			if err != nil {
				log.Fatal().Err(err).Str("trigger", t.Name).Msg("Failed initialising trigger")
			}

			r.Post(pattern(t.Name), trigger(proc.Add).ServeHTTP)
		}
	})

	return r
}

// Other Handlers
func healthHandler(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusOK)
}
