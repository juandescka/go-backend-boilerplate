# MASTER PROMPT — M3A-01
## Mapatu Backend — Swagger UI & API Specification

**Project:** Mapatu Backend Boilerplate  
**Mode:** IMPLEMENTATION + AUTONOMOUS REVIEW & FIX  
**Feature:** `api/swagger`  
**Milestone:** M3A-01  
**Architecture:** Option A — Minimal Preservation

### 1. OBJECTIVE

Analisis dan perbaiki penyebab Swagger UI tidak dapat diakses melalui:

`http://localhost:8099/swagger/index.html`

Current response:

- HTTP 404
- Error code: `ROUTE_NOT_FOUND`

Targetnya adalah membuat Swagger UI dan API specification dapat diakses secara benar pada development environment.

Jangan mengubah implementasi Health & Readiness M1 yang sudah PASS kecuali ditemukan masalah langsung terkait integrasi Swagger.

### 2. REQUIRED READING

Baca:

- `AGENTS.md`
- `develop/README.md` atau nama workflow aktual.
- `develop/FIRST_PROMPT.md`
- `develop/TEMPLATE_PROMPT.md`
- `documentation/architecture.md`
- `documentation/operations.md`
- `documentation/configuration.md`
- `documentation/cli.md`
- `README.md`
- `README.id.md`

Periksa:

- `main.go`
- `cmd/server.go`
- `internal/routes/`
- `internal/handlers/`
- `go.mod`
- Existing Swagger annotations.
- Existing Swagger specification, jika ada.
- Existing generated files.
- Konfigurasi Swagger dan environment, jika ada.

Gunakan implementasi LMS/SPMB hanya sebagai referensi jika memang relevan.

Jangan menyalin struktur mereka secara keseluruhan.

### 3. ROOT CAUSE ANALYSIS

Sebelum mengubah kode, identifikasi:

1. Apakah route Swagger sudah didaftarkan?
2. Apakah dependency Swagger UI tersedia?
3. Apakah API specification sudah dihasilkan?
4. Apakah generated specification sudah terhubung ke runtime?
5. Apakah path Swagger sesuai konfigurasi Gin?
6. Apakah middleware menghalangi akses?
7. Apakah ada konfigurasi environment yang menonaktifkan Swagger?
8. Apakah Swagger annotations mengacu pada endpoint yang benar-benar tersedia?

Jangan langsung menginstal dependency atau membuat package baru sebelum memastikan penyebabnya.

### 4. MINIMAL IMPLEMENTATION

Perbaiki akar masalah menggunakan struktur existing.

Jika menggunakan Swaggo, ikuti mekanisme yang sesuai dengan dependency dan versi aktual repository.

Persyaratan:

- Swagger UI dapat diakses melalui `/swagger/index.html`.
- Spesifikasi API dapat diakses oleh Swagger UI.
- Tidak memerlukan authentication untuk akses lokal sesuai konfigurasi development.
- Tidak mengubah kontrak `/health` dan `/ready`.
- Tidak mengubah middleware global secara sembarangan.
- Tidak memperkenalkan framework dokumentasi baru.
- Tidak membuat duplikasi Swagger generator.
- Tidak menambahkan endpoint bisnis yang belum tersedia.
- Tidak mengubah API response envelope existing.

Prioritaskan penggunaan folder dokumentasi yang sudah ada.

Jika generated Go package memerlukan lokasi tersendiri, gunakan penempatan paling sederhana dan konsisten. Jelaskan alasannya dalam review.

Jangan melakukan restrukturisasi package existing.

Pastikan Swagger tidak terekspos secara tidak sengaja pada production environment. Gunakan mekanisme konfigurasi existing jika tersedia. Jika memerlukan perubahan konfigurasi yang breaking, eskalasikan sebelum implementasi.

### 5. API SPECIFICATION ACCURACY

Pastikan spesifikasi sesuai dengan API aktual.

Khususnya:

- `GET /health`
- `GET /ready`
- HTTP 200.
- HTTP 503 untuk readiness gagal.
- Response envelope existing.

Jangan mencantumkan endpoint LMS/SPMB yang belum tersedia pada Mapatu.

Jangan mengklaim endpoint bekerja hanya karena ada anotasi Swagger.

Perbaiki metadata API yang jelas-jelas tidak sesuai identitas Mapatu, tanpa memperluas scope ke dokumentasi seluruh modul.

### 6. DOCUMENTATION REQUIREMENTS

Evaluasi dan perbarui dokumentasi yang terdampak:

- `README.md`
- `README.id.md`
- `documentation/architecture.md`
- `documentation/operations.md`
- `documentation/configuration.md`
- `documentation/cli.md`

Jika diperlukan, tambahkan dokumentasi Swagger/API secara minimal pada `documentation/api/`.

