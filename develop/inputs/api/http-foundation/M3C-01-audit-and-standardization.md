# MASTER PROMPT — M3C-01
## Mapatu Backend — API Standards & HTTP Foundation

**Project:** Mapatu Backend Boilerplate  
**Repository:** `backend/hris-core-service`  
**Milestone:** M3C — API Standards & HTTP Foundation  
**Starting Iteration:** M3C-01  
**Feature:** `api/http-foundation`  
**Mode:** AUDIT FIRST + TARGETED IMPLEMENTATION + AUTONOMOUS REVIEW & FIX  
**Architecture Strategy:** Pragmatic Clean Architecture — Minimal Preservation

---

# 1. ROLE & OBJECTIVE

Bertindak sebagai Senior Go Backend Engineer dan Software Architect dengan pengalaman pada Go, Gin, REST API, HTTP middleware, request validation, dan reusable backend architecture.

Tugas Anda adalah mengevaluasi dan menetapkan standar HTTP API untuk Mapatu Backend Boilerplate berdasarkan implementasi existing.

**Tujuan utama bukan membuat HTTP framework baru, melainkan memastikan komponen yang sudah tersedia konsisten, sederhana, dan siap digunakan kembali.**

Prioritas:

1. Audit implementasi HTTP foundation existing.
2. Identifikasi standar yang sudah diterapkan.
3. Evaluasi masalah nyata dan inkonsistensi.
4. Tetapkan API conventions berdasarkan implementasi yang valid.
5. Lakukan perbaikan minimal hanya jika diperlukan.
6. Dokumentasikan standar API yang disepakati.
7. Jalankan autonomous review dan fix sampai PASS atau BLOCKED.

Jika implementasi existing sudah memadai, jangan melakukan refactoring hanya untuk menghasilkan perubahan kode.

**Documentation-only outcome is acceptable.**

---

# 2. PROJECT CONTEXT

Mapatu menggunakan pendekatan:

- Go dan Gin.
- Pragmatic Clean Architecture.
- Minimal Preservation.
- Developer-friendly implementation.
- Explicit dependencies.
- Minimal abstraction.
- Risk-based testing.
- Feature-based development workflow.

Milestone sebelumnya:

- M0 — Architecture Audit & Planning.
- M0D — Documentation & AI Workflow.
- M1 — Operational Health & Readiness (PASS).
- M3A — Swagger UI & API Specification (PASS).
- M3B — Scalar API Reference Integration (dilaporkan berfungsi; verifikasi status review dari repository).

M3C tidak perlu mengulang milestone tersebut.

Endpoint `/health` dan `/ready` telah digunakan untuk memverifikasi response HTTP.

Jangan mengubah kontraknya kecuali ditemukan bug yang terbukti dan perubahan tersebut disetujui.

---

# 3. REQUIRED READING

## Project Instructions

Baca:

- `AGENTS.md`
- `AGENTS.id.md`
- `develop/FIRST_PROMPT.md`
- `develop/TEMPLATE_PROMPT.md`
- `develop/README.md`

Jika workflow aktual menggunakan `develop/README-develop.md`, periksa file tersebut dan gunakan sumber aturan yang benar.

Jangan membuat duplikasi workflow documentation.

## Technical Documentation

Baca:

- `README.md`
- `README.id.md`
- `documentation/architecture.md`
- `documentation/configuration.md`
- `documentation/operations.md`
- `documentation/cli.md`
- Dokumentasi API existing jika tersedia.

## Source Code

Audit package yang relevan:

```text
internal/
├── routes/
├── handlers/
│   ├── dtos/
│   └── mappers/
├── middlewares/
├── usecases/
├── domain/
├── infrastructure/
└── shared/
    ├── response/
    ├── errors/
    ├── httphelper/
    ├── validation/
    ├── labels/
    └── observability/
```

Periksa juga:

- `cmd/server.go`
- `internal/registry/registry.go`
- Existing HTTP-related tests.
- Swagger/Scalar specifications jika diperlukan untuk mengevaluasi kontrak HTTP aktual.

Jangan menganggap package kosong sebagai implementasi aktif.

Jangan membuat package baru hanya agar struktur terlihat lengkap.

---

# 4. PHASE A — HTTP FOUNDATION DISCOVERY

Mulai dengan pemeriksaan read-only.

Identifikasi:

