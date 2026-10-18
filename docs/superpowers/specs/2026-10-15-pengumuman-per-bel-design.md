# Desain: Pengumuman per Bel (bel terhubung dengan jamnya)

- **Tanggal:** 2026-10-15
- **Status:** Disetujui (user: "gas tulis spec lanjut sampai selesai")
- **Dasar:** [2026-10-15-bel-sekolah-design.md](2026-10-15-bel-sekolah-design.md), [2026-10-15-tampilan-zona-template-design.md](2026-10-15-tampilan-zona-template-design.md), [2026-10-15-tambah-banyak-gabung-design.md](2026-10-15-tambah-banyak-gabung-design.md). Semua aturan di sana tetap berlaku kecuali diubah di sini.

## 1. Ringkasan

Setiap baris jadwal boleh punya satu **pengumuman**: satu suara dari Pustaka Suara yang diputar **setelah** suara jenis bel. Contohnya baris "Jam ke-3" berbunyi Ding Dong → "Jam ke-3 dimulai".

Pengumuman **diisi otomatis** dengan mencocokkan keterangan baris dengan nama suara. Pengisian otomatis ini terjadi di form, saat jadwal dibuat dari template, Tambah banyak, atau impor, dan lewat tombol "Isi pengumuman otomatis" untuk jadwal yang sudah ada. Hasil pengisian otomatis selalu bisa diganti manual.

## 2. Non-Tujuan

- Lebih dari satu suara pengumuman per baris.
- TTS otomatis dari keterangan.
- Pengumuman untuk bel manual atau darurat. Bel manual tetap hanya memutar suara jenis bel.
- Mengubah aturan klasifikasi gabung: kembar tetap ditentukan oleh jenis bel dan keterangan saja.

## 3. Data

- Kolom baru `jadwal.pengumuman_audio_id INTEGER NULL REFERENCES audio(id) ON DELETE RESTRICT`.
- `NULL` berarti tanpa pengumuman.
- **Migrasi:** saat `Store.Open`, setelah skema dijalankan, periksa `PRAGMA table_info(jadwal)`. Bila kolom belum ada, jalankan `ALTER TABLE jadwal ADD COLUMN pengumuman_audio_id INTEGER REFERENCES audio(id) ON DELETE RESTRICT`. Untuk database baru, `schema.sql` sudah memuat kolom ini. Migrasi harus idempoten, sehingga aman bila `Open` dijalankan berkali-kali.
- `schedule.Jadwal` dan `schedule.Bel` mendapat `PengumumanID int64`, dengan 0 berarti tidak ada.
- **Validasi:** `PengumumanID` yang tidak 0 harus menunjuk suara yang ada. Bila tidak, muncul error validasi "suara pengumuman tidak ditemukan".
- **Hapus suara:**
  - Suara yang dipakai sebagai pengumuman ditolak (409) dengan pesan "suara masih dipakai oleh N bel di jadwal".
  - Suara yang dipakai jenis bel ditolak dengan pesan "suara masih dipakai oleh jenis bel".
- Kolom pengumuman ikut terbawa oleh:
  - salin hari
  - duplikat profil
  - Tambah banyak, baik untuk baris baru maupun yang ditimpa
  - backup dan restore (berkas database)

## 4. Bunyi

- Saat bel jadwal berbunyi, urutan berkas yang diputar adalah: semua suara jenis bel, lalu suara pengumuman bila ada.
- Bila suara pengumuman gagal dimuat, bel tetap berbunyi tanpa pengumuman dan error dicatat di log.
- Riwayat tidak berubah.

## 5. Pencocokan otomatis (paket `internal/pengumuman`, fungsi murni)

`Cocokkan(keterangan string, suara []Kandidat) (id int64, ok bool)`, dengan `Kandidat{ID int64; Nama string}`.

**Normalisasi** (`normal(s)`):
- huruf kecil
- `-`, `_`, `.`, `,`, `:`, `(`, `)`, `/` diganti spasi
- spasi berturut-turut dirapatkan, lalu di-trim
- contoh: `"Jam ke-3"` → `"jam ke 3"`, `"Akhir pelajaran (pulang)"` → `"akhir pelajaran pulang"`

