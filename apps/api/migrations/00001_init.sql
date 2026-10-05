-- +goose Up
CREATE TYPE job_status AS ENUM ('queued', 'processing', 'completed', 'partial', 'failed');
CREATE TYPE item_status AS ENUM ('queued', 'processing', 'completed', 'failed');

CREATE TABLE jobs (
    id            uuid PRIMARY KEY,
    owner         text        NOT NULL,
    status        job_status  NOT NULL DEFAULT 'queued',
    webhook_url   text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX jobs_owner_created_idx ON jobs (owner, created_at DESC);

CREATE TABLE job_items (
    id          uuid PRIMARY KEY,
    job_id      uuid        NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    position    integer     NOT NULL,
    source_url  text        NOT NULL,
    status      item_status NOT NULL DEFAULT 'queued',
    output_key  text,
    bytes_in    bigint,
    bytes_out   bigint,
    error       text,
    attempts    integer     NOT NULL DEFAULT 0,
    UNIQUE (job_id, position)
);
CREATE INDEX job_items_job_idx ON job_items (job_id);

-- +goose Down
DROP TABLE job_items;
DROP TABLE jobs;
DROP TYPE item_status;
DROP TYPE job_status;
