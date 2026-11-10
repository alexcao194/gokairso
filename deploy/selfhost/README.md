# Kairso — tự host (web + server chạy chung)

Chạy toàn bộ Kairso (web, API, realtime, PostgreSQL, Redis, MinIO) bằng một lệnh
`docker compose`. Image web/server kéo sẵn từ Docker Hub — không cần build.

## Yêu cầu
- Docker + Docker Compose plugin.
- Một **Firebase project của riêng bạn** (để đăng nhập) — xem mục Firebase bên dưới.

## Chạy nhanh
```bash
# 1. Lấy thư mục này (docker-compose.yml, nginx.conf, .env.example)
cp .env.example .env
# 2. Mở .env, điền: KAIRSO_NS, các secret, và cấu hình Firebase
nano .env
# 3. Đặt service-account Firebase cạnh file này:
#    ./firebase-credentials.json
# 4. MinIO chạy user 1001 — cấp quyền thư mục dữ liệu:
mkdir -p data/minio data/postgres && sudo chown -R 1001:1001 data/minio
# 5. Lên!
docker compose up -d
```
Mở **http://localhost:8080** (hoặc `PUBLIC_URL` bạn đặt).

Migration DB chạy tự động (service `migrate`). Dữ liệu nằm ở `./data/` (giữ khi
nâng cấp).

## Cấu hình (`.env`)
| Biến | Ý nghĩa |
|---|---|
| `COMPOSE_FILE` | Chọn chế độ hạ tầng (xem mục dưới) |
| `KAIRSO_NS` / `KAIRSO_TAG` | Namespace + tag image Docker Hub (vd `alexcao` / `v1.0.0`) |
| `PUBLIC_URL` / `PUBLIC_PORT` | URL + cổng người dùng truy cập (dùng để sinh link file) |
| `MAX_UPLOAD_SIZE` | Giới hạn dung lượng upload (mặc định `200m`) |
| `JWT_SECRET`, `APP_TOKEN_SECRET` | **Đổi** thành chuỗi ngẫu nhiên dài. `APP_TOKEN_SECRET` dùng chung web↔server |
| `DB_*` | PostgreSQL (host/port/user/password/name/sslmode) |
| `MINIO_*` | Object storage S3-compatible (endpoint/keys/bucket/ssl/public-url) |
| `REDIS_HOST` / `REDIS_PORT` | Redis |
| `FIREBASE_*` | Cấu hình Firebase web (xem dưới) |

## Dùng DB / object storage của riêng bạn
- **DB**: phải **tương thích PostgreSQL 15+** (Postgres tự host, AWS RDS, Supabase, Neon…). Không chạy MySQL/khác.
- **Storage**: bất kỳ **S3-compatible** nào (AWS S3, Cloudflare R2, Backblaze B2, Wasabi, MinIO…).

Trong `.env`, đổi dòng đầu thành chỉ dùng lõi (không khởi động Postgres/Redis/MinIO gói sẵn):
```
COMPOSE_FILE=docker-compose.yml
```
rồi trỏ tới dịch vụ ngoài:
```
DB_HOST=your-db-host
DB_PORT=5432
DB_SSLMODE=require
MINIO_ENDPOINT=s3.amazonaws.com        # host:port, không kèm https://
MINIO_USE_SSL=true
MINIO_ACCESS_KEY=...
MINIO_SECRET_KEY=...
MINIO_BUCKET=your-bucket
MINIO_PUBLIC_URL=https://your-bucket.s3.amazonaws.com   # URL public trình duyệt tải file
```
Giữ nguyên `COMPOSE_FILE` mặc định (`docker-compose.yml:docker-compose.bundled.yml`) nếu muốn mọi thứ gói sẵn.

## Firebase (bắt buộc để đăng nhập)
1. Tạo project tại <https://console.firebase.google.com>, bật **Authentication →
   Google** (và/hoặc Email).
2. **Cấu hình web** (Project settings → Your apps → SDK config) → điền 6 biến
   `FIREBASE_*` trong `.env`.
3. **Service account** (Project settings → Service accounts → Generate new
   private key) → lưu thành `./firebase-credentials.json`.
4. Thêm domain của bạn vào **Authentication → Settings → Authorized domains**.

## Kiến trúc
Trình duyệt chỉ nói chuyện với **nginx** (cổng duy nhất):
- `/` → web (Next.js). Web gọi backend **nội bộ** qua `/api/proxy` (ký HMAC).
- `/ws` → server (realtime WebSocket, same-origin).
- `/cdn/` → MinIO (file/ảnh).

Nhờ vậy một image web dựng sẵn chạy ở **mọi domain** không cần build lại:
API/WS same-origin, Firebase nạp lúc chạy qua `/env-config.js`.

## Nâng cấp
```bash
docker compose pull && docker compose up -d
```

## Chạy sau reverse proxy / HTTPS
Đặt một reverse proxy (Caddy/Traefik/nginx) phía trước, trỏ về `PUBLIC_PORT`, và
đặt `PUBLIC_URL=https://your-domain`. Không cần sửa gì khác (same-origin).
