# M1-01 — Operational Health & Readiness Implementation

**Project:** Mapatu Backend Boilerplate  
**Repository:** `backend/hris-core-service`  
**Feature:** `operational/health-readiness`  
**Mode:** IMPLEMENTATION  
**Iteration:** M1-01  
**Architecture Strategy:** Option A — Minimal Preservation  
**Reference:** M0 — Architecture Audit & Planning

---

## 1. ROLE & OBJECTIVE

Bertindak sebagai Senior Go Backend Engineer yang berpengalaman dalam membangun backend production-oriented menggunakan Go, Gin, GORM, MySQL, dan Pragmatic Clean Architecture.

Tugas Anda adalah menyelesaikan **M1 — Operational Health & Readiness**, dengan melengkapi implementasi yang sudah ada agar backend dapat dijalankan dan menyediakan endpoint operasional yang berfungsi.

Tujuan utama:

1. Melengkapi database connection dan readiness checker.
2. Menyelesaikan dependency initialization pada registry.
3. Mendaftarkan route Health dan Readiness.
4. Memastikan startup dan graceful shutdown konsisten.
5. Mempertahankan struktur package dan kontrak yang sudah ada.
6. Memperbarui dokumentasi yang terdampak.
7. Melakukan verifikasi yang relevan tanpa overtesting.

**Jangan membangun ulang fitur yang sebenarnya sudah tersedia.**

Prioritaskan perubahan terkecil yang mampu menyelesaikan masalah aktual.

---

## 2. REQUIRED READING

Sebelum melakukan perubahan, baca dan pahami:

### Project Instructions

- `AGENTS.md`
- `AGENTS.id.md`, jika diperlukan
- `develop/README.md`
- `develop/TEMPLATE_PROMPT.md`
- `develop/FIRST_PROMPT.md`, jika konteks proyek belum dipahami

### Technical Documentation

- `README.md`
- `README.id.md`
- `documentation/architecture.md`
- `documentation/configuration.md`
- `documentation/operations.md`
- `documentation/cli.md`

### Relevant Source Code

Periksa implementasi aktual:

- `main.go`
- `cmd/root.go`
- `cmd/server.go`
- `config/config.go`
- `internal/registry/registry.go`
- `internal/handlers/operational_handler.go`
- `internal/usecases/health/readiness.go`
- `internal/usecases/health/lifecycle.go`
- `internal/middlewares/`
- `internal/shared/response/`
- `internal/shared/observability/`
- `internal/shared/metrics/`
- `internal/routes/`
- `internal/infrastructure/`
- `go.mod`

Periksa pula konfigurasi contoh dan file lain yang berhubungan langsung dengan startup.

Jika tersedia, gunakan laporan M0 sebagai referensi temuan awal.

**Source code aktual adalah sumber kebenaran mengenai implementasi.**

Jangan menganggap laporan M0 masih sepenuhnya sesuai apabila repository telah mengalami perubahan.

Jika dokumentasi berbeda dari source, identifikasi perbedaannya dan perbaiki dokumentasi yang memang berada dalam scope M1.

---

## 3. CURRENT CONTEXT

Berdasarkan audit M0, ditemukan:

### Existing Components

- Operational handler untuk Health dan Readiness.
- `ReadinessChecker` interface.
- Lifecycle state untuk shutdown.
- Registry untuk dependency initialization.
- HTTP middleware dan response helper.
- Metrics component.
- CLI berbasis Cobra.

### Known Problems

- Folder routes belum memiliki implementasi.
- Database connector belum tersedia.
- Database readiness checker belum tersedia.
- Registry belum sepenuhnya sesuai dengan consumer metrics.
- Konfigurasi contoh masih kosong.
- Runtime belum terverifikasi.
- Beberapa komponen LMS/SPMB yang tidak aktif masih terbawa.

Temuan tersebut merupakan baseline audit statis.

**Verifikasi kembali sebelum melakukan perubahan.**

Jangan melakukan refactoring terhadap komponen yang tidak berkaitan dengan M1 hanya karena ditemukan selama inspeksi.

---

## 4. ARCHITECTURE CONSTRAINTS

Gunakan pendekatan:

**Pragmatic Clean Architecture + Minimal Preservation.**

### Mandatory Rules

