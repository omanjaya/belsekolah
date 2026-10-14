# Desain Tahap 2: Tampilan Profesional, Zona Waktu, Cadangan & Template Jadwal

- **Tanggal:** 2026-10-15
- **Status:** Disetujui per bagian (menunggu review spesifikasi)
- **Dasar:** [2026-10-15-bel-sekolah-design.md](2026-10-15-bel-sekolah-design.md). Semua aturan di sana tetap berlaku kecuali diubah di sini.

## 1. Ringkasan

Empat perubahan atas versi v0.1.0:

1. **Tampilan admin layar penuh** — sidebar gelap, bar atas dengan jam & zona, isi selebar layar, form di panel geser kanan.
2. **Zona waktu WIB/WITA/WIT** dapat dipilih dari web.
3. **Cadangan jadwal** — ekspor/impor jadwal satu profil sebagai file JSON, dan duplikat profil.
4. **Template jadwal** — pembuat jadwal bawaan (SD/SMP/SMA-SMK, Normal/Ramadan/Ujian, 5 atau 6 hari) dan template buatan sendiri.

## 2. Non-Tujuan

- Mode gelap.
- Mengubah zona waktu OS (`timedatectl`) dari web.
- Ekspor/impor suara, jenis bel lengkap, libur, atau periode lewat file jadwal (itu tugas backup penuh `.zip`).
- Impor yang menimpa profil yang sudah ada.

## 3. Tampilan (frontend)

### 3.1 Bingkai aplikasi

- **Sidebar kiri gelap** (`slate-900`, ±240px) di layar `md` ke atas. Bisa diciutkan menjadi ikon saja (±64px); pilihan disimpan di `localStorage` (dibungkus try/catch). Menu berurutan: Beranda, Jadwal, Profil & Periode, **Template**, Libur, Jenis Bel, Pustaka Suara, Riwayat, Pengaturan.
- **Bar atas** (putih, menempel di atas saat gulir): judul halaman; pil status (`● Berjalan` hijau, atau kuning/merah bila ada peringatan dari `/api/status`); jam berjalan `HH:MM:SS <WIB|WITA|WIT>`; tombol Keluar.
- **Jam bar atas memakai jam server**, bukan jam browser: offset dihitung dari `sekarang` pada `/api/status` dan disegarkan setiap kali status dimuat, lalu berdetik di browser. Singkatan zona dari `zonaWaktu` di status.
- **Area isi tanpa batas lebar** (`max-w-4xl` dihapus). Padding 24px di laptop, 16px di HP. Tata letak memakai grid yang melebar 1→2→3/4 kolom sesuai lebar layar.
- **HP (< `md`)**: sidebar menjadi laci dari kiri lewat tombol ☰ di bar atas; bar bawah (Beranda, Jadwal, Suara, Riwayat, Lainnya) tetap ada.
- Latar isi `slate-100`, kartu putih bergaris `slate-200`, warna utama biru (`blue-700`). Tidak ada mode gelap.

### 3.2 Pola interaksi

- **Data berbentuk daftar** ditampilkan sebagai **tabel lebar** di laptop (riwayat, suara, libur, periode, jenis bel, template) dan sebagai daftar kartu di HP.
- **Form tambah/edit** dibuka di **panel geser kanan** (lebar ±440px, layar penuh di HP), bukan modal tengah. Panel: judul, isi form, tombol Simpan/Batal menempel di bawah; `Esc` dan klik latar menutup; fokus dikunci di dalam panel dan kembali ke pemicu saat ditutup.
- **Konfirmasi singkat** (hapus, ganti zona, restore) tetap memakai dialog kecil di tengah.
- Komponen bersama baru: `PanelGeser`, `Tabel` (header lengket, baris kosong, pemuatan), `KartuStatistik`, `BarAlat` (baris filter/aksi di atas tabel).

### 3.3 Halaman

