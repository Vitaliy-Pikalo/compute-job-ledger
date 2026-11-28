\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(734211, 1);
SELECT NOT EXISTS (
    SELECT FROM public.schema_migrations WHERE version = 3
) AS apply_v3 \gset
\if :apply_v3
ALTER TABLE public.jobs
    ADD COLUMN lease_owner name,
    ADD COLUMN lease_token uuid,
    ADD COLUMN lease_expires_at timestamptz,
    ADD CONSTRAINT lease_complete CHECK (
        num_nonnulls(lease_owner, lease_token, lease_expires_at) IN (0, 3)
    );
CREATE INDEX jobs_unclaimed_idx ON public.jobs (id) WHERE lease_token IS NULL;

-- Grant EXECUTE only to dedicated worker logins. A zero-row result means
-- that no unclaimed, currently unlocked job was visible to this statement.
CREATE FUNCTION public.claim_job()
RETURNS TABLE(job_id bigint, seed bigint, trials integer, token uuid, expires_at timestamptz)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS $$
    WITH candidate AS (
        SELECT q.id FROM public.jobs q
        WHERE q.lease_token IS NULL
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
INSERT INTO public.schema_migrations(version) VALUES (3);
\endif
COMMIT;
