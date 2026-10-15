# Desain: Tambah Banyak Bel (Gabung ke Profil)

- **Tanggal:** 2026-10-15
- **Status:** Disetujui per bagian (menunggu review spesifikasi)
- **Dasar:** [2026-10-15-bel-sekolah-design.md](2026-10-15-bel-sekolah-design.md) dan [2026-10-15-tampilan-zona-template-design.md](2026-10-15-tampilan-zona-template-design.md). Semua aturan di sana tetap berlaku kecuali diubah di sini.

## 1. Ringkasan

Fitur **"Tambah banyak"** di halaman Jadwal menambahkan banyak bel sekaligus ke profil yang sedang dibuka, dengan cara **menggabungkan** (bukan mengganti). Sumbernya tiga:

1. **Ketik/tempel**: tabel isian banyak baris atau teks tempelan dari Excel, diterapkan ke hari-hari yang dicentang.
2. **Template**: hasil pembuat jadwal bawaan, atau template buatan sendiri.
3. **File/profil lain**: file `.json` hasil ekspor, atau profil lain.

Sebelum disimpan selalu ada **pratinjau**. Pratinjau menampilkan bel baru, bel kembar (dilewati), dan bentrok. Untuk bentrok, pengguna memilih pertahankan yang lama atau timpa dengan yang baru, secara global maupun per baris.

## 2. Non-Tujuan

- Menghapus bel lama yang tidak ada di sumber. Ini mode gabung, bukan ganti; untuk mengganti sudah ada salin hari dan hapus.
- Memindahkan hari sumber ke hari lain, misalnya Senin di template menjadi Selasa di tujuan.
- Urungkan setelah digabung. Gantinya adalah pratinjau dan konfirmasi timpa.
- Menggabungkan ke beberapa profil sekaligus.

## 3. Aturan Gabung

### 3.1 Masukan

- Semua sumber diubah di browser menjadi **format file jadwal** (`formatjadwal.Data`, §5.1 spesifikasi tahap 2) sebelum dikirim ke server.
- Server memvalidasinya persis seperti impor: hari 1–7, jam `HH:MM`, tidak boleh ada dua bel di hari dan jam yang sama dalam masukan, dan maksimal `formatjadwal.MaksBel` (1000) bel.
- Filter "Ambil hari" dijalankan di browser. Server menerima data yang sudah difilter.

### 3.2 Jenis bel

- Nama jenis bel di masukan dicocokkan dengan jenis yang ada **tanpa membedakan huruf besar/kecil**, sama seperti impor.
- Nama yang tidak dikenal harus dipetakan dulu lewat `pemetaan` (`map[nama]PemetaanJenis`, bentuknya sama dengan impor). Pilihannya: ke jenis yang sudah ada (`jenisBelId`), atau buat jenis baru (`baru: true`). Jenis baru dibuat dengan warna dari masukan dan suara "Ding Dong" bawaan (atau suara pertama), seperti impor.
- Logika resolusi dan pembuatan jenis bel dipisah dari `Store.ImporJadwal` menjadi fungsi bersama. Impor dan gabung sama-sama memakainya.

### 3.3 Klasifikasi per slot

Untuk tiap bel masukan, cari bel di profil tujuan pada slot (hari, jam) yang sama:

| Status | Kondisi | Tindakan saat gabung |
|---|---|---|
| `baru` | Slot kosong | Ditambahkan |
| `kembar` | Slot terisi, jenis bel (setelah resolusi) **dan** keterangan (setelah `TrimSpace`, peka huruf besar/kecil) sama | Dilewati |
| `bentrok` | Slot terisi, jenis bel atau keterangannya berbeda | Dipertahankan, **kecuali** slotnya tercantum di `timpa`; kalau tercantum, jenis bel dan keterangan bel lama diganti |

- Saat diperiksa, jenis yang belum dipetakan dianggap berbeda dari semua jenis yang ada. Akibatnya slot yang terisi menjadi `bentrok`, bukan `kembar`.
- Bel lama yang slotnya tidak ada di masukan tidak disentuh.
- Klasifikasi berupa **fungsi murni** di paket baru `internal/gabung`, tanpa akses database.

### 3.4 Batas dan atomisitas

