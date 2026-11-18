# Architecture & Development Conventions

> **Status:** Referensi struktur aktual. Wiring operasional tersedia; hasil verifikasi dan verdict review disimpan terpisah di `develop/`.
>
> **Cakupan:** `backend/hris-core-service`. Dokumen ini menjelaskan struktur yang ditemukan serta pedoman pengembangan minimal; bukan rancangan ulang repository.

## 1. Prinsip arsitektur

Mapatu Backend menggunakan pendekatan **Pragmatic Clean Architecture**: pisahkan tanggung jawab ketika ada manfaat nyata, tetapi jangan menambah package, interface, atau layer hanya untuk memenuhi diagram arsitektur.

- **Pertahankan struktur existing** selama tanggung jawabnya masih masuk akal.
- **Kode Go yang jelas lebih utama** daripada abstraksi, generic wrapper, atau dependency-injection framework.
- **Interface dibuat berdasarkan kebutuhan consumer**, bukan otomatis untuk setiap service.
- **Perubahan struktur perlu dibahas dan disetujui** sebelum dilakukan.
- **Documentation mengikuti implementasi aktual**; rencana dan fitur belum aktif harus diberi label.
- **Pengujian berdasarkan risiko** dan perilaku penting; tidak wajib satu test untuk setiap file.

## 2. Stack yang ditemukan

| Komponen | Implementasi | Catatan |
|---|---|---|
| Go | Directive `go 1.24.0` | Berdasarkan `go.mod` |
| Gin | `1.11.0` | HTTP |
| Cobra | `1.10.2` | CLI |
| GORM | `1.31.2` | ORM yang dituju registry |
| PostgreSQL 17 | Runtime; GORM PostgreSQL + pgx | Satu SQL pool untuk GORM/readiness/metrics/shutdown |
| golang-migrate/v4 | `4.18.3` | Dukungan package migration tersedia |
| Logrus | `1.10.2` | Logging |
| Prometheus client | `1.23.2` | Komponen existing tersambung melalui registry/server |
| Redis client | `v9.22.0` | Dependency tersedia; client belum diinisialisasi |

Ketersediaan dependency **tidak** berarti fiturnya sudah aktif.

## 3. Struktur repository

```text
hris-core-service/
├── main.go
├── cmd/
│   ├── root.go
│   ├── server.go
│   ├── migrate.go
│   ├── seed.go
│   └── bootstrap.go
├── config/
│   └── config.go
├── internal/
│   ├── registry/
│   │   └── registry.go
│   ├── routes/                       # operational/authentication, docs dan metrics
│   ├── handlers/
│   │   ├── authentication_handler.go
│   │   ├── operational_handler.go
│   │   ├── dtos/responses/meta/
│   │   └── mappers/                 # kosong
│   ├── middlewares/
│   │   ├── authentication.go
│   │   ├── authorization.go
│   │   ├── http_hardening.go
│   │   └── metrics_middleware.go
│   ├── usecases/authentication/     # login, token dan typed identity context
│   ├── usecases/authorization/      # explicit grant dan typed resource scope
│   ├── usecases/health/
│   │   ├── readiness.go
│   │   └── lifecycle.go
│   ├── domain/
│   │   ├── models/                  # global user/session identity models
│   │   └── repositories/            # kosong
│   ├── infrastructure/
│   │   ├── authentication/          # concrete GORM auth store
│   │   ├── authorization/           # SQL membership/grant/containment query
│   │   ├── seeding/                 # named fixtures dan atomic first-admin bootstrap
│   │   ├── database.go
│   │   ├── database_readiness.go
│   │   └── migration/
│   └── shared/
│       ├── httphelper/
│       ├── labels/
│       ├── logger/
│       ├── metrics/
│       ├── password/                # bounded Argon2id
│       ├── observability/
│       ├── response/
│       ├── sanitization/
│       ├── security/
│       └── validation/
├── migrations/mysql/                # legacy kosong, dipertahankan
├── migrations/postgres/             # source aktif 000001–000015: identity, tenancy, RBAC
├── docs/                             # generated Swaggo Go/JSON/YAML
├── documentation/
└── develop/
```

Sebagian folder kosong adalah placeholder dan **belum menjadi package Go yang dapat diimpor**. Jangan mengisinya semata-mata agar semua layer tampak lengkap.

## 4. Tanggung jawab package

