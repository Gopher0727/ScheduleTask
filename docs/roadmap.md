# Roadmap

The current goal is a reproducible, reviewable open-source development baseline.
There is no promised release date or production certification.

## Next reliability work

- Define callback idempotency keys, task leases, crash recovery and retry/backoff.
- Design configurable missed-execution handling for extended downtime.
- Add bounded parallel callback workers and benchmarks before making capacity claims.
- Add retention and versioned database migrations with rollback guidance.

## Access and integration

- Authenticated HTTP/gRPC access and a clear tenant model.
- Callback destination/redirect policies and protected TLS deployments.
- Timer listing, editing/deletion and paginated task history when lifecycle semantics
  and compatibility requirements are agreed.
- Release binaries/images and provenance once CI and versioning are established.

Use GitHub issues to discuss a concrete problem and acceptance criteria. Avoid
adding alternate databases before the supported PostgreSQL + Redis combination
has sufficient operational coverage.
