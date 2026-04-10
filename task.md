# Task: Fork rollouts-demo with Prometheus metrics

## Context

The upstream `argoproj/rollouts-demo` app (235-line Go HTTP server) has zero
observability.  We need pod-level request metrics so that KEDA can scale the
Argo Rollout based on true RPS (external + internal traffic), not just CPU.

The forked image will be published to `ghcr.io/yourdevops/rollouts-demo` and
used in the canary-demo Helm chart in `infra-k3s-contabo`.

## Upstream repo

https://github.com/argoproj/rollouts-demo (Go, last updated 2024-07-23)

Source entry point: `main.go` — plain `net/http`, two routes (`/`, `/color`).

## What to do

### 1. Add Prometheus instrumentation to `main.go`

Add `github.com/prometheus/client_golang` as a dependency.

Register a `/metrics` endpoint:

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

router.Handle("/metrics", promhttp.Handler())
```

Wrap the existing handler with a counter + duration middleware.  The minimum
metrics to expose:

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `http_requests_total` | Counter | `method`, `path`, `status_code` | RPS for KEDA scaling |
| `http_request_duration_seconds` | Histogram | `method`, `path` | Latency percentiles |

Implementation approach — use `promhttp.InstrumentHandlerCounter` and
`promhttp.InstrumentHandlerDuration`, or define custom counters and increment
them in a middleware wrapper around the `router`.  The `/metrics` endpoint
itself should NOT be counted.

Keep the existing `COLOR`, `LATENCY`, `ERROR_RATE` env-var behavior unchanged.

### 2. Verify the Dockerfile builds

The upstream Dockerfile is a multi-stage build.  After adding the prometheus
dependency, make a local `docker build` to confirm it still works.  Run
`go mod tidy` before building.

### 3. GitHub Actions workflow for GHCR

**DONE** — see `.github/workflows/build-push.yaml`

- Trigger on push to `main` and `workflow_dispatch`
- Build multi-platform (`linux/amd64`, `linux/arm64`) via `docker/build-push-action@v6`
- Push to `ghcr.io/yourdevops/rollouts-demo` (public images, uses `GITHUB_TOKEN`)

**Layer-reuse optimization — two-job strategy:**

The Dockerfile uses `ARG`/`ENV` at the very bottom (lines 14-19). Everything
above (Go compilation, asset copies) is identical across all 18 variants.
A naive matrix would rebuild Go 18 times.

Instead:

1. **`build-base` job** — builds the `blue` variant with
   `cache-to: type=gha,mode=max`, warming the GHA BuildKit cache with all
   shared layers (Go binary + static assets).
2. **`build-variants` job** (`needs: build-base`) — matrix of the remaining 17
   variants. Each reads the warm cache via `cache-from: type=gha`. Only the
   ENV metadata layers differ, so each variant build takes seconds.

Color variants built (18 total):
`blue`, `green`, `yellow`, `orange`, `purple`, `red`,
`bad-{color}` (COLOR + ERROR_RATE=15),
`slow-{color}` (COLOR + LATENCY=2)

The Dockerfile already supports `COLOR`, `ERROR_RATE`, `LATENCY` as build
args — no changes needed to the Dockerfile for variant builds.

### 4. Verify metrics locally

```bash
docker run -p 8080:8080 -e COLOR=blue ghcr.io/yourdevops/rollouts-demo:blue
curl localhost:8080/color          # generates traffic
curl localhost:8080/metrics        # should show http_requests_total, http_request_duration_seconds
```

## Acceptance criteria

- [ ] `GET /metrics` returns Prometheus text format with `http_requests_total` and `http_request_duration_seconds`
- [ ] Existing `/` and `/color` routes behave identically to upstream
- [ ] `COLOR`, `LATENCY`, `ERROR_RATE` env vars still work
- [ ] GitHub Actions two-job workflow builds all 18 variants with layer cache reuse
- [ ] Image runs on both `amd64` and `arm64`