package emby

import (
	"github.com/cloudbox/autoscan"
	"github.com/rs/zerolog"
)

type Config struct {
	URL       string           `yaml:"url"`
	Token     string           `yaml:"token"`
	Rewrite   autoscan.Rewrite `yaml:"rewrite"`
	Verbosity string           `yaml:"verbosity"`
}

type target struct {
	url       string
	token     string
	libraries []library

	log     zerolog.Logger
	rewrite autoscan.Rewriter
	api     *apiClient
}

func New(c Config) (*target, error) {
	rewriter, err := autoscan.NewRewriter(c.Rewrite)
	if err != nil {
		return nil, err
	}

	api := newAPIClient(c)

	libraries, err := api.Libraries()
	if err != nil {
		return nil, err
	}

	l := autoscan.GetLogger(c.Verbosity).With().
		Str("target", "emby").
		Str("url", c.URL).
		Logger()

	l.Debug().
		Interface("libraries", libraries).
		Msg("Retrieved libraries")

	return &target{
		url:       c.URL,
		token:     c.Token,
		libraries: libraries,

		log:     l,
		rewrite: rewriter,
		api:     api,
	}, nil
}

func (t target) Available() error {
	return t.api.Available()
}