1. Struktur HTTP foundation aktual.
2. Tanggung jawab setiap package.
3. Dependency dan consumer.
4. Komponen yang sudah digunakan.
5. Komponen yang belum memiliki consumer.
6. Konvensi response dan error.
7. Alur request dari route sampai response.
8. Middleware yang terdaftar.
9. Konvensi DTO dan validation.
10. Potensi duplikasi atau coupling yang berdampak nyata.

Gunakan klasifikasi:

- **KEEP:** Sudah baik, tidak perlu perubahan.
- **DOCUMENT:** Sudah baik, tetapi standarnya belum terdokumentasi.
- **FIX:** Ada bug atau inkonsistensi nyata yang perlu diperbaiki.
- **SIMPLIFY:** Kompleksitas yang terbukti tidak diperlukan.
- **DEFER:** Belum diperlukan sekarang.
- **NEEDS DECISION:** Membutuhkan persetujuan pengguna.

Setiap temuan harus didukung referensi file, fungsi, atau perilaku aktual.

Jangan menjadikan preferensi desain sebagai bukti adanya masalah.

---

# 5. PHASE B — API RESPONSE STANDARD

Periksa:

`internal/shared/response/`

Evaluasi:

- Success response.
- Error response.
- Response envelope.
- Pagination metadata.
- HTTP status handling.
- JSON serialization.
- Empty data dan null behavior.
- Konsistensi response helper.
- Hubungan response helper dengan handler DTO.

Gunakan kontrak existing sebagai baseline.

Jangan memaksakan format response baru jika implementasi sekarang sudah konsisten.

Pastikan standar menjelaskan:

1. Kapan menggunakan success response.
2. Kapan menggunakan error response.
3. Bagaimana menentukan HTTP status.
4. Bagaimana menggunakan message dan error code.
5. Bagaimana mengembalikan data kosong.
6. Bagaimana menangani response tanpa data.

Jika `shared/response` masih bergantung pada DTO di package handlers, evaluasi dampaknya.

Jangan langsung memindahkan DTO.

Jika coupling tersebut belum menimbulkan masalah nyata, catat sebagai kandidat M2.

---

# 6. PHASE C — ERROR HANDLING

Audit implementasi error handling existing.

Periksa:

- Error code.
- HTTP status mapping.
- Error response format.
- Internal error handling.
- Validation errors.
- Route-not-found behavior.
- Panic recovery.
- Error logging.

Tujuannya memastikan error HTTP dapat dipahami client tanpa membocorkan informasi internal.

Evaluasi konvensi untuk status seperti:

- 400 Bad Request.
- 401 Unauthorized.
- 403 Forbidden.
- 404 Not Found.
- 409 Conflict.
- 422 Unprocessable Content, apabila memang digunakan.
- 429 Too Many Requests.
- 500 Internal Server Error.
- 503 Service Unavailable.

Jangan membuat seluruh error code modul HRIS sekarang.

Jangan membangun generic error framework baru.

Jangan mengubah global error envelope tanpa analisis kompatibilitas dan persetujuan.

Jika error handling sekarang sudah cukup baik, dokumentasikan saja.

---

# 7. PHASE D — REQUEST BINDING & VALIDATION

Periksa:

- `internal/shared/httphelper/`
- `internal/shared/validation/`
- `internal/shared/labels/`
- DTO request yang tersedia.

Evaluasi:

1. JSON request binding.
2. Query parameter binding.
3. Path parameter handling.
4. Required fields.
5. Type validation.
6. Custom validation jika tersedia.
7. Field error formatting.
8. Error response consistency.

Perhatikan apakah komponen validation masih membawa istilah atau aturan khusus LMS/SPMB.

Jangan menghapusnya secara otomatis.

Jika belum digunakan tetapi tidak menghambat HTTP foundation, klasifikasikan sebagai DEFER atau kandidat M2.

Jangan membuat validator baru jika library dan helper existing memadai.

Jangan membuat generic DTO framework.

Jika diperlukan perbaikan kecil yang aman dan berada dalam scope, implementasikan langsung.

---

# 8. PHASE E — PAGINATION STANDARD

Periksa implementasi pagination existing.

Evaluasi:

- Default page.
- Default limit.
- Maximum limit.
- Invalid parameter behavior.
- Total records.
- Total pages.
- Response metadata.
- Empty result behavior.
- Konsistensi tipe numerik.
- Potensi integer overflow.

