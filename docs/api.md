# API

Default HTTP address is `http://localhost:8082`; gRPC uses port 8083. Access is
unauthenticated and intended for trusted callers only. `app` does not provide
tenant isolation. HTTP JSON requests are limited to 1 MiB.

## HTTP endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/xtimer/createTimer` | Create a disabled timer |
| POST | `/xtimer/enableTimer` | Enable (`1`) or disable (`0`) |
| GET | `/xtimer/getTimer?timer_id=1` | Retrieve a timer |
| GET | `/xtimer/getTasks?timer_id=1&limit=20` | Most recent due execution records |
| GET | `/healthz` | Process liveness |
| GET | `/readyz` | PostgreSQL and Redis connectivity |
| GET | `/metrics` | Prometheus metrics |

Timer API responses use `{"code":0,"message":"ok","data":...}`. Invalid input
returns HTTP 400 / code 90003; store failures return HTTP 500 / code 90002.
Lookup returns 404 for a missing timer. Enable requires matching `app` and returns
403 for a mismatch; this is consistency validation, not authentication.

### Create

```json
{
  "app": "demo",
  "name": "daily-report",
  "cron": "0 9 * * *",
  "notify_http_param": {
    "method": "POST",
    "url": "https://example.com/callback",
    "header": {"Content-Type": "application/json"},
    "body": "{\"event\":\"report\"}"
  }
}
```

`app` and `name` must be non-empty. Cron uses five fields or a standard descriptor,
including `@every`. The callback must have an absolute HTTP(S) URL and method POST
(case-insensitive). No seconds field is accepted in a standard cron expression.
Cron time uses the process time zone; `CRON_TZ=...` prefixes are supported by the
cron parser. `@every` intervals are anchored to the creation timestamp, rounded
to whole seconds. An impossible date has no executions and never loops forever.

Successful response: `{"code":0,"message":"ok","data":{"timer_id":1}}`.

### Enable or disable

```json
{"app":"demo","timer_id":1,"status":1}
```

New timers start disabled. Materialization discovers enabled timers at startup
and once per minute; discovery may take up to a minute, followed by the next
scheduled occurrence. For an every-minute timer, allow up to two minutes. Disabling
prevents later callbacks when executors recheck timer state. An already started
callback may finish. Re-enabling retains the original interval phase.

### Timer and execution lookup

`getTimer` returns ID, app, name, cron, status and the callback object. `getTasks`
returns a JSON array, sorted by `run_timer` descending, excluding future planned
occurrences. Its default limit is 20, range 1–100. Unknown timer history is an
empty array.

Task fields include `task_id`, `timer_id`, `app`, `run_timer` (Unix milliseconds),
`status`, `output`, `cost_time` (start lateness in milliseconds), `create_time` and
`update_time`. Status values are `0` pending, `1` succeeded, `2` failed, `3` running.
HTTP 2xx callback results succeed; other codes, network errors and timeouts fail.
Output stores the response body or error string. Responses over 1 MiB fail.
There are currently no automatic retries.

### Health and metrics

Liveness returns `{"status":"ok"}`. Readiness returns `{"status":"ready"}`
with HTTP 200, or `{"status":"unavailable"}` with HTTP 503 when a dependency
check fails. Checks have a two-second budget. Readiness does not certify cron
progress, task backlog, or delivery guarantees.

Metrics use the `scheduletask_` prefix: task execution counts by outcome,
execution duration, scheduler scans, migration runs, lock acquisition and failure.
Lock failure includes normal contention as well as backend errors.

## gRPC

The contract is [api/proto/xtimer.proto](../api/proto/xtimer.proto); generated Go
types are in [api/pb](../api/pb). Service `xtimer.XTimer` has `CreateTimer`,
`EnableTimer`, and `GetTimer`. Fields match the HTTP create/enable contracts.
Validation failures use `InvalidArgument`, missing timers use `NotFound`, and
app mismatch uses `PermissionDenied`.

With `grpcurl` installed, provide the proto because reflection is not enabled:

```sh
grpcurl -plaintext -import-path api/proto -proto xtimer.proto \
  -d '{"timer_id":"1"}' localhost:8083 xtimer.XTimer/GetTimer
```

There is currently no gRPC execution-history method.
