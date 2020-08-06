package bernard

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/cloudbox/autoscan"
	lowe "github.com/m-rots/bernard"
	ds "github.com/m-rots/bernard/datastore"
	"github.com/m-rots/bernard/datastore/sqlite"
	"github.com/m-rots/stubbs"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

const (
	maxSyncRetries = 5
)

type Config struct {
	AccountPath   string             `yaml:"account"`
	CronSchedule  string             `yaml:"cron"`
	DatastorePath string             `yaml:"database"`
	Priority      int                `yaml:"priority"`
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

	limiter, err := getRateLimiter(auth.Email())
	if err != nil {
		return nil, fmt.Errorf("%v: %w", err, autoscan.ErrFatal)
	}

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
			priority:     c.Priority,
			drives:       drives,
			bernard:      bernard,
			store:        &bds{store},
			limiter:      limiter,
		}

		// start job(s)
		if err := d.StartAutoSync(); err != nil {
			l.Error().
				Err(err).
				Msg("Failed initialising cron jobs")
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
	priority     int
	drives       []drive
	bernard      *lowe.Bernard
	store        *bds
	log          zerolog.Logger
	limiter      *rateLimiter
}

type syncJob struct {
	log      zerolog.Logger
	attempts int
	errors   []error

	cron  *cron.Cron
	jobID cron.EntryID
	fn    func() error
}

func (s *syncJob) Run() {
	// increase attempt counter
	s.attempts++

	// run job
	err := s.fn()

	// handle job response
	switch {
	case err == nil:
		// job completed successfully
		s.attempts = 0
		s.errors = s.errors[:0]
		return

	case errors.Is(err, autoscan.ErrFatal):
		// fatal error occurred, we cannot recover from this safely
		s.log.Error().
			Err(err).
			Msg("Fatal error occurred while syncing drive, drive has been stopped...")

		s.cron.Remove(s.jobID)
		return

	case errors.Is(err, ds.ErrDataAnomaly):
		// data anomaly occurred, this can generally be recovered from, however, retry logic should be applied
		s.log.Trace().
			Err(err).
			Int("attempts", s.attempts).
			Msg("Data anomaly occurred while syncing drive")

		s.errors = append(s.errors, err)
	case err != nil:
		// an un-expected error occurred, this should be retryable with the same retry logic
		s.log.Warn().
			Err(err).
			Int("attempts", s.attempts).
			Msg("Unexpected error occurred while syncing drive")

		s.errors = append(s.errors, err)
	}

	// abort if max retries reached
	if s.attempts >= maxSyncRetries {
		s.log.Error().
			Errs("error", s.errors).
			Int("attempts", s.attempts).
			Msg("Consecutive errors occurred while syncing drive, drive has been stopped...")

		s.cron.Remove(s.jobID)
	}
}

func newSyncJob(c *cron.Cron, log zerolog.Logger, job func() error) *syncJob {
	return &syncJob{
		log:      log,
		attempts: 0,
		errors:   make([]error, 0),
		cron:     c,
		fn:       job,
	}
}

