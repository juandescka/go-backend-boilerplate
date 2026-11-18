# MASTER PROMPT — M3B
## Scalar API Reference Integration

**Project:** Mapatu Backend Boilerplate  
**Repository:** `backend/hris-core-service`  
**Milestone:** M3B — Scalar API Reference Integration  
**Starting Iteration:** M3B-01  
**Feature:** `api/scalar`  
**Mode:** IMPLEMENTATION + AUTONOMOUS REVIEW & FIX  
**Architecture Strategy:** Minimal Preservation  
**Previous Milestone:** M3A — Swagger UI & API Specification (PASS)

---

# 1. ROLE & OBJECTIVE

Bertindak sebagai Senior Go Backend Engineer dan API Platform Engineer yang berpengalaman dalam Go, Gin, OpenAPI, Swaggo, dan Scalar API Reference.

Tugas utama Anda adalah mengintegrasikan **Scalar API Reference** ke dalam backend Mapatu untuk menggantikan Swagger UI sebagai tampilan dokumentasi API.

Pertahankan Swaggo sebagai generator spesifikasi API yang telah digunakan pada M3A.

Tujuan:

1. Menggunakan Scalar sebagai API documentation interface.
2. Menyediakan Scalar melalui endpoint `/docs`.
3. Mempertahankan spesifikasi API existing sebagai single source of truth.
4. Mempertahankan workflow `swag init`.
5. Menghindari penggantian generator yang tidak diperlukan.
6. Memastikan integrasi sederhana, aman, dan mudah dipelihara.
7. Memperbarui dokumentasi yang terdampak.
8. Melakukan review dan perbaikan otomatis sampai PASS.

**Jangan melakukan migrasi penuh dari Swaggo ke generator OpenAPI lainnya.**

---

# 2. REQUIRED READING

Sebelum implementasi, baca:

## Project Instructions

- `AGENTS.md`
- `AGENTS.id.md`
- `develop/README.md` atau nama file workflow aktual
- `develop/FIRST_PROMPT.md`
- `develop/TEMPLATE_PROMPT.md`

## Technical Documentation

- `README.md`
- `README.id.md`
- `documentation/architecture.md`
- `documentation/configuration.md`
- `documentation/operations.md`
- `documentation/cli.md`
- Dokumentasi API existing, jika tersedia

## Source Code

Periksa:

- `main.go`
- `cmd/server.go`
- `internal/routes/`
- `internal/registry/`
- `internal/handlers/`
- `config/`
- `go.mod`
- Swagger annotations.
- Generated Swagger files.
- Swagger UI registration.
- Existing route tests.

Baca juga hasil review M3A yang berstatus PASS apabila tersedia pada `develop/reviews/`.

Jangan membuat asumsi mengenai lokasi generated files atau spesifikasi API sebelum memeriksa repository.

## Official References

Gunakan referensi resmi jika membutuhkan detail integrasi:

- https://scalar.com/products/api-references/getting-started
- https://scalar.com/products/api-references/integrations/html-js
- https://scalar.com/products/api-references/configuration
- https://scalar.com/products/api-references/themes

Hindari mengambil pendekatan dari tutorial lama yang tidak sesuai dengan API Scalar terkini.

---

# 3. CURRENT CONTEXT

Mapatu menggunakan:

- Go.
- Gin.
- Swaggo sebagai Swagger/OpenAPI generator.
- Swagger UI yang telah diintegrasikan pada M3A.
- Operational endpoints yang telah selesai pada M1.

M3A sudah dinyatakan PASS.

Jangan mengulang implementasi M3A.

M3B hanya mengganti presentation layer dokumentasi API dan melakukan penyesuaian kecil yang benar-benar diperlukan.

## Target Architecture

```text
Go Source Code
      |
      v
Swagger Annotations
      |
      v
Swaggo Generator
      |
      v
Swagger JSON / YAML
      |
      v
Scalar API Reference
```

Gunakan generated specification existing.

Jangan memperkenalkan dua spesifikasi API yang harus dikelola secara manual.

---

# 4. PHASE A — EXISTING SWAGGER AUDIT

Identifikasi:

1. Bagaimana Swaggo dijalankan.
2. Lokasi hasil generated specification.
3. Bagaimana generated Go package digunakan.
4. URL aktual Swagger JSON.
5. Bagaimana Swagger UI didaftarkan.
6. Dependency Swagger UI yang digunakan.
7. Apakah route dokumentasi dikendalikan oleh konfigurasi.
8. Apakah spesifikasi dapat diakses tanpa Swagger UI handler.
9. Apakah ada test atau dokumentasi yang bergantung pada route Swagger UI.

