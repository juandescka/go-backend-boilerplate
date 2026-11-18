# Mapatu Backend — First Prompt

> **Tujuan:** menyamakan konteks ChatGPT, Codex, atau AI agent lain saat pertama kali menangani repository Mapatu, setelah pindah akun/model, atau setelah konteks percakapan tidak lagi dapat dipercaya.
>
> **Mode:** **READ-ONLY / ANALYSIS ONLY**. Dokumen ini tidak memberikan izin implementasi.
>
> **Lokasi:** `backend/hris-core-service/develop/FIRST_PROMPT.md` (relatif terhadap workspace HRIS).

## Cara menggunakan

1. Buka percakapan baru dengan AI yang akan membantu proyek ini.
2. Jika AI memiliki akses ke repository, minta AI **membaca dokumen ini dan menjalankan bagian “Prompt untuk dijalankan”**.
3. Jika AI **tidak** memiliki akses filesystem/repository, unggah dokumen ini, `AGENTS.md`, dokumentasi yang relevan, dan input/review terbaru. AI harus membedakan dokumen yang tersedia dan yang belum tersedia; **jangan menebak isi file**.
4. Baca ringkasan yang dihasilkan AI. Setelah konteks benar, berikan **tugas terpisah** untuk analisis, pembuatan Master Prompt, review, atau implementasi.

Gunakan kembali prosedur ini ketika respons AI mulai menyimpang dari keputusan proyek. Untuk mengaudit kualitas Master Prompt yang sudah dibuat, gunakan pula [`TEMPLATE_PROMPT.md`](TEMPLATE_PROMPT.md).

---

## Prompt untuk dijalankan

Anda bertindak sebagai **AI project assistant untuk Mapatu Backend Boilerplate**, sebuah fondasi backend Go bagi HRIS multi-tenant. Fokus Anda pada tahap ini adalah **memahami proyek**, bukan mengubahnya.

### 1. Batas tugas

- **Read-only**: jangan membuat, mengedit, memindahkan, menghapus, atau meng-commit file.
- Jangan menjalankan build, test, lint, migration, seed, server, instalasi dependency, atau operasi yang mengubah state. Perintah eksplorasi read-only boleh dilakukan bila akses tersedia.
- Jangan melakukan refactoring, memberi perintah implementasi kepada agent lain, atau membuat input eksekusi secara otomatis.
- Jangan meminta keputusan teknis yang tidak diperlukan hanya untuk menyelesaikan orientasi.
- Jangan menampilkan credential, token, DSN lengkap, nilai rahasia dalam `.env`, atau data pribadi. Jika perlu, sebutkan nama key saja.

### 2. Kenali lokasi dan batas proyek

- Temukan root repository aktif dan root backend `backend/hris-core-service/` bila tersedia.
- Periksa struktur aktual, `go.mod`, entrypoint, `cmd/`, `config/`, `internal/`, `documentation/`, dan `develop/` secukupnya untuk memahami alur proyek.
- Jangan menganggap path absolut pengguna berlaku di mesin/akun lain.
- **Repository aktif adalah proyek kerja.** `lms-core-service` dan boilerplate `spmb-2026` hanya referensi perbandingan jika tersedia; tidak boleh ditulis atau dianggap sumber kebenaran runtime Mapatu.

### 3. Baca sumber konteks dalam urutan berikut

1. `AGENTS.md` dan, bila perlu, `AGENTS.id.md` — aturan agent.
2. `README.md` atau `README.id.md` — tujuan dan quick start.
3. `documentation/architecture.md` — struktur serta konvensi.
4. `documentation/configuration.md` — sumber konfigurasi dan keterbatasannya.
5. `documentation/operations.md` — health, readiness, lifecycle, dan status implementasi.
6. `documentation/cli.md` — daftar command yang terverifikasi dan yang masih rencana.
7. `develop/README.md` — alur input → eksekusi → review → fix → review sampai PASS.
8. `develop/TEMPLATE_PROMPT.md` — standar penyusunan Master Prompt.
9. Dokumen teknis lain **hanya jika ada dan relevan** dengan pekerjaan yang sedang dibahas.

Jangan menganggap semua file pasti sudah tersedia. Laporkan file yang tidak dapat diakses, tanpa mengarang kontennya.

### 4. Temukan status pekerjaan terbaru

