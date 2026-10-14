# Desain Aplikasi Bel Sekolah

- **Tanggal:** 2026-10-15
- **Status:** Disetujui (menunggu review spesifikasi)

## 1. Ringkasan

Aplikasi bel sekolah otomatis yang berjalan di sebuah **mini PC** (Debian, tanpa monitor) yang tersambung ke **amplifier + speaker/TOA** sekolah lewat jack audio. Admin mengatur semuanya lewat **browser** (HP/laptop) di jaringan lokal sekolah. Aplikasi tidak membutuhkan internet untuk membunyikan bel.

Seluruh aplikasi (web server, UI, penjadwal, pemutar audio, database) dikemas dalam **satu binary Go** dengan frontend Svelte yang di-embed.

## 2. Tujuan & Non-Tujuan

### Tujuan

- Bel berbunyi tepat waktu sesuai jadwal, andal, tanpa ketergantungan internet.
- Jadwal per hari (Senin–Minggu).
- Beberapa **profil** jadwal (mis. Normal, Ujian, Ramadan) dengan satu profil default.
- **Periode khusus**: rentang tanggal yang otomatis memakai profil tertentu.
- **Hari libur**: tanggal/rentang tanggal tanpa bel.
- **Bel manual & darurat** dari browser.
- **Jenis bel** dengan suara berbeda; tiap jenis bel adalah urutan satu atau lebih audio (bel gabungan).
- Sumber audio: **upload** (MP3/WAV), **bawaan** aplikasi, dan **TTS** bahasa Indonesia.
- Satu akun admin dengan login.

### Non-Tujuan (versi pertama)

- Akses dari luar jaringan sekolah / internet.
- Banyak pengguna atau peran.
- Banyak zona speaker / banyak gedung terpisah.
- Pemicu bel listrik/sirene via relay.
- Aplikasi mobile native.
- Tes E2E otomatis frontend.

## 3. Lingkungan & Deployment

| Hal | Keputusan |
|---|---|
| Perangkat | Mini PC x86 (mis. Lenovo ThinkCentre Tiny / Dell OptiPlex Micro bekas), SSD, baterai CMOS/RTC |
| Koneksi audio | Jack 3.5mm mini PC → input amplifier |
| OS | Debian 13, instalasi server tanpa desktop |
| Jaringan | Kabel LAN ke router; IP tetap via **DHCP Reservation** di router |
| Akses | `http://<IP-tetap>` (utama, semua perangkat) dan `http://bel.local` (mDNS, tidak andal di Android) |
| Service | `systemd`, `Restart=always`, `RestartSec=2` |
| Jam | `chrony` (NTP) + zona waktu sesuai lokasi sekolah (`Asia/Jakarta`/`Asia/Makassar`/`Asia/Jayapura`) |
| Audio | `mpv` dengan output ALSA langsung (tanpa PulseAudio/PipeWire) |
| mDNS | `avahi-daemon` |
| Update | `unattended-upgrades` untuk keamanan, **tanpa** reboot otomatis |
| BIOS (manual) | Aktifkan "Power On after AC Loss" |
| Data | `/var/lib/belsekolah/` (`bel.db`, `audio/`, `backup/`) |

Skrip instal (`deploy/install.sh`) memasang dependensi (`mpv`, `ffmpeg`/`ffprobe`, `avahi-daemon`, `chrony`), membuat user sistem `belsekolah` (anggota grup `audio`), menyalin binary, memasang unit systemd, dan mengatur zona waktu.

## 4. Arsitektur

```
                 WiFi / LAN sekolah
   ┌──────────┐                        ┌─────────────────────────────────┐
   │ HP/Laptop│  browser               │  MINI PC (Debian)               │
   │  admin   │ ─────────────────────► │  belsekolah (1 binary Go)       │
   └──────────┘  http://192.168.x.x    │   • Web server + UI (embed)     │
                                       │   • Penjadwal                   │
                                       │   • Pemutar audio (mpv)         │
                                       │   • SQLite                      │
                                       └────────────────┬────────────────┘
                                                        │ jack audio
                                                        ▼
                                              Amplifier → TOA/Speaker
```

### Struktur modul

```
belsekolah/
├── cmd/belsekolah/   main: wiring & start
├── internal/
│   ├── store/        akses SQLite (repositori per entitas, migrasi)
│   ├── schedule/     logika murni: bel apa saja pada tanggal tertentu
│   ├── scheduler/    loop per detik, memicu bel, mencatat riwayat
│   ├── player/       antrean & pemutaran via mpv
│   ├── tts/          teks → MP3 via CLI edge-tts
│   ├── auth/         login, sesi, rate-limit
│   └── api/          REST API JSON + penyajian frontend embed
├── web/              frontend Svelte + Tailwind (build di-embed via go:embed)
└── deploy/           install.sh, belsekolah.service
```

