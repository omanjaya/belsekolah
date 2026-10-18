# Panduan Pemasangan Bel Sekolah

Panduan ini untuk operator sekolah. Ikuti langkahnya berurutan. Tidak perlu paham pemrograman.

## 1. Yang dibutuhkan

- Mini PC x86 dengan SSD dan RAM minimal 2 GB.
- Kabel audio 3.5 mm ke input amplifier (AUX atau LINE IN).
- Kabel LAN (disarankan) atau WiFi. Mini PC harus terhubung ke internet selama pemasangan (untuk GitHub, apt, dan pipx). WiFi bisa disambungkan saat memasang Debian: pilih kartu jaringan berawalan `wl`, pilih nama WiFi, lalu masukkan password-nya.
- Flashdisk minimal 4 GB.
- Akses admin ke router sekolah.

## 2. Pengaturan BIOS

1. Nyalakan mini PC, lalu tekan tombol masuk BIOS berulang kali. Tombol yang umum: F2, Del, F1, atau F10.
2. Cari pilihan **Power On after AC Loss** atau **Restore on AC Power Loss**. Ubah menjadi **Power On**. Dengan ini mini PC menyala sendiri setelah listrik padam.
3. Matikan semua pengaturan sleep atau hibernasi.
4. Simpan dan keluar (biasanya F10).

## 3. Pasang Debian 13

1. Unduh Debian 13 "netinst" untuk amd64 dari situs resmi debian.org.
2. Tulis berkas itu ke flashdisk memakai balenaEtcher atau Rufus.
3. Colok flashdisk ke mini PC, nyalakan, lalu pilih boot dari flashdisk (biasanya F12 atau F11).
4. Ikuti pemasang Debian. Jangan pilih lingkungan desktop.
5. Pada pilihan perangkat lunak, centang **SSH server** dan **standard system utilities**.
6. Saat diminta **password root**, biarkan KOSONG (langsung tekan Enter). Dengan begitu Debian memasang `sudo` dan memasukkan user admin ke grup sudo.
7. Buat satu user admin dan catat password-nya.

Jika password root sudah terlanjur diisi, `sudo` belum ada. Setelah Debian terpasang, jalankan:

1. `su -` (masukkan password root).
2. `apt install sudo`
3. `usermod -aG sudo NAMA_USER` (ganti dengan nama user admin).
4. Keluar (`exit`) dari semua sesi, lalu login lagi.

## 4. IP tetap

Agar alamat mini PC tidak berganti, atur DHCP reservation di router.

1. Di mini PC, jalankan `ip link`. Catat alamat MAC kartu LAN (tulisan setelah `link/ether`), misalnya `aa:bb:cc:dd:ee:ff`.
2. Buka halaman admin router. Cari menu DHCP Reservation atau Static Lease.
3. Masukkan MAC tadi dan pilih IP, misalnya `192.168.1.50`.
4. Simpan, lalu restart mini PC dengan `sudo reboot`.

## 5. Pasang aplikasi

Pastikan mini PC terhubung ke internet (pemasang mengunduh paket lewat GitHub, apt, dan pipx).

### Cara A: git clone (disarankan)

Di mini PC (lewat SSH atau langsung di layar):

```
sudo apt install -y git
git clone https://github.com/omanjaya/belsekolah.git
cd belsekolah
sudo ./pasang.sh
```

`pasang.sh` mengunduh paket rilis terbaru dari GitHub, memeriksa checksum-nya, lalu menjalankan `install.sh` dari paket itu. Untuk memasang versi tertentu, tambahkan `--versi v0.4.0`.

### Cara B: salin paket rilis

1. Unduh `belsekolah-<versi>-linux-amd64.tar.gz` dari halaman Releases GitHub, lalu salin ke mini PC dari komputer lain:
   `scp belsekolah-*-linux-amd64.tar.gz admin@192.168.1.50:`
2. Di mini PC (lewat SSH), ekstrak dan masuk ke foldernya:
   ```
   tar xzf belsekolah-*-linux-amd64.tar.gz
   cd belsekolah-*-linux-amd64/
   ```
3. Jalankan pemasang:
   ```
   sudo ./install.sh
   ```

### Pilihan tambahan

Pilihan berikut berlaku untuk `pasang.sh` maupun `install.sh`:

