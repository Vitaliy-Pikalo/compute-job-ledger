\set ON_ERROR_STOP on
BEGIN;
DO $$
DECLARE
    first_claim record;
    next_claim record;
BEGIN
    SELECT * INTO STRICT first_claim FROM public.claim_job();
    IF EXISTS (SELECT FROM public.claim_job()) THEN
        RAISE EXCEPTION 'unexpired lease was reclaimed';
    END IF;
    UPDATE public.jobs SET lease_expires_at = clock_timestamp() - interval '1 second'
        WHERE id = first_claim.job_id;
    SELECT * INTO STRICT next_claim FROM public.claim_job();
    IF next_claim.job_id <> first_claim.job_id OR next_claim.token = first_claim.token
        OR next_claim.expires_at <= clock_timestamp() THEN
        RAISE EXCEPTION 'reclaim failed to replace the expired lease';
    END IF;
END $$;
ROLLBACK;
