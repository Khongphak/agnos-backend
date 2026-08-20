# agnos-backend

Go + Gin backend service with PostgreSQL, running behind Nginx.

agnos-backend is a hospital middleware system that handles staff authentication, patient search, and proxying requests to Hospital A's upstream API. It exposes real-time WebSocket channels so hospital staff can monitor patient form activity as it happens. The stack is Go 1.23 + Gin + PostgreSQL 16, orchestrated with Docker Compose and fronted by Nginx.

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

**API Explorer (Swagger UI)**

```
http://localhost/api/swagger/index.html
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

---

### POST /staff/refresh

Exchange a valid refresh token for a new token pair.

**Request body**
```json
{ "refresh_token": "<opaque>" }
```

**Response** `200 OK`
```json
{ "access_token": "<jwt>", "refresh_token": "<opaque>", "expires_in": 900 }
```

Rotates the refresh token — the old token is invalidated immediately.

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_INPUT` | Missing or malformed body |
| 401 | `TOKEN_INVALID` | Token expired or revoked |

---

### POST /staff/logout

Revoke a refresh token.

**Request body**
```json
{ "refresh_token": "<opaque>" }
```

**Response** `204 No Content`

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_INPUT` | Missing or malformed body |
| 401 | `TOKEN_INVALID` | Token expired or already revoked |

---

### GET /patient/search

Search patients scoped to the authenticated staff's hospital. Requires `Authorization: Bearer <access_token>`.

**Query parameters** (all optional, combined with AND)

| Param | Format |
|-------|--------|
| `national_id` | string |
| `passport_id` | string |
| `first_name` | string |
| `last_name` | string |
| `dob` | `YYYY-MM-DD` |
| `phone` | string |
| `email` | string |

**Response** `200 OK`
```json
{
  "patients": [
    {
      "id": 1,
      "national_id": "1234567890123",
      "passport_id": null,
      "first_name_th": "สมชาย",
      "last_name_th": "ใจดี",
      "first_name_en": "Somchai",
      "last_name_en": "Jaidee",
      "date_of_birth": "1990-01-15",
      "gender": "M",
      "phone_number": "0812345678",
      "email": "somchai@example.com",
      "patient_hn": "HN000123"
    }
  ]
}
```

| Status | Code | Condition |
|--------|------|-----------|
| 401 | `UNAUTHORIZED` | Missing or invalid access token |

---

### GET /hospital-a/patient/search/{id}

Proxy a patient lookup to the Hospital A upstream API. No authentication required.

**Path parameter:** `id` — HN or national ID of the patient.

**Response** `200 OK` — patient object from the upstream (shape determined by Hospital A).

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_INPUT` | Missing or invalid `id` |
| 404 | `PATIENT_NOT_FOUND` | Upstream returned no result |
| 502 | `UPSTREAM_ERROR` | Hospital A API unreachable or returned an error |

---

### WebSocket: GET /ws/patient?hospital_code=\<code\>

Real-time channel for a patient filling a form. No authentication required.

**On connect** the server sends:
```json
{ "type": "connected", "session_id": "<uuid4>" }
```

**Client → server**
```json
{ "type": "form_update", "status": "filling|submitted", "data": { "<form fields>" } }
```

Messages with any `type` other than `form_update` are silently dropped.

Inactivity timeout: **30 seconds** — if no message is received the server closes the connection and broadcasts an `inactive` event to all staff on that hospital.

---

### WebSocket: GET /ws/staff?token=\<jwt_access_token\>

Real-time channel for staff. Receive-only — the server sends updates, the client does not send messages.

If the token is invalid the server closes with code **1008 (Policy Violation)**.

**Server → client**
```json
{
  "type": "form_update",
  "session_id": "<uuid4>",
  "status": "filling|submitted|inactive",
  "data": { "<form fields>" },
  "timestamp": "<ISO8601>"
}
```

`data` is `null` when `status` is `"inactive"`.

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
| HOSPITAL_A_BASE_URL | `https://hospital-a.api.co.th` | no | Base URL for Hospital A upstream API; override in tests/staging |
| ALLOWED_ORIGINS | `http://localhost:3000` | **yes** | Comma-separated list of allowed CORS origins (e.g. `https://app.vercel.app`) |

Access tokens expire after **15 minutes**. Refresh tokens expire after **30 days**.

---

## Nginx routing

| External path | Backend path |
|---------------|--------------|
| `GET /api/*`  | `GET /*`     |
| `GET /health` | `GET /health`|

Example: `GET http://localhost/api/users` → `GET http://backend:8080/users`

## Bonus features

- **Swagger UI** — interactive API docs at `/api/swagger/index.html`
- **Hospital A proxy** — middleware endpoint (`GET /hospital-a/patient/search/{id}`) that forwards requests to an external upstream, keeping auth concerns separate from the core API
- **Refresh token rotation** — every refresh produces a new token pair and immediately revokes the old refresh token
- **WebSocket inactivity timeout** — if a patient connection goes silent for 30 seconds the server closes it and broadcasts an `inactive` event so staff are notified automatically
- **Test coverage** — unit + integration tests for all handlers; run with `go test ./...`
