// Package worker turns queued job items into compressed images and delivers completion webhooks.
package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	"app/internal/imageproc"
	"app/internal/jobs"
	"app/internal/storage"
)

type Config struct {
	Concurrency   int
	Poll          time.Duration // idle sleep between empty claims
	Lease         time.Duration // how long a claimed item may take before another worker retries it
	MaxAttempts   int
	Image         imageproc.Options
	WebhookSecret string
	WebhookTries  int
}

func DefaultConfig() Config {
	return Config{Concurrency: 4, Poll: 2 * time.Second, Lease: 5 * time.Minute, MaxAttempts: 3,
		Image: imageproc.DefaultOptions(), WebhookTries: 5}
}

type Worker struct {
	Queue   jobs.Queue
	Storage storage.Storage
	Fetch   *http.Client // must be SSRF-safe (safeurl.NewClient)
	Hooks   *http.Client // must be SSRF-safe and must not follow redirects
	Config  Config
	Log     *slog.Logger
}

// Run blocks until ctx is cancelled, running Concurrency image loops and one webhook loop.
func (w *Worker) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < max(1, w.Config.Concurrency); i++ {
		wg.Add(1)
		go func() { defer wg.Done(); w.loop(ctx, w.processOne) }()
	}
	wg.Add(1)
	go func() { defer wg.Done(); w.loop(ctx, w.deliverOne) }()
	wg.Wait()
}

// loop calls step until the context ends, sleeping (with jitter) only when there was no work.
func (w *Worker) loop(ctx context.Context, step func(context.Context) (bool, error)) {
	for ctx.Err() == nil {
		worked, err := step(ctx)
		if err != nil && ctx.Err() == nil {
			w.Log.Error("worker step failed", "err", err)
		}
		if worked && err == nil {
			continue
		}
		sleep := w.Config.Poll + time.Duration(rand.Int64N(int64(w.Config.Poll/2)+1))
		select {
		case <-ctx.Done():
		case <-time.After(sleep):
		}
	}
}

// processOne claims and processes a single item. It reports whether it found work.
func (w *Worker) processOne(ctx context.Context) (bool, error) {
	claim, err := w.Queue.ClaimItem(ctx, w.Config.Lease)
	if err != nil || claim == nil {
		return false, err
	}
	// Record the result even when shutting down, so a finished image is not redone.
	record := context.WithoutCancel(ctx)

	keyExt, bytesIn, bytesOut, perr := w.process(ctx, *claim)
	if perr != nil && ctx.Err() != nil {
		// Shutting down mid-image: not the image's fault. The lease lets another worker pick it up.
		return false, nil
	}
	if perr != nil {
		w.Log.Warn("item failed", "item", claim.ItemID, "attempt", claim.Attempts, "err", perr)
		return true, w.Queue.FailItem(record, *claim, perr.Error(), imageproc.IsPermanent(perr), w.Config.MaxAttempts)
	}
	return true, w.Queue.CompleteItem(record, *claim, keyExt, bytesIn, bytesOut)
}

func (w *Worker) process(ctx context.Context, c jobs.Claim) (key string, in, out int64, err error) {
	data, err := imageproc.Fetch(ctx, w.Fetch, c.SourceURL, w.Config.Image.MaxBytes)
	if err != nil {
		return "", 0, 0, err
	}
	compressed, ext, err := imageproc.Compress(data, w.Config.Image)
	if err != nil {
		return "", 0, 0, err
	}
	key = c.JobID + "/" + strconv.Itoa(c.Position) + "." + ext
	if _, err := w.Storage.Put(ctx, key, bytes.NewReader(compressed)); err != nil {
		return "", 0, 0, fmt.Errorf("store result: %w", err)
	}
	return key, int64(len(data)), int64(len(compressed)), nil
}

type webhookPayload struct {
	JobID  string      `json:"job_id"`
	Status string      `json:"status"`
	Counts jobs.Counts `json:"counts"`
}

// Sign returns the signature header value: "sha256=" + HMAC-SHA256(secret, timestamp + "." + body).
func Sign(secret, timestamp string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(timestamp + "."))
	m.Write(body)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

func (w *Worker) deliverOne(ctx context.Context) (bool, error) {
	claim, err := w.Queue.ClaimWebhook(ctx, time.Minute, w.Config.WebhookTries)
	if err != nil || claim == nil {
		return false, err
	}
	body, _ := json.Marshal(webhookPayload{JobID: claim.JobID, Status: claim.Status, Counts: claim.Counts})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claim.URL, bytes.NewReader(body))
	if err == nil {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-ImageFlow-Timestamp", ts)
		req.Header.Set("X-ImageFlow-Signature", Sign(w.Config.WebhookSecret, ts, body))
		var resp *http.Response
		if resp, err = w.Hooks.Do(req); err == nil {
			resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				err = fmt.Errorf("webhook returned status %d", resp.StatusCode)
			}
		}
	}
	record := context.WithoutCancel(ctx)
	if err != nil {
		w.Log.Warn("webhook failed", "job", claim.JobID, "attempt", claim.Attempts, "err", err)
		retry := time.Duration(claim.Attempts*claim.Attempts) * 30 * time.Second
		return true, w.Queue.FinishWebhook(record, claim.JobID, false, retry)
	}
	return true, w.Queue.FinishWebhook(record, claim.JobID, true, 0)
}
