# Security policy

This is an experimental service. Security fixes target the latest default-branch
code; no older release support or response-time guarantee has been established.

## Reporting

Use [GitHub private vulnerability reporting](https://github.com/Gopher0727/ScheduleTask/security/advisories/new)
if it is enabled. Otherwise contact the [maintainer](https://github.com/Gopher0727)
and request a private reporting channel before sharing exploitation details.
Include affected commit, reproduction, impact and any proposed fix. Do not put
credentials, private data or exploit details in public issues.

## Current trust boundary

The service has no built-in authentication or TLS for HTTP/gRPC. `app` is a logical
grouping field, not an identity or authorization mechanism. APIs can read callback
headers/bodies and task output; access must be restricted to trusted callers.
Callbacks can reach any HTTP(S) destination reachable from the scheduler, including
internal services. There is no destination allowlist or redirect policy.

The development Compose stack publishes ports on loopback. Shared deployments
need authenticated access, protected networking, TLS, dedicated credentials,
an egress policy, and a dedicated Redis database. Response bodies are limited to
1 MiB and callbacks have a ten-second timeout, but these controls do not replace
an access or egress policy.
