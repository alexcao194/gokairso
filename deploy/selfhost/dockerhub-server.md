# kairso-server

The backend for **Kairso** — a workspace, project, KPI, attendance and initiative management platform. This image is the Go API server (REST + realtime WebSocket) and runs database migrations on startup.

- **Multi-arch:** `linux/amd64`, `linux/arm64`
- **Tags:** `latest`, and a version tag per release (e.g. `v0.0.1`)

## Not a standalone image

`kairso-server` is one half of a stack (the other is [`alexcao194/kairso-web`](https://hub.docker.com/r/alexcao194/kairso-web)). It needs PostgreSQL, Redis, S3-compatible storage and a Firebase project. Run the whole stack with the self-host bundle instead of starting this image by hand:

```bash
# in deploy/selfhost
cp .env.example .env         # fill in secrets + Firebase
docker compose up -d         # migrations run automatically
```

## Requirements

- **PostgreSQL 15+** compatible database
- **Redis**
- **S3-compatible** object storage (AWS S3, Cloudflare R2, Backblaze B2, MinIO…)
- A **Firebase project** for authentication

## Key environment variables

| Variable | Purpose |
|---|---|
| `DB_HOST` / `DB_PORT` / `DB_NAME` / `DB_USER` / `DB_PASSWORD` / `DB_SSLMODE` | PostgreSQL connection |
| `REDIS_ADDR` / `REDIS_PASSWORD` | Redis connection |
| `MINIO_ENDPOINT` / `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY` / `MINIO_BUCKET` / `MINIO_USE_SSL` / `MINIO_PUBLIC_URL` | Object storage |
| `JWT_SECRET` | Signs user session tokens — set a long random value |
| `APP_TOKEN_SECRET` | HMAC shared with the web app — must match `kairso-web` |
| `FIREBASE_CREDENTIALS` | Path to the Firebase service-account JSON |

Migrations are embedded and applied automatically at startup (goose).

## Architecture

The browser never talks to this image directly. `kairso-web` proxies REST calls to it (signing `X-App-Token`) and the realtime WebSocket is served at `/ws` on the same origin via nginx.

See the **Install & self-host** guide for the full walkthrough, HTTPS setup and bring-your-own database/storage options.
