package bernard

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cloudbox/autoscan"
	lowe "github.com/m-rots/bernard"
	ds "github.com/m-rots/bernard/datastore"
	"github.com/m-rots/bernard/datastore/sqlite"
	"github.com/m-rots/stubbs"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type Config struct {
	CronSchedule  string `yaml:"cron"`
	AccountPath   string `yaml:"account"`
	DatastorePath string `yaml:"database"`
	DriveID       string `yaml:"id"`
	Verbosity     string `yaml:"verbosity"`
}

func New(c Config) (autoscan.Trigger, error) {
	l := autoscan.GetLogger(c.Verbosity).With().
		Str("trigger", "bernard").
		Str("drive_id", c.DriveID).
		Logger()

	const scope = "https://www.googleapis.com/auth/drive.readonly"
	auth, err := stubbs.FromFile(c.AccountPath, []string{scope}, 3600)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	store, err := sqlite.New(c.DatastorePath)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

	bernard := lowe.New(auth, store)

	trigger := func(callback autoscan.ProcessorFunc) {
		d := daemon{
			log:          l,
			callback:     callback,
			cronSchedule: c.CronSchedule,
			driveID:      c.DriveID,
			bernard:      bernard,
			store:        store,
		}

		if err := d.InitialSync(); err != nil {
			l.Error().Err(err).Msg("Initial sync failed")
			return
		}

		// start job
		if err := d.StartAutoSync(); err != nil {
			l.Error().Err(err).Msg("Cron job could not be created")
			return
		}
	}

	return trigger, nil
}

type daemon struct {
	callback     autoscan.ProcessorFunc
	cronSchedule string
	driveID      string
	bernard      *lowe.Bernard
	store        *sqlite.Datastore
	log          zerolog.Logger
}

func (d daemon) InitialSync() error {
	_, err := d.store.PageToken(d.driveID)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ds.ErrFullSync):
		d.log.Info().Msg("Starting full sync")
		if err := d.bernard.FullSync(d.driveID); err != nil {
			return err
		}
		d.log.Info().Msg("Finished full sync")
	default:
		return err
	}

	return nil
}

type syncJob struct {
	job       func()
	mtx       sync.Mutex
	isRunning bool
}

func (s *syncJob) Do() {
	if s.isRunning {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.isRunning = true
	s.job()
	s.isRunning = false
}

func newSyncJob(job func()) *syncJob {
	return &syncJob{
		job: job,
	}
}

func (d daemon) StartAutoSync() error {
	c := cron.New()

	job := newSyncJob(func() {
		d.log.Trace().Msg("Running partial sync")
		err := d.bernard.PartialSync(d.driveID)
		if err != nil {
			d.log.Error().Err(err).Msg("Partial sync failed")
			c.Stop()
			return
		}
		d.log.Trace().Msg("Partial sync complete")
	})

	_, err := c.AddFunc(d.cronSchedule, job.Do)
	if err != nil {
		return err
	}

	c.Start()
	return nil
}
