# Database URL for local development migrations if not set by env
DB_URL ?= postgres://postgres:postgres@localhost:5432/todo_app?sslmode=disable

.PHONY: all generate format proto build build-server build-scheduler build-worker run run-server run-scheduler run-worker docker-up docker-build docker-down docker-logs migrate-new migrate-up migrate-down lint test tidy clean

all: tidy proto build

# ─── Dependencies & Code Generation ──────────────────────────────────────────

generate: proto

format:
	goimports -w .

proto:
	buf generate

tidy:
	go mod tidy

clean:
	rm -rf bin/

# ─── Build ───────────────────────────────────────────────────────────────────

build: build-server build-scheduler build-worker

build-server:
	go build -o bin/server ./cmd/server

build-scheduler:
	go build -o bin/scheduler ./cmd/scheduler

build-worker:
	go build -o bin/worker ./cmd/worker

# ─── Run Locally ─────────────────────────────────────────────────────────────

run: run-server

run-server:
	go run ./cmd/server

run-scheduler:
	go run ./cmd/scheduler

run-worker:
	go run ./cmd/worker

# ─── Docker Infrastructure ───────────────────────────────────────────────────

docker-up:
	docker compose up -d

docker-build:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# ─── Database Migrations ─────────────────────────────────────────────────────

migrate-new:
	@read -p "Enter migration name (e.g. add_users): " name; \
	migrate create -ext sql -dir db/migration -seq $$name

migrate-up:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migrate-down:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

# ─── Testing & Linting ───────────────────────────────────────────────────────

lint:
	golangci-lint run ./...

test:
	go test ./... -race -cover