| Lokasi | Tanggung jawab saat ini | Pedoman |
|---|---|---|
| `main.go`, `cmd/` | Entrypoint Cobra, HTTP startup/shutdown, migration, seed dan bootstrap | Tetap menjadi titik masuk proses |
| `config/` | Load, defaults, override, validasi | Pisahkan konfigurasi aktif vs sisa fitur lama pada pekerjaan tersendiri |
| `internal/registry/` | Komposisi dependency dan penutupan resource | Gunakan registry konkret; tidak perlu DI framework |
| `internal/routes/` | Operational dan authentication routes (login/me/logout/logout-all), metrics dan Scalar/JSON development | Root router; API docs memakai environment aplikasi existing |
| `internal/handlers/` | Penerjemah HTTP request/response | Handler operasional dan authentication memakai envelope existing |
| `internal/middlewares/` | HTTP hardening, metrics, bearer authentication dan scoped authorization | RequirePermission sesudah bearer dan stored-owner resolver |
| `internal/usecases/authentication/`, `internal/usecases/authorization/` | Global identity/session dan explicit scoped grants | Authentication tidak memberi tenant permission |
| `internal/usecases/health/` | Kontrak readiness dan lifecycle state | Pertahankan boundary kecil yang sudah berguna |
| `internal/infrastructure/` | Connector, readiness, auth/access stores, migrator dan provisioning lokal | Runtime berbagi satu SQL pool; CLI memiliki lifecycle sendiri |
| `internal/domain/` | Identity models schema M5A yang dikonsumsi M5B dan placeholder repositories | Model dipakai auth store; tidak ada generic repository |
| `internal/shared/` | Logging, response, observability, validation, dan utility lain | Utility bersama hanya jika benar-benar dibagi consumer |
| `migrations/postgres/` | Source filesystem migration aktif | 15 pasangan sequential: identity/session, tenancy/organization, membership/RBAC dan policy v1.0 |
| `migrations/mysql/` | Legacy | Kosong pada audit, dipertahankan |
| `docs/` | Package Go registrasi spec dan JSON/YAML hasil generator Swaggo | Lokasi existing dipertahankan; `main.go` berisi satu directive `go:generate` yang dipin |
| `documentation/` | Referensi teknis sistem | Tidak menyimpan riwayat percobaan implementasi |
| `develop/` | Lokasi riwayat inputs dan reviews | Input eksekusi dan review independen dipisahkan |

## 5. Alur aplikasi yang tertulis di kode

```text
main.main
  → cmd.Execute
    → command "server"
      → registry.NewRegistry
        → config.LoadConfig
        → logger.NewLogger
        → metrics.NewComponent
        → ConnectDB: satu pool SQL + startup ping berbatas waktu
        → DB stats metrics + lifecycle + database readiness checker
        → handlers.NewOperationalHandler
      → gin.New
      → middleware HTTP
      → registrasi operational/auth routes + metrics opsional + Scalar dan JSON jika development
      → net.Listen → log started → server.Serve
```

Registry menerima GORM dan pool SQL yang sama dari connector, mendaftarkan DB stats pada komponen metrics existing, lalu membuat lifecycle/checker/handler. `Registry.Close()` memiliki ownership pool dengan `sync.Once`. Connector menutup pool bila startup ping/ORM gagal; registry menutup pool bila registrasi stats gagal. Logrus/metrics tidak membuka client eksternal tambahan.

GORM PostgreSQL menggunakan satu pool pgx `database/sql` yang diberikan connector, tanpa automatic ping atau query versi. Startup dan readiness memakai `PingContext` pada pool yang sama; sesi meminta UTF-8 dan UTC. Startup/connection establishment dibatasi konfigurasi dan readiness memakai deadline request 2 detik. Query aplikasi berikutnya tetap memerlukan context; tidak ada global socket read/write deadline. Tidak ada business/schema query, AutoMigrate, layer baru atau Redis initialization. Detail ada pada [operations.md](operations.md) dan [configuration.md](configuration.md).

Migration CLI menggunakan golang-migrate v4.18.3 dengan driver PostgreSQL resmi, source filesystem `migrations/postgres` dan konfigurasi M4A yang sama. CLI memiliki satu pool pgx SQL dan satu koneksi reserved untuk advisory lock, metadata dan SQL; tidak membangun registry/HTTP atau pool backend. Driver/source/pool dibersihkan pada success maupun failure; cleanup failure memengaruhi exit status. Inisialisasi dapat membuat `schema_migrations`, termasuk command version. Statement/lock timeout hanya berlaku pada session migrator; runtime server M4A tidak berubah. Server tidak memanggil migration otomatis.

