# go-todo-app

A todo application built with hexagonal architecture, gRPC, grpc-gateway, and Google OAuth2 cookie-based sessions.

## Architecture

```
internal/
├── domain/          # Pure business logic, no dependencies
│   ├── todo/        # Todo & Tag aggregates
│   └── user/        # User entity
├── application/     # Use cases (services)
│   ├── auth/        # OAuth2 + session management
│   ├── todo/        # Todo & Tag CRUD
│   └── port/        # Repository interfaces (ports)
└── infrastructure/  # Adapters
    ├── handler/grpc/ # gRPC servers (moved here from internal/grpc)
    │   ├── auth_server.go
    │   ├── todo_server.go
    │   ├── interceptor/
    │   └── grpcerr/
    └── postgres/    # Repository implementations
```

## API Endpoints

### Auth
| Method | Path | Description |
|--------|------|-------------|
| GET | /v1/auth/url | Get Google OAuth2 URL |
| POST | /v1/auth/callback | Exchange OAuth code → set session cookie |
| POST | /v1/auth/logout | Clear session |
| GET | /v1/auth/me | Get current user |

### Tags
| Method | Path | Description |
|--------|------|-------------|
| POST | /v1/tags | Create tag |
| GET | /v1/tags | List tags |
| GET | /v1/tags/{tag_id} | Get tag |
| PUT | /v1/tags/{tag_id} | Update tag |
| DELETE | /v1/tags/{tag_id} | Delete tag |

### Todos
| Method | Path | Description |
|--------|------|-------------|
| POST | /v1/todos | Create todo |
| GET | /v1/todos | List todos (filter: status, tag_id, page, page_size) |
| GET | /v1/todos/{todo_id} | Get todo |
| PUT | /v1/todos/{todo_id} | Update todo |
| DELETE | /v1/todos/{todo_id} | Delete todo |
| POST | /v1/todos/{todo_id}/tags/{tag_id} | Add tag to todo |
| DELETE | /v1/todos/{todo_id}/tags/{tag_id} | Remove tag from todo |

## Authentication

Session-based via HTTP cookie (`session_id`). After `/v1/auth/callback`, the cookie is set automatically and sent with every subsequent request.

## Setup

```bash
cp .env.example app.env
# Fill in GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, DATABASE_URL

docker compose up -d postgres
make migrate-up
make generate   # requires buf CLI
make run
```
