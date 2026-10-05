package jobs

import (
	"context"
	"time"
)

// Expired is a finished job past its retention period, with the stored files that belong to it.
type Expired struct {
	ID   string
	Keys []string
}

// Retention is what the janitor needs; PGStore implements it.
type Retention interface {
	ExpiredJobs(ctx context.Context, createdBefore time.Time, limit int) ([]Expired, error)
	DeleteJobs(ctx context.Context, ids []string) (int64, error)
}

// ExpiredJobs returns up to limit finished jobs created before the cutoff, oldest first. Jobs that are still queued
// or processing are never returned, however old.
func (s *PGStore) ExpiredJobs(ctx context.Context, createdBefore time.Time, limit int) ([]Expired, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT j.id, COALESCE(array_agg(i.output_key) FILTER (WHERE i.output_key IS NOT NULL), '{}')
		FROM jobs j LEFT JOIN job_items i ON i.job_id = j.id
		WHERE j.created_at < $1 AND j.status IN ('completed', 'partial', 'failed')
		GROUP BY j.id ORDER BY j.created_at LIMIT $2`, createdBefore, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Expired
	for rows.Next() {
		var e Expired
		if err := rows.Scan(&e.ID, &e.Keys); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteJobs removes the jobs and, by cascade, their items. Call it after the stored files are gone.
func (s *PGStore) DeleteJobs(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM jobs WHERE id = ANY($1::uuid[]) AND status IN ('completed', 'partial', 'failed')`, ids)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