Dependency GORM MySQL telah dihapus pada M4A. Setelah migrator dipindahkan, `go-sql-driver/mysql` dan dependency uniknya juga dihapus karena tidak ada consumer aktif. `lib/pq` tetap dependency tidak langsung dari driver migrasi PostgreSQL resmi; transport koneksi yang diberikan runner adalah pgx. Directory legacy MySQL yang kosong tetap dipertahankan. Source SQL bisnis, data migration dan production deployment di luar M4B; detail kontrak, empty source dan recovery ada pada [CLI](cli.md#3-postgresql-migration-commands).

`internal/routes/swagger.go` mengimpor package generated `docs` untuk registrasi Swaggo, menyajikan `swag.ReadDoc()` langsung pada `/swagger/doc.json`, dan HTML Scalar pada `/docs`, hanya untuk environment aplikasi `development`. HTML memakai integrasi ESM resmi Scalar v1.69.2 dari jsDelivr. API docs membutuhkan koneksi CDN untuk UI; JSON tetap berasal dari binary aplikasi tanpa pembacaan file runtime.

Alur tetap **anotasi Go → generator Swaggo v1.16.6 → artefak `docs/` → JSON Gin → Scalar**. Swagger 2.0 dan satu specification existing dipertahankan tanpa pipeline konversi atau salinan manual. Scalar hanya presentation layer; tidak ada package/layer/frontend project baru. Nama file route existing dipertahankan karena masih melayani URL specification Swagger.

Dependency UI `swaggo/gin-swagger` dan `swaggo/files` dihapus setelah consumer-nya diganti; `swaggo/swag` tetap diperlukan oleh generated package, generator dan JSON handler. Swagger UI lama tidak lagi didaftarkan. `documentation/` tetap menjadi referensi teknis manual; riwayat M3A di `develop/` dipertahankan.

## 6. Batas tanggung jawab & dependency

[API Standards & HTTP Foundation](api/api-standard.md) menjadi acuan kontrak response/error, helper binding/validation/pagination dan penambahan endpoint. Bedakan middleware/handler aktif dari helper tanpa consumer dan direktori placeholder. Rujukan ini tidak menambah layer atau mengubah kontrak operasional.

**Pola yang diinginkan (pedoman, bukan klaim seluruh kode sudah mengikuti):**

```text
HTTP route → HTTP handler → kebutuhan aplikasi/boundary → implementasi teknis
                    ↘ response helper
Registry menyusun dependency, bukan menyimpan aturan bisnis.
```

Untuk fitur sederhana, alur boleh lebih pendek. Contohnya, liveness yang hanya mengembalikan respons tidak memerlukan `HealthService`, `HealthUsecase`, `HealthRepository`, atau interface tambahan.

Observasi arsitektur: `shared/response` masih mengimpor DTO di `handlers/dtos/responses/meta`. Ini coupling arah package yang layak dipantau, tetapi **bukan alasan melakukan refactoring besar pada M1**.

## 7. Konvensi pengembangan (usulan awal)

| Area | Konvensi |
|---|---|
| Package | Lowercase singkat; gunakan nama yang menggambarkan satu tanggung jawab |
| File Go | Nama deskriptif mengikuti gaya repository, contoh `operational_handler.go` |
| Test | `*_test.go` di dekat package yang diuji; tidak harus ada untuk setiap file |
| Test lintas komponen | Pertimbangkan folder `tests/` hanya jika kelak benar-benar dibutuhkan |
| Test fixtures | `testdata/` di package terkait jika diperlukan |
| Interface | Definisikan saat ada dependency boundary atau consumer yang jelas |
| Constructor | Eksplisit dan mudah diikuti; hindari framework internal |
| Shared utility | Hindari package serbaguna yang tidak memiliki batas tanggung jawab |
| Perubahan folder | Ajukan alasan, dampak, alternatif tanpa relocation, lalu minta persetujuan |
| Penamaan DB | Foreign key relasi berakhiran `_id`; audit actors mengikuti kontrak `created_by`/`updated_by` |

Jangan mengasumsikan seluruh pedoman ini telah menjadi aturan resmi sebelum disetujui.

## 8. Hal yang tidak perlu diadopsi sekarang

- Login limiter aktif pada existing handler dan shared/security; tidak ada framework limiter atau Redis tambahan.
- Redis belum siap pakai: jangan dianggap dependency readiness.
- Kontrak metrics auth/cache/rate limit yang belum punya consumer tidak perlu diaktifkan demi kelengkapan.
- Folder domain, repository, mapper, dan test tidak perlu diisi dengan placeholder kode.
- Perubahan konfigurasi autentikasi dan residu LMS/SPMB dibahas terpisah setelah fondasi operasional stabil.

## PostgreSQL development infrastructure (M4P)

Root `compose.yaml` menyediakan satu service PostgreSQL 17 terpisah untuk development, dengan port host localhost, named volume dan healthcheck. Backend Go dijalankan lokal melalui `go run main.go server` dan kini memakai PostgreSQL tersebut melalui shared pool setelah konfigurasi backend diselaraskan. PostgreSQL menjadi dependency startup/readiness M4A; `POSTGRES_*` Docker tidak otomatis memetakan `DB_*` backend. Tidak ada migration otomatis, custom image, service backend, Redis atau database UI dalam Compose development ini. `Dockerfile` backend existing dan `compose.production.yml` dipertahankan tanpa perubahan. Lihat [konfigurasi](configuration.md#7-postgresql-docker-development-m4p) dan [CLI](cli.md#8-docker--postgresql-development-commands).

## Identity & session foundation (M5A)

Migration PostgreSQL menyediakan `users` sebagai identitas global, bukan employee record. Satu user kelak dapat memiliki membership pada beberapa tenant; users tidak memiliki tenant/company/organization/branch key. `auth_sessions.user_id` mengacu users, dengan banyak sesi per user. M5A membuat dua tabel identity ini; fondasi Module A dijelaskan terpisah di bawah. Membership dan schema RBAC tersedia terpisah di bawah; employee dan audit logs lintas modul belum dibuat.

UUID, audit actors nullable, TIMESTAMPTZ dan normalized-email uniqueness mengikuti [schema identity/session](database/identity-session-schema.md). Semua FK memakai NO ACTION untuk menjaga referensi/riwayat; writer kelak wajib memelihara updated_at. M5B memakai model identity yang memiliki consumer, concrete auth store dan storage interface kecil pada usecase; tidak ada AutoMigrate atau seeding saat startup. SQL diterapkan hanya melalui CLI terpisah ke target terverifikasi; startup/connectivity/readiness M4C tetap sama.

## 9. Cara memperbarui dokumen ini

Setelah perubahan implementasi disetujui dan diverifikasi:

1. Perbarui hanya bagian yang menjadi berbeda dari kode.
2. Ubah penanda `IMPLEMENTASI HILANG` / `usulan` menjadi status aktual **setelah dibuktikan**.
3. Sinkronkan [configuration.md](configuration.md), [operations.md](operations.md), dan [cli.md](cli.md) bila perilakunya berubah.
4. Simpan kronologi perubahan dan review pada `develop/`, **bukan** sebagai riwayat versi panjang di dokumen arsitektur.

**Sumber kebenaran:** source repository aktual. Riwayat audit/input/review berada di `develop/`; jangan menyamakan referensi arsitektur dengan verdict review.

## Authentication foundation (M5B)

`routes/authentication.go` mendaftarkan login publik dan /me yang memakai `middlewares.BearerAuthentication`. `handlers/authentication_handler.go` menangani JSON/envelope serta safe security events; `usecases/authentication` mengelola verifikasi, token dan identity context; `shared/password` menangani Argon2id; concrete `infrastructure/authentication.Store` memakai GORM/pool registry yang sama. Interface Store hanya berisi kebutuhan auth, bukan generic CRUD. DTO response terpisah dari models dan tidak menampilkan hash.

Login memakai transaksi conditional active/hash recheck, audit/login update dan session insert. Middleware melakukan indexed unique hash lookup dengan join user aktif dan validasi expires_at/revoked_at. Context bertipe privat membawa user/session IDs serta data aman untuk /me, tanpa token/hash atau tenant permission. Migration000003 menambah unique token_hash index, tanpa mengubah SQL M5A. Logout dan logout-all memakai store yang sama; JWT, refresh, account default dan migration otomatis tidak tersedia. Permission middleware tersedia terpisah pada fondasi authorization di bawah. Detail kontrak dan batas security ada di [authentication API](api/authentication.md).

## Session lifecycle (M5C)

Handler/usecase/store authentication existing kini menyediakan current logout dan logout-all. Bearer middleware tetap menjadi boundary awal; store melakukan revalidation caller setelah lock user→caller-session dengan timestamp database baru, lalu satu UPDATE terbatasi ownership. Login dan logout-all terserialisasi pada row user yang sama. Login yang terurut setelah logout-all adalah autentikasi baru yang dapat menghasilkan session valid; tidak ada janji account lockout atau penghentian request yang sudah terautentikasi.

Revocation mengisi audit fields tanpa DELETE, trigger, tabel, migration atau layer baru. Expiration/lifetime dan shared pool/config tetap sama. Renewal, rotation, idle timeout dan retention/cleanup belum menjadi kebijakan aktif. Detail kontrak concurrency dan deferred decisions ada di [authentication API](api/authentication.md).

## Tenancy & organization foundation (Module A)

Migration000004–000009 menyediakan enam tabel tenant/organisasi sesuai [kontrak schema resmi](database/tenancy-organization-schema.md). Lima tabel child menyimpan tenant_id; composite FK menjaga konsistensi organization, parent department dan branch lokasi. Department/position berscope organisasi, bukan assignment employee. Multiple head offices diperbolehkan. Audit actor mengacu users global dan tidak memberikan membership/permission.

Ini fondasi schema tanpa consumer aplikasi: tidak ditambah model Go, repository, handler atau endpoint administratif. Authentication global dan lifecycle sesi tetap terpisah dari tenant authorization. PostgreSQL menegakkan scope dan larangan self-parent; siklus hierarki panjang masih memerlukan validasi transaksional pada operasi tulis mendatang. Tidak ada trigger rekursif, AutoMigrate atau geofence evaluation. Membership/RBAC disediakan melalui Module B terpisah.

## Membership & RBAC foundation (Module B)

Migration000010–000014 mengikuti [kontrak membership/RBAC](database/tenant-membership-rbac-schema.md). Membership global user ke tenant terpisah dari employee; uniqueness mempertahankan satu identity untuk reaktivasi. System roles memakai enam tuple UUID/code/scope tetap; role assignments memiliki tepat12 kolom yang dipulihkan pengguna, tanpa assigned_at/expires_at. Composite FK dan scope CHECK menjaga tenant/organization/branch, membership association dan kompatibilitas role. Migration 000015 mengisi delapan permission dan 19 grant eksplisit sesuai RBAC Business Policy v1.0. Otorisasi memerlukan grant dan assignment aktif. API pengelolaan assignment ditunda.

Authorization service berada di internal/usecases/authorization dengan satu Store interface pada consumer boundary, concrete store di internal/infrastructure/authorization menggunakan SQL pool/query timeout existing, dan RequirePermission di middleware sesudah BearerAuthentication. Satu query tanpa cache memverifikasi grant, membership aktif dan kepemilikan resource. Assignment TENANT mencakup resource dalam tenant. Assignment ORGANIZATION mencakup organisasi dan cabang yang dimilikinya langsung. Assignment BRANCH hanya mencakup cabang yang sama dan resource miliknya. Scope PLATFORM terpisah dari data tenant. Hierarki child organization tidak ditelusuri. Hanya hasil yang diizinkan memperoleh typed scope context. Registry menyiapkan fondasi ini; belum ada route business/administratif baru. Error401/403/500 memakai envelope dan completion logger existing.


## Developer provisioning

PostgreSQL migrations own fixed roles, permission codes and grants. Cobra invokes one explicitly named development dataset through a static registry. Its transaction and target validation use the existing PostgreSQL connection infrastructure. The separate bootstrap command reads credentials interactively and uses the same password hashing policy as authentication. Its transaction creates the first platform administrator and assignment together. This separation keeps data fixtures outside the system policy and keeps privileged provisioning outside seeders and HTTP. See the [CLI contract](cli.md#4-development-seeding-and-administrator-bootstrap).

## Login abuse boundary

The authentication handler admits the canonical Gin client IP before decoding and then the PostgreSQL-normalized identifier before invoking password authentication. `shared/security` owns one registry-scoped, concurrency-safe map/expiry heap with bounded digest keys, two explicit fixed-window policies and no workers. Normalization uses the same database `lower(btrim(...))` expression as identity lookup to preserve locale behavior. It adds one bounded query after IP admission, including for throttled identifiers. The two-slot Argon2 resource cap stays in the authentication service. Session creation/expiry/revocation and the membership/explicit-permission authorization boundary are unchanged. Login counters are process-local; fleet-wide guarantees require a deployment control.

HTTP order remains request ID → completion log → optional metrics → safe recovery → security headers (including auth namespace no-store) → CORS → global body boundary → route. Login then applies IP admission → bounded decoder/validation → authoritative normalization/identifier admission → authentication/session creation. Protected routes apply bearer authentication before explicit scoped authorization where applicable. CORS preflight checks method/header policy and merges cache Vary across decisions. Existing server read-header/read/write/idle/drain bounds remain in place; HTTPS/HSTS belong to verified TLS termination, not forwarded-header inference.