- `--tz Asia/Makassar` untuk WITA, atau `--tz Asia/Jayapura` untuk WIT. Tanpa pilihan ini zona waktunya WIB (`Asia/Jakarta`).
- `--hostname NAMA` untuk mengganti nama mini PC (bawaan: `bel`).
- `--perangkat-audio NAMA` untuk memilih keluaran suara (lihat bagian Pemecahan masalah).

Contoh: `sudo ./pasang.sh --tz Asia/Makassar`

Menjalankan `pasang.sh` atau `install.sh` lagi aman. Zona waktu, nama host, dan perangkat audio yang lama tetap dipakai.

**Mengganti zona waktu setelah terpasang.** Zona waktu juga bisa diganti dari web: buka **Pengaturan**, pilih **Zona waktu** (WIB, WITA, atau WIT), lalu simpan. Aplikasi akan mulai ulang sendiri sekitar 2 detik, jadi halaman bisa terputus sebentar. Pilihan di web ini lebih diutamakan daripada `--tz` saat instal. Jam sistem Debian pada mini PC tidak ikut berubah; yang berubah hanya jam yang dipakai aplikasi untuk menjalankan bel.

## 6. Login pertama

1. Dari HP atau laptop di jaringan yang sama, buka `http://192.168.1.50` (ganti dengan IP mini PC) atau `http://bel.local`.
   Catatan: alamat `.local` sering tidak jalan di Android. Pakai IP saja.
2. Ambil password awal di mini PC:
   ```
   sudo cat /var/lib/belsekolah/password-awal.txt
   ```
3. Login, lalu ganti password. Berkas password awal akan terhapus sendiri setelah password diganti.

## 7. Sambungkan ke amplifier dan uji

1. Colok kabel audio dari mini PC ke input AUX atau LINE IN amplifier.
2. Kecilkan volume amplifier lebih dulu.
3. Buka **Beranda**. Pada kartu **Bunyikan bel**, pilih **Jenis bel** (misalnya Masuk), lalu tekan **Bunyikan**.
4. Naikkan volume amplifier pelan-pelan.
5. Jika tidak ada suara, lihat bagian Pemecahan masalah.

Di kartu yang sama ada dua tombol lain:

- **Bel darurat**: tahan tombol merah **Tahan untuk bel darurat** selama 2 detik sampai terisi penuh. Menekan sebentar tidak akan membunyikan apa pun. Tombol ini memakai jenis bel bernama "Darurat". Jika jenis bel itu terhapus atau diganti namanya, tombol nonaktif dan muncul pesan agar Anda membuatnya lagi di menu **Jenis Bel**.
- **Hentikan suara**: menghentikan suara yang sedang diputar.

## 8. Atur jadwal

Isi menu berikut sesuai urutan ini:

1. **Jenis Bel**: buat jenis bel, misalnya masuk, istirahat, pulang.
2. **Jadwal**: atur jam bel tiap hari.
3. **Libur**: isi hari libur.
4. **Profil & Periode**: atur profil jadwal dan periode berlakunya (semester atau ujian).

Di HP, menu bawah hanya menampilkan **Beranda**, **Jadwal**, **Suara**, dan **Riwayat**. **Jenis Bel**, **Profil & Periode**, **Libur**, dan **Pengaturan** ada di bawah tombol **Lainnya**. Di laptop, semua menu tampil di sisi kiri.

Di menu **Jadwal**, pilih profil lewat dropdown **Profil**, lalu pilih hari lewat tab Senin sampai Minggu. Tombol **Tambah bel** menambah bel pada hari yang sedang dipilih, dan **Salin dari hari lain** menyalin jadwal antarhari.

Menu lain: **Suara** untuk suara bel (unggah file atau buat dari teks), **Riwayat** untuk catatan bel yang sudah berbunyi, gagal, atau terlewat, dan **Pengaturan** untuk volume, password, info sistem, dan backup.

## 9. Backup

- Mini PC membuat backup otomatis setiap hari dan menyimpan 7 yang terakhir di `/var/lib/belsekolah/backup`.
- Backup manual dan pemulihan ada di **Pengaturan → Backup & Restore**:
  - **Unduh backup** menyimpan file `.zip` berisi semua data.
  - **Pulihkan dari backup**: pilih file `.zip` pada **File backup (.zip)**, lalu tekan **Pulihkan dari backup** dan konfirmasi.
