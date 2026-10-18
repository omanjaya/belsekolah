# Bel Sekolah

Aplikasi bel sekolah otomatis (Go) dengan antarmuka web, berjalan di mini PC Debian.

## Ringkasan

Bel Sekolah membunyikan bel secara otomatis sesuai jadwal (per hari, per profil jadwal, dengan hari libur dan periode khusus), memutar suara lewat `mpv`, dan bisa membuat pengumuman suara (TTS) lewat `edge-tts`. Tombol "Tambah banyak" di halaman Jadwal menggabungkan bel dari ketikan/tempelan, template, file, atau profil lain ke profil yang dibuka, dengan pratinjau bentrok. Setiap bel bisa punya pengumuman suara sendiri yang diisi otomatis dari keterangannya (suara bernama "{keterangan} dimulai"), dan tombol "Isi pengumuman otomatis" mengisi banyak bel sekaligus. Admin mengelola semuanya lewat antarmuka web yang nyaman dipakai di HP: beranda dan bel manual/darurat, jadwal, profil dan libur, suara dan jenis bel, riwayat, serta pengaturan (password, backup, restore). Backend Go dengan SQLite; frontend Svelte 5 yang hasil build-nya tertanam di binary (`internal/webui/dist`).

## Pengembangan

Lingkungan lengkap di Docker (sudah berisi mpv, ffmpeg, dan edge-tts):

```
docker compose up --build
```

Lalu buka http://localhost:8080 (ubah port host dengan `BEL_PORT=18080 docker compose up --build`). Password awal ada di:

```
docker compose exec belsekolah cat /var/lib/belsekolah/password-awal.txt
```

Hot reload backend dan frontend sekaligus (air untuk Go, Vite untuk Svelte):

```
docker compose -f compose.dev.yaml up --build
```

Lalu buka http://localhost:5173 (ubah port dengan `BEL_DEV_PORT`). Perubahan `.svelte`/`.ts` langsung tampil di browser; perubahan `.go` membuat backend dibuild dan dijalankan ulang otomatis. Data dev terpisah dari `compose.yaml`; password awalnya:

```
docker compose -f compose.dev.yaml exec api cat /var/lib/belsekolah/password-awal.txt
```

Frontend saja tanpa Docker (proxy `/api` ke port 8080, ubah dengan `BEL_API=http://127.0.0.1:18081`):

```
cd web && npm ci && npm run dev
```

Menjalankan backend tanpa Docker:

```
go run ./cmd/belsekolah -data ./tmp-data -addr :8080
```

Setelah mengubah frontend, jalankan `cd web && npm run build` dan commit isi `internal/webui/dist`, karena folder itulah yang ditanam ke binary. Jangan membangun dengan `VITE_E2E`; variabel itu hanya untuk tes Playwright lokal.

## Pengujian

```
go vet ./...
go test -race ./...
cd web && npm run check && npm test     # tipe dan tes unit frontend
cd web && npx playwright test           # tes UI (membangun dan menjalankan server sendiri)
./scripts/e2e.sh                        # Playwright lalu tes Go terhadap container Docker baru
./deploy/uji/uji-install.sh             # uji pemasang di container Debian 13 (butuh Docker)
```

`./scripts/e2e.sh` memakai `BEL_PORT` (bawaan 8080). Skrip menjalankan Playwright pada container baru, mereset container, lalu menjalankan tes Go (termasuk TTS asli dan restore backup).

## Struktur folder

```
cmd/belsekolah/    titik masuk program
internal/api/      handler HTTP dan routing
internal/auth/     password admin dan sesi
internal/backup/   backup dan restore
internal/bawaan/   suara dan data awal instalasi baru
internal/nada/     sintesis suara bel bawaan (WAV)
internal/player/   pemutar audio berantrean (mpv)
internal/schedule/ bel apa yang berbunyi pada suatu tanggal
internal/scheduler/ penjadwal bel
internal/sistem/   pembungkus ffprobe dan timedatectl
internal/store/    penyimpanan SQLite
internal/tts/      edge-tts
internal/webui/    frontend hasil build (dist) tertanam
internal/zona/     zona waktu Indonesia (WIB/WITA/WIT) dan pemilihan zona aktif
internal/formatjadwal/ file jadwal: ekspor/impor profil dan template
internal/pembuat/  penyusun jadwal bel dari parameter sekolah
web/               kode frontend Svelte 5 dan tes Playwright
e2e/               tes end-to-end Go terhadap Docker
deploy/            pemasang dan layanan systemd untuk mini PC
scripts/           skrip e2e dan rilis
pasang.sh          pemasang dari git clone (unduh rilis terbaru lalu install.sh)
docs/superpowers/  spesifikasi dan rencana
```

## Dokumen

- Spesifikasi: [docs/superpowers/specs](docs/superpowers/specs)
- Rencana implementasi: [docs/superpowers/plans](docs/superpowers/plans)

## Pemasangan di mini PC

Panduan lengkap untuk operator: [PANDUAN-INSTAL.md](PANDUAN-INSTAL.md). Singkatnya, di mini PC Debian 13:

```
sudo apt install -y git
git clone https://github.com/omanjaya/belsekolah.git
cd belsekolah && sudo ./pasang.sh
```

`pasang.sh` mengunduh paket rilis terbaru (termasuk pre-release) dari GitHub, memeriksa checksum, lalu menjalankan `install.sh`. Pembaruan: `git pull && sudo ./pasang.sh`.

Untuk membuat paket rilis (amd64 dan arm64):

```
./scripts/rilis.sh [versi]
```

Hasilnya ada di `dist/belsekolah-<versi>-linux-<arch>.tar.gz` beserta `SHA256SUMS`. Paket berisi binary, `install.sh`, `belsekolah.service`, dan `PANDUAN-INSTAL.md`. Salin ke mini PC, ekstrak, lalu jalankan `sudo ./install.sh`.