### Tanggung jawab dan antarmuka modul

- **`schedule`**: fungsi murni tanpa I/O.
  `BelPadaTanggal(tgl Date, data DataJadwal) []BelTerjadwal` — `DataJadwal` berisi profil default, periode khusus, libur, dan jadwal semua profil. Tidak membaca jam sistem atau database.
- **`scheduler`**: menerima `Clock` (antarmuka, agar bisa dipalsukan saat tes), `store`, dan `player`. Tiap detik memeriksa apakah ada bel jatuh tempo; memuat ulang data saat ada perubahan dari API (sinyal reload) dan saat tanggal berganti.
- **`player`**: antarmuka `Play(ctx, urutan []FileAudio, prioritas)` dan `Stop()`. Implementasi asli menjalankan proses `mpv`; implementasi palsu untuk tes.
- **`tts`**: `Buat(ctx, teks, suara) ([]byte mp3, error)` dengan memanggil CLI `edge-tts` (lihat §8).
- **`store`**: CRUD untuk semua entitas; migrasi skema saat start.
- **`auth`**: verifikasi password bcrypt, sesi via cookie `HttpOnly`, `SameSite=Strict`.
- **`api`**: handler HTTP; memanggil `store`, `player`, `tts`; memberi sinyal reload ke `scheduler`.

### Data flow

```
schedule (murni) ──► scheduler (loop) ──► player (mpv) ──► amplifier
       ▲                                     ▲
     store ◄──── api ◄──── browser ──────────┘ (bel manual / tes suara)
```

## 5. Model Data (SQLite)

| Tabel | Kolom utama | Keterangan |
|---|---|---|
| `audio` | `id`, `nama`, `sumber` (`bawaan`/`upload`/`tts`), `file`, `durasi_ms`, `teks_tts`, `suara_tts`, `dibuat` | Semua suara, termasuk TTS yang sudah dirender jadi MP3 |
| `jenis_bel` | `id`, `nama`, `warna` | Mis. Masuk, Istirahat, Pulang, Darurat |
| `urutan_bel` | `jenis_bel_id`, `urutan`, `audio_id` | Urutan audio yang diputar untuk satu jenis bel |
| `profil` | `id`, `nama` | Mis. Normal, Ujian, Ramadan |
| `jadwal` | `id`, `profil_id`, `hari` (1=Senin..7=Minggu), `jam` (`HH:MM`), `jenis_bel_id`, `label` | Unik per (`profil_id`, `hari`, `jam`) |
| `periode_khusus` | `id`, `tgl_mulai`, `tgl_selesai`, `profil_id`, `keterangan` | Rentang inklusif; tidak boleh tumpang tindih |
| `libur` | `id`, `tgl_mulai`, `tgl_selesai`, `keterangan` | Rentang inklusif |
| `pengaturan` | `kunci`, `nilai` | `profil_default_id`, `volume`, `password_hash`, `wajib_ganti_password` |
| `riwayat_bel` | `id`, `waktu`, `tanggal`, `jadwal_id` (null untuk bel manual), `jenis_bel_id`, `label`, `sumber` (`jadwal`/`manual`/`darurat`), `status` (`berhasil`/`gagal`/`terlewat`/`cadangan`/`dihentikan`), `pesan` | Log untuk diagnosis; `tanggal`+`jadwal_id` dipakai agar bel tidak berbunyi dua kali setelah restart |

Penghapusan audio yang masih dipakai di `urutan_bel` ditolak; penghapusan jenis bel yang masih dipakai di `jadwal` ditolak. Profil default tidak bisa dihapus.

### Aturan penentuan bel suatu tanggal

1. Tanggal termasuk **libur** → tidak ada bel terjadwal.
2. Tanggal termasuk **periode khusus** → pakai profil periode tersebut.
3. Selain itu → pakai **profil default**.
4. Ambil `jadwal` dari profil terpilih untuk hari tersebut, urut berdasarkan jam.

Bel manual/darurat tidak terikat aturan ini dan selalu bisa dibunyikan.

## 6. Perilaku Penjadwal

- Loop berdetak tiap 1 detik membandingkan waktu sekarang dengan bel berikutnya.
- Bel dibunyikan jika `jam_bel <= sekarang < jam_bel + 60 detik` dan belum pernah dibunyikan hari itu (dilacak per tanggal+jadwal_id di memori dan dicocokkan dengan `riwayat_bel` saat start).
- Bel yang terlewat lebih dari 60 detik (karena mati listrik, crash, atau jam melompat) **tidak** disusulkan; dicatat `terlewat`.
- Jam mundur (koreksi NTP) tidak menyebabkan bel ganda.
- Data dimuat ulang saat API mengubah data dan saat pergantian tanggal.
- Saat dimuat ulang karena admin mengubah data, bel hari ini yang waktunya sudah lewat lebih dari 60 detik dianggap selesai tanpa dicatat `terlewat` (bel itu memang baru dibuat setelah waktunya lewat).