- Pertahankan struktur folder existing.
- Pertahankan konvensi penamaan package dan file.
- Gunakan registry konkret yang sudah tersedia.
- Gunakan operational handler yang sudah tersedia.
- Gunakan `ReadinessChecker` existing.
- Hindari interface baru jika tidak diperlukan.
- Jangan menambahkan service/usecase untuk liveness sederhana.
- Jangan membuat dependency injection framework.
- Jangan membuat generic repository.
- Jangan membuat package baru hanya untuk satu helper sederhana.
- Jangan mengadopsi kompleksitas LMS/SPMB secara keseluruhan.
- Jangan mengubah kontrak HTTP existing tanpa kebutuhan dan persetujuan.

Penambahan file dalam folder existing berikut merupakan bagian dari rencana M1:

- `internal/routes/`
- `internal/infrastructure/`

Namun, nama file dan bentuk implementasi akhir harus mengikuti hasil pemeriksaan source.

### Changes Requiring Approval

Hentikan bagian pekerjaan terkait dan minta persetujuan apabila diperlukan:

- Memindahkan atau mengubah nama package/folder.
- Mengubah kontrak API yang sudah disepakati.
- Mengganti GORM atau MySQL.
- Menambahkan layer arsitektur baru.
- Mengubah kebijakan konfigurasi secara luas.
- Menghapus modul atau komponen di luar scope.
- Mengubah database schema.
- Memperkenalkan dependency besar yang tidak direncanakan.

Penambahan driver MySQL yang diperlukan untuk GORM diperbolehkan jika belum tersedia dan dapat dijustifikasi.

---

# 5. IMPLEMENTATION SCOPE

## PHASE A — Database Connection

Periksa kebutuhan database pada registry dan configuration.

Lengkapi database connector minimal dengan mempertahankan pendekatan GORM + MySQL.

Persyaratan:

1. Gunakan konfigurasi database existing.
2. Gunakan satu database connection pool yang dikelola oleh registry.
3. Terapkan connection pool settings yang sudah didefinisikan.
4. Gunakan timeout yang terbatas untuk startup connectivity check.
5. Gunakan context bila didukung oleh operasi terkait.
6. Pastikan resource ditutup ketika startup gagal setelah koneksi dibuat.
7. Hindari pembocoran database credentials dalam log dan error.
8. Hindari query ke tabel bisnis.
9. Jangan menjalankan migration secara otomatis.

Jika connector dari LMS/SPMB dijadikan referensi, adaptasi bagian yang benar-benar dibutuhkan.

Jangan menyalin custom instrumentation, logger adapter kompleks, atau abstraction tambahan tanpa kebutuhan nyata.

## PHASE B — Database Readiness Checker

Lengkapi implementasi `ReadinessChecker` menggunakan database pool yang sama.

Expected behavior:

- Menggunakan `PingContext(ctx)` melalui `*sql.DB`.
- Menghormati context cancellation dan deadline.
- Tidak membuat connection pool baru setiap request.
- Tidak memeriksa schema bisnis.
- Tidak menjalankan migration.
- Tidak bergantung pada Redis.
- Tidak menjalankan query tambahan yang tidak diperlukan.

Pertahankan interface existing jika sudah memenuhi kebutuhan.

Jangan membuat interface baru untuk membungkus satu pemanggilan sederhana.

## PHASE C — Registry Wiring

Periksa `internal/registry/registry.go`.

Pastikan dependency initialization memiliki urutan yang benar:

1. Load configuration.
2. Initialize logger.
3. Initialize existing metrics component jika diperlukan oleh wiring.
4. Initialize database connection.
5. Initialize lifecycle state.
6. Initialize database readiness checker.
7. Initialize operational handler.
8. Return registry yang konsisten.

Perbaiki ketidaksesuaian antara `Registry` dan pemanggilan `reg.Metrics`.

Gunakan solusi terkecil yang mempertahankan capability metrics existing.

Jangan mengembangkan metrics baru.

Pastikan `Registry.Close()`:

- Menutup resource yang dimiliki registry.
- Tidak menutup resource yang sama secara tidak aman.
- Aman terhadap pemanggilan berulang jika desain existing mengharuskannya.

Jika initialization gagal, lakukan cleanup terhadap resource yang sudah berhasil dibuat.

## PHASE D — Operational Routes

Lengkapi route registration pada package existing:

`internal/routes/`

Endpoint:

- `GET /health`
- `GET /ready`

Pertahankan `OperationalHandler` existing.

Jangan membuat handler baru jika tidak diperlukan.

Pastikan:

