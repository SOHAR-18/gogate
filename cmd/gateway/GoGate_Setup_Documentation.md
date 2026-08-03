# GoGate — Complete Setup Documentation
### API Gateway & Microservices Orchestrator
**Your progress so far · Phase 0: Environment Setup Complete ✅**

---

## What Is GoGate?

GoGate is a production-grade **API Gateway** you are building from scratch in Go. Think of it as your own mini version of Kong or AWS API Gateway — a single entry point that sits in front of all your backend services and handles:

- Routing requests to the right service
- Authenticating every request (JWT / API keys)
- Rate limiting so no one abuses your API
- Load balancing across multiple instances
- Circuit breaking when a service is down
- Metrics, tracing, and logging for full visibility

By the end of 10 phases, you will have a fully working system that companies like Netflix, Uber, and Airbnb use in production — built by you, understood by you, from zero.

---

## Why Go (Golang)?

Before anything else — why did we choose Go for this project?

**Go is the language of infrastructure.** Docker, Kubernetes, Terraform, Prometheus, and Grafana are all written in Go. When you build a gateway, you need a language that is:

- **Fast** — Go compiles to native machine code, not bytecode
- **Concurrent** — Go has goroutines, which are lightweight threads built into the language. Running 10,000 simultaneous requests is trivial in Go
- **Simple** — Go has very few concepts to learn. No classes, no inheritance, no magic
- **Production-proven** — Go powers Cloudflare, Dropbox, and most of Google's infrastructure

For a project like GoGate — handling thousands of requests per second, running background health checks, managing connection pools — Go is the perfect choice.

---

## What We Did: Step by Step

### Step 1 — Installed Go 1.22

**What we did:** Installed the Go programming language on Windows.

**Why it matters:** Go is the entire foundation of GoGate. Every file we write — the gateway engine, the middleware, the auth system — is Go code. Without Go installed, nothing compiles or runs.

**What it gave us:**
- The `go` command in PowerShell
- The Go compiler that turns our code into a running program
- The Go standard library — thousands of built-in tools we use for HTTP, networking, cryptography, and more

**How to verify it works:**
```
go version
→ go version go1.22.x
```

---

### Step 2 — Installed Docker Desktop

**What we did:** Installed Docker on Windows.

**Why it matters:** GoGate depends on several external services — Redis (for rate limiting and caching), etcd (for service discovery), Prometheus (for metrics), Grafana (for dashboards), and Jaeger (for tracing). Installing all of these directly on your machine would be messy and hard to manage.

Docker solves this by running each service in a **container** — a lightweight, isolated box. You can spin up Redis with one command and tear it down just as easily. Your machine stays clean.

**What it gave us:**
- The `docker` command to run containers
- The `docker compose` command to run multiple containers together
- A way to run our entire GoGate infrastructure locally with `make docker-up`

**Think of Docker like this:** Instead of cooking in your kitchen and making a mess, you order from a restaurant. Each service (Redis, Prometheus, etc.) is a restaurant. Docker is the delivery app. You get exactly what you need, and the kitchen stays clean.

**How to verify it works:**
```
docker --version
→ Docker version 25.x.x

docker compose version
→ Docker Compose version v2.x.x
```

---

### Step 3 — Installed Git & Created GitHub Repository

**What we did:** Installed Git, configured our name/email, and created the `gogate` repository on GitHub.

**Why it matters:** Git is version control — it tracks every change you make to your code. GitHub hosts your code online so it's backed up, shareable, and showcaseable on your resume.

**What it gave us:**
- The `git` command for tracking changes
- A remote GitHub repository at `github.com/SOHAR-18/gogate`
- A professional `main` branch (the modern standard)

**For your resume:** Having the full project on GitHub with proper commits shows employers the entire development journey — not just the final code. They can see you built this from scratch.

**Key Git commands you used:**
```
git init          — start tracking a folder
git add .         — stage all changes
git commit -m ""  — save a snapshot with a message
git push          — upload to GitHub
```

---

### Step 4 — Installed VS Code + Go Extension

**What we did:** Installed Visual Studio Code and the official Go extension from Google.

**Why it matters:** You could write Go in Notepad — but you'd be miserable. VS Code with the Go extension gives you:

