# Changelog

## Unreleased

### Added

- MIT license, contributor/conduct/security policies, API and operations docs.
- Docker Compose development stack with PostgreSQL, Redis and optional callback demo.
- CI for Go formatting, vet, race/integration tests, builds and container smoke checks.
- HTTP timer lookup, execution history, liveness/readiness and Prometheus endpoint.
- Configuration and API validation, persistence and scheduling regression tests.
- Atomic task claims and idempotent materialization.

### Changed

- Go module is `github.com/Gopher0727/ScheduleTask`; entry point is `cmd/scheduletask`.
- Runtime implementation moved to `internal`; protobuf code moved to `api/pb`.
- PostgreSQL stores timers/tasks; Redis supplies locks/time-wheel. Removed MySQL,
  SQLite and PostgreSQL lock/time-wheel alternatives.
- Replaced global client/config singletons with constructor injection.
- Migration runs at startup and every minute; interval schedules keep a stable phase.
- Concurrent bucket scanning, local-zone parsing, PostgreSQL reconciliation of
  missing Redis indexes, cancellation-aware callbacks and bounded shutdown.
- Docker now uses the module's Go floor, checked-in container config and a non-root user.

### Compatibility

- Existing Go imports and backend config require updates. HTTP paths and protobuf
  fields remain compatible; invalid inputs are now rejected.
- Timer/task primary keys, timestamps and task uniqueness require a database migration.
  Back up and inspect old data before upgrading; see docs/operations.md.
- This entry describes unreleased work. No version tag or production-readiness claim.