**Padanan bawaan.** Bila `normal(keterangan)` sama persis dengan salah satu kunci di bawah, gunakan nilainya sebagai keterangan pencarian:

| Kunci | Nilai pencarian |
|---|---|
| `pulang` | `akhir pelajaran` |
| `upacara` | `upacara bendera` |
| `masuk` | `bel masuk sekolah` |
| `ujian selesai` | `ujian waktu telah habis` |

**Tingkat kecocokan.** `k` adalah keterangan pencarian yang sudah dinormalisasi, `n` adalah nama suara yang sudah dinormalisasi:

1. `n == k`
2. `n == k + " dimulai"`
3. `n` diawali `k + " "`

Kecocokan berbasis awalan harus berhenti di batas kata, sehingga "jam ke 1" tidak cocok dengan "jam ke 10 dimulai".

**Aturan pemilihan:**
- Ambil tingkat terbaik yang punya kandidat.
- Bila di tingkat itu ada tepat satu kandidat, hasilnya `(id, true)`.
- Bila tidak ada kandidat sama sekali, atau ada lebih dari satu kandidat di tingkat terbaik, hasilnya `(0, false)` (jangan menebak).
- Keterangan kosong selalu menghasilkan `(0, false)`.

**Kandidat:** semua suara di pustaka (bawaan, upload, tts).

## 6. Pengisian otomatis

- **Form tambah/ubah bel:**
  - Saat keterangan berubah (debounce 300 ms) dan pengguna belum memilih pengumuman secara manual di panel itu, form memanggil `GET /api/pengumuman/cocok?keterangan=...` lalu mengisi pilihan dengan hasilnya.
  - Pilihan manual, termasuk "Tanpa pengumuman", tidak pernah ditimpa oleh pengisian otomatis.
  - Panel ubah untuk baris yang sudah punya pengumuman tidak mengisi otomatis.
  - Panel ubah tidak mengisi otomatis kecuali keterangan diubah di panel itu.
- **Tambah banyak dan impor:**
  - Bel masukan yang tidak membawa `pengumuman` diisi dengan `Cocokkan(label)` saat disimpan, berlaku untuk baris baru maupun yang ditimpa.
  - Bel masukan yang membawa `pengumuman` (nama suara dari file) dicocokkan dengan nama suara di pustaka tanpa membedakan huruf besar/kecil dan dengan `TrimSpace`. Bila tidak ditemukan, kolom dikosongkan.
  - `periksa` (impor maupun gabung) mengembalikan `pengumumanTakDitemukan: string[]` berisi nama-nama unik yang tidak ditemukan. UI menampilkannya sebagai peringatan (bukan galat).
- **Duplikat profil dan salin hari:** menyalin kolom apa adanya.
- **Jadwal yang sudah ada:**
  - Tombol "Isi pengumuman otomatis" di halaman Jadwal membuka panel berisi saran untuk profil terpilih: baris yang **belum** punya pengumuman dan punya kecocokan.
  - Setiap saran berupa baris (hari, jam, keterangan → nama suara) dengan centang yang aktif sebagai bawaan.
  - "Terapkan (N)" hanya menyimpan baris yang dicentang. Baris yang sudah punya pengumuman tidak pernah ditimpa.

## 7. Format file jadwal

- `formatjadwal.Bel` mendapat `Pengumuman string \`json:"pengumuman,omitempty"\``, berisi nama suara, panjang maksimal 100 karakter. Versi file tetap 1, dan aplikasi lama mengabaikan field ini.
- Ekspor profil dan template tersimpan menulis nama suara pengumuman.
- Pembuat jadwal tidak mengisi `pengumuman`, sehingga pengisian otomatis yang dipakai saat disimpan.

## 8. API

