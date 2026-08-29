# OrderFlow

Sistem kasir & antrian dapur berbasis web untuk UMKM makanan kebih tepatnya Warung Mang Robin. Menggantikan pencatatan pesanan manual di kertas dengan alur digital yang menjaga urutan pesanan tetap konsisten dan terlihat pemasak secara realtime.

## Latar Belakang

UMKM makanan masih mencatat pesanan di kertas. Saat ramai, kertas menumpuk acak sehingga pesanan tertukar, terlewat, atau tidak dikerjakan sesuai urutan. OrderFlow memberi tiap pesanan nomor urut otomatis dan menyiarkannya ke pemasak secara realtime.

## Peran Pengguna

- **Kasir**: Login, input pesanan pelanggan, edit/batal pesanan selama masih `pending`, memantau status hingga selesai untuk memanggil pelanggan.
- **Pemasak/Koki**: Login, melihat daftar pesanan terurut nomor antrian secara realtime, mengubah status pesanan (`pending` → `cooking` → `done`).

## Status Pesanan

`pending` → `cooking` → `done`, dengan cabang `pending` → `cancelled` (dibatalkan kasir saat masih pending).

## Tech Stack

- **Bahasa:** Go
- **HTTP framework:** Gin
- **Database:** PostgreSQL
- **Realtime:** WebSocket (gorilla/websocket)
- **Auth:** JWT berbasis peran
- **Testing:** testify
- **Deployment:** Docker + GitHub Actions → GitHub Container Registry (GHCR)

## Struktur Proyek
```text
orderflow/
├── main.go               # Titik masuk aplikasi
├── config/               # Loader konfigurasi dari environment variable
├── controllers/          # HTTP controller
├── database/             # Koneksi database & seeding
├── middleware/           # Middleware autentikasi & otorisasi
├── model/                # Entity/struct data
├── repository/           # Akses data ke database
├── routes/               # Registrasi endpoint
├── service/              # Business logic
├── utils/                # Helper function (JWT, hashing, dll.)
├── ws/                   # WebSocket untuk realtime
└── assets/               # Frontend atau file statis (opsional)
```

## Konvensi Commit

Proyek ini memakai [Conventional Commits](https://www.conventionalcommits.org/):
`feat`, `fix`, `chore`,`docs`. Commit dibuat kecil dan bertahap agar progres terbaca jelas.
