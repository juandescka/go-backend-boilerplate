# Mapatu Backend

**Fondasi backend Go yang minimal, mudah dipahami, dan dikembangkan untuk Mapatu HRIS.**

**Bahasa:** [English](README.md) | Bahasa Indonesia

> **Status proyek:** Fondasi backend mencakup operasi, autentikasi, sesi, tenancy, RBAC, provisioning lokal dan keamanan HTTP. Lihat [review kesiapan M9 kanonis](develop/reviews/foundation-readiness/M9-FINAL-review.md) untuk scope terverifikasi dan prasyarat M10. Produk HRIS belum lengkap dan belum bersertifikasi deployment production. Startup memerlukan konfigurasi lokal valid dan PostgreSQL development yang dapat diakses.

## Gambaran umum

Mapatu mengutamakan Pragmatic Clean Architecture, tanggung jawab package yang jelas, Go idiomatis, dan abstraksi seperlunya. Struktur existing dipertahankan. M5B menyediakan login email/password dan current-user authentication dengan opaque bearer token. Otorisasi tenant mengikuti RBAC Business Policy v1.0. Seeder berdasarkan nama dan bootstrap admin pertama tersedia untuk pengembangan lokal.

## Teknologi

Go 1.24 (`go.mod`), Gin, Cobra, GORM + PostgreSQL, golang-migrate, Logrus dan komponen Prometheus opsional existing.

## Struktur proyek

```text
main.go                 Entrypoint CLI
cmd/                    Perintah server, migration, named seeders dan bootstrap
config/                 Loader dan validasi konfigurasi
internal/
  registry/              Inisialisasi dan penutupan dependency
  routes/                Route operasional, autentikasi, docs dan metrics opsional
  handlers/              HTTP handlers
  usecases/health/       Kontrak readiness dan lifecycle
  usecases/authentication/ Login, sesi dan typed identity
  usecases/authorization/ Evaluasi permission dengan scope eksplisit
  infrastructure/        Pool PostgreSQL, stores, migration dan provisioning lokal
  shared/                Utility infrastruktur bersama
documentation/           Referensi teknis
migrations/postgres/     Versi 000001–000015: identity, tenancy dan kebijakan RBAC
migrations/mysql/        Directory legacy kosong, dipertahankan
develop/                 Input pengembangan dan review independen
```

## Mulai menggunakan repository

Gunakan Go yang kompatibel dengan `go.mod` dan database PostgreSQL Docker development.

```bash
go version
go mod download
```

1. Buat `config.json` lokal dari `config.example.json` **hanya jika belum ada konfigurasi lokal**. Pertahankan file lokal existing.
2. Jalankan PostgreSQL melalui quick start Docker di bawah. Atur backend `database.driver=postgres`, host `127.0.0.1`, port yang dipublikasikan (`5433` pada setup lokal M4P existing), dan `sslMode=disable` secara eksplisit untuk Docker lokal. Isi nama/user/password sesuai database terinisialisasi melalui JSON atau `DB_*`; `POSTGRES_*` hanya untuk Docker dan tidak mengatur Go. Port HTTP aplikasi tetap di JSON.
3. Validator existing masih mewajibkan JWT secret untuk kompatibilitas; opaque token tidak memakai JWT signing. Buat nilai lokal yang sebenarnya (jangan commit):

   ```bash
   export JWT_AUTH_SECRET_KEY="$(openssl rand -hex 32)"
   ```

4. Jalankan server dari root repository ini, lalu periksa `/health` dan `/ready` memakai port HTTP konfigurasi (contoh 8080; setup lokal existing 8099):

   ```bash
   go run main.go server
   ```

   Pada terminal lain, gunakan `curl -i http://localhost:8099/health` dan `curl -i http://localhost:8099/ready` jika port konfigurasi adalah 8099.

`.env.example` mencantumkan override yang didukung; pertahankan `.env` existing. `config.json` dan `.env` lokal sudah diabaikan Git. Contoh JSON sengaja tidak berisi JWT secret. Lihat [konfigurasi](documentation/configuration.md) untuk validasi, precedence, pengaturan pool dan opsi yang belum aktif.

## Quick start PostgreSQL Docker (M4P)

