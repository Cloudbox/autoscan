package triggers

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
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

func WithLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		c := alice.New()
		c = c.Append(hlog.NewHandler(logger))
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
}

func WithAuth(username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Don't check for auth if username or password is missing.
		if username == "" || password == "" {
			return next
		}

		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			l := hlog.FromRequest(r)

			user, pass, ok := r.BasicAuth()
			if !ok || user == "" || pass == "" {
				// prompt auth dialog
				rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				rw.WriteHeader(http.StatusUnauthorized)
				return
			}

			// validate credentials
			if ok &&
				subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1 &&
				subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1 {
				l.Trace().Msg("Successful authentication")
				next.ServeHTTP(rw, r)
				return
			}

			// invalid credentials
			l.Warn().Msg("Invalid authentication")
			rw.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			rw.WriteHeader(http.StatusUnauthorized)
		})
	}
}