Jangan membuat pagination framework baru.

Jangan mengimplementasikan endpoint Employee atau modul bisnis hanya untuk menguji pagination.

Jika helper sudah tersedia dan perilakunya memadai, pertahankan.

Jika standar pagination belum lengkap, dokumentasikan keputusan yang berdasarkan implementasi atau tetapkan proposal untuk hal yang belum diputuskan.

Perubahan default, batas maksimum, atau response contract yang berpotensi breaking harus mendapatkan approval.

---

# 9. PHASE F — ROUTING CONVENTIONS

Periksa package:

`internal/routes/`

Tetapkan konvensi minimal untuk:

- API versioning.
- HTTP methods.
- Resource naming.
- Route grouping.
- Handler registration.
- Middleware assignment.
- Operational endpoints.
- API documentation routes.
- Unknown route handling.

Sebagai prinsip:

- Operational endpoints tidak harus menggunakan `/v1`.
- API documentation endpoints dapat berada di luar versioned business API.
- Business APIs nantinya dapat menggunakan `/v1`.

Gunakan route existing sebagai baseline.

Jangan membuat business routes baru.

Jangan membuat route registry atau generic routing abstraction.

Jangan memindahkan seluruh route ke file baru tanpa kebutuhan nyata.

---

# 10. PHASE G — MIDDLEWARE REVIEW

Audit middleware yang sudah tersedia.

Periksa:

- Request ID.
- Request logging.
- Panic recovery.
- CORS.
- Security headers.
- Request body limits.
- HTTP metrics.
- Unknown route handling.

Evaluasi:

1. Urutan middleware.
2. Tanggung jawab.
3. Hubungan dengan HTTP response.
4. Error handling.
5. Potensi konflik.
6. Dampak terhadap Health dan Readiness.
7. Dampak terhadap Scalar API Reference.

Jangan menambahkan authentication middleware, authorization, Redis, atau rate limiter pada M3C.

Jangan mengulang pengujian menyeluruh M1/M3B tanpa perubahan yang relevan.

Jika ada masalah nyata, lakukan perbaikan kecil dengan regression verification yang sesuai.

---

# 11. PHASE H — API STANDARD DOCUMENTATION

Dokumentasi merupakan deliverable utama M3C.

Jika belum tersedia, buat satu dokumen:

`documentation/api/api-standard.md`

Dokumen ini harus menjadi acuan resmi untuk pengembangan API Mapatu berikutnya.

Isi yang direkomendasikan:

## A. General API Conventions

- API versioning.
- URL naming.
- HTTP methods.
- Route grouping.
- Content type.
- Request/response expectations.

## B. Response Standards

- Success response.
- Error response.
- Empty response.
- Pagination metadata.
- Contoh JSON yang benar-benar sesuai implementasi.

## C. HTTP Status & Error Codes

- Konvensi status.
- Error code naming.
- Internal error safety.
- Validation error behavior.

## D. Request Validation

- DTO conventions.
- Binding.
- Field validation.
- Validation error formatting.

## E. Pagination

- Parameter.
- Defaults dan limits yang telah diverifikasi.
- Response metadata.
- Invalid pagination parameters.

## F. Middleware & Request Lifecycle

- Request ID.
- Logging.
- Recovery.
- Security headers.
- Middleware ownership.

## G. Developer Conventions

- Cara menambahkan endpoint baru.
- Penempatan handler dan DTO.
- Kapan usecase diperlukan.
- Kapan interface diperlukan.
- Pemanfaatan helper existing.
- Testing berdasarkan risiko.

## H. Examples

Berikan contoh minimal yang sesuai kontrak aktual.

Contoh tidak boleh diklaim sebagai endpoint aktif jika hanya ilustrasi.

### Documentation Requirements

Evaluasi dan perbarui apabila diperlukan:

- `README.md`
- `README.id.md`
- `documentation/architecture.md`
- `documentation/configuration.md`
- `documentation/operations.md`
- `documentation/cli.md`

Jangan mengubah dokumen yang tidak terdampak.

Jangan membuat file dokumentasi tambahan tanpa kebutuhan yang jelas.

**Documentation synchronization wajib diperiksa dalam setiap iterasi audit, implementation, review, dan fix.**

Jika tidak ada perubahan dokumentasi yang diperlukan pada suatu iterasi, catat alasannya.