- Kedua endpoint berada di luar authenticated API group.
- Tidak memerlukan JWT atau permission.
- Tidak terkena authentication rate limiting.
- Menggunakan middleware global yang relevan.
- Tidak membuat route alias tanpa kebutuhan.
- Tidak menambahkan API version prefix ke endpoint operasional tanpa persetujuan.

Gunakan nama fungsi route registration yang sederhana dan konsisten dengan consumer.

Jangan menambahkan registry/factory khusus untuk route.

## PHASE E — Health / Liveness

Pertahankan semantik liveness existing.

Expected behavior:

- HTTP 200 ketika HTTP process dapat melayani request.
- Response menggunakan envelope existing.
- Tidak memeriksa database.
- Tidak memeriksa Redis.
- Tidak memeriksa external services.
- Tidak menjalankan logic bisnis.

Jangan melakukan refactoring jika handler sudah benar.

## PHASE F — Readiness

Pertahankan readiness handler existing jika semantiknya sudah sesuai.

Expected behavior:

| Condition | Expected |
|---|---|
| Database healthy | HTTP 200 |
| Database ping failed | HTTP 503 |
| Database checker unavailable | HTTP 503 |
| Database timeout | HTTP 503 |
| Application shutting down | HTTP 503 |

Pertahankan error code existing:

`SERVICE_NOT_READY`

Pertahankan readiness timeout 2 detik jika implementasinya memang sesuai audit.

Jangan mengirim raw database error kepada client.

Jika diperlukan, catat reason internal secara aman melalui logger.

Jangan menambahkan detail sensitif dalam response.

## PHASE G — Startup & Shutdown

Periksa startup dan shutdown pada `cmd/server.go`.

Pertahankan pendekatan **startup fail-fast terhadap database**.

Expected flow:

```text
Configuration
    ↓
Dependency Initialization
    ↓
Database Connectivity Check
    ↓
HTTP Router Setup
    ↓
HTTP Server Start
```

Jika database tidak tersedia saat startup, aplikasi boleh gagal sebelum membuka HTTP listener, sesuai strategi existing.

Setelah startup berhasil:

- Health tetap 200 ketika database mengalami gangguan sementara, selama HTTP masih dapat dilayani.
- Readiness mengembalikan 503 ketika dependency kritis tidak tersedia.

Untuk shutdown:

1. Tangkap signal termination sesuai implementasi existing.
2. Aktifkan lifecycle shutdown state.
3. Lakukan graceful HTTP shutdown.
4. Tutup database pool.
5. Hindari double-close yang tidak aman.

Periksa apakah startup logging menggambarkan status server secara akurat.

Jangan menambahkan drain delay, shutdown endpoint, atau orchestration framework tanpa kebutuhan nyata.

## PHASE H — Configuration Examples

Periksa konfigurasi aktual dan validasinya.

Lengkapi jika diperlukan:

- `config.example.json`
- `.env.example`

Persyaratan:

- Gunakan struktur dan key yang benar-benar didukung loader.
- Jangan membuat key konfigurasi yang belum tersedia.
- Jangan mengubah precedence konfigurasi.
- Jangan menyimpan credential sungguhan.
- Jangan menimpa `.env` dan `config.json` lokal.
- Jangan mengubah konfigurasi production.
- Tandai konfigurasi fitur yang belum aktif jika masih diwajibkan oleh validator.

Jika validasi fitur yang belum diimplementasikan menghalangi startup, cari solusi minimal yang tetap mematuhi kontrak existing.

Jangan menggunakan credential atau secret palsu hanya untuk menyembunyikan masalah validasi.

Jika penyelesaiannya membutuhkan perubahan kontrak konfigurasi yang belum disetujui, laporkan sebagai BLOCKED dan minta keputusan.

---

# 6. DOCUMENTATION REQUIREMENTS

**Documentation synchronization is mandatory when applicable.**

Setiap perubahan implementasi harus dievaluasi dampaknya terhadap dokumentasi.

Periksa dan perbarui dokumen berikut apabila relevan:

### `documentation/operations.md`

Sinkronkan:

- Health dan readiness contracts.
- Response status.
- Timeout.
- Dependency checking.
- Startup behavior.
- Shutdown behavior.
- Middleware implications.
- Keterbatasan yang masih ada.

### `documentation/configuration.md`

Sinkronkan:

