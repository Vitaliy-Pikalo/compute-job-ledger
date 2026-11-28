# Compute Job Ledger

A local compute scheduler for answering who submitted a job, which worker ran it, and whether its recorded history still verifies. The CLI runs deterministic workloads and persists retry-safe job submissions to PostgreSQL. PostgreSQL binds each submission to its authenticated caller and records the decision. Workers can claim queued jobs using a database-issued lease.

## First workload

Run `go run ./cmd/ledger risk -seed 42 -trials 1000` with Go 1.22 or later. Each synthetic trial has a 5% chance of a loss. The command prints the input and loss count as JSON. Run `go test ./...` to check repeatability and input limits.

## Development database

With Docker Compose installed, set `LEDGER_DB_PASSWORD` to a local development password in your shell. PostgreSQL 16 listens on loopback port 5432. Set `LEDGER_DB_PORT` to choose another port. From the repository root in Bash:

```sh
docker compose up -d --wait db
docker compose exec -T db psql -X -U ledger -d ledger < db/migrations/001_jobs.sql
docker compose exec -T db psql -X -U ledger -d ledger -c '\d public.jobs'
docker compose exec -T db psql -X -U ledger -d ledger < db/migrations/002_callers.sql
docker compose exec -T db psql -X -U ledger -d ledger < db/migrations/003_claim.sql
```

The named volume retains data between restarts. The migration records its version in the same transaction that creates the jobs table. Running it again preserves existing rows. CI applies it twice against a fresh service and checks that invalid trial counts and duplicate request IDs are rejected.

## Configure a local caller

The local development database uses PostgreSQL password authentication. As the ledger database administrator, open `docker compose exec db psql -X -U ledger -d ledger` and provision a dedicated login for the CLI:

```sql
CREATE ROLE alice LOGIN;
\password alice
GRANT EXECUTE ON FUNCTION public.submit_risk(text,bigint,integer) TO alice;
INSERT INTO public.callers(login,role) VALUES ('alice','researcher');
```

Use the password prompt to set a unique secret. Use `viewer` for a caller who may reach the submission endpoint but cannot submit. Give callers only the function grant. The database owner controls login provisioning and the role mapping.

The function reads the authenticated login from PostgreSQL. A successful submission records its caller, role, request ID and job ID atomically with the job. Viewer and unmapped CLI submissions exit with status 4 and persist a denial. Request IDs remain globally unique. A retry must match both the original owner and inputs. Jobs created before migration 002 retain an unknown owner and reserve their original request IDs.

The decision table covers completed CLI submission attempts. Connection failures, malformed CLI arguments and manually rolled-back SQL transactions fall outside that history. An administrator can modify the table; signed verification is planned for later milestones.

## Submit a job

Start the database, apply migrations 001 and 002, and provision a caller above. Leave the database running. Set `LEDGER_DATABASE_URL` to its PostgreSQL connection string, for example `postgres://alice:<url-encoded-password>@127.0.0.1:5432/ledger?sslmode=disable` for local Compose.

```sh
go build -o ledger ./cmd/ledger
./ledger submit -request-id demo-001 -seed 42 -trials 1000
```

Expect a JSON object containing the database-assigned `job_id`. An identical retry returns the original ID, including when callers race. Changing the seed or trial count under the same request ID exits with status 3 and preserves the original job. Use a new request ID to submit another job with identical inputs. Submission stores the request for a future worker.

The input digest is SHA-256 of the ASCII text `risk/v1\n<seed>\n<trials>\n`, with decimal integers and LF separators. Request IDs are independent of this digest.

## Claim a job

Apply migration 003 as the database administrator. Provision a dedicated worker login in the same administrator psql session used for caller setup:

```sql
CREATE ROLE worker_a LOGIN;
\password worker_a
GRANT EXECUTE ON FUNCTION public.claim_job() TO worker_a;
```

Set `LEDGER_DATABASE_URL` to the worker's connection string and run `./ledger claim`. It prints the job ID, risk inputs, lease token and expiration as JSON. An empty poll prints `null` and exits successfully. A login lacking the function grant exits with status 4.

PostgreSQL locks the lowest-ID available row, skips rows locked by other transactions, and writes the worker login, token and 60-second expiration together. The database clock sets the deadline. If the transaction rolls back, the job becomes available again. A committed claim stays assigned in this milestone; expired-lease recovery and stale-token checks are milestone 5.

## Week-one milestones

Implement these in dependency order, keeping each change and its focused tests together.

1. **Complete.** PostgreSQL development service with Compose, migrations and integration-test setup.
2. **Complete.** Persist submissions with stable input hashes and idempotent request IDs.
3. **Complete.** Bind a local caller credential to a role. Record both authorized submissions and denials.
4. **Complete.** Claim a queued job transactionally with a worker lease.
5. Reclaim expired leases and reject stale worker tokens.
6. Execute the risk workload in a worker and record attempt identity.
7. Accept exactly one result transactionally across retries and competing completions.
8. Reproduce a worker crash after durable result acceptance but before acknowledgement.
9. Append ordered audit events with canonical encoding and a hash chain in the mutation transaction.
10. Verify event order, hashes and chain linkage independently.
11. Sign a versioned ledger checkpoint with a key held outside PostgreSQL.
12. Verify history against a separately stored, trusted checkpoint, including truncation.
13. Provide a repeatable local demonstration of denial, retry and tamper detection on a copy.
14. Exercise the complete demo in CI and document evidence, trust assumptions and limitations.

The target guarantee is one **accepted result**. A worker can execute a workload more than once after a crash. Signed checkpoints cover history through the signed head and rely on the verifier retaining a trusted public key and checkpoint.
