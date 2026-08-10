# sekolah-cbt-go

Pengganti Extraordinary CBT: ujian engine berbasis Go, satu binary dijalankan
per sekolah (mirip pola provisioning Extraordinary CBT yang sudah ada), tiap
instance punya schema PostgreSQL sendiri (`DB_SCHEMA`) dalam satu database.

Admin/soal-bank tetap dikerjakan lewat Inertia/React di sekolah.co.id — server
ini hanya expose API, dengan admin request dipercaya lewat JWT yang di-issue
Laravel (shared `JWT_SECRET`). Siswa (peserta ujian) login sendiri lewat
token+no_ujian langsung ke server ini.

Status saat ini: skeleton + skema migrasi inti. Belum ada handler ujian
(login peserta, ambil soal, submit jawaban) — itu tahap berikutnya.

## Setup lokal

Butuh Go >= 1.22 dan akses ke proxy.golang.org buat `go mod tidy` (belum
di-tidy di sini karena sandbox tempat file ini dibuat tidak ada akses
internet ke proxy Go).

```bash
go mod tidy          # generate go.sum
cp .env.example .env # isi DB_*, PORT, JWT_SECRET, SEKOLAH_ID
make migrate         # bikin schema + apply migrations/0001_init_schema.sql
make run              # jalan di :PORT, cek GET /health
```

## Struktur

```
cmd/cbtserver/main.go     entrypoint: `migrate` subcommand atau serve HTTP
internal/config/          load .env + env vars, satu Config per instance
internal/db/              pgx pool, search_path dipin ke DB_SCHEMA
internal/migrate/         migration runner + file .sql (embedded ke binary)
deploy/                   template systemd unit buat provisioning
```

## Kenapa schema-per-sekolah (bukan database terpisah)

Satu binary yang sama dipakai untuk semua sekolah — bedanya cuma `.env`
(DB_SCHEMA, PORT, SEKOLAH_ID) dan port reverse-proxy Nginx-nya. Ini pola yang
sama seperti provisioning Extraordinary CBT sekarang, tapi lebih ringan
karena semua sekolah tetap satu koneksi PostgreSQL instance, tinggal beda
schema — tidak perlu N connection pool/database server terpisah.

## Deploy di aaPanel (rencana provisioning, belum otomatis)

Untuk tiap sekolah:

1. `CREATE SCHEMA` otomatis lewat `cbt-server migrate` (bikin schema kalau
   belum ada, lalu apply migration)
2. Copy binary hasil `make build` + `.env` terisi ke folder instance,
   misal `/www/wwwroot/sekolah.co.id/cbt-instances/{sekolah_id}/`
3. Isi `deploy/cbt-server.service.template`, taruh di
   `/etc/systemd/system/cbt-{sekolah_id}.service`, lalu
   `systemctl enable --now cbt-{sekolah_id}`
4. Bikin reverse proxy Nginx (via aaPanel) ke `127.0.0.1:{PORT}`
5. Sync peserta dari Buku Induk (reuse logic sync yang sudah ada)

Port dialokasikan dari range 13000-14000 supaya tidak bentrok dengan
Extraordinary CBT yang masih pakai 12000-13000 kalau migrasi bertahap
per sekolah.
