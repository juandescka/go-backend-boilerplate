# Mapatu Backend — Master Prompt Template

> **Tujuan:** pedoman membuat, mengevaluasi, dan memperbaiki Master Prompt lintas ChatGPT, Codex, dan AI coding agent lainnya.
>
> **Status:** **template**, bukan izin untuk menjalankan pekerjaan. Gunakan bersama [`README-develop.md`](README-develop.md), [`FIRST_PROMPT.md`](FIRST_PROMPT.md), dan [`../AGENTS.md`](../AGENTS.md).

## 1. Prinsip penulisan Master Prompt

- **Satu prompt, satu tujuan yang dapat diverifikasi.** Jangan mencampurkan audit, implementasi, dan review sebagai satu tugas tanpa batas yang jelas.
- **Existing first.** Periksa source dan keputusan yang berlaku; pertahankan struktur bila tidak ada alasan teknis konkret untuk mengubahnya.
- **Scope eksplisit.** Pisahkan `In Scope`, `Out of Scope`, dan perubahan yang perlu persetujuan.
- **Bukti lebih penting daripada klaim.** Jangan menyimpulkan hasil runtime dari kode yang belum dijalankan.
- **Ringkas dan dapat dieksekusi.** Jangan mengulang seluruh isi dokumentasi; cukup arahkan pembaca ke file sumber yang relevan.
- **Tidak memaksakan testing.** Pilih pemeriksaan berdasarkan risiko, bukan satu test untuk setiap file atau target coverage arbitrer.
- **Komentar selektif.** Pertahankan komentar penting; jelaskan alasan keputusan yang tidak langsung jelas dengan bahasa Inggris singkat/natural, bukan operasi yang sudah jelas atau riwayat milestone.
- **Perilaku lebih utama daripada jumlah test.** Prioritaskan kontrak/security/database/concurrency/business rules; gunakan fixture/fake secukupnya, hindari duplikasi dan hapus test hanya dengan bukti coverage relevannya tetap terlindungi atau sudah obsolete.
- **Konvensi tanpa breaking cleanup.** Go PascalCase/camelCase, config JSON camelCase, API JSON/database snake_case dan environment UPPER_SNAKE_CASE. Pertahankan key serta pengecualian kontrak existing; jangan mengganti nama hanya demi style.
- **Security sesuai AGENTS.** Pertahankan normalisasi identifier database, limiter berbatas dan trusted proxy; uji saturation/cancellation/concurrency tanpa bypass, dokumentasikan limit per proses, dan jangan membuka isi file secret bootstrap.
- **Logging sesuai AGENTS.** Native message singkat, stable event code, actual API error code jika tersedia dan request ID konsisten; tanpa dump body/header/secret, duplikasi completion atau abstraksi baru. Pilih level sesuai konteks dan uji risiko perubahan.
- **Tidak memaksakan arsitektur.** Interface, adapter, use case, repository, folder, atau dependency baru hanya jika diperlukan.
- **Git tetap milik pengguna.** Audit perubahan tracked/staged/untracked dan pisahkan pekerjaan sebelumnya; jangan stage atau commit otomatis.
- **Satu sumber aturan.** `AGENTS.md` mengatur agent, `documentation/` mengatur sistem, `develop/README-develop.md` mengatur riwayat eksekusi; template ini mengatur **bentuk dan kualitas instruksi**.

## 2. Kapan memakai setiap mode

| Mode | Tindakan agent | Perubahan file? | Hasil |
|---|---|---|---|
| **ANALYSIS** | Baca, telusuri, bandingkan, usulkan | **Tidak** | Temuan + opsi + keputusan yang perlu approval |
| **IMPLEMENTATION** | Kerjakan scope yang **sudah disetujui** | **Ya, terbatas** | Perubahan aktual + bukti pemeriksaan |
| **REVIEW** | Periksa hasil **eksekusi nyata** | **Tidak**, secara default | Review + PASS / FAIL / BLOCKED |
| **FIX** | Perbaiki temuan dari review tertentu | **Ya, terbatas** | Perbaikan aktual + bukti, dilanjutkan review terpisah |

