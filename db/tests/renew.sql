\set ON_ERROR_STOP on
BEGIN;
CREATE ROLE lease_test_a;
CREATE ROLE lease_test_b;
GRANT EXECUTE ON FUNCTION public.claim_job(), public.renew_lease(bigint,uuid)
    TO lease_test_a, lease_test_b;
CREATE FUNCTION public.assert_lease_test(ok boolean, reason text) RETURNS void
LANGUAGE plpgsql AS $$
BEGIN
    IF ok IS DISTINCT FROM true THEN RAISE EXCEPTION '%', reason; END IF;
END $$;

SET SESSION AUTHORIZATION lease_test_a;
SELECT * FROM public.claim_job() \gset original_
SELECT public.renew_lease(:original_job_id, :'original_token') AS renewed \gset
SELECT public.assert_lease_test(:'renewed'::timestamptz > :'original_expires_at'::timestamptz,
    'live owner could not extend lease');
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION lease_test_b;
SELECT public.assert_lease_test(public.renew_lease(:original_job_id, :'original_token') IS NULL,
    'another worker renewed a live token');
RESET SESSION AUTHORIZATION;
SELECT public.assert_lease_test(lease_expires_at = :'renewed'::timestamptz,
    'stored deadline differs from successful renewal') FROM public.jobs WHERE id = :original_job_id;
UPDATE public.jobs SET lease_expires_at = clock_timestamp() - interval '1 second'
    WHERE id = :original_job_id;
SET SESSION AUTHORIZATION lease_test_a;
SELECT public.assert_lease_test(public.renew_lease(:original_job_id, :'original_token') IS NULL,
    'owner revived an expired lease');
SELECT * FROM public.claim_job() \gset replacement_
SELECT public.assert_lease_test(public.renew_lease(:original_job_id, :'original_token') IS NULL,
    'same login reused its previous token after reclaim');
SELECT public.assert_lease_test(public.renew_lease(:replacement_job_id, :'replacement_token') IS NOT NULL,
    'current token was rejected after reclaim');
RESET SESSION AUTHORIZATION;
ROLLBACK;
