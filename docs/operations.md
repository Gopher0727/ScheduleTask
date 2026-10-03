# Operations

## Configuration

Use `config.example.toml` for a host binary, `config.docker.toml` inside Docker,
and `-config PATH` to choose a file. The loader returns independent validated
configs; hot reload is unsupported. Environment overrides use `SCHEDULETASK_`
and replace dots with underscores, for example `postgres.dsn` becomes
`SCHEDULETASK_POSTGRES_DSN`. Keep keys in the file even when overriding them.

| Setting | Meaning |
| --- | --- |
| `port` | HTTP port; gRPC is port + 1; valid range 1–65534 |
| `postgres.dsn` | PostgreSQL connection string |
| `postgres.max_open_conns`, `max_idle_conns` | SQL pool limits |
| `redis.addr`, `password`, `db`, `pool_size` | Dedicated Redis connection |
| `scheduler.bucket_num` | Shared partition count, range 1–1024 |
| `scheduler.try_lock_seconds` | Bucket lease, at least 70 seconds |
| `scheduler.success_expire_seconds` | Completed bucket lock TTL |
| `trigger.zrange_gap_seconds` | Due scan cadence, 1–60 seconds |
| `migrator.migrate_step_minutes` | Half the future planning window, 1–1440 minutes |
| `migrator.try_lock_minutes` | Minute materialization lease duration |

Large planning windows with frequent intervals can exceed the 10,000-occurrence
per-timer bound; lower the window or increase the interval. The default five-minute
step plans ten minutes ahead. Removed pool/queue/cache settings had no runtime
effect and are no longer advertised.

Set `TZ=UTC` (default in Compose) or the same time zone on every scheduler node.
Calendar cron follows that zone unless it supplies `CRON_TZ`. Changing bucket count
or zone with active nodes/indexes can alter partitioning; restart all nodes
together and inspect pending records. Use a dedicated Redis database to isolate
the scheduler's bucket and lock keys.

## Startup and shutdown

Startup validates config, checks dependency connections, creates/migrates tables,
and binds HTTP/gRPC before launching workers. Errors cause nonzero process exit.
Dependency connection/migration has a ten-second budget. SIGINT/SIGTERM cancels
workers and callbacks, gracefully stops servers, and waits within a 15-second
shutdown budget. Compose grants 20 seconds before forcibly terminating the app.

`/healthz` is liveness; `/readyz` checks dependency connectivity. Inspect logs and
execution records as well as readiness; connectivity alone does not prove work
is progressing. Back up PostgreSQL, monitor task failures/running age and lateness,
and protect credentials. Redis Compose enables AOF; PostgreSQL and Redis use
named volumes. `docker compose down` preserves them.

## Delivery semantics and known limits

- Callbacks have a ten-second timeout, a 1 MiB response limit, and no automatic retries.
- Atomic claims suppress concurrent sends; exactly-once external effects are not
  guaranteed. Consumers should make their operations idempotent.
- Graceful cancellation saves failure status with a separate bounded cleanup context.
  Crashes or database unavailability after the callback can leave status `3`.
  Investigate these records before any manual replay.
- Pending executions in the current/previous minute are reconciled from PostgreSQL
  if Redis indexes are lost. Older missed occurrences require manual recovery.
- Redis failure prevents new distributed lock acquisition. A lock expires without
  periodic renewal while a slow bucket runs; PostgreSQL claims remain the final
  concurrent-execution gate.
- Callbacks run sequentially within a bucket. No throughput SLA or load benchmark
  is provided. Plan a worker/lease design before using this for large backlogs.
- Records are retained indefinitely. Establish a retention policy and back up before
  deleting historical data. The service does not yet provide automated cleanup.
- HTTP/gRPC have no authentication/TLS. Restrict callers and callback egress. See SECURITY.md.

## Upgrading the original prototype

This change removes MySQL/SQLite and PostgreSQL-based locks/time-wheel. Set up
Redis and use the new PostgreSQL/Redis configuration. The Go module/import paths
and executable location also changed.

The original models did not declare primary keys, automatic timestamps or task
uniqueness. Existing tables may contain duplicate/zero IDs, duplicate occurrences,
or missing timestamps. Do not assume `AutoMigrate` repairs that data.

1. Stop all scheduler instances and back up PostgreSQL.
2. Inspect timer/task IDs, duplicates and relationships. For disposable demo data,
   use a fresh database. For retained data, write and review a data migration that
   assigns consistent IDs/timestamps and removes duplicate occurrences.
3. Ensure primary keys and `(timer_id, run_timer)` uniqueness can be added. Existing
   `@every` timers need valid creation timestamps to establish their phase.
4. Start one new instance, verify logs/readiness and a callback, then expand.

Automatic schema changes are suitable for fresh development databases. Versioned
schema migrations and rollback tooling are on the roadmap for managed deployments.

## Troubleshooting

**No callback yet:** verify timer status, cron/time zone, allow up to two minutes
after enable for the every-minute example (other cron schedules wait for their
own occurrence), check `/readyz`, logs and `/xtimer/getTasks`. A container's `localhost`
is the container itself; the demo URL is `http://callback:9090/callback`.

**Readiness fails:** inspect `docker compose ps` and dependency logs; check passwords
and network names. Changing a PostgreSQL environment password does not reset an
existing volume's user password.

**Port occupied:** change the host mappings in Compose or use another local `port`
and its consecutive gRPC port. The service exits if either bind fails.

**Migration fails:** inspect old-data compatibility and schema permissions. A startup
schema failure is reported rather than ignored.