Jangan memberi izin perubahan hanya dengan memilih mode IMPLEMENTATION/FIX. Persetujuan pengguna dan batas scope tetap diperlukan. Jika pengguna hanya meminta **dibuatkan Master Prompt**, hasilnya adalah teks prompt, **bukan eksekusi prompt tersebut**.

## 3. Konvensi identitas dan lokasi

Gunakan convention dari [`develop/README-develop.md`](README-develop.md):

```text
develop/inputs/<domain>/<feature>/<Milestone>-<NN>-<tujuan>.md
develop/reviews/<domain>/<feature>/<Milestone>-<NN>-review.md
```

Contoh hipotetis:

```text
develop/inputs/operational/health-readiness/M1-01-implementation.md
develop/reviews/operational/health-readiness/M1-01-review.md
develop/inputs/operational/health-readiness/M1-02-fix-timeout.md
develop/reviews/operational/health-readiness/M1-02-review.md
```

- Input dan review **berpasangan dengan ID iterasi yang sama**.
- FIX baru mengacu pada temuan review iterasi sebelumnya; **jangan mengubah review FAIL lama menjadi PASS**.
- Gunakan nama `kebab-case` untuk bagian deskriptif dan ID milestone yang telah disetujui.
- Buat folder fitur **hanya saat digunakan**. Jangan membuat direktori `tests/`, `prompts/`, atau framework tambahan hanya untuk memenuhi template.

## 4. Checklist sebelum menyusun prompt

Sebelum menghasilkan Master Prompt, pastikan informasi berikut **diketahui atau ditandai belum terverifikasi**:

1. Repository dan fitur sasaran.
2. Milestone serta tipe tugas (ANALYSIS / IMPLEMENTATION / REVIEW / FIX).
3. Input dan review terakhir yang relevan.
4. Source dan dokumentasi yang harus dibaca.
5. Keputusan yang sudah disetujui vs usulan yang masih menunggu approval.
6. Hasil yang diharapkan dan batas perubahan.
7. Risiko yang membenarkan jenis verifikasi tertentu.

**Jika ada informasi yang tidak tersedia:** jangan mengarangnya. Buat prompt yang memerintahkan agent **memeriksa fakta terlebih dahulu** dan **berhenti pada keputusan yang tidak dapat diambil tanpa persetujuan**. Jangan membatalkan seluruh tugas untuk kekurangan informasi nonkritis.

---

## 5. Kerangka Master Prompt (salin dan sesuaikan)

> Gunakan hanya bagian yang relevan; placeholder dalam `[kurung siku]` harus diganti sebelum dikirim untuk eksekusi. Jangan menambahkan file atau command yang belum terverifikasi sebagai fakta.

