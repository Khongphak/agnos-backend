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

> **Required:** Set `JWT_SECRET` before starting. A missing secret will cause token signing to fail at runtime.
>
> ```bash
> export JWT_SECRET="$(openssl rand -hex 32)"
> docker-compose up --build
> ```

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

### POST /staff/create

Register a new staff member tied to a hospital.

**Request body**
```json
{ "username": "alice", "password": "SecurePass1", "hospital_code": "HOSP01", "role": "staff" }
```
`role` is optional and defaults to `"staff"`. Allowed values: `"staff"`, `"admin"`.

**Response** `201 Created`
```json
{ "id": 1, "username": "alice", "hospital_code": "HOSP01", "role": "staff" }
```

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_INPUT` | Missing required fields or invalid role |
| 404 | `HOSPITAL_NOT_FOUND` | `hospital_code` does not exist |
| 409 | `USERNAME_CONFLICT` | Username already taken in that hospital |

---

### POST /staff/login

Authenticate and receive tokens.

**Request body**
```json
{ "username": "alice", "password": "SecurePass1", "hospital_code": "HOSP01" }
```

**Response** `200 OK`
```json
{ "access_token": "<jwt>", "refresh_token": "<opaque>", "expires_in": 900 }
```
`expires_in` is in seconds (900 = 15 minutes). The refresh token is valid for 30 days.

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_INPUT` | Missing required fields |
| 401 | `INVALID_CREDENTIALS` | Wrong username, password, or hospital_code |
| 403 | `ACCOUNT_INACTIVE` | Account has been deactivated |

---

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

## Environment variables

| Variable      | Default     | Required in prod | Note |
|---------------|-------------|-----------------|------|
| SERVER_PORT   | 8080        | no              | |
| DB_HOST       | localhost   | yes             | |
| DB_PORT       | 5432        | no              | |
| DB_USER       | postgres    | yes             | |
| DB_PASSWORD   | (empty)     | yes             | |
| DB_NAME       | agnos       | yes             | |
| DB_SSLMODE    | disable     | yes             | Set to `require` in prod |
| JWT_SECRET    | (empty)     | **yes**         | HS256 signing key; generate with `openssl rand -hex 32` |

Access tokens expire after **15 minutes**. Refresh tokens expire after **30 days**.

---

## Nginx routing

| External path | Backend path |
|---------------|--------------|
| `GET /api/*`  | `GET /*`     |
| `GET /health` | `GET /health`|

Example: `GET http://localhost/api/users` → `GET http://backend:8080/users`