func (d daemon) StartAutoSync() error {
	c := cron.New()
	cl := newNoLogger()

	for _, drive := range d.drives {
		drive := drive
		fullSync := false
		l := d.withDriveLog(drive.ID)

		// full sync required?
		_, err := d.store.PageToken(drive.ID)
		switch {
		case errors.Is(err, ds.ErrFullSync):
			fullSync = true
		case err != nil:
			return fmt.Errorf("%v: failed determining if full sync required: %v: %w",
				drive.ID, err, autoscan.ErrFatal)
		}

		// create job
		job := newSyncJob(c, l, func() error {
			// acquire lock
			if err := d.limiter.Acquire(1); err != nil {
				return fmt.Errorf("%v: failed acquiring sync semaphore: %v: %w",
					drive.ID, err, autoscan.ErrFatal)
			}
			defer d.limiter.Release(1)

			// full sync
			if fullSync {
				l.Info().Msg("Starting full sync")
				start := time.Now()

				if err := d.bernard.FullSync(drive.ID); err != nil {
					return fmt.Errorf("%v: failed performing full sync: %w", drive.ID, err)
				}

				l.Info().Msgf("Finished full sync in %s", time.Since(start))
				fullSync = false
				return nil
			}

			// create partial sync
			dh, diff := d.store.NewDifferencesHook()
			ph := NewPostProcessBernardDiff(drive.ID, d.store, diff)
			ch, paths := NewPathsHook(drive.ID, d.store, diff, withOldChangedFilesToRemove(true))

			l.Trace().Msg("Running partial sync")
			start := time.Now()

			// do partial sync
			err := d.bernard.PartialSync(drive.ID, dh, ph, ch)
			if err != nil {
				return fmt.Errorf("%v: failed performing partial sync: %w", drive.ID, err)
			}

			l.Trace().
				Int("files_added", len(paths.AddedFiles)).
				Int("files_changed", len(paths.ChangedFiles)).
				Int("files_removed", len(paths.RemovedFiles)).
				Msgf("Partial sync finished in %s", time.Since(start))

			// translate paths to scan tasks
			scans := d.getScanTasks(&(drive), paths)

			// move scans to processor
			if len(scans) > 0 {
				l.Trace().
					Interface("scans", scans).
					Msg("Scans moving to processor")

				err := d.callback(scans...)
				if err != nil {
					return fmt.Errorf("%v: failed moving scans to processor: %v: %w",
						drive.ID, err, autoscan.ErrFatal)
				}
			}

			return nil
		})

		id, err := c.AddJob(d.cronSchedule, cron.NewChain(cron.SkipIfStillRunning(cl)).Then(job))
		if err != nil {
			return fmt.Errorf("%v: failed creating auto sync job for drive: %w", drive.ID, err)
		}

		job.jobID = id
	}

	c.Start()
	return nil
}

func (d daemon) getScanTasks(drive *drive, paths *Paths) []autoscan.Scan {
	pathMap := make(map[string]int)
	scanTasks := make([]autoscan.Scan, 0)

	for _, p := range paths.AddedFiles {
		// rewrite path
		rewritten := drive.Rewriter(p)

		// check if path already seen
		if _, ok := pathMap[rewritten]; ok {
			// already a scan task present
			continue
		} else {
			pathMap[rewritten] = 1
		}

		// add scan task
		dir, file := filepath.Split(rewritten)
		scanTasks = append(scanTasks, autoscan.Scan{
			Folder:   filepath.Clean(dir),
			File:     file,
			Priority: d.priority,
			Retries:  0,
			Removed:  false,
		})
	}

	for _, p := range paths.ChangedFiles {
		// rewrite path
		rewritten := drive.Rewriter(p)

		// check if path already seen
		if _, ok := pathMap[rewritten]; ok {
			// already a scan task present
			continue
		} else {
			pathMap[rewritten] = 1
		}

		// add scan task
		dir, file := filepath.Split(filepath.Clean(rewritten))
		scanTasks = append(scanTasks, autoscan.Scan{
			Folder:   filepath.Clean(dir),
			File:     file,
			Priority: d.priority,
			Retries:  0,
			Removed:  false,
		})
	}

	for _, p := range paths.RemovedFiles {
		// rewrite path
		rewritten := drive.Rewriter(p)

		// check if path already seen
		if _, ok := pathMap[rewritten]; ok {
			// already a scan task present
			continue
		} else {
			pathMap[rewritten] = 1
		}

		// add scan task
		dir, file := filepath.Split(rewritten)
		scanTasks = append(scanTasks, autoscan.Scan{
			Folder:   filepath.Clean(dir),
			File:     file,
			Priority: d.priority,
			Retries:  0,
			Removed:  true,
		})
	}

	return scanTasks
}

func (d daemon) withDriveLog(driveID string) zerolog.Logger {
	return d.log.With().Str("drive_id", driveID).Logger()
}