Periksa khususnya apakah `/swagger/doc.json` bergantung pada handler `gin-swagger`.

Jika penghapusan Swagger UI membuat endpoint spesifikasi tidak tersedia, lakukan perubahan minimum agar spesifikasi tetap dapat disajikan.

Jangan menghapus dependency sebelum memeriksa consumer-nya.

Laporkan root cause atau dependency yang memengaruhi strategi integrasi.

---

# 5. PHASE B — SCALAR INTEGRATION DESIGN

Pilih integrasi yang paling sederhana dan konsisten dengan backend existing.

Prioritas:

1. Gunakan route Gin existing.
2. Hindari framework frontend tambahan.
3. Hindari npm build pipeline jika tidak diperlukan.
4. Hindari package atau abstraction baru hanya untuk satu halaman dokumentasi.
5. Hindari dependency Go pihak ketiga jika HTML/JavaScript sederhana sudah memadai.

Gunakan HTML/JavaScript integration resmi Scalar jika sesuai.

Contoh konseptual:

```javascript
Scalar.createApiReference('#app', {
  url: '/swagger/doc.json',
  agent: {
    disabled: true
  }
})
```

Contoh tersebut bukan keharusan implementasi final.

Periksa API Scalar terkini dan endpoint spesifikasi aktual sebelum menulis kode.

## Asset Strategy

Evaluasi cara menyediakan aset Scalar:

- CDN dengan versi dependency yang ditentukan secara eksplisit.
- Self-hosted static assets jika diperlukan.

Untuk baseline development, boleh menggunakan CDN yang terpercaya dengan versi terkunci apabila pendekatan tersebut paling sederhana.

Jangan menggunakan dependency `latest` secara implisit untuk deployment yang membutuhkan reproducibility.

Jelaskan jika dokumentasi tidak dapat dirender tanpa koneksi internet karena aset CDN.

Jangan membuat asset bundling pipeline hanya untuk satu halaman tanpa kebutuhan nyata.

Jika self-hosted assets diperlukan, gunakan mekanisme static file serving yang sederhana dan konsisten dengan repository.

## Security & Privacy

- Jangan memerlukan akun Scalar.
- Jangan memerlukan Scalar API key.
- Jangan memperkenalkan Scalar Registry sebagai dependency.
- Jangan menggunakan external request proxy secara otomatis.
- Jangan mengaktifkan Scalar AI Agent tanpa kebutuhan dan persetujuan.
- Jangan mengirim credential pengguna ke layanan eksternal.
- Jangan menambahkan telemetry khusus yang tidak diperlukan.

Pertimbangkan Content Security Policy existing jika ada.

Jangan melemahkan middleware security global hanya agar Scalar dapat dimuat.

---

# 6. PHASE C — ROUTE IMPLEMENTATION

Target utama:

`GET /docs`

Jika sesuai dengan konvensi router, boleh mendukung `/docs/` tanpa memperkenalkan terlalu banyak alias.

Expected behavior:

- Scalar ditampilkan melalui server Gin existing.
- Tidak membutuhkan authentication untuk local development.
- Menggunakan specification URL yang benar.
- Tidak memerlukan server frontend terpisah.
- Tidak memerlukan command Scalar CLI pada runtime backend.

Jangan menambahkan handler layer, usecase, repository, atau service hanya untuk menyajikan HTML dokumentasi.

Jika handler sederhana cukup, gunakan handler sederhana.

## Existing Swagger Routes

Tujuan M3B adalah menggantikan Swagger UI dengan Scalar.

Setelah Scalar berfungsi:

- Pertahankan route spesifikasi JSON yang diperlukan.
- Evaluasi penghapusan route Swagger UI lama.
- Hindari dua UI dokumentasi aktif tanpa kebutuhan.
- Hindari redirect atau alias tambahan yang tidak diperlukan.

Jika route lama mempunyai consumer yang harus dipertahankan, laporkan implikasi kompatibilitasnya sebelum menghapus.

Keputusan mengganti Swagger UI sudah termasuk scope M3B, tetapi perubahan kontrak API bisnis tetap membutuhkan persetujuan.

---

# 7. PHASE D — SWAGGO GENERATION

