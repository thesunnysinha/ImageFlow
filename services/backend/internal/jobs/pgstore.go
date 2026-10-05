package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Create inserts the job and all items in one transaction using a single batch.
func (s *PGStore) Create(ctx context.Context, owner string, webhookURL *string, sourceURLs []string) (Job, error) {
	id := uuid.NewString()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Job{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`INSERT INTO jobs (id, owner, webhook_url) VALUES ($1,$2,$3)`, id, owner, webhookURL); err != nil {
		return Job{}, err
	}
	batch := &pgx.Batch{}
	for i, u := range sourceURLs {
		batch.Queue(`INSERT INTO job_items (id, job_id, position, source_url) VALUES ($1,$2,$3,$4)`,
			uuid.NewString(), id, i, u)
	}
	br := tx.SendBatch(ctx, batch)
	for range sourceURLs {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return Job{}, err
		}
	}
	if err := br.Close(); err != nil {
		return Job{}, err
	}
	// TODO(phase 2): enqueue River jobs here, inside the same transaction.
	if err := tx.Commit(ctx); err != nil {
		return Job{}, err
	}
	return s.Get(ctx, owner, id)
}

func (s *PGStore) Get(ctx context.Context, owner, id string) (Job, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Job{}, ErrNotFound
	}
	var j Job
	err := s.pool.QueryRow(ctx,
		`SELECT id, status::text, webhook_url, created_at, updated_at FROM jobs WHERE id=$1 AND owner=$2`,
		id, owner).Scan(&j.ID, &j.Status, &j.WebhookURL, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, err
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, position, source_url, status::text, output_key, bytes_in, bytes_out, error
		 FROM job_items WHERE job_id=$1 ORDER BY position`, id)
	if err != nil {
		return Job{}, err
	}
	defer rows.Close()
	j.Items = []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Position, &it.SourceURL, &it.Status, &it.OutputKey, &it.BytesIn, &it.BytesOut, &it.Error); err != nil {
			return Job{}, err
		}
		j.Items = append(j.Items, it)
	}
	return j, rows.Err()
}

func (s *PGStore) List(ctx context.Context, owner string, limit int, before time.Time) ([]Summary, error) {
	if before.IsZero() {
		before = time.Now().Add(24 * time.Hour)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT j.id, j.status::text, j.created_at, j.updated_at,
		       count(i.id), count(i.id) FILTER (WHERE i.status = 'completed'), count(i.id) FILTER (WHERE i.status = 'failed')
		FROM jobs j LEFT JOIN job_items i ON i.job_id = j.id
		WHERE j.owner = $1 AND j.created_at < $2
		GROUP BY j.id ORDER BY j.created_at DESC LIMIT $3`, owner, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var j Summary
		if err := rows.Scan(&j.ID, &j.Status, &j.CreatedAt, &j.UpdatedAt, &j.Counts.Total, &j.Counts.Completed, &j.Counts.Failed); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}