- Konfigurasi MySQL.
- Connection pool.
- Startup requirements.
- Supported environment overrides.
- Contoh konfigurasi.

### `documentation/architecture.md`

Perbarui hanya jika:

- Tanggung jawab package berubah.
- Dependency initialization berubah.
- Ada file atau komponen arsitektur penting yang ditambahkan.

Jangan menulis ulang seluruh dokumentasi arsitektur.

### `documentation/cli.md`

Perbarui status dan contoh command hanya setelah memeriksa implementasinya.

Jangan mendokumentasikan `seed` sebagai command aktif jika belum tersedia.

### `README.md` & `README.id.md`

Jika startup, setup, atau command berubah:

- Perbarui kedua bahasa.
- Pastikan instruksi konsisten.
- Tetap gunakan format ringkas dan programmer-friendly.

### Documentation Rules

1. Dokumentasi harus sesuai dengan kode aktual.
2. Jangan mengklaim fitur sudah berfungsi jika belum diverifikasi.
3. Jangan menghapus informasi penting tanpa alasan.
4. Jangan membuat file dokumentasi baru bila dokumen existing sudah memadai.
5. Jika dokumen baru benar-benar diperlukan, gunakan `documentation/` dan jelaskan alasannya.
6. Jangan mencampurkan riwayat iterasi implementasi ke dalam dokumentasi teknis.
7. Dokumentasi yang terdampak merupakan bagian dari acceptance criteria, bukan pekerjaan opsional setelah milestone ditutup.

Apabila tidak ada perubahan dokumentasi yang diperlukan, nyatakan alasannya dalam laporan akhir.

**Aturan ini juga berlaku untuk seluruh input FIX berikutnya.**

---

# 7. VERIFICATION STRATEGY

Gunakan **risk-based verification**.

Tidak wajib membuat test untuk setiap file.

Tidak ada target code coverage arbitrer.

Prioritaskan pemeriksaan berikut sesuai kebutuhan:

### Compilation

- `go build ./...`

Pastikan tidak ada package, import, atau symbol yang hilang.

### Static Check

- `go vet ./...` jika relevan.

### Focused Tests

Jika diperlukan, tambahkan pengujian terarah pada:

- Health response.
- Readiness healthy/unhealthy.
- Shutdown readiness.
- Context cancellation atau timeout.
- Route registration.

Gunakan `*_test.go` di package terkait.

Jangan otomatis membuat folder `tests/`.

### Runtime / Smoke Verification

Jika environment development tersedia:

- Jalankan server menggunakan command yang benar-benar didukung.
- Periksa `/health` dan `/ready`.
- Verifikasi response dan status HTTP.

Gunakan database development/test yang terisolasi.

Jangan menggunakan production database.

Jika database tidak tersedia, jangan membuat database palsu atau mengklaim runtime verification PASS.

Catat batasan tersebut secara eksplisit.

### Prohibited Verification

Tanpa persetujuan eksplisit, jangan:

- Menjalankan migration destruktif.
- Menjalankan `migrate down`.
- Menjalankan seed.
- Menghapus data.
- Mengubah production environment.
- Menjalankan load/stress test.
- Membuat testing framework baru.

---

# 8. DEVELOPMENT WORKFLOW

Ikuti `develop/README.md`.

Untuk iterasi ini gunakan:

```text
develop/
├── inputs/
│   └── operational/
│       └── health-readiness/
│           └── M1-01-implementation.md
│
└── reviews/
    └── operational/
        └── health-readiness/
            └── M1-01-review.md
```

### Current Iteration

Buat atau simpan input kerja:

`develop/inputs/operational/health-readiness/M1-01-implementation.md`

File tersebut harus mendokumentasikan scope dan instruksi eksekusi yang disetujui.

Jangan menimpa input yang telah digunakan pada iterasi sebelumnya.

### After Implementation

Laporkan hasil implementasi aktual sebagai handoff untuk REVIEW.

**Jangan membuat `M1-01-review.md` dengan status PASS secara otomatis.**

Review dilakukan sebagai tahap terpisah yang memeriksa source dan perubahan aktual.

Pada tahap REVIEW:

- Jika PASS, milestone dapat ditutup setelah dokumentasi sesuai.
- Jika FAIL, buat input FIX baru menggunakan ID `M1-02`.
- Jika BLOCKED, laporkan dependency atau keputusan yang diperlukan.

Jangan membuat input perbaikan spekulatif sebelum review menemukan masalah.

