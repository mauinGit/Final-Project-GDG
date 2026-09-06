<div align="center">

# 🍜 OrderFlow

**Sistem kasir & antrean dapur untuk warung makan**

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Gin](https://img.shields.io/badge/Gin-Framework-008ECF?style=for-the-badge&logo=gin&logoColor=white)](https://gin-gonic.com)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)](https://postgresql.org)
[![WebSocket](https://img.shields.io/badge/WebSocket-Realtime-FF6B35?style=for-the-badge&logo=socketdotio&logoColor=white)](https://github.com/gorilla/websocket)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://docker.com)
[![Swagger](https://img.shields.io/badge/Swagger-OpenAPI-85EA2D?style=for-the-badge&logo=swagger&logoColor=black)](https://swagger.io)

[![CI](https://img.shields.io/github/actions/workflow/status/mauinGit/Final-Project-GDG/ci.yml?style=flat-square&label=CI)](https://github.com/mauinGit/Final-Project-GDG/actions)
![Coverage](https://img.shields.io/badge/service_coverage-93%25-success?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-blue?style=flat-square)

*Final Project — GDGoC UNSRI, Divisi Backend*

</div>

---

## Masalah yang diselesaikan

Di warung makan yang ramai, pesanan berpindah dari kasir ke dapur lewat tiket kertas. Saat antrean panjang, tiket tertukar, terselip, atau terbaca salah. Kasir tidak tahu pesanan mana yang sudah matang, koki tidak tahu mana yang harus didahulukan, dan pelanggan menunggu tanpa kepastian.

OrderFlow menghilangkan kertas dari alur itu. Satu sumber kebenaran di database, satu kanal realtime, dan aturan status yang dijaga di sisi server — sehingga kasir dan dapur selalu melihat hal yang sama pada saat yang sama.

---

## Cara sistem ini bekerja

Ada dua peran: **kasir** dan **koki**. Keduanya login dan mendapat token; setiap endpoint memeriksa peran sebelum menjalankan aksi.

**Kasir** membuat pesanan dengan memilih menu dan jumlahnya. Sistem mengambil nama dan harga dari data menu, menghitung subtotal, menerapkan diskon, lalu mencatat metode pembayaran — tunai dengan kembalian otomatis, atau non-tunai. Nomor antrean dihitung sendiri oleh sistem dan direset setiap hari.

Begitu pesanan tersimpan, **broadcast WebSocket** dikirim ke semua klien yang terhubung. Layar koki langsung memperbarui daftar antrean tanpa perlu refresh.

**Koki** memproses antrean dan mengubah status mengikuti alur yang sudah ditetapkan:

```
pending ──▶ cooking ──▶ done
   │
   └──▶ cancelled     (hanya kasir, hanya saat masih pending)
```

Perubahan di luar alur itu ditolak. Pesanan yang sudah dimasak tidak bisa dibatalkan, dan pesanan yang sudah selesai tidak bisa dikembalikan ke antrean.

Di akhir hari, kasir membuka **laporan harian**: jumlah pesanan, omzet kotor dan bersih, total diskon, pemisahan uang tunai dan non-tunai untuk mencocokkan isi laci, serta menu terlaris.

---

## Keputusan teknis & alasannya

Bagian ini menjelaskan *kenapa* sistem dibangun seperti ini, bukan sekadar *apa* yang dipakai.

**Harga dibekukan di baris pesanan.** `order_items` menyimpan `price_at_order` dan `menu_name` sebagai salinan, bukan sekadar merujuk ke tabel menu. Kalau harga hanya diambil dari menu saat ditampilkan, menaikkan harga hari ini akan mengubah nilai seluruh riwayat transaksi — dan laporan bulan lalu jadi salah. Yang dicatat adalah *berapa yang benar-benar dibayar*, bukan *berapa harganya sekarang*.

**Harga disimpan sebagai `INT`, bukan `DECIMAL` atau `FLOAT`.** Rupiah tidak punya sen, dan aritmetika floating point pada uang adalah sumber bug klasik — `0.1 + 0.2` tidak sama dengan `0.3`.

**Klien tidak dipercaya soal uang.** Kasir hanya mengirim `menu_item_id` dan `quantity`. Nama, harga, subtotal, dan kembalian dihitung sepenuhnya di server. Harga yang diselipkan di body request diabaikan — ada unit test khusus yang menjaga jaminan ini.

**Liveness dan readiness dipisah.** `/healthz` menandakan proses hidup dan tidak menyentuh database; `/readyz` memeriksa koneksi database dan membalas `503` bila putus. Bedanya penting: database tersendat bukan alasan me-restart aplikasi, tapi cukup alasan untuk berhenti mengirim lalu lintas ke sana.

**Refresh token bukan JWT.** JWT bisa diverifikasi tanpa database — bagus untuk access token, buruk untuk sesi yang harus bisa dicabut. Refresh token di sini adalah nilai acak yang disimpan sebagai hash SHA-256, sehingga bisa ditandai terpakai, dicabut, dan diperiksa setiap kali dipakai.

**Reuse detection.** Setiap refresh token hanya boleh ditukar sekali. Jika token yang sudah ditukar dikirim ulang, itu tanda token bocor — dan karena tidak mungkin membedakan pemilik asli dari pencuri, **seluruh sesi pengguna tersebut dicabut**. Pemilik asli harus login ulang; pencuri ikut terputus.

**Nama menu unik tanpa peduli huruf besar-kecil.** `UNIQUE` bawaan PostgreSQL membandingkan byte, sehingga "Nasi Goreng" dan "nasi goreng" lolos sebagai dua entri berbeda. Diganti dengan unique index pada `LOWER(name)`.

**Menu yang sudah pernah dipesan tidak bisa dihapus.** `ON DELETE RESTRICT` menjaga agar riwayat transaksi tidak putus. Percobaan penghapusan dijawab `409`, bukan error database mentah.

**Migrasi berversi.** Runner mencatat setiap file yang sudah dijalankan di tabel `schema_migrations` dan membungkus tiap file dalam satu transaksi. Migrasi tidak pernah dijalankan dua kali, dan kegagalan di tengah tidak meninggalkan skema setengah jadi.

---

## Menjalankan aplikasi

> ⚠️ Pilih **satu** mode saja. Ketiganya memakai port host yang sama (`8080`), jadi menjalankan lebih dari satu bersamaan akan menyebabkan `port is already allocated`.

### Mode 1 — Docker, tanpa clone repo

Cara tercepat mencoba. Cukup punya Docker; tidak perlu Go, tidak perlu source code.

```bash
# Ambil dua berkas ini saja
curl -O https://raw.githubusercontent.com/mauinGit/Final-Project-GDG/master/docker-compose.public.yml
curl -O https://raw.githubusercontent.com/mauinGit/Final-Project-GDG/master/.env.example
mv .env.example .env    # lalu isi nilainya

docker compose -f docker-compose.public.yml pull
docker compose -f docker-compose.public.yml up -d
```

Image ditarik dari GitHub Container Registry. Migrasi dan seeding akun berjalan otomatis saat container start, jadi aplikasi langsung siap dipakai.

Mematikan: `docker compose -f docker-compose.public.yml down`

### Mode 2 — Docker, build dari source

Untuk memverifikasi bahwa kode di repo ini benar menghasilkan image yang sama.

```bash
git clone https://github.com/mauinGit/Final-Project-GDG.git
cd Final-Project-GDG
cp .env.example .env    # isi nilainya

docker compose up -d --build
```

Mematikan: `docker compose down`

### Mode 3 — Lokal, untuk pengembangan

Aplikasi berjalan langsung di host, hanya PostgreSQL yang di container. Paling nyaman saat mengubah kode — cukup restart terminal, tanpa rebuild image.

```bash
cp .env.example .env    # DB_HOST=localhost
docker compose up -d db
go run main.go
```

Aplikasi berjalan di `http://localhost:8080`.

---

## Konfigurasi

Semua variabel di bawah **wajib diisi** kecuali yang bertanda default. Aplikasi menolak start bila ada yang kosong atau tidak valid, dengan pesan yang menyebutkan variabel mana yang bermasalah — gagal cepat lebih baik daripada error misterius di tengah jalan.

| Variabel | Keterangan |
|---|---|
| `DB_USER` / `DB_PASSWORD` / `DB_NAME` | Kredensial PostgreSQL |
| `DB_HOST` | `localhost` untuk Mode 3, `db` untuk Mode 1 & 2 |
| `DB_PORT` | Port PostgreSQL, harus angka |
| `APP_PORT` | Port aplikasi di host — default `8080` |
| `APP_ENV` | `development` atau `production` — default `development` |
| `JWT_SECRET` | **Minimal 32 karakter.** Generate: `openssl rand -base64 48` |
| `CORS_ORIGINS` | Origin yang diizinkan, dipisah koma — default `http://localhost:8080` |
| `SEED_KASIR_EMAIL` / `SEED_KASIR_PASSWORD` | Akun kasir yang dibuat otomatis |
| `SEED_PEMASAK_EMAIL` / `SEED_PEMASAK_PASSWORD` | Akun koki yang dibuat otomatis |

`APP_ENV` menentukan format log: **JSON** di production agar mudah diproses mesin, **teks berwarna** di development agar nyaman dibaca di terminal.

---

## Struktur proyek

```
Final-Project-GDG/
├── main.go                  # Entry point — memuat config, merangkai dependensi,
│                            #   menjalankan server, menangani graceful shutdown
├── config/                  # Membaca & memvalidasi environment variable saat startup
├── logger/                  # Pabrik slog — memilih format JSON atau teks per environment
│
├── models/                  # Struct data & konstanta domain (status, metode bayar).
│                            #   Tanpa logika, dipakai bersama oleh semua lapisan
├── repository/              # Lapisan akses data — satu-satunya tempat yang menulis SQL.
│                            #   Berisi juga integration test bertag `integration`
├── service/                 # Lapisan aturan bisnis — validasi, perhitungan uang,
│                            #   transisi status, reuse detection. Bergantung pada
│                            #   interface, bukan struct konkret, agar bisa di-unit-test
├── controllers/             # Lapisan HTTP — mem-bind request, memanggil service,
│                            #   menerjemahkan error domain menjadi status HTTP
├── routes/                  # Pendaftaran rute & urutan pemasangan middleware
├── middleware/              # Request ID, structured logging, recovery, rate limiter,
│                            #   CORS, security headers, autentikasi & otorisasi peran
├── utils/                   # Helper lintas lapisan — hashing password, JWT,
│                            #   pembuatan & hashing refresh token
├── ws/                      # Hub WebSocket — mengelola klien dan menyiarkan event
│
├── migrations/              # Skema database berurutan (001, 002, …).
│                            #   Dijalankan otomatis saat startup, dicatat di
│                            #   tabel `schema_migrations` agar tidak terulang
├── docs/                    # Spesifikasi OpenAPI hasil `swag init` — jangan diedit manual
├── frontend/                # Halaman demo sederhana untuk melihat alur realtime
│
├── .github/workflows/       # CI — vet, unit test, integration test, lalu push image
├── Dockerfile               # Multi-stage build → image akhir hanya berisi binary
├── docker-compose.yml       # Mode pengembangan (build dari source)
└── docker-compose.public.yml# Mode distribusi (pull image dari GHCR)
```

Alur satu request menembus lapisan secara berurutan:

```
HTTP  →  middleware  →  controllers  →  service  →  repository  →  PostgreSQL
                                          ↓
                                    ws (broadcast)
```

Setiap lapisan hanya mengenal lapisan tepat di bawahnya. `service` tidak tahu apa itu HTTP; `repository` tidak tahu apa itu aturan bisnis. Pemisahan inilah yang membuat lapisan aturan bisnis bisa diuji tanpa database sama sekali.

---

## Pengujian

Dua lapis, dengan tujuan berbeda.

**Unit test** menguji *keputusan* — validasi, perhitungan subtotal dan kembalian, batas diskon, transisi status, reuse detection. Semuanya memakai mock, berjalan dalam hitungan detik.

```bash
go test ./... -cover
```

**Integration test** menguji *penyimpanan* — SQL, constraint, transaksi, dan agregasi. Testcontainers menyalakan PostgreSQL sungguhan untuk setiap test, lalu menghapusnya. Butuh Docker aktif.

```bash
go test -tags=integration ./repository/... -v
```

Hal-hal yang hanya bisa dibuktikan oleh database sungguhan diuji di sini: rollback transaksi saat penyimpanan item gagal, penolakan hapus menu yang sudah dipesan, unique index case-insensitive, dan agregasi laporan yang mengabaikan pesanan batal.

Kedua lapis dijalankan otomatis oleh CI. Image tidak akan dibangun sebelum seluruhnya lolos.

---

## Dokumentasi API

Setelah aplikasi berjalan, buka Swagger UI:

**`http://localhost:8080/swagger/index.html`**

Semua endpoint terdokumentasi lengkap dengan skema request, kemungkinan status response, dan peran yang diizinkan. Tombol **Authorize** di kanan atas memungkinkan mencoba endpoint langsung dari browser — isi dengan `Bearer {token}` hasil login.

### Ringkasan endpoint

Semua di bawah `/api`. Kolom akses menandakan peran yang diizinkan.

**Autentikasi**

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| `POST` | `/auth/login` | Publik | Login, menerima access + refresh token |
| `POST` | `/auth/refresh` | Publik | Tukar refresh token dengan pasangan baru |
| `GET` | `/auth/me` | Kasir, Koki | Identitas pemilik token saat ini |
| `POST` | `/auth/logout` | Kasir, Koki | Cabut refresh token |

**Menu**

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| `GET` | `/menu` | Kasir, Koki | Daftar menu, bisa disaring per kategori |
| `GET` | `/menu/{id}` | Kasir, Koki | Detail satu menu |
| `POST` | `/menu` | Kasir | Tambah menu |
| `PATCH` | `/menu/{id}` | Kasir | Ubah menu |
| `DELETE` | `/menu/{id}` | Kasir | Hapus menu (ditolak bila pernah dipesan) |

**Pesanan**

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| `GET` | `/orders` | Kasir, Koki | Daftar pesanan — berhalaman, filter status & tanggal |
| `GET` | `/orders/{id}` | Kasir, Koki | Detail satu pesanan |
| `POST` | `/orders` | Kasir | Buat pesanan |
| `PATCH` | `/orders/{id}/status` | Koki | Ubah status pesanan |
| `DELETE` | `/orders/{id}` | Kasir | Batalkan pesanan |

**Laporan & operasional**

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| `GET` | `/reports/daily` | Kasir | Laporan penjualan harian |
| `GET` | `/healthz` | Publik | Liveness — aplikasi hidup |
| `GET` | `/readyz` | Publik | Readiness — database terjangkau |
| `WS` | `/ws/orders` | Publik | Kanal realtime perubahan pesanan |

---

## Keamanan & operasional

**Otorisasi berlapis.** Autentikasi memverifikasi token; otorisasi memeriksa peran. Keduanya middleware terpisah, sehingga satu endpoint bisa terbuka untuk kedua peran sementara aksi tulisnya dibatasi.

**Rate limiting per IP.** Batas longgar (20 req/detik) untuk seluruh endpoint sebagai jaring pengaman, plus batas ketat untuk login dan refresh — cukup untuk orang yang salah ketik password, menyiksa untuk skrip brute force.

**Request ID.** Setiap request diberi UUID yang dikembalikan lewat header `X-Request-ID` dan ikut tercatat di setiap baris log. Kalau ada laporan error, satu ID cukup untuk menelusuri seluruh jejaknya.

**CORS dibatasi daftar origin**, bukan wildcard. Origin di luar daftar tidak menerima header CORS sama sekali.

**Graceful shutdown.** Sinyal `SIGTERM` dari Docker tidak langsung memutus proses: server berhenti menerima request baru, menyelesaikan yang sedang berjalan (maksimal 10 detik), lalu menutup pool database dan pekerjaan latar belakang dengan rapi.

---

<div align="center">

Dibangun oleh **Maulana & Claude** — [GitHub](https://github.com/mauinGit)

</div>