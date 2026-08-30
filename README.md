
## Menjalankan Proyek

### Prasyarat

- Docker & Docker Compose
- (Opsional, untuk development) Go 1.25

### 1. Siapkan environment variable

Salin `env.example` menjadi `.env`, lalu isi nilainya:

```bash
cp env.example .env
```

Variabel yang perlu diisi: kredensial database (`DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`, `DB_HOST`), `APP_PORT`, `JWT_SECRET`, serta email & password seed untuk kasir dan koki.

### 2. Jalankan seluruh sistem (produksi)

```bash
docker compose up -d --build
```

Perintah ini membangun image aplikasi, menyalakan database, menjalankan migrasi otomatis, melakukan seeding akun awal, lalu menyalakan API. Cek kesehatan:

```bash
curl http://localhost:8080/health
```

### 3. Mode development (opsional)

Nyalakan hanya database:

```bash
docker compose -f docker-compose.dev.yml up -d
```

Lalu jalankan aplikasi dari source:

```bash
go run main.go
```

## Menjalankan Test

Unit test berfokus pada service layer (logika bisnis inti):

```bash
go test ./service/ -v -cover
```

## Dokumentasi API

Semua endpoint diawali `/api`. Kecuali login, semua membutuhkan header `Authorization: Bearer <token>`.

### Autentikasi

`POST /api/auth/login` — publik. Body: `email`, `password`. Mengembalikan `token` dan `role`.

### Pesanan

`POST /api/orders` — khusus kasir. Membuat pesanan baru beserta itemnya.

`GET /api/orders` — kasir & koki. Daftar pesanan, terurut waktu masuk. Bisa difilter: `?status=pending`.

`GET /api/orders/{id}` — kasir & koki. Detail satu pesanan.

`PATCH /api/orders/{id}/status` — khusus koki. Mengubah status pesanan (mengikuti aturan transisi).

`DELETE /api/orders/{id}` — khusus kasir. Membatalkan pesanan (hanya saat masih `pending`).

### Realtime

`WS /ws/orders` — koneksi WebSocket. Menyiarkan event `order_created` (pesanan baru) dan `order_updated` (status berubah) ke seluruh client yang terhubung.

## Akun Awal (Seed)

Dibuat otomatis saat aplikasi start, diambil dari environment variable (tidak di-hardcode). Password disimpan dalam bentuk hash bcrypt.

- Kasir — email & password sesuai `SEED_KASIR_*`
- Koki — email & password sesuai `SEED_PEMASAK_*`

## Catatan

Maul & AI is here :)