---

# 9. OUT OF SCOPE

Jangan mengimplementasikan:

- Authentication.
- Authorization/RBAC.
- User management.
- Tenant management.
- Employee management.
- Attendance.
- Rate limiter.
- Redis initialization.
- Seeder.
- Business migrations.
- New metrics capabilities.
- Generic error-handling framework.
- Deployment orchestration.

Jangan membersihkan seluruh sisa LMS/SPMB dalam milestone ini.

Foundation simplification yang tidak langsung menghambat Health dan Readiness merupakan kandidat M2.

---

# 10. ACCEPTANCE CRITERIA

Implementasi M1-01 dianggap siap untuk REVIEW apabila:

- [ ] Database connector tersedia dan menggunakan konfigurasi existing.
- [ ] Satu database pool digunakan dan dikelola dengan benar.
- [ ] Database readiness checker menggunakan context-aware ping.
- [ ] Registry berhasil menginisialisasi dependency secara konsisten.
- [ ] Wiring metrics yang menghambat compilation telah diperbaiki.
- [ ] Operational routes terdaftar.
- [ ] Health tidak bergantung pada database.
- [ ] Readiness mengembalikan status yang sesuai.
- [ ] Startup failure memiliki error handling yang aman.
- [ ] Graceful shutdown dan resource cleanup diperhatikan.
- [ ] Tidak ada perubahan struktur package yang tidak disetujui.
- [ ] Tidak ada pengembangan fitur di luar scope.
- [ ] Contoh konfigurasi diperbarui jika diperlukan.
- [ ] Dokumentasi yang terdampak telah disinkronkan.
- [ ] Pemeriksaan relevan dijalankan atau keterbatasannya dicatat.
- [ ] Seluruh perubahan dapat ditelusuri melalui development input.

Jika salah satu kriteria belum dapat dipenuhi, laporkan status dan alasannya.

Jangan memberikan verdict REVIEW PASS sendiri.

---

# 11. REQUIRED OUTPUT

Setelah implementasi, tampilkan:

## A. Implementation Summary

Ringkasan pekerjaan yang benar-benar selesai.

## B. Files Changed

Tabel:

| File | Action | Reason |
|---|---|---|
| ... | Added/Modified | ... |

## C. Behavior

Jelaskan perilaku aktual:

- Health.
- Readiness.
- Database connectivity.
- Startup.
- Shutdown.

## D. Verification Results

Tabel:

| Check | Status | Evidence |
|---|---|---|
| Build | PASS/FAIL/NOT RUN | ... |
| Static check | PASS/FAIL/NOT RUN | ... |
| Relevant tests | PASS/FAIL/NOT RUN | ... |
| Runtime verification | PASS/FAIL/NOT RUN | ... |

Jangan mengarang hasil command.

## E. Documentation Changes

Tabel:

| Document | Action | Reason |
|---|---|---|
| ... | Updated/Added/Unchanged | ... |

Jelaskan jika tidak ada perubahan yang diperlukan.

## F. Deviations & Blockers

Jelaskan:

- Kebutuhan perubahan di luar approval.
- Masalah yang belum terselesaikan.
- Dependency yang tidak tersedia.
- Risiko teknis yang masih terbuka.

## G. Review Handoff

Nyatakan bahwa implementasi siap untuk diperiksa melalui:

`develop/reviews/operational/health-readiness/M1-01-review.md`

Berikan referensi file, acceptance criteria, dan bukti verifikasi agar reviewer dapat menilai implementasi secara independen.

---

# 12. STOP CONDITIONS

Hentikan bagian pekerjaan terkait dan minta keputusan apabila:

- Diperlukan restructuring yang belum disetujui.
- Diperlukan perubahan public API contract.
- Diperlukan perubahan konfigurasi yang breaking.
- Diperlukan operasi database yang destruktif.
- Diperlukan penggantian teknologi utama.
- Ada konflik antara keputusan proyek dan implementasi yang tidak dapat diselesaikan dalam scope.
- Diperlukan akses production atau credential yang tidak tersedia.

Untuk masalah nonkritis, lanjutkan bagian scope lain yang aman dan laporkan keterbatasannya.

---

# FINAL INSTRUCTION

Implement **M1-01 — Operational Health & Readiness** using the existing Mapatu backend structure.

Preserve existing architecture and behavior wherever possible.

Complete missing operational wiring without introducing unnecessary layers, abstractions, or features.

