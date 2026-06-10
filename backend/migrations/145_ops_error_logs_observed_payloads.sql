-- Store observed request/response payloads for ops error investigation only.
-- These columns are intentionally separate from the removed retry/replay fields.

ALTER TABLE ops_error_logs
  ADD COLUMN IF NOT EXISTS observed_request_headers JSONB,
  ADD COLUMN IF NOT EXISTS observed_request_body TEXT,
  ADD COLUMN IF NOT EXISTS observed_request_body_truncated BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS observed_request_body_bytes INT,
  ADD COLUMN IF NOT EXISTS observed_response_headers JSONB,
  ADD COLUMN IF NOT EXISTS observed_response_body TEXT,
  ADD COLUMN IF NOT EXISTS observed_response_body_truncated BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS observed_response_body_bytes INT;

COMMENT ON COLUMN ops_error_logs.observed_request_headers IS 'Sanitized request headers captured for ops error investigation.';
COMMENT ON COLUMN ops_error_logs.observed_request_body IS 'Sanitized request body captured for ops error investigation. Not used for retry/replay.';
COMMENT ON COLUMN ops_error_logs.observed_request_body_truncated IS 'Whether observed_request_body was truncated before storage.';
COMMENT ON COLUMN ops_error_logs.observed_request_body_bytes IS 'Original request body byte length when known.';
COMMENT ON COLUMN ops_error_logs.observed_response_headers IS 'Sanitized response headers captured for ops error investigation.';
COMMENT ON COLUMN ops_error_logs.observed_response_body IS 'Sanitized response body captured for ops error investigation. Not used for retry/replay.';
COMMENT ON COLUMN ops_error_logs.observed_response_body_truncated IS 'Whether observed_response_body was truncated before storage.';
COMMENT ON COLUMN ops_error_logs.observed_response_body_bytes IS 'Original response body byte length when known.';
