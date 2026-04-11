# Go Todo App

A production-grade, asynchronous Todo tracking system built entirely in Go using strict **Hexagonal Architecture (Ports & Adapters)**.

## 🚀 Key Technologies
- **Core:** Go 1.22+, strict Hexagonal Architecture.
- **Interfaces:** gRPC & HTTP (via `grpc-gateway`), Buf schema definitions.
- **Data & Caching:** PostgreSQL (`pgx`), Memcached, Redis (`go-redis`).
- **Auth & Security:** PASETO v2, Google OAuth2 integration.
- **Background Jobs:** Asynchronous resilient queues via `asynq`.
- **Infrastructure:** Docker, Docker Compose, fully optimized multi-stage Distroless containers.
- **Observability:** Wide-event structured logging (`slog`).

## 🏗 System Architecture

The application is heavily decoupled into three highly-optimized discrete runtime binaries deployed independently via Docker Compose:

1. **`server`**: The primary gRPC & HTTP REST boundary. Handles User flows, Auth, and immediate Todo mutations.
2. **`scheduler`**: A lightweight SQL poller. Identifies upcoming deadlines (e.g., Todos due in <24 hours) and instantly routes queue payloads to Redis.
3. **`worker`**: The Asynq driving adapter. A background processor handling heavy tasks safely (e.g., bouncing SMTP requests using Mailpit, sending HTML reminder emails with exponential backoff).

## 🛠 Quick Start

Everything operates smoothly inside Docker. No local dependencies required except Docker and `make`.

```bash
# 1. Start the entire infrastructure (Postgres, Redis, Memcached, Mailpit, + 3 Binaries)
make docker-up

# 2. View streaming logs
docker compose logs -f

# 3. Access local Mailpit to view sent email Reminders
# Open your browser to http://localhost:8025
```

## 📜 Dev Commands
The system leverages local sandboxing for development:
```bash
make db-up         # Spin up only the DB infra
make migrate-up    # Run database schemas
make run-server    # Run the hot-code local API server
make run-worker    # Run the hot-code local queue processor
make run-scheduler # Run the hot-code local polling cron
```
