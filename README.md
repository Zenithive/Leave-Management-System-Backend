# Leave Management System — Backend

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white)](https://www.docker.com/)

The backend service for the **Leave Management System** — handles authentication, leave requests/approvals, email notifications, and Slack integrations (birthday announcements + daily leave summaries).

> This README covers the **backend** only.

---

## Table of Contents

- [Features](#features)
- [Technology Stack](#technology-stack)
- [Backend Setup](#backend-setup)
  - [1. Clone & Install](#1-clone--install)
  - [2. Environment Variables](#2-environment-variables)
  - [3. Database](#3-database)
  - [4. Resend (Email)](#4-resend-email)
  - [5. Slack Integration](#5-slack-integration)
  - [6. Daily Leave Cron Job](#6-daily-leave-cron-job)
- [Docker Setup](#docker-setup)
- [Running Locally](#running-locally)
- [Troubleshooting](#troubleshooting)

---

## Features

- 🔐 **Authentication** — signup/login restricted to a company email domain
- 📝 **Leave requests** — pending, approve, reject, and track leave history
- 📧 **Email notifications** — transactional emails via Resend (verification, approvals, rejections)
- 💬 **Slack integration** — daily "who's on leave today" summary + automated birthday announcements
- ⏰ **Secure cron endpoint** — token-protected endpoint for an external scheduler to trigger daily jobs
- 🐳 **Containerized** — Dockerfile + docker-compose setup

## Technology Stack

| Layer | Technology |
|---|---|
| Backend language | **Go 1.25** |
| Database | **PostgreSQL** (Railway / Supabase / local via Docker) |
| Email | **Resend** |
| Notifications | **Slack** (Incoming Webhooks) |
| Containerization | **Docker** + **Docker Compose** |
| Scheduling | External cron (e.g. **cron-job.org**) hitting a secured backend endpoint |

---

## Backend Setup

### 1. Clone & Install

```bash
git clone https://github.com/your-username/your-project.git
cd your-project/backend
go mod tidy
```

### 2. Environment Variables

Create a `.env` file in the project root:

```env
# Database
DB_URL=postgresql://user:password@host:port/dbname?sslmode=disable

# Server
APP_PORT=8082
APP_URL=http://localhost:8089
APP_NAME=Leave Management System

# Email (Resend)
RESEND_API_KEY=your_resend_api_key_here
RESEND_FROM=leave@yourdomain.com

# CORS
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8089

# Slack
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
EXTERNAL_API_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Cron security — endpoint: GET /api/cron/daily-leave-slack?token=CRON_SECRET
CRON_SECRET=your-random-cron-secret-min-32-chars-here

# Company rules — restricts signup/login to this email domain
COMPANY_EMAIL_DOMAIN=yourdomain.com

# Seed / demo account password
DEMO_SEED_PASSWORD=Demo@1234
```

Generate a secure `CRON_SECRET` instead of hand-typing one:

```bash
openssl rand -hex 32
```

### 3. Database

**Option A — IN for production**

 exampler Copy the connection string from [Railway](https://railway.app) or [Supabase](https://supabase.com) into `DB_URL`. Use `?sslmode=require` in production.

**Option B — Local Postgres for development**

```bash
docker run --name leave-db \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=dbname \
  -p 5432:5432 \
  -v leave-db-data:/var/lib/postgresql/data \
  -d postgres
```

Then in `.env`:
```env
DB_URL=postgresql://user:password@localhost:5432/dbname?sslmode=disable
```

> **If using the bundled `docker-compose.yml` instead** (see [Docker Setup](#docker-setup)), Postgres is handled for you — don't run the command above separately, and don't use `localhost` as the host. Use the Postgres **service name** (e.g. `ums-postgres`) instead, since `localhost` inside a container refers to the container itself, not your host machine or sibling containers.

### 4. Resend (Email)

1. Create an account at [resend.com](https://resend.com).
2. Verify a sending domain (or use Resend's test domain while developing).
3. Generate an API key → `RESEND_API_KEY`.
4. Set `RESEND_FROM` to an address on your verified domain.

### 5. Slack Integration

1. [api.slack.com/apps](https://api.slack.com/apps) → **Create New App** → **From scratch**.
2. Choose workspace and app name.
3. Under **Incoming Webhooks**, toggle on → **Add New Webhook to Workspace**.
4. Pick the channel for notifications (e.g. `#leave-updates`).
5. Copy the webhook URL into both `SLACK_WEBHOOK_URL` and `EXTERNAL_API_URL`.

### 6. Daily Leave Cron Job

Railway/Supabase don't run scheduled jobs — you need an external scheduler hitting your endpoint:

```
GET https://your-backend-url/api/cron/daily-leave-slack?token=CRON_SECRET
```

**Using cron-job.org (free):**
1. Create an account → **Create cronjob**
2. URL: `https://your-backend-url/api/cron/daily-leave-slack?token=YOUR_CRON_SECRET`
3. Schedule: daily at your preferred time
4. Method: `GET` → Save and enable

Test manually:
```bash
curl "https://your-backend-url/api/cron/daily-leave-slack?token=YOUR_CRON_SECRET"
```
`200 OK` = Slack message sent successfully.

> ⚠️ Treat `CRON_SECRET` like a password — anyone with it can trigger the endpoint. Never commit `.env`.

---

## Docker Setup

**Files:**
- `Dockerfile` — multi-stage build: compiles the Go binary in a `golang:1.25-alpine` build stage, copies only the binary + `pkg/migration` into a minimal `alpine:latest` runtime image
- `docker-compose.yaml` — runs the backend; Postgres can be added as a second service if not using Railway/Supabase

Pick the configuration that matches where your database lives.

### Option A — `DB_URL` points to Railway / Supabase (no local DB container needed)

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

```bash
docker compose up -d --build
```

### Option B — Local Postgres in Docker

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

`depends_on: condition: service_healthy` makes `backend` wait until Postgres is actually accepting connections — not just "container started" — before it tries to connect. This avoids the race condition where the backend crashes because Postgres hasn't finished initializing yet.

Update `.env` to match this service, using the **service name** (`ums-postgres`) as the host — not `localhost`:
```env
DB_URL=postgresql://postgres:yourpassword@ums-postgres:5432/dbname?sslmode=disable
```

```bash
docker compose up -d --build
```

**Stop everything:**
```bash
docker compose down
```

**Stop and delete the local DB volume too:**
```bash
docker compose down -v
```

**View logs (live):**
```bash
docker logs -f ums-backend
```

**Open a shell inside the running container:**
```bash
docker exec -it ums-backend sh
```

---

## Running Locally

**Without Docker** (requires Go installed + a reachable Postgres):
```bash
go mod tidy
go run main.go
```
Server starts on `APP_PORT` (default `8082`).

**With Docker:**
```bash
docker compose up -d --build
```

---

## Troubleshooting

**Container keeps restarting / `Failed to connect to database`**
- `docker logs ums-backend` to see the actual error
- Confirm `DB_URL` is present inside the container: `docker exec -it ums-backend sh -c "env | grep DB_URL"`
- If the host in `DB_URL` is `localhost`, fix it — that resolves to the container itself, not your real database

**`dial tcp [::1]:5432: connect: connection refused`**
- The app is resolving `localhost` to IPv6 (`::1`), where nothing is listening inside the container. Update `DB_URL` to the correct host (external hostname, or the Postgres service name if running locally in Docker).

**`Conflict. The container name "/ums-backend" is already in use`**
- An old container with that name still exists. Remove it: `docker rm ums-backend`

**Container runs `sh` instead of the app**
- Happens only if `sh` was explicitly passed as an override (e.g. `docker run ... ums-backend sh`). The Dockerfile's `CMD ["./ums-backend"]` runs automatically when no command override is given.