Pertahankan generator Swaggo existing.

Jangan menggantinya dengan Scalar CLI.

Pastikan workflow generate tetap menggunakan command yang sudah berlaku pada repository.

Contoh umum:

```bash
swag init -g main.go --parseInternal
```

Namun, gunakan lokasi output dan argumen aktual yang telah ditetapkan pada M3A apabila berbeda.

Jangan mengubah generated specification format hanya karena Scalar mendukung OpenAPI 3.x.

Swagger 2.0 dapat tetap digunakan selama seluruh dokumentasi dapat ditampilkan secara benar.

Jangan menambahkan conversion pipeline ke OpenAPI 3.1 tanpa kebutuhan yang terbukti.

## Single Source of Truth

Pastikan:

- Anotasi Go menjadi sumber deklarasi API jika demikian desain existing.
- Generated specification tetap konsisten.
- Scalar membaca specification tersebut.
- Tidak ada salinan manual specification yang berpotensi berbeda.

Jika regeneration diperlukan, lakukan menggunakan command existing dan periksa perubahan generated files.

Jangan mengubah anotasi endpoint bisnis hanya untuk menyelesaikan masalah tampilan.

---

# 8. PHASE E — SCALAR PRESENTATION

Scalar merupakan dokumentasi developer, sehingga tampilannya harus profesional dan mudah digunakan.

Gunakan pendekatan visual minimal.

Preferensi:

- Identitas Mapatu.
- Layout dokumentasi standar Scalar.
- Navigasi endpoint yang jelas.
- Search yang berfungsi.
- Dark/light theme sesuai kemampuan Scalar.
- Typography dan spacing yang nyaman.
- Responsive layout.

Tidak perlu membuat custom design system.

Tidak perlu menggunakan komponen dari Admin Web Mapatu.

Jangan membuat frontend project baru.

Jika branding memerlukan perubahan kecil pada title atau konfigurasi Scalar, lakukan secara minimal.

Hindari custom CSS berlebihan.

Prioritaskan fungsi dan usability dibanding dekorasi.

---

# 9. PHASE F — DOCUMENTATION SYNCHRONIZATION

**Documentation Impact Assessment wajib dilakukan pada setiap iterasi.**

Periksa dan perbarui dokumen berikut apabila terdampak:

## README.md

- Perbarui URL dokumentasi API.
- Pastikan Quick Start sesuai implementasi.
- Perbaiki referensi Swagger UI yang sudah tidak berlaku.

## README.id.md

- Sinkronkan perubahan dengan versi English.
- Pertahankan struktur dan makna yang konsisten.

## documentation/cli.md

- Dokumentasikan command generate Swaggo aktual.
- Jelaskan bahwa Scalar tidak menggantikan `swag init`.
- Jangan menambahkan command Scalar CLI sebagai kewajiban jika tidak digunakan.

## documentation/architecture.md

- Jelaskan hubungan Swaggo, generated specification, Gin, dan Scalar.
- Perbarui struktur atau dependency flow hanya jika berubah.

## documentation/configuration.md

- Dokumentasikan konfigurasi enable/disable API docs jika benar-benar ada.
- Jelaskan batasan akses development/production.
- Jangan menciptakan environment variable baru tanpa kebutuhan.

## documentation/operations.md

- Dokumentasikan endpoint dokumentasi jika relevan.
- Jangan mengubah penjelasan Health dan Readiness yang sudah benar.

## documentation/api/

Jika sudah ada dokumentasi API khusus, perbarui dokumen yang relevan.

Jika belum ada, pertimbangkan satu dokumen ringkas hanya apabila informasi tidak tepat ditempatkan dalam dokumen existing.

Jangan membuat banyak dokumen baru.

### Mandatory Documentation Rules

1. Setiap perubahan API, configuration, CLI, atau architecture harus dievaluasi dampak dokumentasinya.
2. Perbarui dokumen yang terdampak dalam iterasi pekerjaan yang sama.
3. Jangan membuat dokumentasi spekulatif.
4. Jangan menyatakan endpoint aktif sebelum terverifikasi.
5. Jangan menghapus riwayat keputusan tanpa alasan.
6. Jika tidak ada perubahan dokumentasi yang diperlukan, jelaskan alasannya.
7. Dokumentasi yang tidak sinkron merupakan temuan review.

Aturan ini berlaku untuk IMPLEMENTATION, REVIEW, dan FIX.

