# Mapatu — Development Workflow

> **Cakupan:** `backend/hris-core-service/develop/`.
> **Tujuan:** menyimpan instruksi pekerjaan (*inputs*) dan bukti pemeriksaan hasil eksekusi (*reviews*) berdasarkan fitur, secara berurutan sampai pekerjaan dapat dinyatakan **PASS**.
> **Status:** konvensi kerja yang diusulkan untuk dipakai setelah disetujui. Dokumen ini tidak menyatakan bahwa M1 sudah diimplementasikan.

## 1. Peran `develop/` dan `documentation/`

- **`documentation/`** adalah referensi teknis yang berlaku: arsitektur, konfigurasi, operasi, dan CLI. Isinya harus mengikuti kode yang benar-benar tersedia.
- **`develop/`** menyimpan *jejak pekerjaan*: input yang disetujui, perubahan yang dieksekusi, hasil review, masalah, dan input perbaikan selanjutnya.
- Temuan historis tetap berada pada `develop/`; jangan menyalin seluruh log iterasi ke dokumen teknis.
- Laporan **PASS** tidak menggantikan dokumentasi atau pemeriksaan kode aktual.

## 2. Struktur folder

```text
develop/
├── README.md
├── inputs/
│   ├── foundation/
│   │   └── architecture/                  # jika ada pekerjaan terkait
│   └── operational/
│       └── health-readiness/
│           ├── M1-01-implementation.md
│           ├── M1-02-fix-db-timeout.md
│           └── M1-03-fix-shutdown.md
└── reviews/
    ├── foundation/
    │   └── architecture/
    └── operational/
        └── health-readiness/
            ├── M1-01-review.md
            ├── M1-02-review.md
            └── M1-03-review.md
```

**Ini contoh struktur dan nama file, bukan daftar artefak yang sudah ada.** Buat hanya folder fitur yang sedang dikerjakan. Struktur relatif di `inputs/` dan `reviews/` harus sama agar pasangan file mudah ditemukan.

Gunakan **domain/fitur**, bukan nama agent atau tanggal, sebagai pengelompokan utama. Contoh selanjutnya: `authentication/login/` atau `authorization/rbac/` **jika** fitur tersebut benar-benar mulai dikembangkan.

## 3. Penamaan dan ID iterasi

Format input:

```text
<Milestone>-<NN>-<tujuan-singkat>.md
```

Format review:

```text
<Milestone>-<NN>-review.md
```

Contoh:

| ID iterasi | Input | Review |
|---|---|---|
| `M1-01` | `M1-01-implementation.md` | `M1-01-review.md` |
| `M1-02` | `M1-02-fix-db-timeout.md` | `M1-02-review.md` |
| `M1-03` | `M1-03-fix-shutdown.md` | `M1-03-review.md` |

- `M1` adalah ID milestone yang sudah disepakati; `01`, `02`, dan seterusnya adalah urutan **eksekusi pada fitur/milestone itu**.
- ID yang sama pada *input* dan *review* berarti review tersebut menilai pekerjaan dari input itu.
- Jangan gunakan `final`, `final2`, `fix-final`, `latest`, atau menimpa artefak iterasi lama.
- Nama tujuan singkat memakai **kebab-case**; contoh `fix-db-timeout`.
- Jika milestone yang sama memiliki beberapa fitur, penyimpanan terpisah berdasarkan folder fitur; pastikan ID unik dalam setiap jalur fitur dan jelaskan cakupannya.
- Audit M0 yang sudah pernah dilakukan tidak perlu dibuat ulang atau diberi status eksekusi secara retroaktif. Jika arsip M0 disimpan, beri label jelas sebagai **audit statis**, bukan implementasi yang telah diuji.

## 4. Siklus kerja

```text
Input M1-01 (disetujui)
   ↓
Eksekusi perubahan kode sesuai scope
   ↓
Review M1-01 (berdasarkan diff dan bukti nyata)
   ├─ PASS    → sinkronkan dokumentasi → tutup pekerjaan
   ├─ FAIL    → buat Input M1-02 yang merujuk temuan M1-01
   │             ↓
   │           eksekusi perbaikan → Review M1-02 → ulangi bila perlu
   └─ BLOCKED → catat blocker → tunggu keputusan/dependency
                               → lanjut hanya setelah blocker diselesaikan
```

**Definisi status:**

- **PASS:** seluruh *acceptance criteria* wajib dalam scope terpenuhi berdasarkan bukti yang cukup. Tidak berarti seluruh sistem bebas bug.
- **FAIL:** ada kriteria wajib yang belum terpenuhi atau regresi yang perlu diperbaiki. Review harus menjelaskan temuan dan tindakan berikutnya.
- **BLOCKED:** tidak bisa diverifikasi/dilanjutkan tanpa keputusan, izin, dependency, atau lingkungan yang diperlukan. Jangan menganggapnya PASS atau otomatis mengubah desain untuk menghindari blocker.

Temuan opsional di luar scope dapat dicatat sebagai *follow-up*, tanpa memaksakan FAIL bila kriteria wajib sudah terpenuhi. Jika keputusan arsitektur berubah, hentikan pekerjaan terkait dan minta persetujuan terlebih dahulu.