| Halaman | Isi |
|---|---|
| **Beranda** | Grid: kartu **Bel berikutnya** (jam, label, hitung mundur, profil aktif) lebar 2 kolom; kartu **Bel manual** (tombol per jenis bel, Darurat tahan 2 detik, Hentikan suara); kartu **Jadwal hari ini** (tabel dengan status per bel) lebar 2 kolom; kartu **Status sistem** (speaker/mpv, sinkron jam, disk, backup terakhir). Peringatan tampil sebagai pita di atas grid. |
| **Jadwal** | Bar alat: pilih profil, pengalih **Minggu \| Hari**, Salin hari…, Ekspor, + Tambah bel. **Minggu** (bawaan): satu kolom per hari sekolah (Senin–Jumat; Sabtu/Minggu tampil hanya bila berisi bel), tiap bel kartu kecil berwarna sesuai jenis bel; klik kartu → panel edit. **Hari**: daftar hari di kiri (nama + jumlah bel), tabel di kanan (Jam, Keterangan, Jenis bel, Suara, aksi). Pilihan tampilan disimpan di `localStorage`. |
| **Profil & Periode** | Dua kartu berdampingan: tabel profil (nama, jumlah bel, default, aksi **Duplikat**, **Ekspor**, **Simpan sebagai template**, Ubah, Hapus) dengan tombol **Impor jadwal**; tabel periode khusus. |
| **Template** | Baru, lihat §6.3. |
| **Libur** | Tabel rentang tanggal + panel geser. |
| **Jenis Bel** | Tabel (warna, nama, urutan suara ringkas, Tes); edit urutan suara di panel geser. |
| **Pustaka Suara** | Tab Bawaan/Upload/TTS di atas satu tabel (nama, sumber, durasi, putar di browser, Tes, aksi). |
| **Riwayat** | Bar alat filter (tanggal, status) + tabel lebar. |
| **Pengaturan** | Grid kartu: Volume, **Zona waktu** (§4), Ganti password, Info sistem, Backup & restore. |

Halaman Login dan Ganti Password Awal tetap kartu kecil di tengah layar (satu-satunya pengecualian).

## 4. Zona Waktu

- Nilai yang diterima: `WIB` → `Asia/Jakarta`, `WITA` → `Asia/Makassar`, `WIT` → `Asia/Jayapura`.
- Disimpan di tabel `pengaturan` dengan kunci `zona_waktu` (nilai: nama IANA).
- **Urutan prioritas saat start:** `pengaturan.zona_waktu` bila ada dan valid → flag `-tz` → `Asia/Jakarta`. Karena itu database dibuka **sebelum** `time.Local` diatur; `-reset-password` tidak bergantung pada zona.
- Nilai di database yang tidak dikenali diabaikan dengan log peringatan (pakai `-tz`), bukan menghentikan aplikasi.
- **Mengubah zona** lewat `PUT /api/pengaturan` dengan `zonaWaktu` berbeda dari yang aktif: simpan, balas `200 {"mulaiUlang": true}`, lalu aplikasi keluar dengan kode 3 (mekanisme yang sama dengan restore). Frontend menampilkan "Menerapkan zona waktu…", menunggu `boot` pada `/api/sesi` berubah, lalu memuat ulang halaman.
- Dialog konfirmasi sebelum menyimpan: "Aplikasi akan dimulai ulang (±2 detik). Bel yang jatuh tepat saat itu bisa terlewat."
- Selama restore sedang disiapkan, perubahan zona ditolak 503 seperti perubahan data lainnya.
- Zona ikut di backup penuh (tabel `pengaturan`); restore sudah memulai ulang aplikasi sehingga zona baru langsung berlaku.
- `GET /api/status` dan `GET /api/sistem` menyertakan `zonaWaktu` berupa singkatan (`WIB|WITA|WIT`, atau nama IANA bila di luar ketiganya) dan `/api/sistem` juga nama IANA.
- `install.sh --tz` tetap mengatur zona OS dan nilai awal `-tz`; dokumentasi menjelaskan bahwa pilihan di web lebih diutamakan.

