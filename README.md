<div align="center">

# Leave Management System — Backend

**A role-based leave management API with multi-stage approvals, configurable leave policies, and automated notifications.**

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

[Features](#features) • [Architecture](#architecture) • [Getting Started](#getting-started) • [Docker](#docker-setup) • [Contributing](#contributing)

</div>

---

> This repository contains the **backend** only. The React frontend lives in a [separate repository].

## Overview

This is the backend service for a **Leave Management System (LMS)** built for organizations with multiple roles and structured approval hierarchies. It handles the full lifecycle of a leave request — from policy configuration and submission, through multi-level approval, to reporting and notifications — on top of a Go REST API.

**Why this exists:** leave tracking in spreadsheets or chat threads breaks down once an organization has more than a handful of people or more than one layer of approval. Requests get lost, approvers forget to act, and leave balances drift out of sync with reality. This system makes leave management a single source of truth: every request has a defined approval chain, every policy (paid, unpaid, WFH, early leave) follows its own configured rules, and every status change is tracked end-to-end — with email and Slack notifications removing the need to check the system manually.

## Features

**Roles & Access**
- Role-based permissions across SuperAdmin, Admin, HR, Employee, and Intern
- Manager assignment, role updates, password changes, manual leave balance adjustments

**Leave Management**
- Apply, edit, cancel, and withdraw leave requests
- Multi-stage approval workflow with configurable approver chains per policy
- Full status timeline per request — applied, pending, approved, rejected, withdrawn
- Configurable leave policies: paid, unpaid, early leave, WFH, with yearly/basic entitlement rules

**Calendar & Holidays**
- Org-wide holiday calendar, managed by admins
- Weekly and monthly leave calendar views

**Asset Management**
- Asset assignment to employees, with quantity and slot tracking

**Reporting**
- Monthly and yearly leave reports generated from leave history

**Notifications**
- Email on leave application and on approval/rejection
- Slack: daily "who's on leave today" summary and automated birthday announcements
- Token-protected cron endpoint for an external scheduler to trigger the daily Slack summary

**Infrastructure**
- Multi-stage Dockerfile and Docker Compose setup, backend + optional local Postgres

## Architecture

> _Diagram to be added._

The frontend (separate repository) communicates with this API over REST. The backend persists leave, user, policy, and asset data in PostgreSQL, sends transactional email on leave events, and posts to Slack via Incoming Webhooks for birthdays and the daily leave summary. Since the hosting platform doesn't run scheduled jobs itself, an external scheduler (e.g. cron-job.org) triggers the daily summary through a token-protected endpoint.

## Tech Stack

| | |
|---|---|
| **Language** | Go 1.25 |
| **Web framework** | [Gin](https://github.com/gin-gonic/gin) |
| **Frontend** | React _(separate repository)_ |
| **Database** | PostgreSQL via `sqlx` + `lib/pq` |
| **Migrations** | [Goose](https://github.com/pressly/goose) |
| **Auth** | JWT (`golang-jwt/jwt`), `golang.org/x/crypto` |
| **Validation** | `go-playground/validator` |
| **Reports (PDF)** | `jung-kurt/gofpdf` |
| **Containerization** | Docker, Docker Compose |
| **Scheduling** | External cron (e.g. cron-job.org) calling a secured endpoint |

## Getting Started

### Prerequisites
- Go 1.25+
- PostgreSQL (local, Dockerized, or hosted — Railway/Supabase)
- Docker & Docker Compose (optional, for containerized setup)

### 1. Clone & install

```bash
git clone https://github.com/sanjayk-eng/UserMenagmentSystem_Backend.git
cd UserMenagmentSystem_Backend
go mod tidy
```

### 2. Configure environment

```bash
cp .env.example .env
```

Fill in your values — see [`.env.example`](./.env.example) for the full list (database, server config, Resend, Slack webhooks, cron secret, company email domain).

### 3. Database

**Hosted (recommended for production)** — provision Postgres on [Railway](https://railway.app) or [Supabase](https://supabase.com), copy the connection string into `DB_URL`.

**Local (development)**

```bash
docker run --name leave-db \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=dbname \
  -p 5432:5432 \
  -v leave-db-data:/var/lib/postgresql/data \
  -d postgres
```

```env
DB_URL=postgresql://user:password@localhost:5432/dbname?sslmode=disable
```

Run migrations:

```bash
goose -dir pkg/migration postgres "$DB_URL" up
```

### 4. Run

```bash
go run main.go
```

The server starts on `APP_PORT` (default `8082`).

### 5. Frontend

The React frontend is maintained in its own repository — see that repo's README for setup. Set `ALLOWED_ORIGINS` in `.env` to the frontend's URL so CORS allows the connection.

## Docker Setup

Two ways to run, depending on where your database lives.

**A — External database (Railway / Supabase)**

```yaml
services:
  backend:
    build: .
    container_name: ums-backend
    ports:
      - "8082:8082"
    env_file:
      - .env
    restart: unless-stopped
```

**B — Local Postgres in Docker**

```yaml
services:
  backend:
    build: .
    container_name: ums-backend
    ports:
      - "8082:8082"
    env_file:
      - .env
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    container_name: ums-postgres
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: yourpassword
      POSTGRES_DB: dbname
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

With option B, point `.env` at the service name, not `localhost`:

```env
DB_URL=postgresql://postgres:yourpassword@ums-postgres:5432/dbname?sslmode=disable
```

```bash
docker compose up -d --build
```

| Command | Action |
|---|---|
| `docker compose up -d --build` | Build and start in the background |
| `docker compose down` | Stop services |
| `docker compose down -v` | Stop and remove the local DB volume |
| `docker logs -f ums-backend` | Follow logs live |
| `docker exec -it ums-backend sh` | Shell into the running container |

## Testing

```bash
go test ./...
go test ./... -cover
```

Test suite is a work in progress. Conventions going forward: unit tests alongside source files, integration tests for DB-touching code behind a build tag or test database, and external services (Resend, Slack) mocked rather than called live.

## Contributing

1. Fork the repository
2. Create a branch: `git checkout -b feature/your-feature-name`
3. Commit your changes with clear, focused messages
4. Run `go mod tidy && go vet ./...` before pushing
5. Open a Pull Request describing what changed and why

For larger changes, please open an issue first to discuss the approach.

## License

MIT — see [LICENSE](./LICENSE).