Keep technical documentation synchronized with actual implementation.

Perform verification proportional to the risk.

Record implementation evidence and prepare a review handoff.

**Do not declare the milestone PASS before an independent REVIEW.**

---

# Implementation execution evidence — 11 November 2026

**Execution status:** Implemented; independent REVIEW pending. This section is an implementation handoff, not a review verdict. Approval comes from the active user request containing the input above. No prior input or review was overwritten.

## Source baseline and scope

- The actual workflow file is [`../../../README-develop.md`](../../../README-develop.md); `develop/README.md` does not exist. Its workflow was followed without renaming files or changing unrelated links.
- No separate M0 report was available. Existing source/docs confirmed missing connector/checker/routes and missing `Registry.Metrics`.
- `AGENTS.md`, `AGENTS.id.md`, `README.md`, `README.id.md`, `develop/` and `documentation/` already contained user changes/untracked files before execution. Agent instructions and unrelated workflow files were not edited.
- Existing operational handler, `ReadinessChecker`, lifecycle, response envelope, middlewares and metrics implementation were preserved. No configuration-loader policy changes, migrations, business features, Redis initialization or package moves were made.

## Actual changes

All implementation paths below are relative to the repository root.

| File | Action | Reason |
|---|---|---|
| `internal/infrastructure/database.go` | Added | One SQL pool, configured limits, bounded startup ping, GORM using same pool, safe failure cleanup/messages |
| `internal/infrastructure/database_readiness.go` | Added | Existing readiness interface backed by shared `PingContext` |
| `internal/infrastructure/database_readiness_test.go` | Added | Missing/closed pool and canceled-context behavior without a test DB server |
| `internal/routes/routes.go` | Added | Public root GET probes and existing optional metrics endpoint |
| `internal/routes/routes_test.go` | Added | Probe envelopes/statuses, shutdown, timeout/cancellation, global middleware and metrics route |
| `internal/registry/registry.go` | Modified | Expose existing metrics component and own connector's shared pool; preserve once-only close |
| `cmd/server.go` | Modified | Wire simple route setup; register signals before bind; log started after listener bind |
| `go.mod`, `go.sum` | Modified | Authorized GORM MySQL driver 1.6.0; required SQL driver 1.8.1 and edwards25519 transitive dependency |
| `config.example.json`, `.env.example` | Modified | Supported keys and defaults, no real/shared secrets, mark inactive options |
| `README.md`, `README.id.md` | Updated | Aligned setup, commands and operational behavior in both languages |
| `documentation/operations.md` | Updated | Probe contracts, 2s deadline, fail-fast startup, shutdown, middleware and limitations |
| `documentation/configuration.md` | Updated | Actual overrides/units, pool/timeouts, retained JWT requirement and inactive settings |
| `documentation/architecture.md` | Updated | Existing package map and new wiring/ownership |
| `documentation/cli.md` | Updated | Confirmed commands, migration execution limits, no active seed command |
| `develop/inputs/operational/health-readiness/M1-01-implementation.md` | Added | Preserve approved input and record this execution evidence |

## Verification actually executed

Go version: `go1.24.0`. Commands run from `backend/hris-core-service` unless stated otherwise.

| Check | Status | Evidence |
|---|---|---|
| Dependency update | Completed | `GOPROXY=off GOSUMDB=off go get gorm.io/driver/mysql@v1.6.0`, then `GOPROXY=off GOSUMDB=off go mod tidy`; used cached modules, limited diff |
| Build | PASS | `GOCACHE=/private/tmp/hris-m1-01-go-cache go build ./...`, exit 0 |
| Static check | PASS | `GOCACHE=/private/tmp/hris-m1-01-go-cache go vet ./...`, exit 0 |
| Relevant tests | PASS | `GOCACHE=/private/tmp/hris-m1-01-go-cache go test ./...`, exit 0; infrastructure and routes tests passed |
| CLI help | PASS | Built `/private/tmp/hris-m1-01-mysql/hris-core-service`; ran `--help`, `server --help`, `migrate --help`, exit 0; confirmed up/down/version/force and no seed |
| Runtime smoke | PASS | `python3 /private/tmp/hris-m1-01-mysql/smoke.py`, exit 0; actual isolated MySQL 5.7.24, loopback DB 33317 and HTTP 33318 |
| Diff formatting | PASS | `git diff --check`, exit 0 |
| Local configuration preservation | Confirmed | `.env` and `config.json` remained 0 bytes, as at initial inspection |