---

# 10. PHASE G — VERIFICATION

Gunakan risk-based verification.

Periksa jika relevan:

```bash
go build ./...
go vet ./...
go test ./...
git diff --check
```

Jangan membuat test untuk setiap fungsi sederhana.

## HTTP Verification

Verifikasi:

1. `GET /docs` menampilkan halaman Scalar.
2. Halaman memuat JavaScript dan CSS yang diperlukan.
3. Scalar berhasil membaca spesifikasi API.
4. Endpoint dan schema muncul.
5. Tidak ada error fatal di browser console.
6. Specification JSON tetap valid dan dapat diakses.
7. `/health` tetap berfungsi.
8. `/ready` tetap berfungsi sesuai kondisi dependency.

Jika lingkungan mendukung browser verification, periksa tampilan Scalar secara aktual.

**HTTP 200 untuk halaman HTML saja belum cukup untuk menyatakan PASS.**

Jika UI tidak berhasil memuat specification, hasil review harus FAIL.

## Security Verification

Pastikan:

- API docs tidak terbuka secara tidak sengaja di production.
- Tidak ada secret tertanam dalam HTML.
- Tidak ada external proxy otomatis untuk request API.
- Security middleware tidak dilemahkan.
- CORS tidak diperluas tanpa kebutuhan.

## Limitations

Jika runtime browser atau dependency tertentu tidak tersedia, laporkan keterbatasannya.

Jangan mengarang bukti pengujian.

Jangan melakukan operasi database destruktif.

---

# 11. AUTONOMOUS IMPLEMENTATION & REVIEW LOOP

Gunakan workflow yang sudah disepakati dalam `develop/README.md`.

## Starting Iteration

Buat:

`develop/inputs/api/scalar/M3B-01-implementation.md`

Catat instruksi, perubahan, dan bukti eksekusi sesuai konvensi development workflow.

Setelah implementasi, lakukan review.

Simpan hasilnya pada:

`develop/reviews/api/scalar/M3B-01-review.md`

## Review Process

Reviewer harus memeriksa:

- Source code aktual.
- Kesesuaian dengan M3B scope.
- Dependency yang ditambahkan/dihapus.
- Route registration.
- Specification availability.
- Scalar rendering.
- Documentation changes.
- Verification evidence.

Gunakan reviewer subagent dengan konteks terpisah jika tersedia.

Jika tidak tersedia, lakukan structured self-review dan nyatakan statusnya secara jelas.

Jangan menyebut self-review sebagai independent review.

## Automatic Fix

Jika review FAIL:

1. Catat temuan dengan evidence.
2. Buat input FIX baru menggunakan ID berikutnya.
3. Perbaiki root cause dalam scope.
4. Perbarui dokumentasi terkait.
5. Jalankan verification.
6. Review ulang.

Contoh:

```text
develop/
├── inputs/
│   └── api/scalar/
│       ├── M3B-01-implementation.md
│       └── M3B-02-fix-[issue].md
│
└── reviews/
    └── api/scalar/
        ├── M3B-01-review.md
        └── M3B-02-review.md
```

Jika masih FAIL, lanjutkan ke M3B-03 dan seterusnya.

Jangan meminta pengguna mengirim prompt tambahan untuk setiap masalah rutin.

Jangan menimpa riwayat input atau review.

Hentikan jika:

- PASS.
- Ada blocker nyata.
- Memerlukan keputusan arsitektur di luar persetujuan.
- Tidak ada kemajuan setelah beberapa iterasi.
- Lima iterasi perbaikan telah dilakukan tanpa PASS.

Jangan memaksakan PASS.

---

# 12. OUT OF SCOPE

Jangan melakukan:

- Penggantian Swaggo generator.
- Migrasi seluruh API ke OpenAPI 3.1.
- Pengembangan modul bisnis.
- Perubahan Authentication/RBAC.
- Perubahan database schema.
- Refactoring seluruh routes.
- Penggantian framework Gin.
- Penambahan frontend framework.
- Pembuatan API documentation platform baru.
- Pengintegrasian Scalar Registry.
- Pengembangan API management system.
- Implementasi fitur Scalar AI Agent.
- Refactoring komponen M1 yang sudah PASS.

Hanya lakukan perubahan yang berkaitan langsung dengan Scalar integration.

---

# 13. ACCEPTANCE CRITERIA

M3B dapat dinyatakan PASS apabila:

