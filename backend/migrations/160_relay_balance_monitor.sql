CREATE TABLE IF NOT EXISTS relay_balance_stations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    base_url TEXT NOT NULL,
    script TEXT NOT NULL,
    package_json TEXT NOT NULL DEFAULT '{"type":"module"}',
    cron_expression VARCHAR(100) NOT NULL DEFAULT '0 * * * *',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_balance NUMERIC(18, 6),
    last_currency VARCHAR(16),
    last_status VARCHAR(20),
    last_error TEXT,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS relay_balance_runs (
    id BIGSERIAL PRIMARY KEY,
    station_id BIGINT NOT NULL REFERENCES relay_balance_stations(id) ON DELETE CASCADE,
    station_name VARCHAR(120) NOT NULL,
    balance NUMERIC(18, 6),
    currency VARCHAR(16),
    status VARCHAR(20) NOT NULL,
    stdout TEXT,
    stderr TEXT,
    error TEXT,
    raw JSONB,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_relay_balance_stations_enabled ON relay_balance_stations(enabled);
CREATE INDEX IF NOT EXISTS idx_relay_balance_runs_station_started ON relay_balance_runs(station_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_relay_balance_runs_started ON relay_balance_runs(started_at DESC);
