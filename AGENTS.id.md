# Mapatu Backend — Instruksi untuk Coding Agent

[English](AGENTS.md) | [Bahasa Indonesia](AGENTS.id.md)

> Cakupan: `backend/hris-core-service`. Dokumen ini mengarahkan cara kerja coding agent di repository. Dokumen ini **tidak otomatis memberikan izin perubahan**; ikuti permintaan pengguna dan input pengembangan yang telah disetujui.

## Konteks project

Mapatu merupakan fondasi backend Go untuk HRIS multi-tenant. Prinsip utama: **Pragmatic Clean Architecture**, kode Go yang idiomatis dan mudah dibaca, abstraksi seminimal mungkin, serta mempertahankan struktur repository yang sudah ada. Audit M0 menemukan beberapa wiring operasional belum lengkap; jangan menyamakan fitur yang dideklarasikan dengan fitur yang sudah berjalan.

Baca referensi berikut sesuai kebutuhan:

- [`README.id.md`](README.id.md): pengenalan dan quick start.
- [`documentation/architecture.md`](documentation/architecture.md): struktur package dan konvensi arsitektur.
- [`documentation/configuration.md`](documentation/configuration.md): perilaku konfigurasi aktual.
- [`documentation/operations.md`](documentation/operations.md): health/readiness serta lifecycle aplikasi.
- [`documentation/cli.md`](documentation/cli.md): command CLI dan status verifikasinya.
- [`develop/README-develop.md`](develop/README-develop.md): siklus input → eksekusi → review → perbaikan.

**Source code aktual menjadi acuan untuk perilaku yang benar-benar tersedia.** Jika dokumentasi berbeda dengan kode, laporkan perbedaannya; jangan diam-diam menyatakan fitur yang belum dibuat sudah berjalan. Usulan tidak sama dengan implementasi.

## Sebelum melakukan perubahan

1. Baca file terkait, caller yang relevan, serta input yang telah disetujui di `develop/inputs/` jika tersedia.
2. Jelaskan scope dan file yang diperkirakan terdampak.
3. Pertahankan struktur folder/package, naming, kontrak API, dan perilaku existing jika memungkinkan.
4. **Diskusikan perubahan struktur terlebih dahulu.** Pemindahan/rename package, penambahan layer, perubahan kontrak API publik, pergantian dependency inti, atau cleanup luas membutuhkan persetujuan eksplisit.
5. Untuk permintaan yang hanya berupa audit/perencanaan: **jangan mengubah file atau menjalankan build, test, migration, maupun instalasi dependency** kecuali diminta secara terpisah.
6. Untuk cleanup, catat perubahan tracked, staged dan untracked sebelum mengedit; pertahankan pekerjaan sebelumnya dan pisahkan dari tugas baru. **Jangan melakukan staging atau commit otomatis.**

## Prinsip implementasi

- Pilih perubahan terkecil yang benar-benar menyelesaikan masalah teridentifikasi.
- Ikuti tanggung jawab existing: routing di `internal/routes/`, translasi HTTP di `internal/handlers/`, serta inisialisasi/penutupan dependency di `internal/registry/`.
- Gunakan `internal/usecases/` dan `internal/infrastructure/` jika tanggung jawabnya jelas; jangan membuat use case, repository, interface, factory, atau layer tambahan untuk endpoint sederhana.
- Buat interface ketika memang ada batas consumer atau kebutuhan implementasi alternatif, bukan secara otomatis.
- Jangan menyalin kompleksitas LMS/SPMB secara utuh. Keduanya referensi, bukan repository yang boleh diubah.
- Jangan membuat package placeholder, fitur spekulatif, atau generic helper yang belum memiliki pemakai.
- Pertahankan konsistensi konfigurasi, logging, error response, dan penamaan; jangan membocorkan secret maupun detail error internal melalui respons HTTP.
- Jangan memperluas pekerjaan atau melakukan refactoring besar sebagai efek samping task yang sempit.

### Komentar kode dan penamaan

- Jelaskan keputusan penting atau tidak langsung jelas, terutama security, transaksi, concurrency, kompatibilitas dan error handling. Jelaskan alasannya; jangan menarasikan operasi yang jelas atau memberi komentar setiap baris.
- Tulis komentar singkat, natural dan berbahasa Inggris. Pertahankan komentar existing yang bermakna dan perbarui saat perilaku berubah. Ikuti GoDoc jika dokumentasi berguna; hindari marker AI, anotasi milestone dan riwayat perubahan dalam source.
- Gunakan PascalCase untuk identifier Go exported dan camelCase untuk local/unexported, dengan initialism Go yang idiomatis. Config JSON memakai camelCase, API JSON publik dan nama PostgreSQL memakai snake_case, environment variable memakai UPPER_SNAKE_CASE.
- Pertahankan key/kontrak existing serta pengecualian yang disetujui (termasuk legacy JWT keys dan audit FK `created_by`/`updated_by`). Jangan mengganti identifier publik/config/schema hanya demi style; ajukan breaking change terlebih dahulu.

### Kebijakan logging

