package autoscan

import (
	"github.com/rs/zerolog"

	"github.com/cloudbox/autoscan"
)

type Config struct {
	URL       string             `yaml:"url"`
	User      string             `yaml:"username"`
	Pass      string             `yaml:"password"`
	Rewrite   []autoscan.Rewrite `yaml:"rewrite"`
	Verbosity string             `yaml:"verbosity"`
}

type target struct {
	url  string
	user string
	pass string

	log     zerolog.Logger
	rewrite autoscan.Rewriter
	api     apiClient
}

func New(c Config) (autoscan.Target, error) {
	l := autoscan.GetLogger(c.Verbosity).With().
		Str("target", "autoscan").
		Str("url", c.URL).Logger()

	rewriter, err := autoscan.NewRewriter(c.Rewrite)
	if err != nil {
		return nil, err
	}

	return &target{
		url:  c.URL,
		user: c.User,
		pass: c.Pass,

		log:     l,
		rewrite: rewriter,
		api:     newAPIClient(c.URL, c.User, c.Pass, l),
	}, nil
}

func (t target) Scan(scan autoscan.Scan) error {
	scanFolder := t.rewrite(scan.Folder)

	// send scan request
	l := t.log.With().
		Str("path", scanFolder).
		Logger()

	l.Trace().Msg("Sending scan request")

	if err := t.api.Scan(scanFolder); err != nil {
		return err
	}

	l.Info().Msg("Scan moved to target")
	return nil
}

func (t target) Available() error {
	return t.api.Available()
}