---

# 12. PHASE I — TARGETED IMPLEMENTATION

Setelah audit dan standardization:

### CASE A — Existing Foundation Already Good

Jika komponen HTTP existing sudah memadai:

1. Jangan melakukan refactoring.
2. Jangan menambahkan abstraction.
3. Jangan membuat helper baru.
4. Dokumentasikan standar API.
5. Review kesesuaian dokumentasi dengan kode.
6. Selesaikan milestone apabila seluruh acceptance criteria terpenuhi.

**Dokumentasi saja merupakan hasil M3C yang valid.**

### CASE B — Minor Verified Issues

Jika ditemukan bug atau inkonsistensi kecil:

1. Identifikasi root cause.
2. Pilih perubahan terkecil.
3. Pertahankan struktur folder.
4. Pertahankan kontrak existing.
5. Implementasikan perbaikan.
6. Perbarui dokumentasi terdampak.
7. Verifikasi perubahan.
8. Lakukan review.

Tidak perlu meminta approval untuk perbaikan rutin non-breaking yang berada dalam scope M3C.

### CASE C — Structural or Breaking Changes

Jika solusi membutuhkan:

- Perubahan response envelope publik.
- Perubahan HTTP status contract yang breaking.
- Perubahan default API behavior yang berpengaruh pada consumer.
- Pemindahan package.
- Penghapusan komponen lintas fitur.
- Penambahan dependency besar.
- Perubahan arsitektur yang signifikan.

Jangan implementasikan perubahan tersebut secara otomatis.

Dokumentasikan alasan, dampak, dan alternatif minimalnya.

Tandai NEEDS DECISION dan minta persetujuan pengguna.

Lanjutkan bagian lain yang aman apabila memungkinkan.

---

# 13. VERIFICATION STRATEGY

Gunakan **risk-based verification**, bukan exhaustive testing.

M3C bukan milestone pengujian menyeluruh.

Jika tidak ada perubahan kode, jangan membuat unit test baru hanya untuk memenuhi formalitas.

Jika ada perubahan kode, jalankan pemeriksaan yang relevan:

```bash
go build ./...
go vet ./...
go test ./...
git diff --check
```

Pilih test sesuai area yang berubah.

Contoh:

- Response helper berubah → uji kontrak response terkait.
- Validation berubah → uji kasus valid dan invalid.
- Pagination berubah → uji boundary yang relevan.
- Middleware berubah → uji efek middleware terkait.
- Routing berubah → uji registrasi route terkait.

Jangan mengulang seluruh smoke test Scalar atau skenario database outage M1 jika tidak terdampak.

Jika hanya dokumentasi yang berubah, verifikasi akurasi terhadap source, tautan, contoh JSON, dan konsistensi dokumen.

Jangan memaksakan code coverage tertentu.

Jangan membuat file test untuk setiap file source.

Jangan menjalankan migration, seed, atau operasi database destruktif.

---

# 14. AUTONOMOUS DEVELOPMENT & REVIEW LOOP

Gunakan konvensi `develop/README.md` atau sumber workflow aktual yang ditemukan.

## Starting Input

Buat:

`develop/inputs/api/http-foundation/M3C-01-audit-and-standardization.md`

Catat:

- Objective.
- Scope.
- Existing implementation findings.
- Standardization plan.
- Perubahan yang benar-benar diperlukan.
- Documentation impact.
- Acceptance criteria.

## First Review

Setelah audit dan pekerjaan yang berada dalam scope selesai, buat:

`develop/reviews/api/http-foundation/M3C-01-review.md`

Review harus memeriksa:

- Akurasi analisis terhadap source.
- Konsistensi response.
- Error handling.
- Validation.
- Pagination.
- Routing.
- Middleware.
- Perubahan kode, jika ada.
- Dokumentasi API.
- Verification evidence.

Gunakan reviewer subagent berkonteks terpisah jika tersedia.

Jika tidak tersedia, lakukan structured self-review dan nyatakan keterbatasannya.

Jangan mengklaim independent review jika tidak dilakukan secara independen.

## Automatic Fix Loop

Jika review FAIL:

1. Catat temuan spesifik.
2. Buat input FIX M3C-02.
3. Perbaiki hanya masalah terverifikasi.
4. Perbarui dokumentasi yang terdampak.
5. Jalankan pemeriksaan yang relevan.
6. Review ulang.
7. Simpan M3C-02-review.md.

