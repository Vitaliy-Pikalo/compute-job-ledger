\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(734211, 1);
SELECT NOT EXISTS (
    SELECT FROM public.schema_migrations WHERE version = 4
) AS apply_v4 \gset
\if :apply_v4
CREATE OR REPLACE FUNCTION public.claim_job()
RETURNS TABLE(job_id bigint, seed bigint, trials integer, token uuid, expires_at timestamptz)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS $$
    WITH candidate AS (
        SELECT q.id FROM public.jobs q
        WHERE q.lease_token IS NULL OR q.lease_expires_at <= statement_timestamp()
        ORDER BY q.id
        LIMIT 1 FOR UPDATE SKIP LOCKED
    )
    UPDATE public.jobs j
    SET lease_owner = session_user, lease_token = gen_random_uuid(),
        lease_expires_at = clock_timestamp() + interval '60 seconds'
    FROM candidate
    WHERE j.id = candidate.id
    RETURNING j.id, j.seed, j.trials, j.lease_token, j.lease_expires_at;
$$;
REVOKE ALL ON FUNCTION public.claim_job() FROM PUBLIC;
INSERT INTO public.schema_migrations(version) VALUES (4);
\endif
COMMIT;