- **Autocomplete** — it suggests code as you type, including function names and parameters
- **Error highlighting** — red underlines show mistakes before you even run the code
- **Auto-formatting** — every time you save, your code is perfectly formatted automatically (this is `gofmt`, Go's formatter)
- **Inline documentation** — hover over any function to see what it does
- **Integrated debugger** — step through code line by line when something goes wrong

**Extensions installed:**
- `Go` by Go Team at Google — the core language support
- `Docker` by Microsoft — see your containers inside VS Code
- `REST Client` — test your API endpoints without leaving the editor
- `GitLens` — advanced Git history and blame
- `Error Lens` — shows errors inline next to the broken line

**VS Code setting we added:**
```json
"editor.formatOnSave": true
```
This auto-formats every Go file on save. In Go, formatting is not optional — it's enforced by the language community. This setting means you never have to think about it.

---

### Step 5 — Created the Project Folder & Initialized Go Module

**What we did:** Created the `gogate` folder at `C:\Users\rosha\gogate` and ran `go mod init`.

**Why it matters:** `go mod init` creates `go.mod` — the project manifest. This is Go's equivalent of `package.json` in Node.js or `requirements.txt` in Python. It tells Go:
- What this project is called (`github.com/SOHAR-18/gogate`)
- What version of Go we're using
- What external packages we depend on

**Files created:**
- `go.mod` — the project manifest (what the project is and what it depends on)
- `go.sum` — a security file that locks exact versions of every dependency (auto-generated)

**Why the module name matters:** We named it `github.com/SOHAR-18/gogate`. This matches our GitHub URL. When we import code between our own packages (e.g., importing the auth package into the gateway), Go uses this name as the base path.

---

### Step 6 — Created the Full Folder Structure

**What we did:** Created 19 folders inside `gogate` using PowerShell.

**Why it matters:** A well-organized folder structure is the difference between a project that scales and one that becomes a mess. We used the **standard Go project layout** — the same structure used by Docker, Kubernetes, and every major Go project.

**The structure and what each folder is for:**

```
gogate/
│
├── cmd/                        ← Entry points (runnable programs)
│   ├── gateway/                ← The main API gateway (our core program)
│   │   └── main.go
│   ├── user-service/           ← Dummy user microservice
│   ├── product-service/        ← Dummy product microservice
│   └── order-service/          ← Dummy order microservice
│
├── internal/                   ← Core business logic (private to this project)
│   ├── proxy/                  ← Reverse proxy engine (Phase 2)
│   ├── middleware/              ← Middleware chain (Phase 3)
│   ├── auth/                   ← JWT & API key auth (Phase 4)
│   ├── ratelimit/              ← Rate limiting with Redis (Phase 5)
│   ├── loadbalancer/           ← Load balancing strategies (Phase 6)
│   ├── discovery/              ← Service discovery with etcd (Phase 7)
│   ├── circuitbreaker/         ← Circuit breaker pattern (Phase 8)
│   ├── config/                 ← Configuration loading (Phase 1)
│   └── admin/                  ← Admin REST API (Phase 10)
│
├── pkg/                        ← Reusable utilities
│   ├── logger/                 ← Structured logging
│   ├── metrics/                ← Prometheus metrics
│   └── tracing/                ← Jaeger distributed tracing
│
├── deployments/
│   └── docker/                 ← Docker Compose files
│
├── scripts/                    ← Helper scripts
├── tests/
│   └── integration/            ← End-to-end tests
│
├── .gitignore                  ← Files Git should ignore
├── .env                        ← Secret config (never committed)
├── Makefile                    ← Shortcut commands
├── go.mod                      ← Go project manifest
└── go.sum                      ← Dependency lock file
```

**Why `internal/` is special:** In Go, any code inside a folder named `internal` can ONLY be imported by code in the same project. This prevents other projects from importing your internal business logic — enforced by the Go compiler itself.

**Why `cmd/` has multiple folders:** Each folder inside `cmd/` has its own `main.go` and compiles into a separate runnable binary. This means we can build and run the gateway, user-service, product-service, and order-service all as independent programs — exactly like real microservices.

---

### Step 7 — Installed All 13 Go Dependencies

**What we did:** Ran `go get` for each package and `go mod tidy` to clean up.

**Why it matters:** These libraries save us from reinventing the wheel. Each one is production-tested, widely used, and solves a specific problem.

**Every dependency explained:**

| Package | What it does | Used in Phase |
|---|---|---|
| `go-chi/chi` | HTTP router — matches URL paths to handlers (like Express.js) | Phase 1 |
| `redis/go-redis` | Redis client — connect to Redis for caching and rate limiting | Phase 4, 5 |
| `spf13/viper` | Config management — reads `.env` files and environment variables | Phase 1 |
| `golang-jwt/jwt` | JWT authentication — create and validate JSON Web Tokens | Phase 4 |
| `rs/zerolog` | Structured logging — fast JSON logging (used at Cloudflare) | Phase 3 |
| `sony/gobreaker` | Circuit breaker — stops sending traffic to failing services | Phase 8 |
| `prometheus/client_golang` | Prometheus metrics — exposes metrics endpoint for monitoring | Phase 9 |
| `prometheus/promhttp` | HTTP handler for Prometheus metrics scraping | Phase 9 |
| `opentelemetry.io/otel` | OpenTelemetry — distributed tracing standard | Phase 9 |
| `otel/exporters/jaeger` | Sends traces to Jaeger for visualization | Phase 9 |
| `etcd/client/v3` | etcd client — for service discovery and dynamic config | Phase 7 |
| `avast/retry-go` | Retry logic with exponential backoff | Phase 8 |

**What `go mod tidy` does:** After installing packages, tidy removes any unused ones and adds any missing ones. It also generates the `go.sum` file which contains a cryptographic hash of every dependency — ensuring no one can tamper with the packages you use. This is Go's security model for dependencies.

---

### Step 8 — Created Base Files (.gitignore, .env, Makefile)

**What we did:** Created three essential configuration files.

#### .gitignore

**Why it matters:** Tells Git which files to never commit to GitHub.

The most important line is `.env` — your `.env` file contains secret keys (JWT secret, Redis password, API keys). If you accidentally commit this to a public GitHub repo, anyone can use your secrets. The `.gitignore` prevents this mistake.

```
bin/          ← compiled binaries (no need to commit these)
*.exe         ← Windows executables
.env          ← SECRET — never commit this
vendor/       ← downloaded packages (go.sum handles this instead)
```

#### .env

**Why it matters:** Stores all configuration and secrets in one place, separate from code.

This is the **12-Factor App** methodology — configuration that changes between environments (development, staging, production) should never be hardcoded in code. It goes in environment variables.

```
GATEWAY_PORT=8080         ← our gateway listens here
ADMIN_PORT=9090           ← admin API listens here
REDIS_HOST=localhost      ← where Redis is running
JWT_SECRET=...            ← secret for signing JWT tokens
ADMIN_API_KEY=...         ← key for accessing admin endpoints
```

In Phase 1, we use Viper to read these values into Go structs. Change a value in `.env` and the whole system updates — no code changes needed.

#### Makefile

**Why it matters:** Shortcuts for common commands so you never have to remember long commands.

Instead of typing `go run cmd/gateway/main.go` every time, you just type `make run`. Instead of typing the full docker compose command, you type `make docker-up`.

```makefile
make run        → go run cmd/gateway/main.go
make build      → compiles to bin/gateway
make test       → runs all tests
make docker-up  → starts Redis, etcd, Prometheus, Grafana
make docker-down → stops everything
```

This is how professional teams work — Makefiles are standard in Go projects.

---

### Step 9 — Wrote First Working Go Code

**What we did:** Created `cmd/gateway/main.go` with a basic HTTP server and ran it.

**The code:**
```go
package main

import (
    "fmt"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "GoGate is alive!")
    })

    log.Println("GoGate starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

**Line by line explanation:**

- `package main` — every runnable Go program must have `package main`. It tells Go this file is an entry point.
- `import (...)` — we import three standard library packages: `fmt` for printing, `log` for logging, `net/http` for the HTTP server
- `http.HandleFunc("/", ...)` — register a handler for the `/` path. When any request comes in to `/`, run this function
- `func(w http.ResponseWriter, r *http.Request)` — the handler receives two things: `w` to write the response, `r` which contains the incoming request
- `http.ListenAndServe(":8080", nil)` — start an HTTP server on port 8080
- `log.Fatal(...)` — if the server crashes, log the error and exit

**What we proved:** Running `curl http://localhost:8080` returned `GoGate is alive!` — confirming Go, VS Code, our folder structure, and our dependencies are all working correctly.

---

### Step 10 — First Git Commit & Push to GitHub

**What we did:** Committed all files and pushed to GitHub. Fixed the `master` vs `main` branch issue.

**The commands:**
```
git add .                                       ← stage all files
git commit -m "feat: initial project setup"     ← save snapshot
git push origin main --force                    ← upload to GitHub
```

**What the commit message format means:**
`feat: initial project setup and scaffold`

This follows **Conventional Commits** — a standard used by professional teams. The prefix describes the type of change:
- `feat:` — a new feature
- `fix:` — a bug fix
- `docs:` — documentation changes
- `refactor:` — code restructuring
- `chore:` — maintenance tasks

Using this format makes your Git history readable and professional. Tools can even auto-generate changelogs from it.

**The master vs main issue:** When Git was originally created, it called the default branch `master`. In 2020, the industry moved to `main` as the standard name. Your local Git used the old default (`master`) while GitHub expected `main`. We renamed it with `git branch -m master main` — a one-time fix. From now on, all branches default to `main`.

---

## Current Project State

After all setup steps, here is exactly what you have:

```
C:\Users\rosha\gogate\
├── cmd/
│   ├── gateway/
│   │   └── main.go          ✅ First working Go server
│   ├── user-service/         (empty — Phase 1)
│   ├── product-service/      (empty — Phase 1)
│   └── order-service/        (empty — Phase 1)
├── internal/                 (all empty — Phase 1 onwards)
├── pkg/                      (all empty — Phase 3 onwards)
├── deployments/docker/       (empty — Phase 1)
├── .gitignore                ✅ Created
├── .env                      ✅ Created (never committed)
├── Makefile                  ✅ Created
├── go.mod                    ✅ All 13 dependencies listed
└── go.sum                    ✅ All versions locked
```

**On GitHub:** `github.com/SOHAR-18/gogate` — `main` branch — 1 commit

---

## What Phase 1 Will Build

Now that the environment is ready, Phase 1 turns GoGate into a real working system:

**Config Loader** — Viper reads `.env` into a Go struct. Every part of the gateway reads config from one place. Change `.env`, restart, everything updates.

**Three Dummy Microservices** — `user-service` (port 8081), `product-service` (port 8082), `order-service` (port 8083). These simulate real backend services. The gateway will forward requests to them.

**Docker Compose** — one `docker-compose.yml` file that starts all 4 services (gateway + 3 services) with one command. This is how you'll run the full system locally.

**Real Router** — replace the simple `http.HandleFunc` with the `chi` router. Configure routes like:
- `GET /users/*` → forward to user-service
- `GET /products/*` → forward to product-service
- `POST /orders/*` → forward to order-service

**By the end of Phase 1**, you'll run `make docker-up` and have four services talking to each other. You'll `curl http://localhost:8080/users/1` and the gateway will forward it to user-service and return the response. That is a real API gateway doing real work.

---

## The 10-Phase Journey Ahead

| Phase | What You Build | Key Concepts Learned |
|---|---|---|
| **0** ✅ | Dev environment, project scaffold | Go, Docker, Git, project structure |
| **1** | Config, dummy services, Docker Compose, router | Go modules, HTTP routing, Docker |
| **2** | Reverse proxy engine | httputil.ReverseProxy, request forwarding |
| **3** | Middleware chain & plugin system | Go interfaces, context propagation |
| **4** | JWT & API key authentication | RS256 JWT, Redis lookups, auth patterns |
| **5** | Rate limiting with Redis | Token bucket, sliding window, Lua scripts |
| **6** | Load balancing & health checks | Round-robin, goroutines, atomic operations |
| **7** | Service discovery with etcd | Distributed KV stores, watch API, leases |
| **8** | Circuit breaker & resilience | Circuit breaker pattern, retry, bulkhead |
| **9** | Observability (Prometheus, Grafana, Jaeger) | Metrics, tracing, dashboards |
| **10** | Admin REST API & live dashboard | Runtime config, RBAC, htmx |

---

## Key Things to Remember

**Every time you start working:**
1. Open PowerShell inside `C:\Users\rosha\gogate`
2. Make sure Docker Desktop is running
3. Open VS Code with `code .`

**Every time you finish a feature:**
```
git add .
git commit -m "feat: describe what you built"
git push
```

**The `.env` file is secret.** Never commit it. Never share it. The `.gitignore` protects you, but always double-check before pushing.

**`go mod tidy` is your friend.** Run it any time you add or remove imports. It keeps `go.mod` and `go.sum` clean and correct.

---

*Documentation generated after Phase 0 completion — GoGate API Gateway project*
*GitHub: github.com/SOHAR-18/gogate*