Dengan Docker Desktop / Docker Engine dan Compose tersedia, tambahkan nilai `POSTGRES_*` yang belum ada dari `.env.example` ke `.env` lokal yang diabaikan Git, lalu ganti placeholder password dengan password lokal yang dibuat secara acak. Pertahankan pengaturan existing. Jalankan dari root repository ini:

```bash
docker compose up -d
docker compose ps
```

Command ini hanya menjalankan PostgreSQL 17 pada `127.0.0.1:5432` (atau `POSTGRES_PORT`), dengan named storage persisten dan healthcheck. Backend Go berjalan lokal menggunakan PostgreSQL ini setelah konfigurasi koneksi `DB_*` / JSON disesuaikan. Lihat [konfigurasi](documentation/configuration.md#7-postgresql-docker-development-m4p) untuk inisialisasi/precedence dan [command Docker](documentation/cli.md#8-docker--postgresql-development-commands) untuk lifecycle serta pemeriksaan SQL.

## Referensi perintah singkat

| Perintah | Kegunaan |
|---|---|
| `go run main.go --help` | Bantuan CLI |
| `go run main.go server` | Periksa konektivitas DB, lalu layani HTTP |
| `go run main.go migrate --help` | Bantuan command migration yang tersedia |
| `go build ./...` | Build semua package |
| `go test ./...` | Jalankan test terarah |
| `go vet ./...` | Pemeriksaan statis |

Migration CLI PostgreSQL memakai konfigurasi backend yang sama dan `migrations/postgres`. M5A menyediakan migration `users` dan `auth_sessions`; periksa target efektif sebelum menjalankan `go run main.go migrate up`. `version` melaporkan state terpasang dan dapat menginisialisasi metadata. Users adalah identitas global yang terpisah dari employee/tenant; login dan bearer current-user authentication tersedia setelah schema diterapkan secara eksplisit. Lihat [schema identity/session](documentation/database/identity-session-schema.md). `down` melakukan rollback satu versi; `force` hanya untuk recovery metadata manual. Startup server tidak menjalankan migration. Rincian perilaku ada di [referensi CLI](documentation/cli.md).

## Seeder lokal dan bootstrap administrator

```bash
go run main.go seed list
go run main.go seed run demo-tenant --target 127.0.0.1:5433/mapatu_dev
go run main.go bootstrap super-admin --target 127.0.0.1:5433/mapatu_dev
```

Terapkan migration sebelum command yang menulis data dan gunakan target efektif yang tepat. Command tulis memerlukan environment development dan koneksi PostgreSQL lokal yang terverifikasi. Bootstrap memerlukan terminal interaktif, password minimal 15 karakter (UTF-8 valid, maksimal 1024 byte), dan hanya membuat administrator platform pertama. Gunakan cluster lokal yang Anda kelola. Lihat [CLI dan urutan dependensi](documentation/cli.md#4-development-seeding-and-administrator-bootstrap) dan [panduan operasional](documentation/operations.md#local-seeding-and-bootstrap).

## Fondasi tenancy & organization

Modul A menambah migration SQL000004–000009 untuk tenants, organizations, branches, departments, positions dan work_locations. Composite reference menegakkan konsistensi tenant/organisasi; multiple head offices dan lokasi kerja tingkat organisasi didukung. Lihat [kontrak schema](documentation/database/tenancy-organization-schema.md) untuk kolom, scoped uniqueness dan validasi siklus hierarki panjang yang ditunda. Migration membership dan schema RBAC000010–000014 menambah asosiasi tenant, enam system roles dan assignment dengan scope valid. Migration 000015 menambah delapan kode permission dan 19 grant eksplisit. Kebijakan ini memeriksa kepemilikan resource dan cakupan assignment. Lihat [kontrak RBAC](documentation/database/tenant-membership-rbac-schema.md). CRUD organisasi, employee assignment belum diimplementasikan. Deploy migration secara eksplisit setelah memeriksa target; pengujian disposable tidak memigrasikan data development.

## Authentication foundation

`POST /v1/auth/login` memverifikasi email/password dan mengembalikan opaque bearer token; `GET /v1/auth/me` memvalidasi token, expiration/revocation sesi dan aktivasi user. Lihat [API authentication](documentation/api/authentication.md) untuk input, errors, lifetime sesi dan persyaratan deployment. M5C menambah protected `POST /v1/auth/logout` dan `/v1/auth/logout-all`, tanpa menghapus record sesi. Registration, renewal/refresh, tenant permissions dan default user belum diimplementasikan. Logout-all terserialisasi dengan pembuatan sesi; password login yang terurut setelahnya dapat membuat sesi baru. Login memakai batas percobaan IP dan identifier ternormalisasi per proses dengan HTTP 429 dan `Retry-After`. Konfigurasikan origin CORS eksplisit dan IP proxy tepercaya sebelum deployment; deployment publik multi-instance memerlukan perlindungan abuse bersama. Lihat [konfigurasi keamanan](documentation/configuration.md#login-abuse-policy) dan [operasi](documentation/operations.md#authentication-abuse-operations).

## Endpoint operasional

| Method | Path | Perilaku |
|---|---|---|
| `GET` | `/health` | HTTP 200 untuk liveness tanpa pemeriksaan DB |
| `GET` | `/ready` | HTTP 200 jika ping DB berhasil; HTTP 503 saat gagal, timeout, checker tidak tersedia atau shutdown |

Probe berada pada root router dengan middleware global, tanpa authentication/rate limiting login. Readiness memakai SQL pool yang sama dengan deadline 2 detik dan `SERVICE_NOT_READY` saat gagal. DB gagal saat startup mencegah HTTP listener dibuka. SIGINT/SIGTERM mengaktifkan shutdown, melakukan drain HTTP dan menutup pool sekali. Lihat [operasi](documentation/operations.md) untuk respons dan implikasi middleware.

## Referensi API (development)

Dengan `app.env` / `APP_ENV` bernilai `development`, buka [http://localhost:8099/docs](http://localhost:8099/docs) jika `app.port` adalah 8099. Jika berbeda, gunakan port konfigurasi Anda (contoh JSON memakai 8080). Restart server setelah membangun perubahan.

Scalar menggantikan Swagger UI dan membaca specification existing `/swagger/doc.json`. Isinya health/readiness dan keempat operasi autentikasi (login, current user, logout dan logout-all), termasuk readiness 503 dan login 429. `/swagger/index.html` dan aset Swagger UI dihapus; consumer URL JSON tetap memakai path yang sama. Docs dan JSON tidak tersedia di luar development. Persyaratan konfigurasi/database startup existing tetap berlaku.

Scalar v1.69.2 dimuat dari modul jsDelivr dengan versi terkunci, sehingga UI memerlukan akses internet/CDN dan JavaScript. Tidak perlu server frontend, npm build, akun/key Scalar, Registry atau Scalar CLI. Request API langsung menuju backend; AI Agent, telemetry dan proxy eksternal dinonaktifkan. Font sistem menghindari CDN font terpisah.

Swaggo tetap satu-satunya generator. Hasilkan ulang artefak `docs/` existing setelah anotasi berubah:

```bash
go generate .
```

Command ini tetap menjalankan Swaggo v1.16.6 dengan command M3A. Lihat [CLI](documentation/cli.md#5-api-specification-generation).

## Dokumentasi dan proses pengembangan

- [Standar API dan fondasi HTTP](documentation/api/api-standard.md): kontrak response/error, batas binding dan pagination, serta konvensi routing dan middleware.
- [Arsitektur](documentation/architecture.md), [konfigurasi](documentation/configuration.md), [operasi](documentation/operations.md), [CLI](documentation/cli.md).
- [Workflow pengembangan](develop/README-develop.md): input disetujui di `develop/inputs/`, review independen di `develop/reviews/`.
- [Kesiapan fondasi dan prasyarat M10](develop/reviews/foundation-readiness/M9-FINAL-review.md): bukti verifikasi, risiko dan penerimaan development; [checklist lokal aman](documentation/operations.md#foundation-development-checklist).
- Hasil test, bukti runtime dan batasan tidak menggantikan review independen.

## Prinsip pengembangan

Pertahankan struktur, utamakan Go yang mudah dibaca, tambahkan abstraksi bila dibutuhkan, lakukan verifikasi sesuai risiko, dan sinkronkan dokumentasi dengan implementasi. Diskusikan perubahan struktur atau kontrak sebelum diterapkan.
