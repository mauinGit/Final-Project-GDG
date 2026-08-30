# ---------- Tahap 1: Build ----------
# Pakai image Go lengkap untuk meng-compile aplikasi.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Salin file dependency dulu agar layer cache Docker efisien:
# selama go.mod/go.sum tidak berubah, download tidak diulang.
COPY go.mod go.sum ./
RUN go mod download

# Salin sisa kode lalu compile menjadi satu binary statis.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./main.go

# ---------- Tahap 2: Runtime ----------
# Image akhir super kecil, hanya berisi binary hasil compile.
FROM alpine:latest

WORKDIR /app

# Sertifikat CA agar koneksi TLS (bila ada) berfungsi.
RUN apk --no-cache add ca-certificates

# Salin HANYA binary dari tahap build, plus folder migrasi.
COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations

# Port yang diekspos aplikasi.
EXPOSE 8080

# Jalankan aplikasi.
CMD ["./server"]