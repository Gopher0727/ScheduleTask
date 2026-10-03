# Open-source readiness design

The owner requested end-to-end improvement of the existing public repository,
including clone, implementation, validation and push. They selected MIT, required
PostgreSQL + Redis only, authorized conventional directory restructuring and
constructor injection, and required all repository content/comments in English.
Delivery is a branch and reviewable PR, without merging or tagging a release.

## Scope and success criteria

A contributor can clone, launch the PostgreSQL + Redis + Go Compose stack, create,
query, enable/disable a timer, observe a callback and query execution history.
Keep HTTP route and protobuf field compatibility. Use cmd/scheduletask for the
CLI, internal for runtime code and api/pb for external protocol clients. Keep
Go 1.26.2 as the floor and the canonical GitHub module path.

## Design

PostgreSQL owns timer/task state; Redis provides expiring locks and time indexes.
Pass validated configuration and clients through constructors; startup returns
errors and owns dependency cleanup. Claim tasks atomically, deduplicate scheduled
occurrences, preserve interval phase, terminate impossible schedules, scan buckets
concurrently, reconcile partial Redis indexes against durable pending tasks and
persist terminal cancellation results with a bounded cleanup context.

Supply a non-root Docker image, loopback development stack, callback example and
smoke script, CI with real PostgreSQL/Redis and race detector, MIT/community files,
English API/architecture/operations/roadmap docs, templates and Dependabot.

## Reliability boundaries

The current/previous minute is the catch-up window. Callbacks run sequentially
within each bucket with a ten-second timeout and 1 MiB response bound. Materialize
at startup/every minute with a ten-minute default horizon; discovery can take up to a minute, then the next scheduled occurrence applies. No exactly-once, automatic retries, crash
reclaim, retention, authentication or egress allowlist claim. Document original
schema incompatibilities and backup/manual migration requirements.

## Verification

Write regression tests before behavioral fixes. Use isolated PostgreSQL schemas
and unique Redis keys. Run formatting, vet, full race/integration tests, module
consistency and builds. An independent runtime review found four issues (impossible
cron loops, interval drift, cache-loss skips, canceled result writes); each needs
covering tests and fixes. Validate complete create/enable/callback/history/disable
flow locally and through Compose CI. Push and inspect PR checks.
