package triggers

import (
	"net/http"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

func detailHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := zerolog.Ctx(r.Context())
		logger.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.
				Str("method", r.Method).
				Str("url", r.URL.Path)
		})

		next.ServeHTTP(w, r)
	})
}

func WithLogger(next http.Handler) http.Handler {
	c := alice.New()
	c = c.Append(hlog.NewHandler(log.Logger))
	c = c.Append(hlog.RequestIDHandler("request_id", "Request-Id"))
	c = c.Append(detailHandler)

	c = c.Append(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Debug().
			Int("status", status).
			Dur("duration", duration).
			Msg("Request processed")
	}))

	return c.Then(next)
}