- `jadwalDTO` mendapat `pengumumanId: number | null`, dipakai saat menerima maupun mengirim. Bila field tidak dikirim pada permintaan buat atau ubah, nilainya `null`.
- `GET /api/pengumuman/cocok?keterangan=<teks>` mengembalikan `200 {"suara": {"id": 12, "nama": "Jam ke-3 dimulai"} | null}`.
- `GET /api/profil/{id}/pengumuman/saran` mengembalikan `200 [{"jadwalId", "hari", "jam", "label", "suara": {"id", "nama"}}]`, diurutkan menurut hari lalu jam. Profil yang tidak ada → 404.
- `POST /api/profil/{id}/pengumuman/terapkan` dengan body `{"isi": [{"jadwalId": 1, "audioId": 12}]}` mengembalikan `200 {"diisi": N}`.
  - Hanya baris milik profil itu yang **masih kosong** pengumumannya yang diisi. Selainnya dilewati tanpa error.
  - Satu transaksi.
  - Penjadwal dimuat ulang.
- `belHariIniDTO` (di `/api/status`) mendapat `pengumuman: {"id", "nama"} | null`.
- Respons periksa impor dan gabung mendapat `pengumumanTakDitemukan: string[]`.
- Semua rute baru didaftarkan lewat `rute(...)`, jadi wajib login, melewati cekOrigin, dan ikut terblokir saat restore.

## 9. Tampilan

- **Form bel** (panel geser di Jadwal):
  - Medan "Pengumuman" berupa komponen `PilihSuara`: kotak cari, daftar suara yang tersaring, opsi "Tanpa pengumuman", dan tombol ▶ untuk mencoba.
  - Di bawahnya ada keterangan kecil "Diisi otomatis dari keterangan" bila nilai berasal dari pengisian otomatis.
- **Tampilan Hari:** kolom "Pengumuman" berisi nama suara atau "—". Di HP tabel bergulir menyamping seperti kolom lain.
- **Tampilan Minggu:** kartu bel yang punya pengumuman menampilkan ikon speaker kecil dengan `aria-label` "dengan pengumuman ‹nama›".
- **Beranda:** bel berikutnya menampilkan "+ ‹nama pengumuman›" bila ada.
- **Bar alat Jadwal:** tombol "Isi pengumuman otomatis" membuka `PanelGeser` saran (§6). Bila tidak ada saran, muncul pesan "Semua bel sudah punya pengumuman atau tidak ada suara yang cocok".
- **Alur impor dan Tambah banyak:** peringatan kuning "Pengumuman tidak ditemukan di pustaka: A, B — dikosongkan" bila `pengumumanTakDitemukan` tidak kosong.

## 10. Pengujian

- **Go `internal/pengumuman`:**
  - normalisasi
  - tiga tingkat kecocokan
  - batas kata (jam ke 1 vs jam ke 10)
  - padanan bawaan
  - ambigu menghasilkan tidak ada
  - keterangan kosong
- **Go store:**
  - migrasi kolom pada database lama (buat tabel tanpa kolom, lalu `Open`) bersifat idempoten
  - CRUD membawa `PengumumanID`
  - pengumuman yang tidak ada → galat validasi
  - hapus suara yang dipakai → `ErrDipakai` dengan jumlah bel
  - salin hari dan duplikat profil membawa kolom
  - gabung dan impor: isi otomatis, nama dari file, nama yang tidak ditemukan
  - ekspor menulis nama
  - saran dan terapkan tidak menimpa baris yang sudah terisi
- **Go scheduler:** berkas yang diputar adalah urutan jenis bel lalu pengumuman. Pengumuman yang gagal dimuat tidak menggagalkan bunyi bel.
- **Go API:** semua endpoint baru, termasuk 401 dan 404, `pengumumanId` di jadwal, dan status.
- **Vitest:** logika murni baru di frontend, bila ada.
- **Playwright:**
  - tambah bel "Jam ke-3": pengumuman terisi "Jam ke-3 dimulai", tersimpan, dan tampil di tabel
  - "Isi pengumuman otomatis" untuk profil berisi beberapa bel tanpa pengumuman
  - hapus suara yang dipakai ditolak
  - suara uji diunggah lewat API di tes, sebagai berkas WAV kecil yang dibuat di tes
