# Mapatu — API Standards & HTTP Foundation

Dokumen ini menjadi acuan pengembangan HTTP API Mapatu. **Kontrak yang sudah berjalan**, **helper yang tersedia tetapi belum dipakai**, dan **konvensi endpoint berikutnya** dibedakan secara eksplisit. Jangan mengubah kontrak publik hanya untuk menyesuaikan preferensi penamaan atau bentuk respons.

Sumber utama: [server](../../cmd/server.go), [routes](../../internal/routes/routes.go), [response](../../internal/shared/response/response.go), [binding](../../internal/shared/httphelper/bind.go), [validation](../../internal/shared/validation/validator.go), dan [HTTP middleware](../../internal/middlewares/http_hardening.go). Detail konfigurasi/lifecycle tetap di [configuration](../configuration.md) dan [operations](../operations.md).

## 1. Cakupan dan ownership

Alur aktif:

```text
Gin router → middleware global → route handler → response
                                  └→ readiness boundary → shared SQL pool
Registry menyusun dependency; handler menerjemahkan HTTP.
```

| Lokasi | Tanggung jawab dan status |
|---|---|
| `internal/routes/` | Registrasi route konkret, termasuk dokumentasi; tidak menyimpan business logic |
| `internal/handlers/` | Handler operasional aktif; handler menentukan data, status/code publik dan kapan berhenti |
| `internal/shared/response/` | Envelope JSON dan builder metadata; bukan global error mapper |
| `internal/shared/httphelper/` | Helper binding/validation tersedia; login memakai decoder endpoint khusus; helper ini belum memiliki consumer |
| `internal/shared/validation/`, `labels/` | Formatter error field dan label; belum menjadi alur validasi endpoint aktif |
| `internal/middlewares/`, `shared/observability/` | Boundary HTTP, korelasi, log, recovery, metrics |
| `handlers/dtos/responses/meta/` | DTO metadata existing yang diimpor response helper |
| `internal/shared/errors/`, `internal/domain/repositories/`, `internal/handlers/mappers/` | Direktori placeholder, bukan implementasi Go aktif |

M5B menyediakan login dan bearer current-user API dengan DTO tersendiri. HRIS business API dan parser pagination belum aktif. Scoped RBAC tersedia dan rate limiter login aktif. Komponen yang dideklarasikan atau folder kosong tidak membuktikan kemampuan tersebut tersedia.

## 2. Routing dan representasi

### Kontrak aktif

| Method/path | Perilaku |
|---|---|
| `POST /v1/auth/login` | Login publik, envelope200/400/401/413/415/422/429/500/503 |
| `GET /v1/auth/me` | Opaque bearer;200 atau401, operational failure500 |
| `POST /v1/auth/logout` | Opaque bearer; revoke current session,200/data:null |
| `POST /v1/auth/logout-all` | Opaque bearer; revoke own active sessions,200/data:null |
| `GET /health` | JSON envelope, 200, tanpa cek dependency |
| `GET /ready` | JSON envelope, 200 atau 503; deadline 2 detik; menolak saat shutdown |
| `GET /docs` | Scalar HTML, hanya development |
| `GET /swagger/doc.json` | Swagger 2.0 mentah dari Swaggo, hanya development |
| `GET <metrics.path>` | Format Prometheus jika metrics aktif; default path `/metrics` |

Tidak ada prefix `/v1` untuk probe/dokumentasi. Swagger UI lama tidak aktif. Docs dan spec memakai environment aplikasi existing; bukan `GIN_MODE`. Opaque bearer melindungi /v1/auth/me; tidak ada JWT pada route aktif. Startup tetap fail-fast terhadap DB; jangan menyimpulkan liveness berjalan saat inisialisasi DB gagal.

