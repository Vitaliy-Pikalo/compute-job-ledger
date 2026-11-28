\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(734211, 1);
SELECT NOT EXISTS (
    SELECT FROM public.schema_migrations WHERE version = 5
) AS apply_v5 \gset
\if :apply_v5
CREATE FUNCTION public.renew_lease(p_job_id bigint, p_token uuid)
RETURNS timestamptz
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS $$
DECLARE
    deadline timestamptz;
    observed_at timestamptz;
BEGIN
    SELECT lease_expires_at INTO deadline FROM public.jobs
        WHERE id = p_job_id AND lease_owner = session_user AND lease_token = p_token
        FOR UPDATE;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;
    -- Time spent waiting for the row lock counts against the lease.
    observed_at := clock_timestamp();
    IF deadline <= observed_at THEN
        RETURN NULL;
    END IF;
    deadline := observed_at + interval '60 seconds';
    UPDATE public.jobs SET lease_expires_at = deadline WHERE id = p_job_id;
    RETURN deadline;
END $$;
REVOKE ALL ON FUNCTION public.renew_lease(bigint,uuid) FROM PUBLIC;
INSERT INTO public.schema_migrations(version) VALUES (5);
\endif
COMMIT;
