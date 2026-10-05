package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Claim is one image a worker has leased.
type Claim struct {
	ItemID    string
	JobID     string
	Position  int
	SourceURL string
	Attempts  int // including this one
}

// WebhookClaim is one completed job whose callback is due.
type WebhookClaim struct {
	JobID    string
	URL      string
	Status   string
	Attempts int
	Counts   Counts
}

type Counts struct {
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Total     int `json:"total"`
}

// Queue is what the worker needs; PGStore implements it with FOR UPDATE SKIP LOCKED so any
// number of workers (or replicas) can run against one database.
type Queue interface {
	ClaimItem(ctx context.Context, lease time.Duration) (*Claim, error)
	CompleteItem(ctx context.Context, c Claim, outputKey string, bytesIn, bytesOut int64) error
	FailItem(ctx context.Context, c Claim, reason string, permanent bool, maxAttempts int) error
	ClaimWebhook(ctx context.Context, lease time.Duration, maxAttempts int) (*WebhookClaim, error)
	FinishWebhook(ctx context.Context, jobID string, delivered bool, retryIn time.Duration) error
}

// ClaimItem leases the next due item. An item still "processing" after the lease (a crashed
// worker) becomes claimable again. It returns (nil, nil) when nothing is due.
func (s *PGStore) ClaimItem(ctx context.Context, lease time.Duration) (*Claim, error) {
	var c Claim
	err := s.pool.QueryRow(ctx, `
		UPDATE job_items SET status = 'processing', attempts = attempts + 1, locked_at = now()
		WHERE id = (
			SELECT id FROM job_items
			WHERE (status = 'queued' AND next_attempt_at <= now())
			   OR (status = 'processing' AND locked_at < now() - make_interval(secs => $1))
			ORDER BY next_attempt_at, position
			FOR UPDATE SKIP LOCKED
			LIMIT 1)
		RETURNING id, job_id, position, source_url, attempts`, lease.Seconds()).
		Scan(&c.ItemID, &c.JobID, &c.Position, &c.SourceURL, &c.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_, err = s.pool.Exec(ctx, `UPDATE jobs SET status = 'processing', updated_at = now() WHERE id = $1 AND status = 'queued'`, c.JobID)
	return &c, err
}

func (s *PGStore) CompleteItem(ctx context.Context, c Claim, outputKey string, bytesIn, bytesOut int64) error {
	return s.finishItem(ctx, c, `UPDATE job_items SET status = 'completed', output_key = $2, bytes_in = $3, bytes_out = $4,
		error = NULL, locked_at = NULL WHERE id = $1 AND status = 'processing'`, c.ItemID, outputKey, bytesIn, bytesOut)
}

// FailItem records a failure. A transient failure is retried with exponential backoff
// until maxAttempts; a permanent one fails the item at once.
func (s *PGStore) FailItem(ctx context.Context, c Claim, reason string, permanent bool, maxAttempts int) error {
	if len(reason) > 500 {
		reason = reason[:500]
	}
	if permanent || c.Attempts >= maxAttempts {
		return s.finishItem(ctx, c, `UPDATE job_items SET status = 'failed', error = $2, locked_at = NULL
			WHERE id = $1 AND status = 'processing'`, c.ItemID, reason)
	}
	backoff := time.Duration(1<<min(c.Attempts, 8)) * 5 * time.Second
	return s.finishItem(ctx, c, `UPDATE job_items SET status = 'queued', error = $2, locked_at = NULL,
		next_attempt_at = now() + make_interval(secs => $3) WHERE id = $1 AND status = 'processing'`,
		c.ItemID, reason, backoff.Seconds())
}

// finishItem updates the item and recomputes the job in one transaction. The job row is locked
// first so concurrent items of one job cannot both miss that the other finished.
func (s *PGStore) finishItem(ctx context.Context, c Claim, sql string, args ...any) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, `SELECT 1 FROM jobs WHERE id = $1 FOR UPDATE`, c.JobID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE jobs j SET updated_at = now(),
			status = (CASE WHEN c.pending > 0 THEN 'processing'
			               WHEN c.failed = 0 THEN 'completed'
			               WHEN c.completed = 0 THEN 'failed'
			               ELSE 'partial' END)::job_status,
			webhook_pending = (j.webhook_url IS NOT NULL AND c.pending = 0 AND j.webhook_delivered_at IS NULL),
			webhook_next_at = CASE WHEN c.pending = 0 THEN now() ELSE j.webhook_next_at END
		FROM (SELECT count(*) FILTER (WHERE status IN ('queued', 'processing')) AS pending,
		             count(*) FILTER (WHERE status = 'failed') AS failed,
		             count(*) FILTER (WHERE status = 'completed') AS completed
		      FROM job_items WHERE job_id = $1) c
		WHERE j.id = $1`, c.JobID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PGStore) ClaimWebhook(ctx context.Context, lease time.Duration, maxAttempts int) (*WebhookClaim, error) {
	var w WebhookClaim
	err := s.pool.QueryRow(ctx, `
		UPDATE jobs SET webhook_attempts = webhook_attempts + 1, webhook_next_at = now() + make_interval(secs => $1)
		WHERE id = (SELECT id FROM jobs
		            WHERE webhook_pending AND webhook_next_at <= now() AND webhook_attempts < $2
		            ORDER BY webhook_next_at FOR UPDATE SKIP LOCKED LIMIT 1)
		RETURNING id, webhook_url, status::text, webhook_attempts`, lease.Seconds(), maxAttempts).
		Scan(&w.JobID, &w.URL, &w.Status, &w.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	err = s.pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE status = 'completed'), count(*) FILTER (WHERE status = 'failed')
		FROM job_items WHERE job_id = $1`, w.JobID).Scan(&w.Counts.Total, &w.Counts.Completed, &w.Counts.Failed)
	return &w, err
}

func (s *PGStore) FinishWebhook(ctx context.Context, jobID string, delivered bool, retryIn time.Duration) error {
	if delivered {
		_, err := s.pool.Exec(ctx, `UPDATE jobs SET webhook_pending = false, webhook_delivered_at = now() WHERE id = $1`, jobID)
		return err
	}
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET webhook_next_at = now() + make_interval(secs => $2) WHERE id = $1`, jobID, retryIn.Seconds())
	return err
}

// OutputKey returns the storage key for one finished item owned by owner.
func (s *PGStore) OutputKey(ctx context.Context, owner, jobID string, position int) (string, error) {
	var key *string
	err := s.pool.QueryRow(ctx, `
		SELECT i.output_key FROM job_items i JOIN jobs j ON j.id = i.job_id
		WHERE j.id = $1 AND j.owner = $2 AND i.position = $3 AND i.status = 'completed'`, jobID, owner, position).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && key == nil) {
		return "", ErrNotFound
	}
	return derefOrEmpty(key), err
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
