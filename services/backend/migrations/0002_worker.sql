ALTER TABLE job_items
    ADD COLUMN next_attempt_at timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN locked_at       timestamptz;

-- Workers claim with FOR UPDATE SKIP LOCKED; this index serves the claim query.
CREATE INDEX job_items_claim_idx ON job_items (next_attempt_at) WHERE status IN ('queued', 'processing');

ALTER TABLE jobs
    ADD COLUMN webhook_pending      boolean     NOT NULL DEFAULT false,
    ADD COLUMN webhook_attempts     integer     NOT NULL DEFAULT 0,
    ADD COLUMN webhook_next_at      timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN webhook_delivered_at timestamptz;

CREATE INDEX jobs_webhook_idx ON jobs (webhook_next_at) WHERE webhook_pending;