JSON API memakai `application/json; charset=utf-8`. HTML dokumentasi, spec mentah, Prometheus, redirect Gin dan respons dari HTTP transport adalah pengecualian envelope aplikasi. `NoRoute` menghasilkan 404 `ROUTE_NOT_FOUND`; `HandleMethodNotAllowed=true` dan `NoMethod` menghasilkan 405 `METHOD_NOT_ALLOWED` untuk method yang tidak terdaftar pada path yang cocok, setelah middleware. Jangan menjanjikan envelope JSON untuk error/redirect yang terjadi sebelum handler aplikasi.

### Konvensi untuk endpoint berikutnya

- Business API dapat dikelompokkan di `/v1` dengan Gin group biasa. Group /v1/auth aktif; endpoint HRIS bisnis belum tersedia.
- Gunakan resource noun plural dalam lowercase, kata majemuk dengan hyphen, dan path parameter bernama deskriptif. Contoh **ilustratif, belum aktif**: `GET /v1/examples/:example_id`.
- Daftarkan HTTP method secara eksplisit sesuai perilaku: GET membaca; POST melakukan create/action; PUT/PATCH memperbarui; DELETE menghapus. Ini bukan penambahan route, alias HEAD/OPTIONS atau kontrak status baru.
- Pertahankan key envelope dan metadata existing. Gunakan tag `json` eksplisit pada DTO baru; jangan menormalisasi key publik existing secara massal. Pagination memakai `has_next`, sorting memakai `order_by`, sedangkan filter existing memiliki `isActive`.
- Definisikan query, path parameter, optional/required fields, status, error code dan aturan empty result sebelum implementasi endpoint. Policy 201/204, pagination default dan batas baru harus ditetapkan pada kontrak fitur terkait, bukan diasumsikan dari helper.

## 3. Response standard

### Success

`response.SuccessResponse(c, data, message, pagination, filters, sorting)` selalu menulis **200**. Tidak ada parameter status, created-response helper atau no-content helper. `message` ditujukan untuk manusia; client memakai status dan struktur data, bukan parsing kalimat.

Respons `/health` yang dibentuk handler:

```json
{"success":true,"message":"Service is healthy.","data":{"status":"ok"}}
```

Respons `/ready` saat siap:

```json
{"success":true,"message":"Service is ready.","data":{"status":"ready"}}
```

| Nilai yang diberikan caller | JSON yang dihasilkan |
|---|---|
| `data=nil` | `"data":null`, key tetap ada |
| Slice nil, termasuk slice nil di dalam interface | `"data":null` |
| Slice kosong yang diinisialisasi | `"data":[]` |
| Map/struct kosong | Sesuai tipe/tag JSON; bukan otomatis null/array |
| Semua argumen metadata nil | Key `meta` tidak dikirim |
| Sebagian metadata non-nil | `meta` hanya memuat pointer metadata yang non-nil |

Untuk list kosong pada endpoint berikutnya, pilih slice terinisialisasi jika kontraknya array. Jangan mengganti perilaku nil secara global. Untuk operasi tanpa payload yang memakai helper existing, data tetap null dan status200; **ini berbeda dari HTTP204**. Endpoint baru yang memerlukan 201 atau 204 harus menyepakati status/envelope-nya terlebih dahulu; jangan menyetel status lalu memanggil helper200 atau menulis JSON pada respons204.

### Error

`ErrorResponseWithCode(c, status, code, message, details)` memakai status caller tanpa memetakan error domain secara otomatis. Bentuknya hanya `error`, tanpa top-level `success:false`.

Respons readiness gagal:

```json
{"error":{"code":"SERVICE_NOT_READY","message":"Service is not ready."}}
```

- `error.code`: identifier stabil, `UPPER_SNAKE_CASE`, tidak sama dengan event code log.
- `error.message`: penjelasan publik yang aman. Jangan kirim DSN, SQL, stack, token, password atau raw internal `err.Error()`.
- `details`: opsional, hanya field publik yang disetujui. Interface nil dihilangkan oleh `omitempty`; typed nil yang tersimpan dalam interface dapat terserialisasi sebagai `"details":null`. Jangan mengandalkan semua typed nil akan dihilangkan.
- Helper response menulis JSON; **tidak** otomatis `return` atau `c.Abort()`. Handler harus berhenti setelah respons terminal. Middleware yang menolak request harus abort agar handler berikutnya tidak menulis respons lagi.

