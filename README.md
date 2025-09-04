# GO-API-K8S
[![CI](https://github.com/s1natex/tasks-api-GO/actions/workflows/ci.yml/badge.svg)](https://github.com/s1natex/tasks-api-GO/actions/workflows/ci.yml)
![Docker Pulls](https://img.shields.io/docker/pulls/s1natex/tasks-api-go)

A learning project: production-ready Go REST API with observability, CI/CD, and Kubernetes deployment
Built step-by-step to explore DevOps practices end-to-end

## Features
- Go + chi
- In-memory → SQLite storage
- /tasks CRUD (POST, GET) with validation
- Middleware: request ID, panic recovery, timeouts, CORS
- Auth stub: API key / Bearer token via env vars
- Rate limiting with configurable RPS & burst

- Observability:
    - /health endpoint
    - /metrics (Prometheus)
    - Structured JSON logging (slog)
    - OpenTelemetry tracing

- OpenAPI spec (/openapi.json) + Swagger UI (/docs)
- Graceful shutdown with signal handling
- Docker & docker-compose
- Kubernetes deployment with probes & PVC

- GitHub Actions CI/CD:
    - run tests + lint
    - build & push Docker image to Docker Hub

## Run locally
- Go:
```
go run .
curl -s http://localhost:8080/health
```
- Docker:
``` 
docker build -t tasks-api-go:dev .
docker run --rm -p 8080:8080 -p 8081:8081 tasks-api-go:dev
```
- Docker Compose:
```
docker compose up -d
curl -s http://localhost:8080/health
```
- Kubernetes (Docker Desktop):
```
kubectl apply -f k8s/
kubectl -n tasks-api port-forward deploy/tasks-api 8080:8080 8081:8081
```
## Auth & Config
| Var                | Default         | Description                      |
| ------------------ | --------------- | -------------------------------- |
| `AUTH_MODE`        | `none`          | `apikey`, `bearer`, or `none`    |
| `API_KEY`          | *empty*         | API key if `AUTH_MODE=apikey`    |
| `BEARER_TOKEN`     | *empty*         | Token if `AUTH_MODE=bearer`      |
| `RATE_LIMIT_RPS`   | `0`             | Requests per second (0 = off)    |
| `RATE_LIMIT_BURST` | `0`             | Burst size (defaults to 2×RPS)   |
| `DB_PATH`          | `data/tasks.db` | SQLite database file             |
| `LOG_LEVEL`        | `info`          | `debug`, `info`, `warn`, `error` |
## CI/CD
- PRs → run tests + lint + build (no push)
- Push to main → build, tag, and push Docker image:
    - s1natex/tasks-api-go:latest
    - s1natex/tasks-api-go:sha-<commit>
- Tag release (e.g., v1.0.0) → also push s1natex/tasks-api-go:v1.0.0
## Observability
- Health: GET /health
- Metrics: GET /metrics (Prometheus format)
- Tracing: OpenTelemetry spans (export to OTLP or stdout)
- OpenAPI: GET /openapi.json
- Docs: GET /docs
## Example Usage
```
# Create a task
curl -s -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"my task"}'

# List tasks
curl -s http://localhost:8080/tasks
```