- Pemulihan mengganti semua data, termasuk password admin: password yang berlaku adalah password saat backup itu dibuat. Zona waktu yang tersimpan di backup juga ikut dikembalikan. Semua sesi login berakhir. Aplikasi memulai ulang dan halaman dimuat ulang sendiri (tunggu hingga 2 menit), lalu Anda login lagi dengan password yang berlaku saat backup dibuat. Jika lupa password itu, pakai perintah reset di bagian "Lupa password".
- Unduh satu backup setiap awal semester dan simpan di luar mini PC (flashdisk atau komputer lain). Jika SSD rusak, backup di dalam mini PC ikut hilang.

## 10. Pembaruan aplikasi

Jika memasang dengan git clone (cara A):

```
cd ~/belsekolah
git pull
sudo ./pasang.sh
```

Jika memasang dari paket (cara B): salin paket versi baru ke mini PC, ekstrak, lalu jalankan `sudo ./install.sh` lagi.

Data jadwal tetap aman di kedua cara.

## 11. Pemecahan masalah

### Tidak ada suara

1. Lihat daftar kartu suara: `aplay -l`.
2. Lihat nama perangkat untuk aplikasi: `mpv --audio-device=help`.
3. Pilih perangkat analog (keluaran jack 3.5 mm), lalu pasang ulang dengan nama itu. Contoh:
   `sudo ./pasang.sh --perangkat-audio "alsa/plughw:CARD=PCH,DEV=0"`
   Pilihan ini disimpan di `/etc/default/belsekolah` sebagai `BELSEKOLAH_MPV_DEVICE`.
4. Jalankan `alsamixer`. Pastikan kanal tidak mute (tanda MM). Tekan `M` untuk membuka mute, panah atas untuk menaikkan volume.
5. Periksa kabel dan input amplifier.

### Alat belum terpasang

Buka **Pengaturan → Info sistem**. Di sana ada status **Pemutar audio (mpv)**, **Pembaca audio (ffprobe)**, dan **Suara TTS (edge-tts)**, masing-masing **Terpasang** (hijau) atau **Belum terpasang** (merah). Jika ada yang merah, jalankan `sudo ./install.sh` lagi dengan mini PC tersambung ke internet. Info sistem juga menampilkan **Ruang disk kosong** dan **Backup harian terakhir**; jika ruang disk hampir habis atau backup harian tidak pernah bertambah, periksa SSD mini PC.

### Jam salah

- Jalankan `timedatectl` dan `chronyc tracking`.
- Pastikan mini PC tersambung ke internet.
- Jika muncul peringatan "Jam belum tersinkron" di Beranda, tunggu beberapa menit setelah internet tersambung.

### Suara dari teks (TTS) gagal

- Pastikan internet tersambung.
- Perbarui komponen TTS:
  `sudo env PIPX_HOME=/opt/pipx PIPX_BIN_DIR=/usr/local/bin pipx upgrade edge-tts`

### Lupa password

```
sudo -u belsekolah belsekolah -data /var/lib/belsekolah -reset-password
sudo systemctl restart belsekolah
```
Lalu ambil password baru seperti di langkah 6.

### Tidak bisa dibuka dari HP

1. Pastikan HP memakai WiFi yang sama dengan mini PC (bukan WiFi tamu).
2. Pastikan IP-nya benar.
3. Cek layanan: `sudo systemctl status belsekolah`.
4. Lihat log: `sudo journalctl -u belsekolah -n 50`.

### Listrik padam

Mini PC menyala sendiri (karena pengaturan BIOS). Bel yang terlewat saat listrik mati tercatat sebagai "terlewat" di menu **Riwayat**.

## 12. Checklist uji di lokasi

Lakukan sebelum dipakai sungguhan:

- [ ] Cabut listrik mini PC, lalu colok lagi. Mini PC menyala sendiri dan web bisa dibuka.
- [ ] Cabut kabel LAN. Bel tetap berbunyi sesuai jadwal.
- [ ] Jam di Beranda sama dengan jam sebenarnya.
- [ ] Web bisa dibuka dari HP Android.
- [ ] Web bisa dibuka dari iPhone.
- [ ] Uji bel darurat, lalu uji tombol hentikan.