- Jumlah bel profil setelah digabung (lama + baru) tidak boleh melebihi `formatjadwal.MaksBel`, supaya profil tetap bisa diekspor. Pelanggaran → 400 dengan pesan yang jelas.
- `gabung` berjalan dalam **satu transaksi**. Klasifikasi dihitung ulang dari isi database saat itu, bukan dari pratinjau.
- Slot di `timpa` yang ternyata bukan `bentrok` (sudah kosong, sudah kembar, atau tidak ada di masukan) diabaikan tanpa error.
- Gagal di langkah mana pun berarti tidak ada yang berubah.
- Setelah berhasil, penjadwal memuat ulang data (`ubahData()`).
- Mengikuti aturan database satu koneksi: semua pembacaan dilakukan sebelum transaksi dibuka, atau di dalam transaksi lewat `tx`.

## 4. API (tambahan atas §7 spesifikasi tahap 2)

Keduanya wajib login, dan pengubah data mengikuti `cekOrigin` serta blokir restore seperti rute lain.

### `POST /api/profil/{id}/gabung/periksa`

Hanya membaca, tidak mengubah apa pun.

```json
// permintaan
{ "data": { ...formatjadwal.Data... }, "pemetaan": { "Nama Jenis": { "jenisBelId": 3, "baru": false } } }
// jawaban 200
{
  "jenisBelTakDikenal": ["Literasi"],
  "ringkasan": { "baru": 12, "kembar": 3, "bentrok": 2 },
  "baris": [
    { "hari": 1, "jam": "07:00", "status": "bentrok",
      "lama": { "jenisBel": "Masuk", "label": "Masuk" },
      "baru": { "jenisBel": "Upacara", "label": "Upacara" } },
    { "hari": 1, "jam": "07:15", "status": "baru", "lama": null,
      "baru": { "jenisBel": "Masuk", "label": "Sholat Dhuha" } }
  ]
}
```

- `pemetaan` boleh kosong atau tidak dikirim.
- `jenisBelTakDikenal` berisi nama-nama yang tidak cocok dengan jenis yang ada, **tanpa memandang** apakah sudah dipetakan. Dengan begitu UI tahu langkah pemetaan perlu ditampilkan.
- `baris` diurutkan menurut hari lalu jam. `jenisBel` pada `lama` adalah nama jenis yang ada di database. Pada `baru`, isinya nama dari masukan.
- 400 untuk data tidak valid, 404 untuk profil yang tidak ada.

### `POST /api/profil/{id}/gabung`

```json
// permintaan
{ "data": { ... }, "pemetaan": { ... }, "timpa": [ { "hari": 1, "jam": "07:00" } ] }
// jawaban 200
{ "ditambah": 12, "ditimpa": 1, "dilewati": 4 }
```

- `dilewati` = kembar + bentrok yang dipertahankan.
- 400 untuk data tidak valid, jenis yang belum dipetakan, tujuan pemetaan yang tidak ada, atau melebihi 1000 bel. 404 untuk profil yang tidak ada.

## 5. Tampilan (frontend)

### 5.1 Pintu masuk

- Tombol **"Tambah banyak"** di bar alat halaman Jadwal, di sebelah "+ Tambah bel", tersedia di tampilan Minggu maupun Hari.
- Tombol itu membuka `PanelGeser` lebar berjudul "Tambah banyak ke profil *‹nama›*". Profil sasarannya adalah profil yang sedang dipilih di halaman Jadwal.
- Ada tiga langkah: **Sumber → Pemetaan (bila perlu) → Pratinjau**.

### 5.2 Langkah Sumber

Ada tiga tab.

- **Ketik/tempel**
  - Tabel isian dengan kolom Jam, Keterangan, dan Jenis bel (pilihan dari jenis yang ada), plus tombol "+ baris" dan hapus per baris.
  - Ada kotak **"Tempel dari Excel/teks"**:
    - Satu bel per baris. Kolom dipisah TAB, `;`, atau `,`, dengan urutan `jam, keterangan, jenis bel`.
    - Jam boleh ditulis `7.15`, `7:15`, atau `07:15`, lalu dinormalkan menjadi `HH:MM`.
    - Kolom jenis bel boleh kosong; isinya diambil dari pilihan **jenis bawaan** di atas kotak. Nama jenis yang tidak dikenal dibiarkan dan ditangani langkah Pemetaan.
    - Baris kosong diabaikan. Baris yang tidak terbaca ditampilkan merah dengan alasannya dan tidak ikut.
    - Hasil tempelan ditambahkan ke tabel isian supaya bisa diperbaiki.
  - **Hari tujuan** berupa centang Sen–Min, dengan Sen–Jum dicentang sebagai bawaan. Setiap baris dibuat untuk setiap hari yang dicentang.
  - Kalau ada baris dengan jam yang sama, browser menampilkan galat sebelum Periksa.
