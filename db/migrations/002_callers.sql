\set ON_ERROR_STOP on
BEGIN;
SELECT pg_advisory_xact_lock(734211, 1);
SELECT NOT EXISTS (
    SELECT FROM public.schema_migrations WHERE version = 2
) AS apply_v2 \gset
\if :apply_v2
CREATE TABLE public.callers (
    login name PRIMARY KEY,
    role text NOT NULL CHECK (role IN ('researcher', 'viewer'))
);

ALTER TABLE public.jobs ADD COLUMN submitted_by name;

CREATE TABLE public.submission_decisions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    login name NOT NULL,
    role text CHECK (role IN ('researcher', 'viewer')),
    request_id text NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('accepted', 'denied', 'conflict')),
    job_id bigint REFERENCES public.jobs,
    CHECK ((outcome = 'accepted') = (job_id IS NOT NULL))
);

-- Privileged logic is confined to this entry point. The administrator grants
-- EXECUTE explicitly to each local caller. SESSION_USER is the original login.
CREATE FUNCTION public.submit_risk(p_request text, p_seed bigint, p_trials integer)
RETURNS TABLE(outcome text, job_id bigint)
LANGUAGE plpgsql SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS $$
DECLARE
    caller_role text;
    digest bytea;
BEGIN
    SELECT c.role INTO caller_role FROM public.callers c WHERE c.login = session_user FOR SHARE;
    IF caller_role IS DISTINCT FROM 'researcher' THEN
        outcome := 'denied';
    ELSE
        digest := sha256(convert_to('risk/v1' || chr(10) || p_seed::text || chr(10)
                                    || p_trials::text || chr(10), 'UTF8'));
        INSERT INTO public.jobs(request_id,input_hash,seed,trials,submitted_by)
            VALUES(p_request,digest,p_seed,p_trials,session_user)
            ON CONFLICT(request_id) DO NOTHING RETURNING id INTO job_id;
        IF job_id IS NULL THEN
            SELECT j.id INTO job_id FROM public.jobs j WHERE j.request_id = p_request
                AND j.submitted_by = session_user AND j.input_hash = digest
                AND j.seed = p_seed AND j.trials = p_trials;
        END IF;
        outcome := CASE WHEN job_id IS NULL THEN 'conflict' ELSE 'accepted' END;
    END IF;
    INSERT INTO public.submission_decisions(login,role,request_id,outcome,job_id)
        VALUES(session_user,caller_role,p_request,outcome,job_id);
    RETURN NEXT;
END;
$$;
REVOKE ALL ON FUNCTION public.submit_risk(text,bigint,integer) FROM PUBLIC;
INSERT INTO public.schema_migrations(version) VALUES(2);
\endif
COMMIT;