Dokumentasikan:

- URL Swagger.
- Cara mengaktifkan Swagger jika menggunakan konfigurasi.
- Cara menghasilkan ulang specification.
- Command generator dan versi/tool yang diperlukan.
- Keterbatasan development/production.
- Perilaku aktual setelah implementasi.

Jangan membuat file dokumentasi baru jika dokumen existing sudah cukup.

**Aturan sinkronisasi dokumentasi berlaku pada setiap iterasi IMPLEMENTATION, REVIEW, dan FIX.**

### 7. VERIFICATION

Lakukan pemeriksaan yang relevan:

- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `git diff --check`

Verifikasi HTTP:

- `GET /swagger/index.html` menghasilkan Swagger UI.
- `GET /swagger/doc.json` menghasilkan specification valid apabila menggunakan path Swaggo default.
- Swagger UI berhasil memuat spesifikasi.
- `GET /health` tetap berfungsi.
- `GET /ready` tetap berfungsi sesuai kondisi dependency.

Jangan berhenti pada HTTP 200 jika Swagger UI menampilkan kesalahan pemuatan specification.

Lakukan verifikasi runtime jika lingkungan aman dan tersedia.

Jangan mengubah production database atau menjalankan migration.

### 8. AUTONOMOUS DEVELOPMENT LOOP

Gunakan workflow:

`IMPLEMENT → REVIEW → FAIL → FIX → REVIEW → ... → PASS`

Simpan input awal di:

`develop/inputs/api/swagger/M3A-01-implementation.md`

Simpan review awal di:

`develop/reviews/api/swagger/M3A-01-review.md`

Jika review menemukan masalah:

1. Catat temuan aktual.
2. Buat input perbaikan dengan ID M3A-02.
3. Implementasikan perbaikan.
4. Perbarui dokumentasi terdampak.
5. Jalankan pemeriksaan relevan.
6. Review ulang dan simpan hasilnya.
7. Lanjutkan iterasi bila masih FAIL.

Gunakan reviewer subagent berkonteks terpisah jika tersedia.

Jangan meminta pengguna menangani bug rutin yang masih berada dalam scope.

Eskalasi hanya untuk breaking changes, perubahan struktur besar, operasi destruktif, atau blocker yang benar-benar membutuhkan keputusan pengguna.

Jika tidak ada kemajuan setelah beberapa percobaan atau lima iterasi perbaikan, laporkan BLOCKED beserta buktinya.

### 9. ACCEPTANCE CRITERIA

- [ ] Root cause Swagger 404 ditemukan.
- [ ] Swagger route berfungsi.
- [ ] Swagger UI berhasil dimuat.
- [ ] API specification valid dan berhasil dimuat.
- [ ] Metadata API sesuai Mapatu.
- [ ] Operational endpoints tetap berfungsi.
- [ ] Tidak ada perubahan arsitektur berlebihan.
- [ ] Dokumentasi relevan disinkronkan.
- [ ] Verification evidence tersedia.
- [ ] Review terakhir menghasilkan PASS.

### 10. FINAL REPORT

Setelah PASS atau BLOCKED, tampilkan:

1. Root cause.
2. Implementation summary.
3. Files changed.
4. Verification results.
5. Documentation changes.
6. Jumlah iterasi.
7. Final verdict.
8. Recommended Git Conventional Commit.

Jangan membuat Git commit otomatis.

### FINAL INSTRUCTION

Restore Swagger UI and API specification access with the smallest justified changes.

Preserve the existing Mapatu architecture and completed M1 behavior.

Autonomously review, fix, and re-review until PASS or a genuine blocker.

Keep documentation synchronized throughout the entire workflow.

---

## Approved execution decisions and actual evidence — 12 November 2026

The active user request approves implementation and autonomous separate-context review/fix for this scope. This input was saved before edits. Source baseline was clean at commit `625005a` (`feat(operations): wire health and MySQL readiness probes`). M1's PASS input/reviews are preserved.

### Root cause verified before changes

1. `internal/routes/routes.go` registered only `/health` and `/ready`; `cmd/server.go` had no Swagger setup call. Gin therefore used the existing `ROUTE_NOT_FOUND` fallback for `/swagger/index.html`.
2. `go.mod` had no `swaggo/swag`, `gin-swagger`, or UI asset dependency. Baseline `go build ./...` failed at `docs/docs.go:4` because `github.com/swaggo/swag` was missing.
3. `docs/docs.go`, `swagger.json`, and `swagger.yaml` already existed and were committed; generation was not wholly absent. They described the two operational routes but retained LMS identity, unused Auth/Users tags and BearerAuth metadata.
4. No runtime import registered the generated package; generated artifacts alone do not publish HTTP routes.
5. There was no alternative Swagger route/prefix or existing Swagger configuration/gate. Global middleware was not the cause of the no-Origin 404.
6. Local app settings were development/port 8099. Only these nonsecret settings were inspected; local files were not overwritten.