Lanjutkan M3C-03 dan seterusnya bila diperlukan.

Gunakan pola:

`AUDIT → STANDARDIZE → REVIEW → FIX → REVIEW → PASS`

Jangan meminta pengguna menangani bug rutin yang dapat diperbaiki dalam scope.

Jangan menghapus atau menimpa riwayat iterasi.

Jika lima iterasi perbaikan belum menghasilkan PASS, atau pekerjaan tidak menunjukkan kemajuan, hentikan dan laporkan BLOCKED beserta alasannya.

Jangan memaksakan PASS.

---

# 15. OUT OF SCOPE

Jangan mengimplementasikan:

- Authentication.
- Authorization/RBAC.
- Rate limiting.
- Redis.
- Employee APIs.
- Tenant APIs.
- Business database migrations.
- Seeder.
- Generic repository framework.
- New API gateway.
- New frontend project.
- Replacement of Gin.
- Replacement of Swaggo/Scalar.
- Complete restructuring.
- Broad LMS/SPMB cleanup.

Temuan di luar scope cukup dicatat sebagai kandidat M2 atau milestone berikutnya.

---

# 16. ACCEPTANCE CRITERIA

M3C dinyatakan PASS apabila:

- [ ] HTTP foundation existing telah dianalisis.
- [ ] Response contract sudah jelas.
- [ ] Error handling conventions sudah jelas.
- [ ] Request binding dan validation telah dievaluasi.
- [ ] Pagination conventions telah dievaluasi.
- [ ] Routing conventions telah ditetapkan.
- [ ] Middleware responsibilities telah diperiksa.
- [ ] Tidak terdapat masalah blocking dalam scope.
- [ ] Masalah kecil yang terverifikasi telah diperbaiki jika diperlukan.
- [ ] Tidak ada breaking change yang tidak disetujui.
- [ ] Tidak ada unnecessary abstraction.
- [ ] `documentation/api/api-standard.md` tersedia dan akurat.
- [ ] Dokumentasi lain yang terdampak telah disinkronkan.
- [ ] Verification sesuai perubahan telah dilakukan atau keterbatasannya dicatat.
- [ ] Input dan review setiap iterasi tersimpan.
- [ ] Review terakhir menghasilkan PASS berdasarkan bukti aktual.

Milestone dapat PASS tanpa perubahan source code jika hasil audit menunjukkan implementasi existing sudah memadai.

---

# 17. REQUIRED FINAL REPORT

Setelah PASS atau BLOCKED, tampilkan:

## A. Final Status

PASS / BLOCKED.

## B. HTTP Foundation Assessment

| Area | Finding | Decision |
|---|---|---|
| Response | ... | KEEP/FIX/DOCUMENT |
| Error | ... | ... |
| Validation | ... | ... |
| Pagination | ... | ... |
| Routing | ... | ... |
| Middleware | ... | ... |

## C. Files Changed

Cantumkan file source, konfigurasi, dan dokumentasi yang benar-benar berubah.

Jika tidak ada perubahan kode, nyatakan secara eksplisit.

## D. Documentation Changes

Jelaskan dokumen yang ditambahkan atau diperbarui.

## E. Verification Results

Laporkan command yang benar-benar dijalankan dan hasilnya.

## F. Review Iterations

- Jumlah iterasi.
- Temuan yang ditemukan.
- Temuan yang diperbaiki.
- Final review reference.
- Remaining non-blocking items.

## G. Deferred Items

Daftar singkat masalah yang layak dipertimbangkan pada M2 atau milestone berikutnya.

## H. Recommended Git Conventional Commit

Berikan commit message berdasarkan perubahan aktual.

Jangan melakukan Git commit otomatis.

---

# FINAL INSTRUCTION

Audit and standardize the existing Mapatu HTTP API foundation.

Prefer documenting valid existing conventions over implementing unnecessary abstractions.

Make only small, justified, non-breaking fixes within the approved scope.

Preserve the repository structure and existing operational contracts.

Autonomously review, fix, and re-review until PASS or a genuine blocker.

Keep documentation synchronized with actual implementation during every iteration.

Do not expand the milestone into authentication, database migration, or business API development.

Do not automatically commit changes.

---

## Execution input after read-only discovery — 12 November 2026