## 5. Cadangan Jadwal

### 5.1 Format file jadwal

Satu format untuk ekspor profil, template buatan sendiri, dan hasil pembuat jadwal bawaan:

```json
{
  "format": "belsekolah-jadwal",
  "versi": 1,
  "nama": "Normal",
  "jenisBel": [{ "nama": "Masuk", "warna": "#2563eb" }],
  "jadwal": [{ "hari": 1, "jam": "07:00", "jenisBel": "Masuk", "label": "Masuk" }]
}
```

- `hari` 1=Senin … 7=Minggu; `jam` `HH:MM`; `jenisBel` merujuk `jenisBel[].nama`.
- Validasi impor: `format` dan `versi` cocok; maksimal 1000 bel; ukuran file maksimal 256 KB; setiap bel lolos validasi yang sama dengan `POST /jadwal`; tidak ada (hari, jam) ganda; setiap `jenisBel` yang dirujuk ada di daftar `jenisBel`. Pesan error dalam bahasa Indonesia menyebut baris/bel yang salah.
- Nama file ekspor: `jadwal-<nama-profil-dislug>.json`.

### 5.2 Impor

Dua langkah:

1. `POST /api/impor/periksa` dengan isi file → `{nama, jumlahBel, jenisBelTakDikenal: [nama], namaTersedia}`. `namaTersedia` = nama profil yang belum dipakai (menambah " (impor)", lalu " (impor 2)", dst. bila bentrok).
2. `POST /api/impor` `{data, nama, pemetaan}` — `pemetaan` memetakan setiap nama di `jenisBelTakDikenal` ke `{"jenisBelId": n}` atau `{"baru": true}`. `baru` membuat jenis bel dengan nama & warna dari file dan urutan suara satu audio bawaan "Ding Dong". Jenis bel yang namanya sudah ada dipakai langsung (cocok tanpa membedakan huruf besar/kecil).
   - Semua dalam satu transaksi: profil baru + jenis bel baru + bel. Gagal → tidak ada yang tersimpan.
   - Nama profil bentrok → 409. Pemetaan kurang → 400 menyebut nama yang belum dipetakan.
   - Balas `201 {profilId}`.

Frontend: tombol **Impor jadwal** → pilih file → layar periksa (nama profil yang bisa diubah, jumlah bel, tabel pemetaan jenis bel bila perlu) → Impor → pindah ke Jadwal dengan profil baru terpilih.

### 5.3 Duplikat profil

`POST /api/profil/{id}/duplikat` `{nama}` → salin seluruh bel profil (jenis bel sama) dalam satu transaksi, balas `201 {profilId}`. Nama bawaan di form: "<nama> (salinan)".

## 6. Template Jadwal

### 6.1 Template buatan sendiri

- Tabel baru:
  ```sql
  CREATE TABLE IF NOT EXISTS template (
    id     INTEGER PRIMARY KEY,
    nama   TEXT NOT NULL UNIQUE,
    data   TEXT NOT NULL,   -- JSON format §5.1
    dibuat TEXT NOT NULL
  );
  ```
- `POST /api/template` `{nama, profilId}` (dari profil) atau `{nama, data}` (dari file) → 201. Nama ganda → 409.
- `GET /api/template` → daftar `{id, nama, jumlahBel, hari: [1..7 yang berisi], dibuat}`; `GET /api/template/{id}` → termasuk `data`; `PUT /api/template/{id}` `{nama}`; `DELETE /api/template/{id}`; `GET /api/template/{id}/ekspor` → unduh file.
- **Pakai template** = alur impor §5.2 dengan `data` dari template (frontend memanggil periksa lalu impor).
- Template tidak merujuk ID apa pun, jadi menghapus jenis bel/profil tidak merusaknya.

### 6.2 Pembuat jadwal bawaan

