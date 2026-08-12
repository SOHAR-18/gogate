# GoGate — Production-minded API Gateway in Go

[![Go Version](https://img.shields.io/badge/go-1.26.3-blue.svg)]()
[![License](https://img.shields.io/badge/license-MIT-green.svg)]()
[![Build Status](https://img.shields.io/badge/build-pending-lightgrey.svg)]()

Table of contents
- [What this project is](#what-this-project-is)
- [Goals & Audience](#goals--audience)
- [Feature summary](#feature-summary)
- [Architecture — components & diagrams](#architecture---components--diagrams)
  - [High-level component diagram (ASCII)](#high-level-component-diagram-ascii)
  - [Sequence: request lifecycle](#sequence-request-lifecycle)
- [Code layout & responsibilities](#code-layout--responsibilities)
- [Design details (deep dive)](#design-details-deep-dive)
  - [Routing & proxying behavior](#routing--proxying-behavior)
  - [Load balancing & health checks](#load-balancing--health-checks)
  - [Circuit breaker & retries](#circuit-breaker--retries)
  - [Authentication & authorization](#authentication--authorization)
  - [Rate limiting](#rate-limiting)
  - [Service discovery](#service-discovery)
  - [Observability (metrics, traces, logs)](#observability-metrics-traces-logs)
  - [Admin & runtime configuration](#admin--runtime-configuration)
- [Configuration reference (env & files)](#configuration-reference-env--files)
- [Quickstart — run locally and with Docker Compose](#quickstart---run-locally-and-with-docker-compose)
- [Examples — requests and expected responses](#examples---requests-and-expected-responses)
- [Prometheus & tracing minimal configs](#prometheus--tracing-minimal-configs)
- [Testing and CI recommendations](#testing-and-ci-recommendations)
- [Production hardening & operational guidance](#production-hardening--operational-guidance)
- [Troubleshooting & common errors](#troubleshooting--common-errors)
- [Extensibility & roadmap](#extensibility--roadmap)
- [Contributing, style & commit conventions](#contributing-style--commit-conventions)
- [License & contact](#license--contact)

---

## What this project is

GoGate is a production-minded API Gateway implemented in Go. It demonstrates patterns and components you would expect in a modern gateway: routing to multiple backend services, authentication (JWT/API key), Redis-backed rate limiting, round-robin load balancing with health checks, per-route circuit breakers, service discovery (etcd), Prometheus metrics and OpenTelemetry traces, and an admin surface for runtime inspection and route management.

It includes:
- A gateway binary (cmd/gateway) — the main entrypoint handling ingress traffic
- Three example backend services (cmd/user-service, cmd/product-service, cmd/order-service)
- Internal packages implementing proxying, load balancing, circuit breakers, middleware, and discovery

This project is suitable for:
- Developers learning how gateways and microservice ingress work
- Teams prototyping routing and resilience behavior for microservices
- Reference material for implementing best-practice gateway features in Go

---

## Goals & Audience

Primary goals:
- Provide a clear, well-structured reference implementation of an API gateway
- Show how real libraries (chi, viper, go-redis, prometheus, opentelemetry, etcd client, gobreaker) are composed
- Make runtime behavior observable and administrable (health, metrics, traceability)
- Be extendable: middleware chain, alternative load-balancers, different discovery backends

Audience:
- Backend engineers learning gateway patterns
- Students and instructors for production-ready Go infrastructure
- DevOps engineers evaluating patterns to apply to in-house gateways

---

## Feature summary

- Path-prefix based routing to multiple upstreams
- Reverse proxy with request rewriting (strip prefix, forwarding headers)
- Per-route round-robin load-balancer with health checking
- Circuit breaker per-route (open/close with configurable thresholds)
- Middleware chain: request ID, logging, auth, rate-limiting, metrics, tracing
- Redis-backed rate limiting (sliding window/token-bucket primitives)
- Service discovery using etcd (pull/upstream registration)
- Admin API for route CRUD, status, and metrics
- Prometheus metrics + OpenTelemetry tracing integration
- Makefile for quick dev tasks and docker-compose manifests for full local stack

---

## Architecture — components & diagrams

High level, the system looks like:

- External clients → GoGate (ingress) → backend services (user/product/order)
- External infra (Redis, etcd, Prometheus, OTLP collector) run alongside (docker/k8s)

### High-level component diagram (ASCII)

    +------------+         +-------------------+           +---------------------+
    |  Client(s) |  ---->  |     GoGate        |   ---->   |   Backend services  |
    | (curl/browser)       |  (gateway process) |           |  user/product/order |
    +------------+         +-------------------+           +---------------------+
                                |    |   |   |
                                |    |   |   +--> admin API (runtime)
                                |    |   +------> Prometheus (/metrics)
                                |    +---------> OpenTelemetry (OTLP)
                                +-----------> Redis (rate-limits)
                                             etcd (service discovery)

### Sequence: request lifecycle (detailed)

1. Client issues HTTP request to gateway.
2. Listener receives request on gateway port.
3. Request ID middleware ensures requests carry X-Request-ID (new or propagated).
4. Logging middleware logs incoming request metadata.
5. Auth middleware checks JWT or API key; reject (401) on failure.
6. Rate-limit middleware consults Redis to allow/deny the request (429 on deny).
7. Observability middleware starts a trace span.
8. Router matches path prefix → finds route configuration and upstream list.
9. Load balancer selects a healthy upstream instance (round-robin).
10. Circuit breaker checks route state; if open, return 503/short-circuit.
11. Reverse proxy sets header context and forwards request to upstream instance via httputil.ReverseProxy with per-request timeout.
12. Response flows back; gateway annotates response headers (X-Served-By, X-Upstream, X-Request-ID), records metrics, and completes trace.

---

## Code layout & responsibilities

Top-level (important parts only):

```
cmd/
  gateway/                # main application: bootstrap, router, admin
  user-service/           # simple backend service (example)
  product-service/        # simple backend service (example)
  order-service/          # simple backend service (example)

internal/
  proxy/                  # Reverse proxy & Upstream model (proxy.go, upstream.go)
  loadbalancer/           # RoundRobin + health checker
  circuitbreaker/         # Manager & per-route breakers
  middleware/             # Middleware chain (logging, auth, rate-limit, tracing)
  auth/                   # JWT & API-key helpers
  ratelimit/              # Redis helpers and limiter primitives
  discovery/              # etcd discovery / registration
  config/                 # configuration loading (viper)
  admin/                  # admin REST handlers + dashboard
pkg/
  logger/ metrics/ tracing # shared helpers; instrumentation
Makefile
go.mod
deployments/docker/       # docker-compose manifests to run full stack locally
```

Files to inspect (start here):
- cmd/gateway/main.go — app bootstrap and server lifecycle
- internal/proxy/proxy.go — reverse proxy mechanics, error handling, header management
- internal/loadbalancer/* — balancer and health checking logic
- internal/circuitbreaker/* — circuit breaker manager and wrappers
- internal/middleware/* — chain construction and middleware implementations

(You can find proxy implementation in internal/proxy/proxy.go and the module dependencies in go.mod.)

---

## Design details (deep dive)

This section explains why things are implemented the way they are and how to extend or modify behavior.

### Routing & proxying behavior
- Routing is path-prefix based (e.g. /users → user-service).
- For each route, configuration controls:
  - Upstreams (host:port or URL)
  - Health check path
  - Timeout
  - StripPrefix flag
- The ReverseProxy wraps net/http/httputil.ReverseProxy and customizes:
  - Director: to set scheme/host, strip prefix, and set custom headers (X-Forwarded-By, X-Original-Path)
  - ErrorHandler: marks upstream unhealthy and returns a 502/504
  - Timeouts: each upstream call runs with a context timeout to avoid hung requests

Why: Using the standard ReverseProxy gives robust, battle-tested proxy semantics while letting us control behavior via Director and ErrorHandler.

### Load balancing & health checks
- Default strategy: RoundRobin. The balancer maintains a ring of upstreams and returns the next healthy instance.
- HealthChecker: background goroutine that periodically requests the health path and toggles the healthy flag for each instance.
- Instances marked unhealthy are skipped by the balancer. When they recover, they are reincorporated.

Extensibility: add least-connections or weighted balancer by implementing the same interface as RoundRobin.

### Circuit breaker & retries
- Circuit breaker library is used as a wrapper for outbound proxy calls per-route.
- Parameters configurable: failure threshold, timeout before half-open, success counts, and metrics integration.
- Behavior: when the breaker is open, requests get short-circuited with 503 and metric increments; on half-open, a probe request will attempt to re-enable.

Retries: Exponential backoff with a capped retry count is recommended for idempotent requests — implemented as an optional middleware or inside proxy wrappers, not enabled by default for non-idempotent verbs.

### Authentication & authorization
- Supports JWT tokens and API keys (configurable).
- JWT: uses signing secret / public key configured via env or config file. Validate algorithms (prefer RS256 for production).
- API keys: a store (in-memory or Redis-backed) maps keys to allowed scopes. Admin endpoints should only be callable by admin-scoped tokens.

Best practice: Put auth as an early middleware in the chain to avoid wasted downstream work.

### Rate limiting
- Redis-backed token-bucket or sliding-window implementation.
- Rate limits configurable per-route or per-client (API key / IP).
- Failure mode: missing Redis → gateway can either fail-closed (deny all) or fail-open (allow all); default safe approach is fail-open for availability with an alert.

Tuning: set Redis expiry keys appropriately and ensure Redis is scaled for expected QPS.

### Service discovery
- Discovery uses etcd client to register/listen for upstream instances.
- Optionally, static configuration in configs/ is supported for simpler setups or demos.

Flow:
- Instances register themselves (e.g., service name and address) in etcd with a TTL lease.
- Gateway watches key prefixes to update route upstream lists automatically.

### Observability (metrics, traces, logs)
- Metrics: Prometheus client for counters, histograms, and gauges:
  - http_requests_total{route,method,status}
  - http_request_duration_seconds{route}
  - upstream_health_status{route,instance}
  - circuit_breaker_state{route}
- Tracing: OpenTelemetry spans around inbound request and outbound HTTP calls. Configure OTLP exporter to send traces to an OTEL/Jaeger collector.
- Logs: structured logs include request ID, timestamp, route, upstream, latency, status code.

Expose: /metrics for Prometheus and admin endpoints for status. Ensure OTLP endpoint reachable for traces.

### Admin & runtime configuration
Admin API enables:
- List, add, remove routes at runtime
- View upstream health and circuit breaker states
- Force-remove or drain instances
- Inspect metrics or admin logs

Admin API must be protected (API key or JWT with admin claim). Use RBAC for multi-operator environments.

---

## Configuration reference (env & files)

The gateway reads configuration from environment variables, .env files, or a configuration YAML via Viper.

Recommended environment variables (defaults shown where applicable):

- GATEWAY_PORT=8080
  - Port for main gateway HTTP listener.
- ADMIN_PORT=9090
  - Port for admin API and dashboard.
- ROUTES_CONFIG_PATH=configs/routes.yaml
  - Path to static routes config (if not using etcd).
- REDIS_HOST=redis
- REDIS_PORT=6379
- REDIS_PASSWORD=
- ETCD_ENDPOINTS=http://etcd:2379
  - Comma-separated list if multiple endpoints.
- JWT_SECRET=   (or JWT_PUBLIC_KEY_PATH=)
- ADMIN_API_KEY=supersecretadminkey
- PROMETHEUS_METRICS_PORT=9092
- OTLP_ENDPOINT=http://otel-collector:4318
- LOG_LEVEL=info
- RATE_LIMIT_DEFAULT=100/min  (example shorthand)

Sample YAML (configs/routes.yaml):

```yaml
routes:
  - path: /users
    strip_prefix: true
    timeout_ms: 5000
    health_path: /health
    upstreams:
      - http://user-service:8081
      - http://user-service-2:8081
  - path: /products
    strip_prefix: true
    timeout_ms: 5000
    health_path: /health
    upstreams:
      - http://product-service:8082
```

Use `viper.AutomaticEnv()` and `viper.ReadInConfig()` to combine env and file-based configuration.

---

## Quickstart — run locally and with Docker Compose

Prerequisites:
- Go (>= 1.26)
- Docker & Docker Compose (for the full stack)
- Make (optional)

1) Clone and enter repo:
```bash
git clone https://github.com/SOHAR-18/gogate.git
cd gogate
```

2) Run gateway only (developer flow):
```bash
# ensure env values are set (create a .env or export them)
make run
# or
go run cmd/gateway/main.go
```

3) Full demo (recommended — runs Redis, etcd, Prometheus, example services):
```bash
make docker-up
# Wait until services are healthy, then:
curl http://localhost:8080/          # gateway root
curl http://localhost:8080/users/1   # proxied to user-service
# stop:
make docker-down
```

If your deployments/docker/docker-compose.yml is missing or you want a minimal compose for demo, create one like:

```yaml
version: "3.8"
services:
  redis:
    image: redis:7
    ports: ["6379:6379"]

  etcd:
    image: quay.io/coreos/etcd:v3.7
    command: ["etcd","--advertise-client-urls","http://0.0.0.0:2379","--listen-client-urls","http://0.0.0.0:2379"]
    ports: ["2379:2379"]

  user-service:
    build: ./cmd/user-service
    ports: ["8081:8081"]

  product-service:
    build: ./cmd/product-service
    ports: ["8082:8082"]

  order-service:
    build: ./cmd/order-service
    ports: ["8083:8083"]

  gateway:
    build: ./cmd/gateway
    ports: ["8080:8080","9090:9090"]
    environment:
      - REDIS_HOST=redis
      - ETCD_ENDPOINTS=http://etcd:2379
    depends_on: ["redis","etcd","user-service","product-service","order-service"]
```

Note: in production use specialized images and avoid building locally on production hosts.

---

## Examples — requests and expected responses

Assume gateway listens on port 8080 with default route prefixes.

1) Health / root:
```bash
curl -v http://localhost:8080/
# Response headers include X-Request-ID and X-Served-By
```

2) Forwarded request:
```bash
curl -v http://localhost:8080/users/42
# gateway forwards to user-service, returns the user's payload
# Example success response:
# HTTP/1.1 200 OK
# X-Request-ID: 1234abcd
# X-Upstream: http://user-service:8081
```

3) Admin: list routes (authenticated with ADMIN_API_KEY header)
```bash
curl -H "X-Admin-Key: supersecretadminkey" http://localhost:9090/admin/routes
# Response:
# [
#   {"path":"/users","upstreams":["http://user-service:8081"],"timeout_ms":5000,...},
#   ...
# ]
```

4) Add a route:
```bash
curl -X POST -H "Content-Type: application/json" -H "X-Admin-Key: supersecretadminkey" \
  -d '{"path":"/payments","strip_prefix":true,"upstreams":["http://payments:8085"],"health_path":"/health","timeout_ms":5000}' \
  http://localhost:9090/admin/routes
```

---

## Prometheus & tracing minimal configs

Prometheus scrape job example (prometheus.yml snippet):

```yaml
scrape_configs:
  - job_name: 'gogate'
    metrics_path: '/metrics'
    static_configs:
      - targets: ['gateway:9092']   # or localhost:9092
```

OpenTelemetry:
- Configure OTLP endpoint (OTLP/HTTP or gRPC) via OTLP_ENDPOINT env var.
- Example: use `otel/opentelemetry-collector` or Jaeger backend for local testing.

---

## Testing and CI recommendations

Local:
```bash
# Unit tests
go test ./...

# Integration (requires docker)
make docker-up
# run ./tests/integration/... or a harness
make docker-down
```

CI pipeline (example steps):
1. go vet ./...
2. go test ./... (with race detector if compiled)
3. golangci-lint run
4. Build artifact: go build -o bin/gateway cmd/gateway/main.go
5. (optional) Build Docker image and push to registry
6. (optional) Run a small integration smoke test in ephemeral environment

Add GitHub Actions or other CI to run the above on PRs.

---

## Production hardening & operational guidance

- TLS: Terminate TLS at load balancer (ALB/Ingress) or add TLS support to gateway (prefer cert-manager/Ingress in k8s).
- Secrets: do not store JWT_SECRET or passwords in repo. Use Kubernetes Secrets, Vault, or cloud KMS.
- Redundancy: run multiple gateway replicas behind a front load-balancer/load balancer and share state (e.g., use etcd or central config).
- Rate limiting: scale Redis appropriately or use cloud-managed rate-limiters for very high QPS.
- Health & readiness: provide k8s readiness and liveness endpoints and configure proper thresholds.
- Autoscaling: use k8s HPA with CPU/memory or custom metrics (request latency).
- Rollouts: use canary deploys / blue/green for gateway changes; admin API can help in runtime route migration.

---

## Troubleshooting & common errors

- 502 / 504: upstream service unavailable; check upstream container logs and health endpoint.
- 429: rate limit hit. Verify client identity and rate-limit settings; check Redis connectivity.
- 401: invalid JWT or missing API key. Verify token claims, issuer and signature (use correct secret/key).
- High latency: inspect gateway metrics (request_duration_seconds), check backend p99 timings and network.
- Circuit-breaker open frequently: investigate upstream health and consider adjusting thresholds (or fix upstream failures).

Useful debugging:
```bash
docker compose -f deployments/docker/docker-compose.yml logs -f gateway
curl -v -H "Authorization: Bearer $TOKEN" http://localhost:8080/users/1
```

---

## Extensibility & roadmap

Possible next steps / extension points:
- Plugin system to load middleware dynamically
- Weighted or least-connections balancer
- Native TLS termination and certificate hot-reload
- OAuth2/OIDC integration for richer auth flows
- Rate-limit policies persisted / per-tenant quotaing
- Policy-as-code integration for custom routing decisions

---

## Contributing, style & commit conventions

- Fork -> branch -> PR
- Commit message format: Conventional Commits (e.g., `feat: add weighted balancer`, `fix(proxy): handle headers correctly`)
- Tests: add unit tests for logic; add integration test for end-to-end behavior when changing routing/proxying code
- Formatting: `gofmt` / `goimports` — please run `gofmt -w .` before committing

Suggested PR checklist:
- [ ] Unit tests added or updated
- [ ] Integration smoke test (if relevant)
- [ ] `go vet` / `golangci-lint` pass
- [ ] Documentation updated (README, inline comments)

---

## License & contact

This project does not include a license file by default. Add an appropriate license (MIT, Apache-2.0, etc.). If you'd like a LICENSE file added automatically, tell me which license you want and I will add it.

Maintainer / contact:
- Repository: https://github.com/SOHAR-18/gogate
- Open an issue for questions, feature requests, or bugs.

---

Appendix — quick reference snippets

Admin route JSON schema (example):

```json
{
  "path": "/payments",
  "strip_prefix": true,
  "timeout_ms": 5000,
  "health_path": "/health",
  "upstreams": ["http://payments-1:8085", "http://payments-2:8085"]
}
```

Sample .env for local development:

```bash
GATEWAY_PORT=8080
ADMIN_PORT=9090
REDIS_HOST=redis
REDIS_PORT=6379
ETCD_ENDPOINTS=http://etcd:2379
JWT_SECRET=changeme
ADMIN_API_KEY=adminkey123
LOG_LEVEL=debug
PROMETHEUS_METRICS_PORT=9092
OTLP_ENDPOINT=http://otel-collector:4318
```

---