- **Template**: pilih "Pembuat jadwal" atau "Template saya".
  - Form pembuat jadwal diambil dari halaman Template menjadi komponen yang dipakai bersama. Hasilnya dari `POST /api/template/buat`.
  - Template tersimpan diambil datanya dari `GET /api/template/{id}`.
- **File/profil lain**: unggah `.json` (dibaca di browser, dibungkus try), atau pilih profil lain (tanpa profil sasaran). Profil lain diambil lewat `GET /api/profil/{id}/ekspor`.
- Untuk tab Template dan File/profil lain ada centang **"Ambil hari"** yang hanya menampilkan hari yang memang ada isinya di sumber. Semuanya dicentang sebagai bawaan.
- Kaki panel menampilkan ringkasan sumber ("18 bel dari 5 hari") dan tombol **Periksa**. Tombol ini nonaktif kalau sumbernya kosong.

### 5.3 Langkah Pemetaan

- Hanya muncul kalau `jenisBelTakDikenal` tidak kosong.
- Bagian pemetaan dari `AlurImpor` dipisah menjadi komponen bersama (`PemetaanJenis.svelte`) dan dipakai oleh impor maupun gabung.
- Setelah semua dipetakan, `periksa` dipanggil ulang dengan `pemetaan`, lalu lanjut ke Pratinjau.

### 5.4 Langkah Pratinjau

- **Kartu ringkasan**: baru, kembar, bentrok.
- **Pilihan global untuk bentrok**: ◉ Pertahankan yang lama (bawaan) / ○ Timpa dengan yang baru. Mengubah pilihan global akan menyetel ulang semua pilihan per baris.
- **Tabel per hari** dengan kolom Jam, Status (lencana: baru hijau, kembar abu-abu, bentrok kuning), Lama, Baru, dan pilihan per baris (hanya untuk bentrok). Baris kembar disembunyikan sebagai bawaan, dengan saklar "Tampilkan kembar". Di HP tabelnya menjadi kartu.
- Tombol **"Gabungkan (N perubahan)"**, dengan N = baru + bentrok yang ditimpa. Tombol nonaktif kalau N = 0.
- Kalau ada yang ditimpa, muncul konfirmasi singkat ("2 bel lama akan diganti") sebelum dikirim.

### 5.5 Selesai

Panel tertutup, muncul notifikasi "12 ditambah, 2 ditimpa, 3 dilewati", lalu data halaman Jadwal dimuat ulang. Kalau gagal, panel tetap terbuka di langkah Pratinjau dan pesan dari server ditampilkan.

## 6. Pengujian

- **Go, `internal/gabung`** (tes tabel):
  - baru
  - kembar
  - bentrok karena beda label
  - bentrok karena beda jenis
  - nama jenis yang beda huruf besar/kecil tetap kembar
  - jenis belum dipetakan menjadi bentrok
  - `timpa` untuk slot yang bukan bentrok diabaikan
  - urutan `baris`
- **Go, store**:
  - gabung menambah dan menimpa dengan benar
  - transaksi gagal berarti tidak ada perubahan
  - batas 1000 bel setelah digabung
  - jenis baru dibuat dengan suara bawaan
  - tes impor yang sudah ada tetap hijau setelah pemetaan dipisah
- **Go, API**:
  - kedua endpoint bisa dipakai
  - 400 untuk data tidak valid dan jenis yang belum dipetakan
  - 404 untuk profil yang tidak ada
  - wajib login
- **Vitest**:
  - pengurai tempelan: TAB, `;`, `,`, `7.15`, kolom jenis kosong, baris rusak
  - perluasan baris × hari
  - deteksi jam ganda
- **Playwright**:
  - alur ketik/tempel sampai selesai: bentrok dipertahankan lalu satu ditimpa, hasilnya terlihat di tampilan Minggu
  - alur dari profil lain dengan filter hari
  - alur jenis tak dikenal yang memunculkan langkah pemetaan
  - tampilan HP (pratinjau sebagai kartu)