### Aturan penting antariterasi

1. **Satu input = satu cakupan eksekusi yang dapat direview.** Review menilai kode dan hasil nyata setelah input dijalankan, bukan sekadar mengomentari teks input.
2. **Review harus menyebut input sumber** dan bukti: file berubah, perilaku, command yang benar-benar dijalankan, hasil, serta yang belum diverifikasi.
3. **Input perbaikan mengacu pada review sebelumnya.** Fokus pada temuan yang belum selesai; jangan menyalin ulang seluruh master prompt jika tidak perlu.
4. **Jaga sejarah:** jangan mengedit review FAIL menjadi PASS. Buat pasangan input/review baru setelah perubahan berikutnya.
5. **Perubahan struktur folder/package, kontrak API, atau dependency besar memerlukan persetujuan** sebelum eksekusi, termasuk jika disarankan oleh reviewer.
6. **Penulisan dokumentasi mengikuti perilaku yang terbukti.** Periksa perubahan dokumen sebelum PASS bila termasuk acceptance criteria.
7. **Testing berbasis risiko.** Tidak wajib ada test untuk setiap file; review harus jujur mengenai pengujian yang tidak dijalankan dan alasannya.

## 5. Struktur minimum file input

Gunakan bagian berikut secara ringkas; buang subbagian yang benar-benar tidak relevan.

```markdown
# M1-02 — Fix Database Readiness Timeout

- **Feature:** `operational/health-readiness`
- **Iteration:** `M1-02`
- **Type:** Fix
- **Based on:** `../../../reviews/operational/health-readiness/M1-01-review.md`
- **Status:** Proposed / Approved / Executed

## Objective
Masalah yang harus diselesaikan.

## Findings to Address
Temuan terverifikasi dari review sebelumnya beserta rujukan file.

## In Scope
Perubahan yang diizinkan.

## Out of Scope / Constraints
Perubahan yang tidak diizinkan; perubahan struktur perlu approval.

## Acceptance Criteria
Kondisi konkret untuk menyatakan pekerjaan selesai.

## Verification Guidance
Pemeriksaan yang proporsional terhadap risiko perubahan.
```

`Based on` pada contoh di atas adalah referensi konseptual. Dalam file sebenarnya, gunakan **relative link Markdown yang valid** dari lokasi file tersebut; jangan menyalin path contoh tanpa memeriksanya.

## 6. Struktur minimum file review

```markdown
# M1-02 — Review

- **Feature:** `operational/health-readiness`
- **Iteration:** `M1-02`
- **Input:** `../../../inputs/operational/health-readiness/M1-02-fix-db-timeout.md`
- **Verdict:** PASS / FAIL / BLOCKED

## Scope Reviewed
Bagian yang benar-benar diperiksa.

## Actual Changes
Daftar file/komponen yang berubah dan ringkasan dampaknya.

## Verification Evidence
Command/check yang dijalankan beserta hasilnya.
Jika tidak dijalankan, tulis "Not run" dan alasan.

## Findings
Temuan bernomor dan referensi file/baris jika ada.

## Acceptance Criteria Assessment
Masing-masing: PASS / FAIL / NOT VERIFIED.

## Next Action
Tutup pekerjaan atau rincikan input perbaikan berikutnya.
```

Seperti contoh input, **periksa dan sesuaikan relative link** sebelum digunakan.

## 7. Prinsip review yang dapat dipercaya

- Cocokkan hasil eksekusi dengan **kode saat ini** dan acceptance criteria, bukan hanya ringkasan yang ditulis agent.
- Pisahkan **observasi**, **dugaan**, dan **rekomendasi**.
- Jangan mengarang hasil build/test atau mengklaim PASS runtime hanya dari audit statis.
- Catat keterbatasan akses (misalnya tidak tersedia database test) secara eksplisit.
- Review boleh menemukan pekerjaan tambahan, tetapi tidak otomatis memberi izin untuk memperluas scope.
- Tetapkan verdict pada **scope yang dinilai**, bukan pada keseluruhan project.

## 8. Aturan praktis menyelesaikan fitur

Pekerjaan dapat ditutup ketika review terakhir PASS, perubahan sesuai approval, tidak ada blocker wajib, dan dokumentasi terkait telah diselaraskan. Riwayat iterasi tetap disimpan sebagai referensi. Setelah ditutup, fitur baru atau perubahan besar sebaiknya menggunakan input baru dengan scope yang jelas.

## 9. Referensi

- [`../documentation/architecture.md`](../documentation/architecture.md) — batas tanggung jawab package dan prinsip arsitektur.
- [`../documentation/operations.md`](../documentation/operations.md) — kontrak health, readiness, startup, dan shutdown.
- [`../documentation/configuration.md`](../documentation/configuration.md) — konfigurasi aktual dan batasannya.
- [`../AGENTS.md`](../AGENTS.md) — instruksi kerja coding agent (English).
- [`../AGENTS.id.md`](../AGENTS.id.md) — padanan Bahasa Indonesia.
