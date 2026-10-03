# ScheduleTask

[![CI](https://github.com/Gopher0727/ScheduleTask/actions/workflows/ci.yml/badge.svg)](https://github.com/Gopher0727/ScheduleTask/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-%E2%89%A5%201.26.2-00ADD8)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A Go service that turns cron schedules into HTTP POST callbacks. PostgreSQL
stores timers and execution results; Redis supplies distributed bucket locks and
time-wheel indexes. HTTP and gRPC expose timer creation, lookup and enable/disable.

**Status: experimental.** This project provides a runnable development stack and
regression tests. It does not yet promise exactly-once delivery, automatic retries,
or recovery of executions after extended downtime. See [operations](docs/operations.md)
before using it for important work.

[API](docs/api.md) ·
[Architecture](docs/architecture.md) · [Contributing](CONTRIBUTING.md)

## Quick start with Docker

Requires Docker Engine/Desktop and Docker Compose v2 with `--wait` support. Go is
only required when developing outside the container.

```sh
git clone https://github.com/Gopher0727/ScheduleTask.git
cd ScheduleTask
cp .env.example .env
docker compose --profile demo up --build -d --wait
curl -fsS http://localhost:8082/readyz
```

Compose starts PostgreSQL, Redis, the scheduler, and a demo callback receiver.
HTTP listens on `localhost:8082`, gRPC on `localhost:8083`. Published ports bind
to loopback. PostgreSQL and Redis data persist in named volumes.

Create a disabled timer:

```sh
curl -fsS http://localhost:8082/xtimer/createTimer \
  -H 'Content-Type: application/json' \
  -d '{"app":"demo","name":"hello","cron":"* * * * *","notify_http_param":{"method":"POST","url":"http://callback:9090/callback","header":{"Content-Type":"application/json"},"body":"{\"hello\":\"world\"}"}}'
```

Use the returned `data.timer_id` to enable it (replace `1` below):

```sh
curl -fsS http://localhost:8082/xtimer/enableTimer \
  -H 'Content-Type: application/json' -d '{"app":"demo","timer_id":1,"status":1}'
curl -fsS 'http://localhost:8082/xtimer/getTimer?timer_id=1'
curl -fsS 'http://localhost:8082/xtimer/getTasks?timer_id=1&limit=20'
docker compose logs -f callback
```

The migrator runs at startup and every minute. For this every-minute example,
allow up to two minutes for discovery and the first callback. Other schedules
wait for their own next occurrence. Standard cron uses five fields (minute through weekday),
with standard descriptors including `@every 10s`. Interval schedules keep the
timer's creation-time phase. Disabling uses the same endpoint with `status: 0`.

For an automated end-to-end check (Python 3, standard library only):

```sh
python3 scripts/smoke.py --callback-url http://callback:9090/callback
docker compose down
```

`down` preserves data. Only use `docker compose down -v` when you deliberately
want to delete development data.

## Local development

Requires Go 1.26.2 or newer. Start just the infrastructure in Docker:

```sh
docker compose up -d --wait postgres redis
cp config.example.toml config.toml
make build
./bin/scheduletask -config config.toml
```

In another terminal, `python3 scripts/smoke.py` starts a temporary local callback
receiver and verifies creation, lookup, execution history and disabling. If you
change `.env` passwords, also update `config.toml` or use environment overrides:
`SCHEDULETASK_POSTGRES_DSN`, `SCHEDULETASK_REDIS_ADDR`,
`SCHEDULETASK_REDIS_PASSWORD`, and `SCHEDULETASK_PORT`. A config file is always
required; overrides replace existing keys. See [configuration](docs/operations.md#configuration).

```sh
make check
export SCHEDULETASK_TEST_POSTGRES_DSN='host=localhost user=scheduletask password=scheduletask dbname=schedule_task port=5432 sslmode=disable'
export SCHEDULETASK_TEST_REDIS_ADDR=localhost:6379
export SCHEDULETASK_TEST_REDIS_PASSWORD=scheduletask
make integration
```

Without test connection variables, database/Redis tests explicitly skip; CI sets
them and runs the full race-enabled suite. Use a disposable database and Redis
instance. Integration tests create isolated PostgreSQL schemas and unique Redis
keys, then clean them up.

## Repository layout

```text
cmd/scheduletask/       executable entry point
api/proto/, api/pb/     gRPC contract and generated Go client/server types
internal/app/          dependency wiring and service lifecycle
internal/config/       validated configuration
internal/core/         models, provider contracts, scheduling and execution
internal/infra/        PostgreSQL and Redis clients
internal/server/       HTTP and gRPC adapters
docs/                  API, architecture, operations and roadmap
examples/, scripts/    callback example and end-to-end smoke check
.github/               CI, dependency updates, issue and PR templates
```

Dependencies are passed through constructors. Runtime packages stay inside
`internal`; the protobuf package remains available to external Go clients.

## Scope and roadmap

Current functionality includes validated create/enable/disable, HTTP/gRPC lookup,
HTTP execution history, PostgreSQL task deduplication and atomic execution claims,
Redis lock ownership, callback cancellation/timeouts, health checks and Prometheus
metrics. Only PostgreSQL plus Redis is supported.

Follow-up priorities are authenticated access, callback egress policy, retries and
crash recovery, retention, schema migration tooling and load testing. Track scope
and compatibility notes in the [roadmap](docs/roadmap.md) and [changelog](CHANGELOG.md).

Contributions are welcome under the [MIT license](LICENSE). Read
[CONTRIBUTING.md](CONTRIBUTING.md), [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) and
[SECURITY.md](SECURITY.md) before opening a report or pull request.
