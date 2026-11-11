# Kairso — Server

The backend for **Kairso**, a platform for managing workspaces, projects, KPIs, attendance and initiatives for teams. This repository is the Go API server: a REST API, a realtime WebSocket hub, and database migrations.

- 🌐 Website & docs: **https://home.kairso.com** (see the [Install & self-host guide](https://home.kairso.com/install))
- 🚀 Hosted app: **https://app.kairso.com**
- 🧩 Web front end: **[kairso-web](https://gitlab.alexcao.tech/kairso/nextkairso)** — the other half of the stack
- 🐳 Image: **[`alexcao194/kairso-server`](https://hub.docker.com/r/alexcao194/kairso-server)** (multi-arch `amd64` + `arm64`)

## Stack

| | |
|---|---|
| Language | Go 1.25 |
| HTTP | standard library `net/http` (Go 1.22+ routing) |
| Database | PostgreSQL 15+ via [pgx](https://github.com/jackc/pgx) + [sqlc](https://sqlc.dev) |
| Migrations | [goose](https://github.com/pressly/goose) (embedded, auto-applied on boot) |
| Cache / pub-sub | Redis |
| Object storage | S3-compatible (MinIO, AWS S3, Cloudflare R2, Backblaze B2…) |
| Auth | Firebase |
| Realtime | WebSocket (`/ws`) |
| API docs | Swagger at `/swagger/` |

The server listens on `:8080`.

---

## Run from source (development)

### Prerequisites
- [Go 1.25+](https://go.dev/dl/)
- Docker + Docker Compose
- A **Firebase project** (service-account JSON) for authentication
- Dev tools (optional, for codegen): [`sqlc`](https://docs.sqlc.dev/en/latest/overview/install.html), [`goose`](https://github.com/pressly/goose#install), [`swag`](https://github.com/swaggo/swag#getting-started), [`air`](https://github.com/air-verse/air)

### Quickest path — full dev stack in Docker
Spins up PostgreSQL, Redis, MinIO, the app (with hot-reload via `air`) and nginx:

```bash
make dev        # build + start everything
make log        # tail the app logs
make dev-down   # stop everything
```

### Manual — run the Go process directly
With PostgreSQL, Redis and MinIO already running locally:

```bash
# 1. Create a .env (the Makefile loads it automatically)
#    set DB_*, REDIS_*, MINIO_*, JWT_SECRET, APP_TOKEN_SECRET, FIREBASE_CREDENTIALS_PATH
# 2. Apply migrations
make migrate-up
# 3. Run (hot-reload)
air
#    …or a plain run
go run ./cmd/server
```

Open http://localhost:8080/swagger/ for the API explorer.

### Make targets

| Target | Does |
|---|---|
| `make dev` / `make dev-down` | Start / stop the full dev stack (Docker) |
| `make log` | Tail app logs |
| `make migrate-up` / `migrate-down` / `migrate-status` | Run goose migrations |
| `make migrate-create name=<x>` | New migration file |
| `make sqlc` | Regenerate type-safe DB code from `db/queries` |
| `make swagger` | Regenerate `docs/swagger.json` from handler annotations |
| `make build` | Compile to `tmp/server` |
| `make tidy` | `go mod tidy` |

### Key environment variables

| Variable | Purpose |
|---|---|
| `DB_HOST` `DB_PORT` `DB_NAME` `DB_USER` `DB_PASSWORD` `DB_SSLMODE` | PostgreSQL |
| `REDIS_HOST` `REDIS_PORT` | Redis |
| `MINIO_ENDPOINT` `MINIO_ACCESS_KEY` `MINIO_SECRET_KEY` `MINIO_BUCKET` `MINIO_USE_SSL` `MINIO_PUBLIC_URL` | Object storage |
| `JWT_SECRET` | Signs user session tokens (long random value) |
| `APP_TOKEN_SECRET` | HMAC shared with the web app — must match `kairso-web`'s `APP_SIGNATURE_SECRET` |
| `FIREBASE_CREDENTIALS_PATH` | Path to the Firebase service-account JSON |

---

## Run with Docker (production / self-host)

The server is not meant to run alone — it pairs with `kairso-web` plus PostgreSQL, Redis and S3-compatible storage. Use the self-host bundle in [`deploy/selfhost`](deploy/selfhost), which runs the published images with a single nginx entry point:

```bash
cd deploy/selfhost
cp .env.example .env         # fill in secrets + Firebase
docker compose up -d         # migrations run automatically
```

Then open `http://localhost:8080` (or your `PUBLIC_URL`). You can bring your own PostgreSQL and S3-compatible storage, or use the bundled ones — see the [Install & self-host guide](https://home.kairso.com/install) for the full walkthrough, HTTPS setup and configuration.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