Helper error juga menyimpan hanya code/status sebagai metadata privat pada Gin context untuk completion logger. `error_code` log sama dengan code yang benar-benar ditulis oleh helper pada respons error pertama, selama status akhir cocok dan code memakai format aman: huruf besar ASCII awal, lalu huruf besar/angka/underscore, maksimal 64 byte. Code kosong/tidak aman, error tanpa helper dan response sukses tidak diberi `error_code`; logger tidak menebaknya dari status atau membaca body. Pemeriksaan ini hanya membatasi telemetry, tanpa mengubah JSON/status/error code publik. Pemanggilan helper setelah respons committed tidak mengganti metadata respons pertama. Detail format, level dan privasi log ada di [operations](../operations.md#51-structured-logging).

## 4. Status HTTP dan error code

`ErrorResponse` memilih code generik berikut dari status; `ErrorResponseWithCode` mempertahankan code eksplisit caller.

| Status | Code dari `ErrorResponse` | Penggunaan yang tersedia |
|---|---|---|
| 400 | `BAD_REQUEST` | Mapping helper; malformed/type/parameter error dapat diterjemahkan oleh handler berikutnya |
| 401 | `UNAUTHORIZED` | Login dan bearer middleware memakai generic401 |
| 403 | `FORBIDDEN` | Mapping tersedia; CORS aktif memakai code khusus di bawah |
| 404 | `NOT_FOUND` | Mapping tersedia; unknown route aktif memakai code khusus |
| 409 | `CONFLICT` | Mapping tersedia; belum ada konflik bisnis aktif |
| 422 | `VALIDATION_FAILED` | Login memakai422 untuk invalid input |
| 429 | `RATE_LIMIT_EXCEEDED` | Login ditolak karena batas IP/identifier atau kapasitas record; header `Retry-After` dalam detik bulat dibulatkan ke atas |
| 500 | `INTERNAL_SERVER_ERROR` | Recovery dan kegagalan internal yang aman |
| Status lain, termasuk 405/413/415/503 | `ERROR` | Fallback generik, bukan otomatis code operasional |

Code eksplisit yang dipakai source aktif:

| Status/code | Consumer |
|---|---|
| 404 `ROUTE_NOT_FOUND` | `cmd/server.go` NoRoute |
| 405 `METHOD_NOT_ALLOWED` | `cmd/server.go` NoMethod |
| 403 `CORS_ORIGIN_NOT_ALLOWED` | CORS menolak Origin |
| 403 `CORS_PREFLIGHT_NOT_ALLOWED` | Requested method/header preflight tidak diizinkan |
| 413 `REQUEST_BODY_TOO_LARGE` | Boundary body size/read failure; lihat batasan pada bagian middleware |
| 415 `UNSUPPORTED_MEDIA_TYPE` | Boundary media type |
| 503 `SERVICE_NOT_READY` | Readiness gagal/tidak tersedia/timeout/shutdown |
| 500 `INTERNAL_SERVER_ERROR` | Panic recovery, spec tidak tersedia atau operational auth failure |
| 503 `AUTHENTICATION_UNAVAILABLE` | Local password-verification capacity penuh |

Jangan memakai `ErrorResponse(503, ...)` dan menganggap code menjadi `SERVICE_NOT_READY`; gunakan code eksplisit seperti handler existing. Status/code baru dipilih sesuai kontrak fitur, bukan dibuat menjadi daftar error seluruh HRIS.

Formatter validation mengembalikan error Go, bukan HTTP response. Sebagai **konvensi untuk handler DTO berikutnya**, pisahkan malformed JSON/type/path/query sebagai400, validation field sebagai422 dengan code `VALIDATION_FAILED`, dan `IsRequestBodyTooLarge(err)` sebagai413 sebelum fallback parsing. Ini bukan global mapper yang sudah terpasang, dan tidak mengubah status endpoint existing. Terjemahkan kegagalan internal ke500 aman; simpan detail diagnostik di log terkontrol.

## 5. Request binding dan validation

### Helper yang tersedia

`httphelper.BindAndValidate[T](c, &dto, bindType)`:

| BindType | Pemanggilan actual |
|---|---|
| `BindJSON` | `c.ShouldBindJSON` |
| `BindQuery` | `c.ShouldBindQuery`; parameter menggunakan tag `form` Gin |
| `BindForm` atau nilai lain | `c.ShouldBind`: pemilihan binder otomatis berdasarkan method/content type, bukan pemaksaan form |

Gin melakukan validation dengan tag **`binding`**. Setelah binding berhasil, helper melakukan `validator.New().Struct(target)` dengan tag default **`validate`**. Kedua engine terpisah; tidak ada registrasi custom validator dalam proyek. Aturan `validate` tambahan tidak dijalankan jika binding sudah gagal. Jangan menganggap salah satu tag otomatis menggantikan yang lain atau menduplikasi semua rules tanpa alasan.

DTO sebaiknya struct request yang jelas, dengan tag JSON/form sesuai transport. Untuk membedakan nilai hilang dari zero/false, pilih pointer/tipe optional sesuai kontrak; `required` pada scalar zero-value memiliki konsekuensi validation. Login memakai DTO request dan decoder endpoint khusus; helper path belum aktif. Path mentah dari Gin perlu diparse dan dicek tipe/range/format secara eksplisit sebelum dipakai; jangan menambah generic DTO framework hanya untuk itu.

Perilaku JSON existing memakai default Gin1.11.0: unknown fields tidak ditolak dan decoder membaca satu nilai JSON tanpa pemeriksaan EOF tambahan oleh helper. Jangan menjanjikan strict JSON, penolakan semua trailing values, atau query/path policy yang belum diimplementasikan. Evaluasi perubahan strictness sebagai perubahan kontrak fitur terkait.

### Error field

`validation.FormatValidationErrors` membentuk `ValidationErrorDetail` dengan `Message="Kesalahan validasi"` dan `Details map[string]string`. Key berasal dari tag `json` tanpa opsi, atau lowercase nama field jika tag tidak ada; formatter **tidak** memakai tag `form`. Label dicari lewat `labels.Get`, lalu fallback nama field. Messages existing berbahasa Indonesia; respons operasional memakai bahasa Inggris. Tidak ada global kebijakan terjemahan baru.

Rules yang memiliki message khusus: `required`, `email`, `min`, `max`, `eqfield`, `gte`, `lte`, `oneof`; sisanya memakai `<label> tidak valid`. `min/max` saat ini menyebut karakter tanpa membedakan angka/slice. Formatter hanya membuat peta field **top-level**: namespace nested/index tidak ditelusuri, key dapat kosong/bertabrakan, dan beberapa error untuk key sama saling menimpa. Field dengan tag `json:"-"`, embedded fields, nested DTO, atau slice root tidak boleh diasumsikan memiliki formatter yang benar. Gunakan target struct non-nil; helper tidak menyediakan kontrak target arbitrary yang aman untuk formatter.

**Batas adopsi:** belum ada consumer endpoint, sehingga tidak ada respons aktif yang diperbaiki/diubah di sini. Sebelum menerima nested DTO atau aturan min/max non-string, perbaiki formatter dan tambahkan regression verification pada fitur yang membutuhkannya. Label/rules khusus LMS/SPMB tidak dihapus tanpa consumer/approval.

### Contoh minimal, belum menjadi endpoint aktif

DTO flat untuk ilustrasi:

```go
type ExampleRequest struct {
    Name string `json:"name" form:"name" binding:"required,min=2"`
}
```

Field `name` kosong dapat menghasilkan error Go dengan details `{"name":"Nama wajib diisi"}`. Jika handler mengikuti konvensi422 di atas dan mengirim `ValidationErrorDetail.Details`, contoh HTTP body-nya:

```json
{"error":{"code":"VALIDATION_FAILED","message":"Request validation failed.","details":{"name":"Nama wajib diisi"}}}
```

Contoh tersebut **bukan** output endpoint yang diregistrasikan. Binding helper tidak mengirimnya sendiri. Jangan memasukkan `ValidationErrorDetail.Error()` mentah sebagai respons internal atau membungkus objek details sekali lagi tanpa kontrak yang jelas.

## 6. Pagination

Tidak ada endpoint list aktif atau parser pagination. `BuildPaginationResponse(c, page, limit, hasNext)` hanya membentuk [DTO metadata](../../internal/handlers/dtos/responses/meta/meta_response.go) dan link.

| Hal | Perilaku source existing |
|---|---|
| `page`, `limit` | `int` yang diberikan caller; tidak dibaca, divalidasi, dinormalisasi atau diberi default oleh builder |
| Default page/limit, maximum limit | **Belum ditetapkan**; jangan menganggap contoh angka sebagai default |
| Invalid/missing parameter | Belum ada kebijakan HTTP; builder sendiri tidak mengembalikan error |
| `has_next` | Nilai bool dari caller; bukan hasil query di builder |
| `has_previous` | `page > 1` |
| `next_url`, `prev_url` | Path relatif + query yang di-encode; argumen page/limit menggantikan query yang sama untuk link yang dibuat |
| Query lainnya | Dipertahankan termasuk nilai berulang, melalui `url.Values` |
| Context/request/URL nil | Fallback path `/`, query kosong |
| Link tidak tersedia | String kosong; field tetap ada karena bukan `omitempty` |
| `total` | `*int64` opsional; builder membiarkannya nil sehingga dihilangkan. Caller dapat mengisi pointer ke0 untuk menampilkan0 |
| `total_pages` | Tidak ada field/perhitungan existing |

Sebelum memakai builder, caller harus menyepakati dan memvalidasi input positif, batas maksimum, serta arithmetic pagination/offset. `page+1` dan `page-1` memakai `int`; builder tidak melindungi overflow. Jangan meneruskan nilai tak terbatas atau `MaxInt` dengan `hasNext=true`, dan jangan menghitung offset tanpa pemeriksaan range. Menetapkan clamp/reject/default universal belum disetujui oleh kontrak endpoint mana pun; lakukan bersama consumer pertama, tanpa mengarang angka global pada dokumentasi ini.

Contoh **ilustratif**: caller memilih page1/limit10 (bukan default), `hasNext=false`, membuat slice hasil kosong dan memberi pagination non-nil. Helper success dapat membentuk:

```json
{"success":true,"message":"Results retrieved.","data":[],"meta":{"pagination":{"page":1,"limit":10,"has_next":false,"has_previous":false,"next_url":"","prev_url":""}}}
```

Total dan total pages tidak boleh diklaim sudah dihitung. Keputusan invalid parameter, sorting/filter input, total count dan offset bounds dibuat sebelum endpoint list diperkenalkan. Jangan menambah business route hanya untuk menguji metadata.

## 7. Middleware dan request lifecycle

Urutan dari `cmd/server.go`:

```text
Request ID → Request completion log → HTTP metrics (jika aktif)
→ Panic recovery → Security headers → CORS → Body limit → Handler / 404 / 405
```

| Komponen | Ownership dan batas perilaku |
|---|---|
| Request ID | Header `X-Request-ID` valid diteruskan setelah trim: 1–128 karakter ASCII dari huruf/angka/`.`/`_`/`:`/`-`; lainnya diganti UUID. Disimpan pada Gin context dan standard request context. Tidak ditambah ke envelope |
| Completion log | Satu event `http.request.completed`, native message yang mudah dibaca, status/durasi/ukuran/path/route/request ID/IP/user-agent dan actual `error_code` jika tersedia dari helper. Bukan raw body/query, dump header atau token. Path/user-agent dibersihkan dan dibatasi; `event_code` berbeda dari public error code |
| Metrics | Aktif hanya jika dikonfigurasi; memakai resolved route atau `unmatched`, mengecualikan path metrics sendiri. Tidak menambah auth/rate limiter |
| Recovery | Mencatat event, tipe panic dan stack internal tanpa mengirim nilai panic/stack ke client; menulis500 aman jika writer belum committed, lalu abort. Respons yang sudah committed tidak dapat diganti menjadi envelope500 baru |
| Security headers | `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`; tidak menyediakan CSP/HSTS otomatis; namespace `/v1/auth` mendapat `Cache-Control: no-store` dan `Pragma: no-cache` sebelum early rejection |
| CORS | Origin exact allowlist atau wildcard sesuai validator config. Tanpa Origin diteruskan. Origin ditolak memberi403. OPTIONS dengan Origin memvalidasi requested method/header; valid204 tanpa body, invalid403 `CORS_PREFLIGHT_NOT_ALLOWED`; tanpa Origin mengikuti routing biasa. Vary digabung untuk semua keputusan |
| Body limit | Diterapkan hanya untuk POST/PUT/PATCH/DELETE dengan body; bukan GET probes/docs. Batas bytes dari konfigurasi. Handler harus menangani error reader/binder dan tidak menganggap seluruh body selalu sudah dibaca oleh middleware |
| 404/405 | Fallback aplikasi setelah middleware; CORS/body/recovery dapat menghasilkan status lain lebih awal |

Body boundary mengenali JSON dan `application/*+json` untuk body non-kosong yang diperiksa, menolak media lain dengan415. Setelah lolos gate multipart, body dengan Content-Length nol atau EOF saat peek diteruskan untuk ditentukan handler. Multipart dengan body non-nil di luar allowlist dapat ditolak415 meskipun body kosong; pemeriksaan method/body-nil tetap berjalan lebih dulu. Content-Length melebihi batas dapat ditolak413 sebelum binding; unknown-length stream dibatasi ketika dibaca, sehingga `httphelper.IsRequestBodyTooLarge` perlu diprioritaskan ketika mengubah error binder menjadi HTTP. Peek read error non-EOF saat ini juga dipetakan413; jangan menyamakan setiap413 dengan bukti byte count telah diukur penuh.

Multipart hanya dilewatkan untuk path allowlist legacy di `cmd/server.go`; path tersebut **tidak** membuat upload endpoint aktif atau menjamin batas file. Endpoint upload nanti harus mendefinisikan batas/validation file sendiri sebelum bypass digunakan. `X-LMS-Client` masih ada pada allow-header CORS; itu residu, bukan fitur auth/tenant aktif dan tidak diubah di sini.

Global middleware tetap berlaku pada docs/probe, sehingga status normal200/503 bukan jaminan untuk semua request dengan Origin/body/transport yang berbeda. Lihat [operations](../operations.md) untuk fail-fast startup, shutdown, metrics dan Scalar.

## 8. Menambah endpoint secara minimal

1. Tetapkan kontrak route/method, tipe data, status/code, validation dan empty/list semantics. Jangan mengasumsikan policy yang belum ada pada bagian5/6.
2. Tempatkan HTTP parsing/translation di handler dan DTO transport di bawah folder existing jika memang diperlukan. Jangan expose model GORM/internal error secara otomatis.
3. Tambahkan usecase hanya bila ada pekerjaan aplikasi/bisnis; liveness sederhana tidak memerlukannya. Interface hanya untuk boundary consumer yang nyata atau alternatif implementasi, bukan untuk satu pemanggilan trivial.
4. Registry menginisialisasi dependency eksplisit; route menerima handler konkret. Middleware global tetap di server; middleware fitur diberikan ke group yang tepat saat fitur itu disetujui.
5. Gunakan response/helper existing dalam batas kontraknya; setelah respons terminal, `return`, dan gunakan `c.Abort()` ketika menghentikan middleware chain. Jangan membuat generic response/error/repository/router framework.
6. Update anotasi, lalu [command generator existing](../cli.md#5-api-specification-generation) hanya jika kontrak spec berubah. Scalar membaca spec yang sama; dokumen ini bukan sumber spec kedua.
7. Uji risiko yang berubah: binding valid/invalid, envelope/status, pagination bounds, atau middleware/route relevan. Tidak ada target coverage atau keharusan satu test per file; jangan mengulang outage/Scalar smoke jika tidak terdampak.

## 9. Batas yang perlu diselesaikan sebelum perluasan

- Coupling `shared/response` ke handler DTO belum menimbulkan siklus/bug aktif; pemindahan package memerlukan keputusan terpisah.
- Validation nested/embedded, key berbasis query/form, non-string min/max dan kebijakan bahasa perlu dikerjakan sebelum consumer yang memerlukannya.
- Pagination defaults, max, invalid input, total/offset/overflow guard belum menjadi kebijakan API aktif. Tetapkan dengan endpoint list pertama dan approval kontraknya.
- Strict JSON/trailing-value rejection, created/no-content statuses dan file-upload boundary bukan kemampuan global yang sudah disediakan helper.
- Residu label/multipart/header/config LMS/SPMB dan placeholder package adalah kandidat cleanup terpisah. Jangan mengaktifkan Redis/refresh atau menghapus lintas fitur demi kelengkapan standar; auth/RBAC dan limiter login memiliki consumer aktif tersendiri.

## 10. Authentication API (M5B)

[Authentication contract](authentication.md) memakai success/error helper existing. Credential/session denials401 memakai UNAUTHORIZED generik; operational failures500 tetap dibedakan; crypto-capacity503 memakai AUTHENTICATION_UNAVAILABLE. Login memakai decoder khusus dengan batas16 KiB dan EOF check, validator yang sama untuk email, serta password UTF-8 max1024 byte. Malformed JSON/type400, invalid fields422, body-size413 dan media-type415 mengikuti konvensi. Ini tidak mengubah strictness helper binding global. Request/response DTO auth berada di folder DTO existing; model/hash tidak diserialisasi. Authentication tidak memberikan tenant/RBAC permission.

## 11. Session lifecycle (M5C)

Logout routes memakai envelope/helper existing, tanpa request DTO atau free user/session identifier. Scope berasal hanya dari typed authenticated context. Repeat/revoked/expired/inactive caller401 memakai UNAUTHORIZED generik, operational failure500 tetap aman. Response200/data:null tidak mengandung token/hash/session count. Revalidation setelah database lock waits mempertahankan kontrak ketika dua request concurrent melewati middleware. Tidak ada exception envelope, tenant permission atau endpoint renewal. Lihat [authentication API](authentication.md) untuk batas concurrency/idempotency.

## Authorization foundation

For a future protected business route, apply `BearerAuthentication` and then `RequirePermission` with an explicit permission and a resolver that derives ownership from the stored resource. Verify any requested owner IDs against that resource. RBAC v1.0 requires active owners, membership for tenant data and a matching assignment with an explicit grant. Tenant and organization assignments cover contained scopes. Platform assignments cover platform operations only. See the [canonical contract](../database/tenant-membership-rbac-schema.md).

Missing or invalid authentication returns 401. Missing grants and invalid owner context return 403. Internal lookup failure returns 500. The middleware aborts the request on failure and attaches the verified resource scope on success.

### Authentication throttle responses

Login returns generic 401 `UNAUTHORIZED` for unknown/inactive/no-password/wrong-password accounts. Protected routes return 401 with `WWW-Authenticate: Bearer` for invalid sessions. Authenticated permission/membership/scope denials return 403 `FORBIDDEN`; platform grants do not bypass tenant checks. Login admission rejection returns 429 `RATE_LIMIT_EXCEEDED` and `Retry-After: <seconds>` rounded up, without account-state details. Malformed/invalid input remains 400/422, operational control/database failures remain safe 500, and concurrent KDF capacity exhaustion remains 503 `AUTHENTICATION_UNAVAILABLE`. Earlier middleware may reject a request before login admission. See [authentication](authentication.md) and [actual policy](../configuration.md#login-abuse-policy).
