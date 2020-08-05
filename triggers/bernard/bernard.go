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
	cron  *cron.Cron
	log   zerolog.Logger
	jobID cron.EntryID
	fn    func() error
}

func (s syncJob) Run() {
	_ = s.fn()
	// todo: error is fatal ? stop job
	// todo: error is not fatal but data anomaly ? increase try counter (stop after N tries)
}

func newSyncJob(c *cron.Cron, log zerolog.Logger, job func() error) *syncJob {
	return &syncJob{
		cron: c,
		log:  log,
		fn:   job,
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
					Msg("Scan tasks to be moved to processor")
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