Initial default-cache build/vet/test attempts ended with a sandbox error writing `~/Library/Caches/go-build/trim.txt`. Checks were rerun successfully with a writable temporary cache. Initial MySQL bind was sandbox-blocked; approved execution outside the sandbox enabled only the isolated local smoke test. No production/shared database was used.

### Runtime observations

- Initialized a dedicated MySQL data directory in `/private/tmp/hris-m1-01-mysql/data`; created only empty `hris_m1_01_smoke` and a dedicated local user with a generated password. No business tables, migrations, or seeds were run.
- Config was derived from `config.example.json` in a separate temporary working directory; port/DB name/user were changed for isolation. DB password and JWT secret were generated for this run and passed through supported environment overrides without printing them.
- `/health` 200 and `/ready` 200 with existing envelope, request ID, security headers and no auth.
- Metrics disabled: `/metrics` 404. Disallowed CORS Origin: `/health` 403, consistent with global middleware.
- Starting a second HTTP instance on the occupied test port failed without a false `application.server.started` event.
- Metrics enabled: `/metrics` 200; existing `hris_core_service_db_connections_max_open` gauge was 25; ready route observations were present.
- Temporarily paused the real isolated MySQL process: `/ready` returned 503 in **2.00s** with `SERVICE_NOT_READY`; `/health` remained 200. Resuming MySQL restored `/ready` 200.
- Stopped only the temporary MySQL instance: health remained 200, readiness returned safe 503. Starting a fresh application with DB unavailable failed in **0.05s** before opening HTTP.
- SIGTERM shutdown returned exit 0 twice, with shutdown-started then shutdown-completed events. All application processes and the temporary MySQL instance were stopped after smoke verification.
- Application logs/startup error output did not contain the generated DB password or JWT secret.

The temporary smoke script/logs are local execution artifacts, not a new repository test framework. Unit checks are reproducible from tracked source; live checks require an isolated MySQL environment and actual local configuration.

## Acceptance-criteria evidence for reviewer

| Criterion group | Implementation/evidence |
|---|---|
| Connector, one owned pool, bounded connectivity, cleanup | `database.go`, registry wiring, runtime healthy/outage/fail-fast checks; pool settings inspected and max-open gauge verified |
| Context-aware readiness and expected responses | `database_readiness.go`; missing/closed/canceled unit checks; handler route healthy/error/unavailable/shutdown/timeout checks; real MySQL 2s timeout/recovery |
| Registry/metrics compilation | Build/vet, enabled metrics smoke, shared pool stats |
| Public operational routes and DB-independent health | `routes.go`, focused tests, real DB outage health 200/ready 503 |
| Safe errors and shutdown | Fixed connector messages, unchanged readiness envelope; secret scans, SIGTERM checks; existing `sync.Once` retained |
| Preservation and scope | No package move/API envelope/config policy/schema change; no unrelated feature implementation |
| Examples/docs/input traceability | Files listed above; examples validated by live startup after supplying genuine local values |

These are implementation observations for independent assessment, not acceptance verdicts.

## Limitations and decisions

- The existing JWT/cookie/rate-limit validators are intentionally retained. A real local JWT secret is still required although authentication is inactive. Examples leave the secret blank and document generation; local config was not silently made runnable using a fake/shared secret.
- Driver addition is expressly authorized by this input. It requires SQL driver 1.8.1 and edwards25519; no other dependency upgrades were made.
- ORM automatic ping/version query and SQL logging are disabled. No transaction wrapper, business-query deadline wrapper, slow-query instrumentation, or debug logging was added; retained settings are documented accordingly. Future business SQL must assess version-specific behavior.
- Live verification used MySQL 5.7.24, not a production environment or every MySQL version. Forced HTTP shutdown, SIGINT, and every pool setting were not individually exercised. Shutdown flag behavior is covered by unit tests; repeated close remains protected by existing `sync.Once`.
- No unresolved implementation blocker or unapproved structural/configuration change was needed. Independent review remains outstanding.

## Review handoff

Review the actual source and changes against the acceptance criteria in the approved input above. Record the separate verdict in `develop/reviews/operational/health-readiness/M1-01-review.md`. That review file has **not** been created by implementation. Do not close the milestone until independent review and documentation assessment are complete. If review finds a failure, the next fix input uses M1-02; no speculative fix input was created.
