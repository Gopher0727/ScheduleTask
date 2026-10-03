# Open-source readiness implementation plan

> **For agentic workers:** Use superpowers:executing-plans inline. The owner
> authorized implementation and push; deliver a reviewable PR. Scope refinements
> require PostgreSQL + Redis only, constructor injection, cmd/internal/api layout
> and English-only repository content.

**Goal:** A runnable, tested, documented public scheduler development baseline.
**Architecture:** PostgreSQL owns durable timer/task state; Redis provides locks
and due-time indexes. internal/app wires dependencies and owns lifecycle.
**Tech stack:** Go, Echo, gRPC, GORM, PostgreSQL, Redis, Compose, GitHub Actions.
**Spec:** ../specs/2026-10-03-open-source-readiness-design.md

## Global constraints

- Module github.com/Gopher0727/ScheduleTask; Go floor 1.26.2.
- Existing HTTP routes/protobuf fields stay compatible; callbacks use POST.
- MIT selected by owner; repository content/comments are English.
- Branch and PR only; no main merge or release tag.

## Review focus

- Invalid/missing config and dependencies return startup errors.
- Invalid create/enable/lookup inputs do not mutate durable state.
- Repeated materialization preserves occurrence cadence and completed results.
- Lock contenders and task claim contenders have one winner.
- Redis cache loss and cancellation do not silently lose pending/terminal state.

### Task 1: Persistence, configuration and injection

Files: internal/config, internal/infra, internal/core/model and service packages.
Interfaces: Config.Load returns independent validated config; constructors consume
explicit clients/config; provider TaskStore adds atomic Claim and bounded history.

- [x] Reproduce invalid reload/ranges, missing IDs and duplicate materialization.
- [x] Add primary keys/timestamps, unique occurrences and idempotent inserts.
- [x] Support PostgreSQL + Redis only; remove alternate providers and singleton clients.
- [x] Test real PostgreSQL CRUD, atomic claims/history and Redis ownership/TTL.

### Task 2: Scheduling, callbacks and transport

Files: internal/core/{scheduler,trigger,migrator,executor}, internal/server,
internal/app and cmd/scheduletask.
Interfaces: constructor config injection; shared create/enable validation; HTTP
lookup/history/health adapters; context-aware callbacks and bounded shutdown.

- [x] Reproduce sequential buckets, zone mismatch, startup delay and invalid API input.
- [x] Fix concurrent scanning, catch-up, minute migration and validation.
- [x] Independently review runtime and reproduce cache loss, interval drift,
  impossible dates and terminal cancellation persistence.
- [x] Fix the four findings and bound callback responses and HTTP request bodies.
- [x] Run final full race/integration suite, vet, build and callback smoke.

### Task 3: Contributor experience and delivery

Files: README/community documents, docs, .github, Dockerfile, Compose, examples,
scripts, config samples, Makefile and canonical module/import paths.
Interfaces: make check/build/integration; docker compose --profile demo up --build.

- [x] Add English-only docs, MIT/policies/templates and dependency updates.
- [x] Add Compose PostgreSQL/Redis, non-root app image, example and smoke script.
- [x] Add pinned CI actions with integration/race tests and Compose smoke job.
- [x] Verify links, formatting, module tidiness and workflow/YAML config; Compose runtime is checked by remote CI.
- [ ] Commit, push, create/attach PR and inspect remote CI.