- Gunakan structured logger dan middleware existing. Tulis native message singkat yang mudah dibaca dan pertahankan nilai `event_code`; jangan menambah field message duplikat atau abstraksi/dependency logging tanpa kebutuhan.
- Sertakan `error_code` aplikasi/API aktual jika tersedia secara andal melalui metadata per request. Jangan menebaknya dari status HTTP, buffer/parse response body untuk logging, atau mengubah kontrak respons demi telemetry.
- Pertahankan request ID pada respons, completion log dan diagnostik terkait. Emit satu HTTP completion event; pertahankan event security/operasional terpisah hanya untuk tujuan berbeda, tanpa pengulangan log di setiap layer.
- Pilih level berdasarkan dampak operasional: penolakan autentikasi rutin dapat INFO, penolakan kebijakan/kapasitas WARN, kegagalan internal ERROR. Pertahankan fallback konservatif untuk hasil tanpa klasifikasi; jangan menganggap setiap 4xx sebagai insiden keamanan.
- Jangan log password/hash, raw token/token hash, Authorization/cookie, credential, config rahasia atau dump body/header. Pertahankan sanitasi metadata tak tepercaya menjadi satu baris dengan panjang terbatas; hilangkan nilai query sensitif.
- Gunakan pemeriksaan berbasis risiko untuk akurasi error code, korelasi/isolasi, kebijakan level, duplikasi, privasi serta perilaku HTTP/auth yang tetap sama. Format dan batas implementasi ada di [operations](documentation/operations.md).

### Menjaga boundary keamanan

- Pertahankan normalisasi identifier otoritatif database, admission login berbatas sebelum verifikasi password, boundary trusted proxy eksplisit dan error autentikasi/throttle generik. Jangan evict limit aktif untuk menyamarkan saturasi atau mengklaim kontrol per proses sebagai terdistribusi. Health/readiness tetap di luar limit login.
- Jangan tampilkan file secret bootstrap lokal; periksa ignore/history/permission melalui metadata saja. Pertahankan minimum 15 karakter pada input interaktif dan guard admin pertama khusus lokal.

## Verifikasi dan keamanan

- Pilih verifikasi sesuai risiko perubahan. Tidak wajib membuat test untuk setiap file, dan tidak perlu target coverage persentase sembarang.
- Uji perilaku teramati dan kontrak kritis, dengan prioritas security/authentication/authorization, integritas database, concurrency dan business rules. Jangan membuat test hanya untuk mengisi template.
- Utamakan fixture sederhana dan fake kecil pada batas consumer; hindari mock berlebihan, setup duplikat dan test yang hanya mengulang perilaku framework. Unit dan integration test dapat melindungi risiko berbeda walaupun memanggil fungsi sama.
- Merge/simplify hanya jika keterbacaan membaik tanpa kehilangan perlindungan regresi. Hapus test hanya dengan bukti bahwa coverage relevannya sudah obsolete atau tetap terlindungi secara setara; catat perilaku sebelum/sesudah dan jalankan pemeriksaan relevan. Jumlah test/file bukan target keberhasilan.
- Bedakan hasil audit statis, build, unit/integration/API test, dan pemeriksaan manual. Catat hanya command yang benar-benar dijalankan beserta hasil sebenarnya.
- Jangan mengklaim PASS runtime jika aplikasi tidak dapat dijalankan atau dependency penting belum tersedia.
- Jangan menjalankan migration destruktif/down/reset, mengubah data production, atau cleanup luas tanpa izin eksplisit dan verifikasi environment target.
- Jangan menimpa secret lokal, `.env`, atau konfigurasi runtime. Untuk dokumentasi, gunakan contoh tanpa credential asli.
- Hindari perubahan `go.mod`/`go.sum` atau file generated yang tidak berkaitan. Setiap perubahan dependency harus jelas alasannya dan sesuai scope.

## Artefak pengembangan dan review

Untuk pekerjaan dengan workflow tercatat, simpan **input eksekusi yang telah disetujui** di `develop/inputs/<domain>/<feature>/` dan **review hasil eksekusi nyata** di `develop/reviews/<domain>/<feature>/`. Pasangkan ID, misalnya `M1-01-implementation.md` ↔ `M1-01-review.md`; perbaikan memakai ID berikutnya dan merujuk review sebelumnya. Jangan menimpa review FAIL lama atau membuat klaim PASS historis yang tidak pernah diverifikasi.

Status review: **PASS**, **FAIL**, atau **BLOCKED**. Review harus menjelaskan perubahan, bukti pemeriksaan, kondisi acceptance criteria, masalah tersisa, dan langkah berikutnya. Rincian workflow terdapat pada [`develop/README-develop.md`](develop/README-develop.md). Buat/perbarui artefak hanya dalam scope pekerjaan yang disetujui.

Jika tugas yang disetujui mencakup autonomous review–fix, lanjutkan fix rutin dalam scope dengan pasangan input/review baru sampai PASS atau blocker nyata. Pisahkan tahap implementasi dan review; utamakan reviewer dengan context terpisah jika tersedia, selain itu nyatakan structured self-review secara eksplisit. Jangan menyebut self-review independen atau menimpa review FAIL. Perubahan di luar approval tetap membutuhkan keputusan.

## Perubahan dokumentasi

- Jaga `README.md` dan `README.id.md` tetap selaras jika instruksi bersama berubah.
- Pastikan dokumentasi menggambarkan implementasi aktual; tandai fitur yang masih pending atau proposed.
- README berisi ringkasan command; penggunaan CLI detail ada di `documentation/cli.md`.
- Riwayat proses disimpan di `develop/`, bukan `documentation/`.
- Jangan menambah folder dokumentasi atau standar baru jika belum diperlukan.

## Laporan penyelesaian

Setelah task selesai, ringkas: **scope**, **file yang berubah**, **perilaku**, **verifikasi yang benar-benar dijalankan**, **keterbatasan**, dan **keputusan yang masih diperlukan**. Untuk audit/perencanaan, sampaikan temuan dan matriks usulan perubahan, lalu berhenti untuk mendapatkan persetujuan.
