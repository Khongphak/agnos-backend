# agnos-backend

Go + Gin backend service with PostgreSQL, running behind Nginx.

## Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose
- Go 1.23+ (for local development without Docker)

## Setup

```bash
cp .env.example .env
# Edit .env and set DB_PASSWORD and other secrets
```

## Run with Docker

```bash
docker-compose up --build
```

All three services start together: **Nginx** (port 80) → **Go/Gin backend** (port 8080, internal) → **PostgreSQL** (port 5432, internal).

## Verify the system is healthy

```bash
# Through Nginx (production path)
curl http://localhost/health

# Direct backend (local dev only)
curl http://localhost:8080/health
```

Expected response:
```json
{ "status": "ok", "database": "connected" }
```

## Run locally without Docker

```bash
go mod tidy
# Ensure PostgreSQL is running and .env values are set in your shell
go run ./cmd
```

## Run tests

```bash
go test ./...
```

## Project structure

```
agnos-backend/
├── cmd/                         # Entry point (main.go)
├── internal/
│   ├── config/                  # Environment variable loading
│   ├── handler/                 # HTTP handlers (Gin)
│   ├── service/                 # Business logic
│   ├── repository/              # Database access
│   └── model/                   # Domain structs (expand per feature)
├── pkg/
│   └── response/                # Shared response helpers
├── migrations/                  # SQL migration files
├── nginx/
│   └── default.conf             # Nginx reverse proxy config
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## API

### GET /health

Returns service and database status.

**Response** `200 OK`
```json
{ "status": "ok", "database": "connected" }
```

`database` is `"disconnected"` when the DB cannot be reached (service still returns 200 — callers should check the field value).

### Error response format

All error responses follow a consistent envelope:
```json
{ "error": { "code": "ERROR_CODE", "message": "human-readable message" } }
```

## Nginx routing

| External path | Backend path |
|---------------|--------------|
| `GET /api/*`  | `GET /*`     |
| `GET /health` | `GET /health`|

Example: `GET http://localhost/api/users` → `GET http://backend:8080/users`
