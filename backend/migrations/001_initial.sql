-- +goose Up
CREATE TABLE IF NOT EXISTS clusters (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    kubeconfig  TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS git_repos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id  UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    branch      TEXT NOT NULL DEFAULT 'main',
    token       TEXT,
    path        TEXT NOT NULL DEFAULT '/',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS drift_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id      UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    resource_kind   TEXT NOT NULL,
    resource_name   TEXT NOT NULL,
    namespace       TEXT NOT NULL,
    severity        TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    drift_type      TEXT NOT NULL CHECK (drift_type IN ('modified', 'missing', 'unexpected')),
    desired_state   JSONB,
    live_state      JSONB,
    diff            JSONB,
    resolved        BOOLEAN NOT NULL DEFAULT FALSE,
    detected_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id  UUID REFERENCES clusters(id) ON DELETE SET NULL,
    action      TEXT NOT NULL,
    actor       TEXT NOT NULL DEFAULT 'system',
    resource    TEXT,
    detail      JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS drift_events_cluster_detected ON drift_events(cluster_id, detected_at DESC);
CREATE INDEX IF NOT EXISTS audit_log_cluster_created ON audit_log(cluster_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS drift_events;
DROP TABLE IF EXISTS git_repos;
DROP TABLE IF EXISTS clusters;