- [ ] Integrasi Scalar tersedia pada backend Gin.
- [ ] `/docs` dapat diakses pada development environment.
- [ ] Scalar UI berhasil dirender.
- [ ] Specification API berhasil dimuat.
- [ ] Metadata Mapatu tampil dengan benar.
- [ ] Swaggo tetap menjadi generator.
- [ ] Command generation existing tetap berfungsi.
- [ ] Specification endpoint tetap tersedia.
- [ ] Swagger UI lama ditangani sesuai keputusan kompatibilitas.
- [ ] Tidak ada dependency berlebihan.
- [ ] Tidak ada perubahan package structure yang tidak disetujui.
- [ ] Health dan Readiness tetap berfungsi.
- [ ] Tidak ada pelemahan security middleware.
- [ ] Dokumentasi yang terdampak telah diperbarui.
- [ ] Verification evidence tersedia.
- [ ] Review terakhir menghasilkan PASS.

---

# 14. REQUIRED FINAL OUTPUT

Setelah PASS atau BLOCKED, tampilkan:

### A. Final Status

PASS / BLOCKED.

### B. Implementation Summary

Ringkas perubahan yang benar-benar dilakukan.

### C. Files Changed

| File | Action | Reason |
|---|---|---|
| ... | ... | ... |

### D. Runtime Result

Tampilkan URL aktual:

- Scalar UI.
- API specification.
- Health.
- Readiness.

### E. Verification

| Check | Status | Evidence |
|---|---|---|
| Build | PASS/FAIL/NOT RUN | ... |
| Tests | PASS/FAIL/NOT RUN | ... |
| API specification | PASS/FAIL/NOT RUN | ... |
| Scalar rendering | PASS/FAIL/NOT RUN | ... |
| Security | PASS/FAIL/NOT RUN | ... |

### F. Documentation Changes

Sebutkan seluruh dokumen yang diperbarui dan alasannya.

### G. Review Iterations

- Jumlah iterasi.
- Review terakhir.
- Temuan yang diperbaiki.
- Remaining limitations.

### H. Git Conventional Commit

Berikan satu rekomendasi commit message.

Jangan melakukan commit otomatis.

---

# FINAL INSTRUCTION

Integrate Scalar API Reference into the existing Mapatu Go/Gin backend.

Preserve the working Swaggo generation pipeline from M3A.

Replace Swagger UI with Scalar while keeping the API specification available.

Prefer minimal changes, avoid unnecessary dependencies, and maintain a simple developer experience.

Autonomously implement, review, fix, and re-review until PASS or a genuine blocker.

Update relevant technical documentation in every iteration.

Do not modify unrelated business logic or completed milestones.

Do not automatically commit changes.

---

## Approved execution and evidence — 12 November 2026

The active user request authorizes Scalar replacement, minimal routing/spec/dependency changes, relevant documentation synchronization, and autonomous review/fix. Saved before edits; no previous input/review was overwritten. Baseline was clean commit `cf5303e` (M3A Swagger integration); M1/M3A historical PASS records remain unchanged.

### Existing Swagger audit and integration decision

- `main.go` already has one pinned `go:generate` directive: `go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs --parseInternal`.
- Existing generated `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` describe Mapatu probes in Swagger2.0. The generated Go import registers a single runtime spec.
- M3A `SetupSwaggerRoutes` used `gin-swagger` with `swaggo/files` at `/swagger/*any`; `/swagger/doc.json` depended on that UI adapter, not an independent JSON handler.
- Runtime consumers of the adapters were only route setup. Repo tests/docs referenced the old UI; no additional in-repo UI consumer was found. Replacing the UI is expressly authorized; retain JSON compatibility and remove old HTML/assets without custom redirects/second UI. External bookmarks to old UI must use `/docs`; no claim is made that all external consumers are known.
- Preserve development-only existing App.Env gate and global middleware. Existing security headers have no CSP to relax.
- Keep the current file/package layout: evolve `internal/routes/swagger.go` to serve Scalar plus the Swagger JSON URL rather than adding a new layer/package. Use `SetupAPIDocsRoutes` for the actual combined responsibility; update its only production consumer and focused tests.

### Asset and privacy choices

Used official Scalar HTML/JS integration with ESM `@scalar/api-reference@1.69.2/esm.js` from jsDelivr. Version availability verified from the public CDN package metadata before implementation. No npm/build pipeline, frontend server, Go Scalar library, runtime CLI, Registry, account/API key or format conversion.

