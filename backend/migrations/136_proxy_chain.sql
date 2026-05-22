-- Add optional upstream proxy relation for chained proxy routing.

ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS upstream_proxy_id BIGINT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'proxies_upstream_proxy_id_fkey'
          AND table_name = 'proxies'
    ) THEN
        ALTER TABLE proxies
            ADD CONSTRAINT proxies_upstream_proxy_id_fkey
            FOREIGN KEY (upstream_proxy_id)
            REFERENCES proxies (id)
            ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'proxies_upstream_not_self_check'
          AND table_name = 'proxies'
    ) THEN
        ALTER TABLE proxies
            ADD CONSTRAINT proxies_upstream_not_self_check
            CHECK (upstream_proxy_id IS NULL OR upstream_proxy_id <> id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_proxies_upstream_proxy_id
    ON proxies (upstream_proxy_id)
    WHERE upstream_proxy_id IS NOT NULL;