### Minimal implementation

- Preserve existing `docs/` location; generated Go package stays there because Swaggo registers the specification through its `init`, and it already existed. No new layer, package relocation or second generator.
- Add `internal/routes/swagger.go`: import generated package and serve `/swagger/*any` through standard Swaggo adapters only when existing `App.Env` is development (trimmed/case-insensitive). All other environment values register no UI/spec/assets. `GIN_MODE` is not the exposure gate.
- Wire setup into `cmd/server.go` under existing global middleware. Preserve M1 handlers, readiness/checker, pool, lifecycle, registry, response envelope and configuration precedence.
- Correct main metadata to Mapatu, Operations tag and root base path; remove inactive auth metadata. Use a single pinned `go:generate` directive for Swaggo v1.16.6.
- Runtime dependencies: gin-swagger v1.6.1, files v1.0.1, swag v1.16.6. No existing dependency versions were upgraded. The existing Swaggo transitive `go-openapi/spec` package is also used by focused spec tests; no new test framework.
- Regenerate all three existing artifacts; leave host unset so UI uses the current origin/port. Specs describe only GET health/ready with existing response envelopes and readiness 503.
- Update both READMEs and architecture/operations/configuration/CLI reference. Examples, local/production config and actual development workflow remain unchanged.

### Execution and verification

- Added dependencies from cached modules. The first generation with `GOPROXY=off` could not perform Go's deprecation lookup; reran successfully with the local file module proxy. `go mod tidy` was rerun with a writable temporary build cache after a sandbox cache error. No source fix was needed for those environment constraints.
- A focused test draft initially had a Go map-value pointer-method compile error; it was corrected before reviewer assessment. Final focused/full tests pass; no historical review FAIL was overwritten.
- `GOCACHE=/private/tmp/hris-m1-01-go-cache go build ./...`, `go vet ./...`, `go test ./...`: all exit 0. `git diff --check`: exit 0.
- Reviewer `/root/review_swagger` separately read current source/docs/generated files and ran `go test ./internal/routes -run Swagger -count=1`, PASS (0.814s). No actionable finding.
- `go generate .` succeeded using `GOCACHE=/private/tmp/hris-m1-01-go-cache`, `GOPROXY=file:///Users/kapantow.juan/go/pkg/mod/cache/download`, `GOSUMDB=off`. Repeat generation left all three generated-file SHA-256 hashes unchanged. The globally installed swag v1.16.4 was not used; the directive runs v1.16.6.
- Fresh real isolated MySQL 5.7.24 and app on ports 33337/33338: `python3 -u /private/tmp/hris-m3a-swagger.grywSe/smoke.py`, exit 0. Development UI/spec/assets were 200; production UI/spec/assets returned existing 404 code while M1 probes stayed 200. Runtime JSON paths/definitions matched generated JSON. CORS/global headers still applied.
- Browser loaded UI/spec successfully, displayed Mapatu and both operations without load/console errors. Executing both operations through Swagger returned real 200 envelopes. Screenshot: `/private/tmp/hris-m3a-swagger.grywSe/swagger-ui.png` (isolated test).
- Paused only isolated MySQL: readiness 503 in 2.01s, health and Swagger remained 200; readiness recovered after resume. After actual DB shutdown, health stayed 200, readiness 503 and Swagger JSON 200. SIGTERM exited 0; no production DB/migration/seed was involved. Temporary app/DB stopped.
- The requested local URL was initially served by the old process, verified via listener PID/cwd. Preflight with the current binary/local development config reached DB initialization and only failed on occupied 8099. Stopped that exact old development process gracefully, then started `GOCACHE=/private/tmp/hris-m1-01-go-cache APP_ENV=development go run main.go server` from repository root. New development server is intentionally left running on 8099.
- On the requested URL: `/swagger/index.html`, `/swagger/doc.json`, `/health`, `/ready` all returned 200. Browser loaded the spec at `http://localhost:8099/swagger/index.html` with no warning/error messages. Screenshot: `/private/tmp/hris-m3a-swagger.grywSe/swagger-local-8099.png`.
- Local `.env` and `config.json` hashes are unchanged. `git diff --quiet HEAD` confirms operational handlers, health usecase/lifecycle, infrastructure, registry, configuration loader, middleware and shared helpers are untouched.

### Handoff

One implementation/review iteration was sufficient; no speculative M3A-02 input is needed. Final actual assessment is recorded separately in [M3A-01 review](../../../reviews/api/swagger/M3A-01-review.md). No Git commit was made.