- Periksa `develop/inputs/` dan `develop/reviews/` berdasarkan **fitur dan milestone**, bukan hanya urutan nama file secara global.
- Temukan review terbaru pada scope yang relevan, ID iterasinya, pasangan input, verdict **PASS / FAIL / BLOCKED**, temuan terbuka, dan pekerjaan berikutnya.
- Jika belum ada input/review atau tidak jelas milestone yang sedang aktif, nyatakan **belum dapat dipastikan**; jangan menetapkan milestone aktif secara sepihak.
- Review historis adalah **bukti pekerjaan pada saat review tersebut**, bukan jaminan kode masih sama. Audit statis M0 tidak setara dengan hasil build atau runtime test.
- Periksa apakah ada keputusan arsitektur atau perubahan kontrak yang masih menunggu persetujuan.

### 5. Selaraskan informasi, jangan menyamakan rencana dengan fakta

Gunakan kaidah berikut:

- **Untuk perilaku aktual:** periksa source code yang dapat diakses; jangan menyimpulkan fungsi aktif hanya karena terdokumentasi atau memiliki anotasi.
- **Untuk aturan proyek:** gunakan keputusan pengguna yang sudah disetujui, `AGENTS.md`, dan dokumen teknis yang relevan.
- **Untuk proses:** gunakan `develop/README.md` dan histori input/review.
- **Untuk rencana:** pisahkan jelas `PROPOSED`, `APPROVED`, `IMPLEMENTED`, dan `VERIFIED`.
- Bila sumber bertentangan, sebutkan konflik, lokasi sumber, dan dampaknya. Jangan diam-diam memilih interpretasi yang memperluas scope.

Prinsip Mapatu: **Minimal Preservation**, Pragmatic Clean Architecture, idiomatic Go, struktur existing diutamakan, interface/layer hanya bila perlu, testing berbasis risiko, dan perubahan struktur/kontrak/dependency harus didiskusikan sebelum eksekusi.

### 6. Lakukan pemeriksaan keselarasan konteks

Jawab secara ringkas:

- Apa yang sudah ada di source, apa yang belum terhubung, dan apa yang masih rancangan?
- Adakah komponen warisan LMS/SPMB yang belum dibutuhkan? **Catat saja**, jangan langsung membersihkan.
- Apakah rekomendasi atau keputusan yang akan datang berpotensi mengubah struktur folder, public API, database, security, atau dependency?
- Apakah dokumentasi dan status review terakhir masih selaras dengan source yang terlihat?

Jika akses source tidak tersedia, batasi penilaian pada dokumen yang diberikan dan tandai seluruh keterbatasan verifikasi.

### 7. Format output wajib

Berikan hasil orientasi dengan bagian berikut; **singkat, spesifik, dan berdasarkan bukti**:

1. **Identitas & akses proyek** — root yang ditemukan; file/referensi dapat dibaca dan yang tidak.
2. **Ringkasan arsitektur** — package dan alur startup/routing/dependency yang relevan.
3. **Current State** — pisahkan *Verified in source*, *Documented but not verified*, dan *Proposed*.
4. **Development Status** — milestone/fitur/iterasi terakhir yang benar-benar dapat ditemukan; verdict dan issue terbuka.
5. **Keputusan yang harus dihormati** — batas arsitektur, scope, dan approval yang masih berlaku.
6. **Inkonsistensi atau ketidakpastian** — hanya yang konkret dan penting.
7. **Recommended Next Step** — satu langkah logis, **sebagai usulan, bukan eksekusi**.

Cantumkan path atau referensi file pada temuan penting. Jangan mengarang path, hasil test, status PASS, atau perintah CLI.

**STOP setelah output orientasi.** Tunggu instruksi pengguna untuk pekerjaan berikutnya.

---

## Perintah pemulihan konteks (opsional)

Gunakan bila jawaban atau Master Prompt dari AI mulai tidak konsisten:

> Baca ulang `develop/FIRST_PROMPT.md`, `develop/TEMPLATE_PROMPT.md`, `AGENTS.md`, dokumen teknis relevan, dan input/review terbaru. Bandingkan rekomendasi terakhir Anda dengan source aktual dan keputusan yang sudah disetujui. Tunjukkan secara spesifik mana yang berupa fakta, asumsi, usulan, atau penyimpangan; perbaiki rekomendasi tanpa memperluas scope. Jangan mengubah file atau menjalankan implementasi.

## Hubungan dengan dokumen lain

- [`../AGENTS.md`](../AGENTS.md) — aturan kerja coding agent.
- [`../documentation/architecture.md`](../documentation/architecture.md) — konteks arsitektur.
- [`README.md`](README.md) — aturan pencatatan pekerjaan/iterasi.
- [`TEMPLATE_PROMPT.md`](TEMPLATE_PROMPT.md) — standar Master Prompt yang akan dibuat setelah orientasi.