## 7. Pemutar Audio

- Satu antrean pemutaran; bel diputar satu per satu, audio dalam satu jenis bel diputar berurutan.
- Prioritas: **darurat** menghentikan suara yang sedang diputar dan mengosongkan antrean; **jadwal** dan **manual** masuk antrean.
- Tombol **Hentikan** menghentikan suara dan mengosongkan antrean.
- Volume global dari `pengaturan` diteruskan ke `mpv`.
- File hilang/rusak → putar **nada cadangan** yang di-embed di binary, status `cadangan`.
- `mpv` gagal → coba ulang 1 kali, lalu status `gagal`.

## 8. TTS

- Layanan: edge-tts (suara Microsoft `id-ID-GadisNeural` dan `id-ID-ArdiNeural`).
- Implementasi: aplikasi memanggil program CLI `edge-tts` (paket Python resmi, dipasang via `pipx`), sama seperti `mpv`/`ffprobe`. Alasan: protokol Microsoft memakai token DRM yang berubah-ubah (kini "v2" dengan hash khusus); CLI resmi mengikutinya dan cukup diperbarui dengan `pipx upgrade edge-tts`, tanpa mengubah aplikasi.
- Alur: admin mengetik teks + memilih suara → pratinjau → simpan sebagai `audio` (`sumber=tts`) berupa file MP3 (`tts-<hex>.mp3`). Saat bel berbunyi tidak ada panggilan internet.
- Teks 1–1000 karakter. Teks TTS bisa diedit lalu dirender ulang (menimpa file yang sama).
- Tanpa internet atau CLI belum terpasang → error jelas (HTTP 503) "Gagal membuat suara TTS; pastikan mini PC terhubung ke internet" / "program edge-tts belum terpasang".
- API: `GET /api/tts/suara`, `POST /api/tts/pratinjau` `{teks, suara}` → `audio/mpeg`, `POST /api/audio/tts` `{nama, teks, suara}` → 201, `PUT /api/audio/{id}/tts` `{teks, suara}` → 200 (hanya untuk audio bersumber `tts`).

## 9. Antarmuka Pengguna

Svelte + Tailwind, bahasa Indonesia, mobile-first, tombol besar.

1. **Beranda** — jam sekarang, profil aktif hari ini, bel berikutnya + hitung mundur, tombol **Bel Manual** (pilih jenis bel), **Bel Darurat** (tahan 2 detik), **Hentikan Suara**, jadwal hari ini dengan status, peringatan (jam belum tersinkron, bel gagal).
2. **Jadwal** — pilih profil → tab per hari → daftar (jam, jenis bel, label); tambah/edit/hapus; "Salin dari hari lain".
3. **Profil & Periode** — CRUD profil, set default; CRUD periode khusus.
4. **Libur** — CRUD tanggal/rentang libur.
5. **Jenis Bel** — CRUD, susun urutan audio (tambah, naik/turun, hapus), tombol **Tes** ke speaker.
6. **Pustaka Suara** — tab Bawaan / Upload / TTS; preview di browser dan tes di speaker.
7. **Riwayat** — daftar bel dengan filter status/tanggal.
8. **Pengaturan** — volume, ganti password, info sistem (IP, jam, status NTP, versi), backup/restore `.zip`.

## 10. Keamanan

- Satu akun admin; password default dibuat saat instal dan **wajib diganti** saat login pertama.
- Password di-hash bcrypt.
- Rate-limit login: 5 gagal → kunci 5 menit.
- Sesi via cookie `HttpOnly`, `SameSite=Strict`; semua endpoint API kecuali login butuh sesi.
- Upload: hanya MP3/WAV, maks 20 MB, divalidasi `ffprobe`; nama file disimpan dengan ID acak (bukan nama asli) untuk mencegah path traversal.
- Restore backup memvalidasi isi zip (hanya `bel.db` dan `audio/*`).

## 11. Penanganan Error & Ketahanan

| Kejadian | Penanganan |
|---|---|
| Listrik padam → menyala | BIOS auto-on, systemd start; bel terlewat dicatat `terlewat`, tidak disusulkan |
| Aplikasi crash | systemd restart dalam 2 detik |
| File audio hilang/rusak | Nada cadangan + peringatan di Beranda |
| mpv gagal | Retry 1x, lalu `gagal` + peringatan merah |
| Jam melompat | Toleransi 60 detik; tanpa bel ganda/susulan |
| Jam belum tersinkron | Peringatan kuning di Beranda |
| Internet mati saat TTS | Pesan error jelas; audio TTS tersimpan tetap berfungsi |
| Database | SQLite mode WAL; backup otomatis harian, simpan 7 terakhir di `/var/lib/belsekolah/backup/` |

