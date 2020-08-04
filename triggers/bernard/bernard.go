package bernard

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cloudbox/autoscan"
	lowe "github.com/m-rots/bernard"
	ds "github.com/m-rots/bernard/datastore"
	"github.com/m-rots/bernard/datastore/sqlite"
	"github.com/m-rots/stubbs"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type Config struct {
	AccountPath   string             `yaml:"account"`
	CronSchedule  string             `yaml:"cron"`
	DatastorePath string             `yaml:"database"`
	Verbosity     string             `yaml:"verbosity"`
	Rewrite       []autoscan.Rewrite `yaml:"rewrite"`
	Drives        []struct {
		ID      string             `yaml:"id"`
		Rewrite []autoscan.Rewrite `yaml:"rewrite"`
	} `yaml:"drives"`
}

func New(c Config) (autoscan.Trigger, error) {
	l := autoscan.GetLogger(c.Verbosity).With().
		Str("trigger", "bernard").
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

	limiter := getRateLimiter(c.AccountPath)

	bernard := lowe.New(auth, store,
		lowe.WithPreRequestHook(limiter.Wait),
		lowe.WithSafeSleep(120*time.Second))

	var drives []drive
	for _, d := range c.Drives {
		rewriter, err := autoscan.NewRewriter(append(d.Rewrite, c.Rewrite...))
		if err != nil {
			return nil, err
		}

		drives = append(drives, drive{
			ID:       d.ID,
			Rewriter: rewriter,
		})
	}

	trigger := func(callback autoscan.ProcessorFunc) {
		d := daemon{
			log:          l,
			callback:     callback,
			cronSchedule: c.CronSchedule,
			drives:       drives,
			bernard:      bernard,
			store:        &bds{store},
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

type drive struct {
	ID       string
	Rewriter autoscan.Rewriter
}

type daemon struct {
	callback     autoscan.ProcessorFunc
	cronSchedule string
	drives       []drive
	bernard      *lowe.Bernard
	store        *bds
	log          zerolog.Logger
}

func (d daemon) InitialSync() error {
	for _, drive := range d.drives {
		l := d.withDriveLog(drive.ID)

		_, err := d.store.PageToken(drive.ID)
		switch {
		case errors.Is(err, ds.ErrFullSync):
			l.Info().Msg("Starting full sync")
			start := time.Now()

			if err := d.bernard.FullSync(drive.ID); err != nil {
				return err
			}
			l.Info().Msgf("Finished full sync in %s", time.Since(start))
		case err != nil:
			return err
		}
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
		for _, drive := range d.drives {
			l := d.withDriveLog(drive.ID)

			dh, diff := d.store.NewDifferencesHook()
			ph := NewPostProcessBernardDiff(drive.ID, d.store, diff)
			ch, paths := NewPathsHook(drive.ID, d.store, diff)

			l.Trace().Msg("Running partial sync")
			start := time.Now()

			err := d.bernard.PartialSync(drive.ID, dh, ph, ch)
			if err != nil {
				d.log.Error().
					Err(err).
					Msg("Partial sync failed")
				c.Stop()
				return
			}

			l.Trace().
				Int("added_files", len(paths.AddedFiles)).
				Int("changed_files", len(paths.ChangedFiles)).
				Int("removed_files", len(paths.RemovedFiles)).
				Msgf("Partial sync finished in %s", time.Since(start))

			// do something with the results

		}
	})

	_, err := c.AddFunc(d.cronSchedule, job.Do)
	if err != nil {
		return err
	}

	c.Start()
	return nil
}

func (d daemon) withDriveLog(driveID string) zerolog.Logger {
	return d.log.With().Str("drive_id", driveID).Logger()
}
