# Compute Job Ledger

A local compute scheduler for answering who submitted a job, which worker ran it, and whether its recorded history still verifies. The CLI runs deterministic workloads and persists retry-safe job submissions to PostgreSQL. Caller authorization is the next milestone.

## First workload

Run `go run ./cmd/ledger risk -seed 42 -trials 1000` with Go 1.22 or later. Each synthetic trial has a 5% chance of a loss. The command prints the input and loss count as JSON. Run `go test ./...` to check repeatability and input limits.

## Development database

With Docker Compose installed, set `LEDGER_DB_PASSWORD` to a local development password in your shell. PostgreSQL 16 listens on loopback port 5432. Set `LEDGER_DB_PORT` to choose another port. From the repository root in Bash:

```sh
docker compose up -d --wait db
docker compose exec -T db psql -X -U ledger -d ledger < db/migrations/001_jobs.sql
docker compose exec -T db psql -X -U ledger -d ledger -c '\d public.jobs'
docker compose down
```

The named volume retains data between restarts. The migration records its version in the same transaction that creates the jobs table. Running it again preserves existing rows. CI applies it twice against a fresh service and checks that invalid trial counts and duplicate request IDs are rejected.

## Submit a job

Start the database and apply migration 001 above. Leave the database running. Set `LEDGER_DATABASE_URL` to its PostgreSQL connection string, for example `postgres://ledger:<url-encoded-password>@127.0.0.1:5432/ledger?sslmode=disable` for local Compose.

```sh
go build -o ledger ./cmd/ledger
./ledger submit -request-id demo-001 -seed 42 -trials 1000
```

Expect a JSON object containing the database-assigned `job_id`. An identical retry returns the original ID, including when callers race. Changing the seed or trial count under the same request ID exits with status 3 and preserves the original job. Use a new request ID to submit another job with identical inputs. Submission stores the request for a future worker.

The input digest is SHA-256 of the ASCII text `risk/v1\n<seed>\n<trials>\n`, with decimal integers and LF separators. Request IDs are independent of this digest.

## Week-one milestones

Implement these in dependency order, keeping each change and its focused tests together.

1. **Complete.** PostgreSQL development service with Compose, migrations and integration-test setup.
2. **Complete.** Persist submissions with stable input hashes and idempotent request IDs.
3. Bind a local caller credential to a role. Record both authorized submissions and denials.
4. Claim a queued job transactionally with a worker lease.
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