**Approval:** active M3C user request. **Baseline:** clean `98b61d0`. **Plan:** documentation-only standardization; no demonstrated active endpoint defect requires a non-breaking source fix. Actual workflow is `develop/README-develop.md`; no duplicate workflow file. M3B-01 review exists and has PASS. Preserve all historical inputs/reviews.

### Existing foundation findings

| Area / source evidence | Finding | Decision |
|---|---|---|
| `internal/shared/response/response.go:13–68` | Active success/error envelopes, success helper always200, optional meta/details | KEEP / DOCUMENT |
| `response.go:71–96`, handler/server/middleware consumers | Generic status mapping differs from explicit route/middleware/readiness codes; unlisted statuses map to ERROR | KEEP / DOCUMENT exact behavior, no generic framework |
| `response.go:8`, `handlers/dtos/responses/meta/meta_response.go` | Shared response depends on handler DTOs; no import cycle or demonstrated active failure | DEFER package cleanup to M2 |
| `shared/httphelper/bind.go:23–55` and repository consumer search | JSON/query/automatic binding then validator; no current call sites outside definition | DOCUMENT as available helper, not active endpoint behavior |
| `shared/validation/validator.go:22–109` | JSON-key mapping/Indonesian field messages; top-level mapping only, nested namespaces may yield empty/colliding keys, min/max wording assumes character counts | DEFER hardening before nested/non-string adoption; no active request DTO/consumer |
| `shared/labels/field_label.go` | Legacy education labels/enums remain; Name fallback available | DEFER broad LMS/SPMB cleanup |
| `response.go:99–143` and pagination DTO | Metadata/link builder only, no consumers/parser/defaults/max/page-count policy; page arithmetic assumes caller bounds, total remains nil | DOCUMENT exact contract; DEFER future pagination policy and bounds with first consumer |
| `internal/routes/`, `cmd/server.go:44–71` | Root probes, development docs/JSON, optional metrics; existing404/405; no /v1 business group | KEEP / DOCUMENT; future resource conventions clearly distinguished |
| `cmd/server.go:45–55`, `middlewares`, `shared/observability/http.go` | Active middleware order, public safe errors, request-ID/logging/recovery/CORS/body limits/metrics | KEEP / DOCUMENT with boundary limitations |
| `http_hardening.go:29–90` | Known-length limit may reject early; unknown length bound as consumed; JSON Content-Type checked only for a nonempty body on write methods | DOCUMENT; no blanket promise of eager full-body/strict JSON validation |
| `cmd/server.go:148–158` | Legacy multipart bypass paths retained but no registered upload handler/file-size enforcement | DEFER cleanup/file-aware boundary to upload feature; do not claim uploads active |
| `shared/errors`, `domain/models/repositories`, `handler/mappers` directories | Empty placeholders, not active Go packages | DEFER, do not fill with abstractions |

No SIMPLIFY action is justified by an actual consumer problem. No current change needs approval. Future pagination defaults/maximums, new success statuses and nested validation contracts require a separate feature decision before adoption; they are not silently imposed or blockers to documenting current behavior.

### Standardization plan and actual required changes

1. Add the expressly requested single `documentation/api/api-standard.md` as the authoritative developer reference, distinguishing implemented behavior, future conventions and unavailable/deferred capabilities.
2. Preserve response envelopes, /health,/ready, status contracts, config/module/generated files and all source. Add links from both READMEs and architecture so the standard is discoverable.
3. Cover response null/empty semantics, precise codes, safe error translation, helper validation tag behavior/limitations, pagination absence of defaults/policies, actual routing/middleware and focused developer examples.
4. Evaluate configuration/operations/CLI; leave unchanged if behavior is unchanged and they remain accurate. Keep audit history here rather than in the technical standard.
5. Separate-context reviewer checks source accuracy, examples, classifications and limits. Fix only verified findings, with sequential FIX/review artifacts if needed.

### Acceptance and verification guidance

All original M3C acceptance criteria above apply. The standard must not advertise unused helpers as active HTTP features, invent defaults/routes or claim old runtime checks were rerun. Since source does not change, verify source alignment, JSON examples, links/anchors and documentation consistency; `git diff --check` and source/config/generated/dependency preservation checks. No new tests or repeated build/vet/test/database/Scalar smoke merely for prose. No DB operations, server restart, dependency update, generation or commit required.