Logika di Go (`internal/template`), murni dan teruji tabel. `POST /api/template/buat` `{parameter}` → `200` berisi data format §5.1 (tidak menyimpan apa pun). Frontend memakainya untuk pratinjau, lalu menyimpan lewat alur impor.

**Parameter:**

| Field | Arti | Batas |
|---|---|---|
| `jenjang` | `SD` / `SMP` / `SMA` (hanya untuk nama & nilai awal di form) | wajib |
| `varian` | `normal` / `ramadan` / `ujian` | wajib |
| `hariSekolah` | 5 (Sen–Jum) atau 6 (Sen–Sab) | 5/6 |
| `jamMasuk` | `HH:MM` | wajib |
| `lamaJP` | menit per jam pelajaran | 10–90 |
| `jumlahJP` | JP Senin–Kamis | 1–12 |
| `jumlahJPJumat` | JP hari Jumat | 1–12 |
| `jumlahJPSabtu` | JP hari Sabtu (bila 6 hari) | 1–12 |
| `istirahat` | daftar `{setelahJP, menit}` | 0–2 item, menit 5–60, `setelahJP` naik & < jumlah JP hari itu (dilewati di hari yang JP-nya lebih sedikit) |
| `upacaraSenin` | `{aktif, menit}` | menit 10–90 |
| `kegiatanJumat` | `{aktif, menit, label}` | menit 10–90, label bawaan "Kegiatan Jumat" |
| `ujian` | `{jamMulai, jumlahSesi, lamaSesi, jeda}` (hanya varian ujian) | sesi 1–4, lama 30–180, jeda 5–60 |

**Nilai awal di form** (frontend; dapat diubah):

| | SD | SMP | SMA/SMK |
|---|---|---|---|
| `lamaJP` normal / Ramadan | 35 / 25 | 40 / 30 | 45 / 30 |
| `jumlahJP` / Jumat / Sabtu | 6 / 4 / 4 | 7 / 5 / 5 | 8 / 5 / 5 |
| istirahat | setelah JP 3 (15 mnt), JP 5 (15 mnt) | setelah JP 3 (15), JP 6 (15) | setelah JP 4 (15), JP 6 (15) |

Bersama: `jamMasuk` 07:00 (Ramadan 07:30), `hariSekolah` 5, upacara Senin aktif 40 menit, kegiatan Jumat nonaktif. Ujian: mulai 07:30, 2 sesi × 120 menit, jeda 30 menit.

**Aturan pembentukan (varian normal & ramadan)**, untuk setiap hari sekolah:

1. Bel **Masuk** di `jamMasuk`, label "Masuk" (Senin dengan upacara: "Upacara"; Jumat dengan kegiatan: label kegiatan).
2. Waktu mulai JP 1 = `jamMasuk` (+ menit upacara/kegiatan bila aktif di hari itu). Bila ada upacara/kegiatan, bel **Ganti Pelajaran** label "Jam ke-1" di waktu mulai JP 1.
3. Setiap JP berikutnya mulai tepat setelah JP sebelumnya. Di awal JP k (k ≥ 2): bel **Ganti Pelajaran** label "Jam ke-k", **kecuali** bila tepat sebelumnya ada istirahat.
4. Istirahat setelah JP n: bel **Istirahat** label "Istirahat" di akhir JP n; bel **Masuk** label "Masuk setelah istirahat" di akhir istirahat (sekaligus awal JP n+1).
5. Akhir JP terakhir: bel **Pulang** label "Pulang". Istirahat yang `setelahJP` ≥ jumlah JP hari itu dilewati.

**Varian ujian**, untuk setiap hari sekolah: setiap sesi i — bel **Masuk** "Sesi i dimulai" di awal sesi; di akhir sesi bel **Istirahat** "Sesi i selesai", kecuali sesi terakhir bel **Pulang** "Ujian selesai". Sesi berikutnya mulai setelah `jeda`.

