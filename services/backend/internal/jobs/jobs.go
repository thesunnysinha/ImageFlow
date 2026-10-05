// Package jobs holds the job domain types and persistence.
package jobs

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("job not found")

type Item struct {
	ID        string  `json:"id"`
	Position  int     `json:"position"`
	SourceURL string  `json:"source_url"`
	Status    string  `json:"status"`
	OutputKey *string `json:"output_key,omitempty"`
	BytesIn   *int64  `json:"bytes_in,omitempty"`
	BytesOut  *int64  `json:"bytes_out,omitempty"`
	Error     *string `json:"error,omitempty"`
}

type Job struct {
	ID         string    `json:"id"`
	Status     string    `json:"status"`
	WebhookURL *string   `json:"webhook_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Items      []Item    `json:"items"`
}

// Summary is a job without its items, for listings.
type Summary struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Counts    Counts    `json:"counts"`
}

// Store is the persistence contract; implemented by PGStore and test fakes.
type Store interface {
	Create(ctx context.Context, owner string, webhookURL *string, sourceURLs []string) (Job, error)
	Get(ctx context.Context, owner, id string) (Job, error)
	OutputKey(ctx context.Context, owner, id string, position int) (string, error)
	// List returns the owner's jobs, newest first. before is an exclusive created_at cursor (zero: from the start).
	List(ctx context.Context, owner string, limit int, before time.Time) ([]Summary, error)
	Ping(ctx context.Context) error
}
