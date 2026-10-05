// Package janitor deletes finished jobs, and the images stored for them, once they are older than the retention
// period. Without it the database and the object store only ever grow.
package janitor

import (
	"context"
	"log/slog"
	"time"

	"app/internal/jobs"
	"app/internal/storage"
)

type Janitor struct {
	Store     jobs.Retention
	Storage   storage.Storage
	Retention time.Duration
	Interval  time.Duration
	BatchSize int
	Log       *slog.Logger
}

// Result summarises one sweep.
type Result struct {
	Jobs    int64
	Objects int
}

// RunOnce deletes expired jobs in batches until none are left, and reports what it removed. Stored files go first: if
// the process stops between the two steps the job row remains and the next sweep finishes the work.
func (j *Janitor) RunOnce(ctx context.Context, now time.Time) (Result, error) {
	var total Result
	batch := j.BatchSize
	if batch <= 0 {
		batch = 100
	}
	cutoff := now.Add(-j.Retention)
	for ctx.Err() == nil {
		expired, err := j.Store.ExpiredJobs(ctx, cutoff, batch)
		if err != nil {
			return total, err
		}
		if len(expired) == 0 {
			return total, nil
		}
		ids := make([]string, 0, len(expired))
		for _, e := range expired {
			for _, key := range e.Keys {
				if err := j.Storage.Delete(ctx, key); err != nil {
					return total, err
				}
				total.Objects++
			}
			ids = append(ids, e.ID)
		}
		n, err := j.Store.DeleteJobs(ctx, ids)
		if err != nil {
			return total, err
		}
		total.Jobs += n
		if len(expired) < batch {
			return total, nil
		}
	}
	return total, ctx.Err()
}

// Run sweeps every Interval until ctx ends. A failed sweep is logged and retried on the next tick.
func (j *Janitor) Run(ctx context.Context) {
	interval := j.Interval
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		res, err := j.RunOnce(ctx, time.Now())
		switch {
		case err != nil && ctx.Err() == nil:
			j.Log.Error("retention sweep failed", "err", err)
		case res.Jobs > 0:
			j.Log.Info("retention sweep", "jobs_deleted", res.Jobs, "files_deleted", res.Objects)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