- Jam yang melewati 23:59 → 400 "jadwal melewati tengah malam".
- `jenisBel` di data hasil: Masuk, Ganti Pelajaran, Istirahat, Pulang dengan warna bawaan (§`bawaan`). Saat dipakai, nama yang sudah ada dipakai langsung; yang hilang diperlakukan seperti `jenisBelTakDikenal` di §5.2 (layar pemetaan, bawaan "buat baru").
- Nama profil bawaan: `"<Jenjang> <Normal|Ramadan|Ujian> <5|6> hari"`, mis. "SMP Normal 5 hari".

### 6.3 Halaman Template

- Bagian atas: **Pembuat jadwal** — form parameter di kiri (pilihan jenjang & varian mengisi ulang nilai awal), **pratinjau mingguan** di kanan (diperbarui otomatis ±300 ms setelah isian berubah, memakai komponen tampilan Minggu dari Jadwal). Tombol **Buat profil** → alur impor (nama bisa diubah) → pindah ke Jadwal.
- Bagian bawah: tabel **Template saya** (nama, jumlah bel, hari, dibuat, aksi Pakai / Ekspor / Ganti nama / Hapus) dengan tombol **Impor template** (dari file §5.1).

## 7. Perubahan Kontrak API (tambahan atas §14 spesifikasi dasar)

| Metode & Path | Keterangan |
|---|---|
| `GET/PUT /pengaturan` | Tambah `zonaWaktu` (`WIB`/`WITA`/`WIT`). PUT yang mengubah zona membalas `{mulaiUlang: true}` lalu memulai ulang aplikasi. |
| `GET /status`, `GET /sistem` | Tambah/ubah `zonaWaktu` sesuai §4. |
| `GET /profil` | Tambah `jumlahBel` per profil. |
| `POST /profil/{id}/duplikat` `{nama}` | §5.3 |
| `GET /profil/{id}/ekspor` | Unduh file jadwal §5.1 (`Content-Disposition: attachment`). |
| `POST /impor/periksa`, `POST /impor` | §5.2 |
| `GET/POST /template`, `GET/PUT/DELETE /template/{id}`, `GET /template/{id}/ekspor` | §6.1 |
| `POST /template/buat` | §6.2, tanpa efek samping |

Semua endpoint baru wajib login, mengikuti aturan Origin/CSRF, dan ditolak 503 selama restore disiapkan (kecuali yang hanya membaca). Perubahan data memicu muat ulang penjadwal.

## 8. Pengujian

- **Go `internal/template`**: tes tabel pembentukan — tiap jenjang × varian, 5/6 hari, upacara/kegiatan aktif/nonaktif, istirahat dilewati di hari pendek, ujian, batas tengah malam, validasi parameter.
- **Go format jadwal**: ekspor → impor menghasilkan bel identik (round-trip); validasi file rusak/versi salah/terlalu besar/bel ganda/rujukan jenis bel hilang; impor transaksional (gagal di tengah → tidak ada sisa).
- **Go API**: duplikat, ekspor, periksa/impor dengan pemetaan `jenisBelId` dan `baru`, bentrok nama, CRUD template, zona waktu (validasi, `mulaiUlang`, prioritas pengaturan > flag, nilai rusak diabaikan).
- **Vitest**: perhitungan offset jam server dan format jam + singkatan zona.
- **Playwright** (Docker, port 1808x): semua tes lama disesuaikan dengan tata letak baru; tambahan — sidebar ciut/laci HP, panel geser (Esc, fokus), jadwal Minggu/Hari, impor/ekspor jadwal, duplikat profil, pembuat template → buat profil, simpan & pakai template sendiri, ganti zona (menunggu restart, jam bar atas berganti singkatan), tidak ada gulir horizontal di 375px dan isi memenuhi lebar di 1440px.
- **e2e Go**: ganti zona lewat API → aplikasi mulai ulang → `/api/status` melaporkan zona baru.