```markdown
# [MILESTONE-ITERATION] — [JUDUL]

- Project: Mapatu Backend Boilerplate (`backend/hris-core-service`)
- Feature: `[domain/feature]`
- Mode: [ANALYSIS | IMPLEMENTATION | REVIEW | FIX]
- Iteration: `[M#-NN]`
- Based on: `[path input/review sebelumnya, jika ada]`
- Approval status: [PROPOSED | APPROVED — sertakan rujukan keputusan]

## Objective
[Hasil tunggal yang ingin dicapai dan alasan singkat.]

## Required Reading
- `AGENTS.md`
- `develop/README-develop.md`
- `documentation/[dokumen-relevan].md`
- `[source-file yang relevan]`
- `[input/review terakhir, jika ada]`

Jika file tidak tersedia, jelaskan keterbatasan; jangan menebak isinya.

## Current Context
[Fakta yang sudah diketahui; tandai apa yang masih perlu diverifikasi dari kode.]

## In Scope
[Pekerjaan yang diizinkan secara eksplisit.]

## Out of Scope
[Pekerjaan yang tidak boleh dilakukan, termasuk perluasan fitur yang tidak terkait.]

## Architecture & Change Constraints
- Utamakan struktur dan konvensi existing.
- Jangan menambah layer, interface, package, dependency, atau tool tanpa kebutuhan nyata.
- Jika perlu mengubah struktur folder, public API contract, schema, security behavior,
  atau dependency penting di luar approval: jelaskan dampak dan minta keputusan dahulu.
- Jangan mengubah repository referensi LMS/SPMB.
- Jangan mengklaim command/fitur tersedia sebelum memeriksa implementasinya.
- Pertahankan komentar penting; tulis komentar why secara selektif sesuai AGENTS, tanpa marker AI atau anotasi milestone.
- Pertahankan naming/key existing; konvensi tidak mengizinkan breaking API/config/schema renaming.
- Ikuti logging policy AGENTS: gunakan logger existing, native message yang mudah dibaca, stable event_code dan request_id; teruskan actual error_code melalui metadata per request tanpa menebak dari status atau membaca response body. Pertahankan privasi, satu completion event dan level sesuai dampak operasional.

## Execution Instructions
[Langkah sesuai MODE; lihat bagian khusus mode di dokumen template.]

## Verification
[Pemeriksaan berbasis risiko/perilaku dan batas environment. Bedakan executed, not run, dan blocked.
Tidak wajib test per source atau target coverage. Jika merge/remove/simplify test diperlukan,
catat risiko/perilaku sebelum-sesudah dan bukti coverage kritis tetap tersedia.]

## Documentation Impact
[Dokumen yang perlu diselaraskan, atau 'none' bila tidak ada perubahan kontrak.]

## Acceptance Criteria
- [ ] [Kriteria terukur 1]
- [ ] [Kriteria terukur 2]

## Required Output
[Ringkasan, file yang dibaca/diubah, bukti, temuan, keputusan dan next step.
Rekomendasi model/reasoning effort sesuai ketersediaan nyata dan risiko pekerjaan.
Rekomendasi Conventional Commit sesuai seluruh diff aktual, dengan scope pekerjaan lama/baru
dipisahkan bila perlu; jangan stage atau commit otomatis.]

## Stop Conditions
[Persetujuan hilang, konflik kontrak, dependency tidak tersedia, risiko destruktif,
 atau fakta penting tidak dapat diverifikasi.]

STOP pada batas mode dan scope ini.
```

### Catatan penyesuaian

- Untuk **ANALYSIS**, `Approval status` biasanya `PROPOSED`, `Execution Instructions` berarti **inspeksi read-only** dan verifikasi bersifat statis; jangan menjalankan build/test/migration kecuali diminta eksplisit.
- Untuk **IMPLEMENTATION** atau **FIX**, pastikan scope perubahan telah disetujui. Perbolehkan pemeriksaan relevan setelah implementasi, tetapi **jangan otomatis menjalankan test berat, migration destruktif, atau seed production**.
- Untuk **REVIEW**, hanya telaah perubahan dan pemeriksaan read-only/non-destruktif yang memang dibutuhkan; **jangan diam-diam memperbaiki kode**. Audit dapat menyatakan BLOCKED jika bukti belum cukup.

---

## 6. Petunjuk khusus per mode

### A. ANALYSIS — Audit & planning

**Tujuan:** memahami kondisi aktual dan mengusulkan pekerjaan, bukan mengimplementasikannya.

Masukkan instruksi:

- Telusuri struktur, dependency flow, source, dan dokumentasi relevan.
- Pisahkan `Current State`, `Finding`, `Recommendation`, `Impact`, dan `Confidence/Verification Limit`.
- Bandingkan referensi hanya jika ada dan berhubungan langsung dengan scope.
- Usulkan **Option A — Minimal Preservation** sebagai baseline pembanding; berikan alternatif hanya bila ada manfaat nyata.
- Cantumkan daftar keputusan yang membutuhkan approval, perubahan file yang diusulkan, dan acceptance criteria implementasi berikutnya.
- **STOP tanpa perubahan file.** Jangan memberi status PASS runtime atas audit statis.

**Output ringkas:** Executive summary, current structure, findings berbukti, opsi, impact matrix, usulan milestone, pending decisions.

### B. IMPLEMENTATION — Implementasi scope yang disetujui

**Tujuan:** menyelesaikan satu input kerja, menjaga behavior existing, dan memberi bukti perubahan.

Masukkan instruksi:

- Baca input yang disetujui dan cek source terkini sebelum perubahan.
- Kerjakan hanya scope; gunakan package existing dan perubahan terkecil yang layak.
- Jangan membangun abstraction/framework baru hanya demi generalisasi.
- Jika menemukan kebutuhan perubahan besar di luar scope, **STOP bagian terkait dan minta approval**; jangan menyamakan temuan baru dengan izin otomatis.
- Sinkronkan dokumentasi yang memang terdampak oleh perubahan.
- Evaluasi kebutuhan named seeder untuk setiap modul. Daftarkan hanya dataset yang diperlukan, pisahkan reference data yang dimiliki migration, pertahankan idempotency dan hindari credential hardcoded. Jalankan hanya melalui command eksplisit. Tambah risk-based tests dan perbarui [CLI](../documentation/cli.md#4-development-seeding-and-administrator-bootstrap) dan [operations](../documentation/operations.md#local-seeding-and-bootstrap) saat menambah seeder.
- Jalankan pemeriksaan relevan sesuai risiko dan environment; dokumentasikan yang tidak dijalankan.
- Audit nilai test existing sebelum menambah test; hindari duplikasi/mocks/fixtures berlebihan dan pertahankan negative/security paths. Jangan membuat test hanya untuk melengkapi template.
- Lapor file berubah, perilaku, hasil command/test aktual, risiko tersisa, dan bagian yang belum bisa diverifikasi.
- Pisahkan tahap implementasi dan review pada **iterasi yang sama**. Utamakan reviewer dengan context terpisah; jika tidak tersedia, nyatakan structured self-review beserta keterbatasannya, tanpa klaim independent review.
- Jika input secara eksplisit menyetujui autonomous review–fix, lanjutkan fix rutin dalam scope dan review baru sampai PASS atau blocker nyata. Pertahankan FAIL, perbarui dokumentasi terdampak dan minta keputusan hanya untuk kebutuhan di luar approval; jangan menganggap mode IMPLEMENTATION saja mengizinkan loop ini.

**Output ringkas:** Changed files, implementation summary, verification evidence, deviations/blockers, handoff for review.

### C. REVIEW — Pemeriksaan hasil eksekusi

**Tujuan:** mengevaluasi kode yang benar-benar berubah, bukan sekadar menilai narasi agent sebelumnya.

Masukkan instruksi:

- Temukan **input dengan ID sama**, approval, diff/source aktual, dan acceptance criteria.
- Periksa perubahan beserta efek samping terhadap kontrak yang sudah ada.
- Catat evidence yang **benar-benar diperoleh** dan batas yang belum diperiksa.
- Status **PASS** jika semua kriteria wajib memiliki bukti cukup; **FAIL** jika ada kriteria wajib gagal; **BLOCKED** jika penilaian tergantung akses/keputusan/dependency yang tidak tersedia.
- Setiap finding harus menyebut lokasi sumber, dampak, dan rekomendasi sempit.
- Temuan nonblocking di luar scope dicatat sebagai follow-up, bukan otomatis FAIL.
- Jangan mengedit kode dan jangan menimpa review historis.

**Output ringkas:** Reviewed scope, evidence, acceptance criteria matrix, findings, verdict, next action.

### D. FIX — Perbaikan berdasarkan review

**Tujuan:** menutup masalah terverifikasi dari **review sebelumnya**, tanpa mengulang seluruh scope awal.

Masukkan instruksi:

- Referensikan path review dan ID finding yang hendak diselesaikan.
- Verifikasi bahwa finding masih berlaku pada source saat ini.
- Perbaiki hanya finding yang termasuk scope dan telah diizinkan.
- Jangan menghapus test atau menurunkan standar acceptance criteria untuk memperoleh PASS semu.
- Dokumentasikan perubahan dan pemeriksaan terarah.
- Setelah FIX, buat **review iterasi baru** yang berpasangan dengan input FIX; jangan mengubah verdict historis.

**Output ringkas:** Findings addressed, exact changes, evidence, remaining issues, handoff for new review.

---

## 7. Pemeriksaan kualitas Master Prompt sebelum digunakan

Gunakan checklist ini untuk mencegah *prompt drift*:

- [ ] Mode dan scope jelas; tidak ada tugas campuran yang saling bertentangan.
- [ ] Membaca source, `AGENTS.md`, dan dokumen relevan yang benar.
- [ ] Status `PROPOSED`, `APPROVED`, `IMPLEMENTED`, `VERIFIED` tidak dicampur.
- [ ] Tidak mengklaim command, route, fitur, atau test yang belum dibuktikan.
- [ ] Tidak menambah folder/layer/interface/dependency tanpa alasan konkret.
- [ ] Perubahan struktur/kontrak/security di luar approval membutuhkan keputusan pengguna.
- [ ] Tidak mengadopsi kompleksitas LMS/SPMB hanya karena tersedia sebagai referensi.
- [ ] Acceptance criteria relevan dan dapat dinilai.
- [ ] Verification proporsional terhadap risiko dan tidak memaksakan test seluruh file.
- [ ] Test melindungi perilaku/kontrak kritis, bukan jumlah file; cleanup memiliki bukti coverage yang setara dan tidak menambah fixture/mock berlebihan.
- [ ] Komentar penting dipertahankan dan why-comment ditulis selektif; naming/key/pengecualian kompatibilitas existing tetap dijaga.
- [ ] Dokumentasi yang terdampak disebutkan; riwayat pengembangan tetap di `develop/`.
- [ ] Format output dan STOP conditions eksplisit.
- [ ] Review tidak otomatis mengubah kode atau mengklaim PASS tanpa bukti.
- [ ] Autonomous review–fix hanya dalam approval; no automatic staging/commit; rekomendasi model/effort dan Conventional Commit sesuai konteks aktual.

## 8. Prompt untuk memperbaiki Master Prompt yang menyimpang

> Evaluasi **Master Prompt terakhir**, jangan menjalankannya. Baca `develop/FIRST_PROMPT.md`, `develop/TEMPLATE_PROMPT.md`, `AGENTS.md`, dokumentasi terkait, serta input/review terakhir yang tersedia. Audit kesesuaian terhadap struktur existing, persetujuan, kontrak, scope, testing, dan bukti implementasi. Tampilkan: (1) penyimpangan yang spesifik, (2) dampaknya, (3) perbaikan yang diperlukan, dan (4) **Master Prompt revisi lengkap** sesuai template ini. Jangan menambah pekerjaan, menebak status, atau mengubah repository.

## 9. Rekomendasi model, reasoning effort dan commit

Saat **menyajikan Master Prompt di chat**, tuliskan **setelah blok prompt**, bukan di dalam instruksi eksekusi:

```text
Rekomendasi model: [model yang benar-benar tersedia bagi pengguna]
Reasoning effort: [Medium | High | opsi yang tersedia]
Alasan: [1 kalimat sesuai kompleksitas audit/implementasi/review]
```

Jangan mengunci nama model tertentu di dokumen ini; pilihan dan ketersediaan model dapat berubah. Isi ini merupakan rekomendasi percakapan, **bukan syarat eksekusi agent**.

Pada laporan akhir pekerjaan, sertakan rekomendasi model/reasoning effort untuk pekerjaan berikutnya sesuai ketersediaan nyata dan kompleksitasnya; jika tidak diketahui, nyatakan keterbatasan dan jangan mengarang kemampuan/model. Berikan Conventional Commit berdasarkan final diff yang benar-benar ada. Jika beberapa milestone belum di-commit, sarankan pemisahan scope; jika satu commit gabungan diperlukan, berikan judul dan body singkat yang mencakup semuanya. Rekomendasi ini tidak mengizinkan staging atau commit otomatis.

## 10. Referensi

- [`FIRST_PROMPT.md`](FIRST_PROMPT.md) — orientasi/recovery konteks AI.
- [`README-develop.md`](README-develop.md) — workflow, ID dan lokasi iterasi.
- [`../AGENTS.md`](../AGENTS.md) — aturan agent.
- [`../documentation/architecture.md`](../documentation/architecture.md) — struktur dan konvensi.
- [`../documentation/operations.md`](../documentation/operations.md) — contoh scope operational health/readiness.