## 12. Pengujian

- **`schedule`**: tes tabel — libur vs periode khusus vs default, hari tanpa jadwal, rentang lintas bulan/tahun, batas inklusif rentang.
- **`scheduler`**: fake clock — tepat waktu, tanpa bel ganda, tanpa susulan > 60 detik, jam mundur, pergantian tanggal, reload data.
- **`player`**: implementasi palsu — antrean, darurat memotong, nada cadangan, retry.
- **`store` & `api`**: tes integrasi dengan SQLite in-memory/temporary; validasi input & aturan penghapusan.
- **`auth`**: rate-limit, sesi, wajib ganti password.
- **`tts`**: program `edge-tts` tiruan (skrip shell); satu tes e2e di container ke layanan asli.
- **Frontend**: verifikasi manual di browser.
- **Perangkat**: checklist manual di mini PC — cabut listrik, cabut LAN, ubah jam, bunyi di amplifier, akses dari Android & iPhone.

## 13. Lingkungan Development (Docker)

Docker **hanya** dipakai untuk development dan pengujian; deploy di mini PC tetap binary + systemd (§3).

- `Dockerfile` multi-stage: build Go (dan frontend setelah Rencana 4) → runtime `debian:trixie-slim` berisi `mpv`, `ffmpeg` (untuk `ffprobe`), `ca-certificates`, `tzdata` — paket yang sama dengan mini PC.
- Audio di container diarahkan ke perangkat ALSA `null` (`/etc/asound.conf`), sehingga `mpv` benar-benar berjalan tanpa speaker.
- `compose.yaml`: port `8080`, volume data `/var/lib/belsekolah`, `TZ=Asia/Jakarta`, `restart: unless-stopped`.
- Dipakai untuk tes end-to-end API dan browser terhadap binary yang sama dengan produksi.

## 14. Kontrak API (JSON)

Semua di bawah `/api`, body dan respons JSON (kecuali upload/unduh). Error: `{"error": "<pesan Indonesia>"}` dengan kode 400 (validasi), 401 (belum login), 403 (wajib ganti password), 404, 409 (dipakai/duplikat/tumpang tindih), 429 (login terkunci). Semua perubahan data memicu muat ulang penjadwal.

Tambahan tahap 2: lihat `2026-10-15-tampilan-zona-template-design.md` §7.

| Metode & Path | Keterangan |
|---|---|
| `POST /login` `{password}` | Set cookie sesi; balas `{wajibGantiPassword}` |
| `POST /logout`, `GET /sesi` | Sesi saat ini |
| `POST /password` `{lama, baru}` | Ganti password (min. 8 karakter) |
| `GET /status` | Jam, profil/libur hari ini, bel hari ini + status, bel berikutnya, peringatan |
| `POST /bel/manual` `{jenisBelId}`, `POST /bel/darurat` `{jenisBelId}`, `POST /bel/stop` | Bel manual/darurat/hentikan |
| `GET/POST /profil`, `PUT/DELETE /profil/{id}` | Profil |
| `GET /jadwal?profil=`, `POST /jadwal`, `PUT/DELETE /jadwal/{id}`, `POST /jadwal/salin` | Jadwal; salin mengganti isi hari tujuan |
| `GET/POST /periode`, `PUT/DELETE /periode/{id}` | Periode khusus |
| `GET/POST /libur`, `PUT/DELETE /libur/{id}` | Libur |
| `GET/POST /jenis-bel`, `PUT/DELETE /jenis-bel/{id}`, `POST /jenis-bel/{id}/tes` | Jenis bel + urutan audio |
| `GET /audio`, `POST /audio` (multipart), `PUT/DELETE /audio/{id}`, `GET /audio/{id}/file`, `POST /audio/{id}/tes` | Pustaka suara |
| `GET /riwayat?tanggal=&status=&batas=` | Riwayat |
| `GET/PUT /pengaturan` `{volume, profilDefaultId}` | Pengaturan |
| `GET /sistem` | Versi, IP, zona waktu, jam, status NTP |
| `GET /backup`, `POST /restore` (multipart zip) | Backup/restore; restore memulai ulang aplikasi |

- Password awal dibuat acak saat database baru dan ditulis ke `<data>/password-awal.txt` (mode 0600, dihapus setelah diganti). Log hanya menyebut lokasi file, tidak pernah isi password. Flag `-reset-password` membuat password acak baru lalu keluar.
- Sesi disimpan di memori (hilang saat aplikasi restart), kedaluwarsa 7 hari sejak terakhir dipakai.
- Permintaan yang mengubah data wajib `Origin` sama dengan host (atau tanpa `Origin`), sebagai perlindungan CSRF tambahan selain `SameSite=Strict`.
- Restore: aplikasi keluar dengan kode 3 setelah mengganti data; systemd/Docker memulainya lagi.
