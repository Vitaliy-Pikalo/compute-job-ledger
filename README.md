# Compute Job Ledger

A local compute scheduler for answering who submitted a job, which worker ran it, and whether its recorded history still verifies. The first milestone is a deterministic Go workload. The scheduler and verification features below are planned.

## First workload

Run `go run ./cmd/ledger risk -seed 42 -trials 1000` with Go 1.22 or later. Each synthetic trial has a 5% chance of a loss. The command prints the input and loss count as JSON. Run `go test ./...` to check repeatability and input limits.

## Week-one milestones

Implement these in dependency order, keeping each change and its focused tests together.

1. PostgreSQL development service with Compose, migrations and integration-test setup.
2. Persist submissions with stable input hashes and idempotent request IDs.
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