The HTML uses same-origin `/swagger/doc.json`; external proxy is explicitly empty, `agent.disabled=true`, telemetry false, `persistAuth=false`, default remote fonts false. System font CSS is minimal. No credentials are embedded and no analytics plugin is loaded. JavaScript and internet/CDN module access are required for rendering; HTML200 alone is insufficient. Optional Scalar promotion controls were not used and do not constitute an account/Registry dependency.

Official references consulted:
- https://scalar.com/products/api-references/getting-started
- https://scalar.com/products/api-references/integrations/html-js
- https://scalar.com/products/api-references/configuration
- https://scalar.com/products/api-references/themes

### Actual changes

- Existing route file now serves `/docs` HTML and `swag.ReadDoc()` JSON at retained `/swagger/doc.json`; non-development registers neither. Old Swagger UI paths return existing404 fallback.
- Removed only unused UI dependencies `github.com/swaggo/gin-swagger` and `github.com/swaggo/files`, with `go mod tidy` removing their obsolete sums; retained `swaggo/swag` and all existing dependency versions.
- Reserved `/docs` and its namespace in the existing metrics-path validator to prevent a docs/metrics route collision. This is a directly related startup guard; no new configuration key, precedence or global policy was introduced. Added focused collision validation checks.
- Updated existing route tests to cover Scalar HTML/privacy config, old UI absence, unchanged spec/envelope references and development gate. Existing M1 tests retained.
- Updated both READMEs plus architecture, configuration, operations and CLI. Existing references suffice; no new technical documentation file/folder. Generation/examples/local config/workflow unchanged.

### Verification performed

- `GOCACHE=/private/tmp/hris-m1-01-go-cache go build ./...`, `go vet ./...`, `go test ./...`: PASS, exit0. Config test0.465s, route tests3.002s; other unchanged checks reported pass/no tests as appropriate.
- Reviewer `/root/review_scalar`, separate context: `go test ./internal/routes ./config -count=1` PASS (2.514s/0.297s) and `git diff --check` PASS. No actionable source/docs finding.
- `go generate .` using the existing directive and cached-file module proxy: PASS; all three generated-file SHA-256 values stayed identical. No generated-file/main.go diff or second spec/pipeline.
- `git diff --check`: PASS. Local `.env`/`config.json` hashes unchanged. Diff confirms handlers, registry, infrastructure, lifecycle/health usecase, global middleware and shared code unchanged.
- Fresh real MySQL5.7.24 test instance at33347 and app33348: `python3 -u /private/tmp/hris-m3b-scalar.UVBb45/smoke.py`, exit0. Scalar HTML/privacy config and spec200; old Swagger UI404; production docs/spec404 with unchanged health/ready200; `/docs` metrics collision rejected during configuration validation.
- Browser at `/docs` loaded pinned CDN ESM and the existing Swagger2.0 JSON, rendered Mapatu, both operational endpoints and response models, and showed no warning/error messages. Search for readiness returned the correct operation; dark/light toggle changed state. Scalar Test Request called the local `/ready` and returned real200 JSON; backend completion log recorded the browser request directly from loopback with route `/ready`.
- Paused only the isolated DB: readiness503 in2.01s, health/docs/JSON200; readiness recovered after resume. Stopped only isolated DB: health200, readiness503, JSON200. SIGTERM cleanup succeeded and temporary processes stopped. No migration/seed/production operation.
- Screenshot of isolated UI: `/private/tmp/hris-m3b-scalar.UVBb45/scalar-test.png`.
- Local development process on8099 was verified by PID/cwd. Preflight confirmed config/DB initialization before only failing on occupied port. Gracefully replaced the verified old process with `GOCACHE=/private/tmp/hris-m1-01-go-cache APP_ENV=development go run main.go server`.
- On actual localhost8099: `/docs`, `/swagger/doc.json`, `/health`, `/ready` all200; `/swagger/index.html`404 `ROUTE_NOT_FOUND`. Browser rendered Mapatu/models using the correct8099 server origin with no console warning/error. Screenshot `/private/tmp/hris-m3b-scalar.UVBb45/scalar-local-8099.png`. The user's development server remains running intentionally.

### Result

One iteration; no verified finding required a FIX input. Final actual verdict is recorded in [M3B-01 review](../../../reviews/api/scalar/M3B-01-review.md). No Git commit was performed.
