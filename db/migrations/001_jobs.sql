\set ON_ERROR_STOP on
BEGIN;
-- All ledger migrations use this transaction-scoped lock.
SELECT pg_advisory_xact_lock(734211, 1);

CREATE TABLE IF NOT EXISTS public.schema_migrations (
    version integer PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);

SELECT NOT EXISTS (
    SELECT FROM public.schema_migrations WHERE version = 1
) AS apply_v1 \gset
\if :apply_v1
CREATE TABLE public.jobs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    request_id text NOT NULL UNIQUE CHECK (length(request_id) > 0),
    input_hash bytea NOT NULL CHECK (octet_length(input_hash) = 32),
    seed bigint NOT NULL,
    trials integer NOT NULL CHECK (trials BETWEEN 1 AND 1000000),
    created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO public.schema_migrations (version) VALUES (1);
\endif
COMMIT;
