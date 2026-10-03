# Contributing

Start with the README and architecture document. Search existing issues before
opening a new report. Discuss interface changes or large features in an issue;
small fixes can arrive as a focused pull request.

## Development workflow

1. Fork the repository, clone your fork and create a descriptive branch.
2. Start PostgreSQL and Redis with `docker compose up -d --wait postgres redis`.
3. Copy `config.example.toml` to `config.toml`. Never commit credentials or `.env`.
4. Add a regression test for behavior changes, implement the change, and run
   `make check` plus the integration commands in the README.
5. For scheduling/API changes, run `python3 scripts/smoke.py` against the local
   binary. For container changes, run the complete Compose demo stack.
6. Describe the problem, resulting behavior and verification in your PR. Keep
   unrelated refactoring in separate changes and update relevant docs.

`make test` runs the race detector. Integration tests skip when their connection
variables are absent; `make integration` requires explicit test infrastructure.
Test schemas and keys are temporary. Do not point these tests at production.

## Go and protocol conventions

Use `gofmt`, pass context to I/O, return initialization errors, and inject dependencies
through constructors. Keep domain behavior in `internal/core`, transport concerns
in `internal/server`, and wiring in `internal/app`. Avoid new global clients or
runtime configuration singletons. Tests should assert observable behavior rather
than implementation text.

Generated files in `api/pb` should be regenerated, not hand edited. Install the
Protocol Buffers compiler (`protoc`) and the pinned plugins:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
export PATH="$(go env GOPATH)/bin:$PATH"
make proto
```

The proto uses `go_package = "./pb"`; `make proto` writes into `api/pb`. Generated
headers record compiler/plugin versions. Go users import
`github.com/Gopher0727/ScheduleTask/api/pb`.

## Reports and review

Bug reports should include the version/commit, config with secrets removed,
reproduction, expected result, actual result and relevant logs. Use the feature
template for proposals. Security reports follow SECURITY.md rather than public
issue templates.

CI must pass before merge. Maintainers review API compatibility, correctness,
test coverage and operational effects. By submitting a contribution, you agree
that it is distributed under this repository's MIT license.
