# Go Todo App

A todo application built with hexagonal architecture, gRPC, and grpc-gateway.

## Table of Contents
- [Preview](#preview)
- [Key Technologies](#key-technologies)
- [System Architecture](#system-architecture)
- [Hexagonal Architecture](#hexagonal-architecture-ports--adapters)
- [Quick Start](#quick-start)
- [Feature Addition Workflow](#feature-addition-workflow)
- [Dev Commands](#dev-commands)

## Preview

![Preview 1](docs/ss/1.png)
![Preview 2](docs/ss/2.png)
![Preview 3](docs/ss/3.png)

## Key Technologies
- Core: Go 1.22+, strict Hexagonal Architecture.
- Interfaces: gRPC & HTTP (via grpc-gateway), Buf schema definitions.
- Data & Caching: PostgreSQL (pgx), Memcached, Redis (go-redis).
- Auth & Security: PASETO v2, Google OAuth2 integration.
- Background Jobs: Asynchronous resilient queues via asynq.
- Infrastructure: Docker, Docker Compose, optimized multi-stage Distroless containers.
- Observability: Wide-event structured logging (slog).

## System Architecture

The application is heavily decoupled into three highly-optimized discrete runtime binaries deployed independently via Docker Compose:

1. server: The primary gRPC & HTTP REST boundary. Handles User flows, Auth, and immediate Todo mutations.
2. scheduler: A lightweight SQL poller. Identifies upcoming deadlines (e.g., Todos due in <24 hours) and instantly routes queue payloads to Redis.
3. worker: The Asynq driving adapter. A background processor handling heavy tasks safely (e.g., bouncing SMTP requests using Mailpit, sending HTML email with exponential backoff).

## Hexagonal Architecture (Ports & Adapters)

This project strictly follows the **Hexagonal Architecture** pattern to decouple core business logic from external concerns like databases, transport layers, and third-party services.

- **Domain Layer (`internal/domain`)**: The "heart" of the application. Contains pure business logic, entities (e.g., `Todo`), and value objects. It has zero dependencies on any other layer or external framework.
- **Application Layer (`internal/application`)**: Implements specific use cases (e.g., `CreateTodo`, `Login`). It orchestrates domain objects and coordinates the flow of data.
- **Ports (`internal/port`)**: Defines the boundaries via interfaces.
    - **Input Ports**: Interfaces that the application exposes to the outside world (e.g., `AuthUseCase`).
    - **Output Ports**: Interfaces that the application needs from the outside world (e.g., `UserRepository`, `EmailSender`).
- **Adapters (`internal/adapter`)**:
    - **Driving Adapters (Input)**: Handlers that "drive" the application, such as gRPC/REST servers (`internal/adapter/driving/grpc`) or background task processors.
    - **Driven Adapters (Output)**: Implementations of the output ports that interact with external infrastructure like PostgreSQL, Redis, or SMTP.

This decoupled structure ensures that core business rules remain testable, maintainable, and independent of infrastructure changes.

## Quick Start

Everything operates smoothly inside Docker. No local dependencies required except Docker and make.

```bash
# 1. Clone the repository
git clone https://github.com/SemmiDev/go-todo-app.git
cd go-todo-app

# 2. Set up your environment variables
# Note: You MUST update .env with your own Google OAuth client credentials for login to work!
cp .env.example .env

# 3. Start the entire infrastructure (Postgres, Redis, Memcached, Mailpit, + 3 Binaries)
make docker-up

# 4. Access the different services
# Frontend Application:   http://localhost:8080
# Backend API Gateway:    http://localhost:8080/v1/...
# Mailpit (Email Logs):   http://localhost:8025

# 5. View streaming logs
docker compose logs -f
```

## Feature Addition Workflow

Follow this standard pipeline to introduce new domain features:

1. Database Migrations
   Create a new migration file to represent your domain schema.
   Create new queries within your data layer. Ensure `make migrate-up` runs cleanly.

2. Protobuf Definitions
   Define your services and messages in `api/proto/v1`.
   Run `make buf-generate` to predictably map your proto logic into Go stubs.

3. Core Domain Layer Update
   Add logic strictly scoped to the pure `internal/application/` bounds. Do not leak DB or HTTP frameworks here.

4. Dependency Injection & Ports
   Plug the logic into your hexagonal HTTP or gRPC handlers within `internal/adapter/driving/grpc/`. Add background payloads to `internal/adapter/driving/worker/` handlers if processing occurs offline.

5. Verification
   Deploy locally via `make run-server` or `make docker-up` to inspect.

## Dev Commands

The system leverages local sandboxing for development:
```bash
make db-up         # Spin up only the DB infra
make migrate-up    # Run database schemas
make run-server    # Run the hot-code local API server
make run-worker    # Run the hot-code local queue processor
make run-scheduler # Run the hot-code local polling cron
```
