# Catatan lanjutan (setelah Rencana 2–5)

Temuan review yang sengaja ditunda. Tidak ada yang memblokir pemakaian; urut kira-kira menurut manfaat.

## Perlu diuji di perangkat asli
- Unit systemd dengan semua pengerasan (port 80 via capability, audio ke speaker sebagai user `belsekolah`, edge-tts sebagai user layanan), `timedatectl`/`hostnamectl`, `bel.local` dari HP, persistensi setelah reboot. Container uji tidak mencakup systemd.
- Uji dengar nada bawaan (kemungkinan bunyi "klik" di akhir nada; belum ada fade-out).

## Backend
- ffprobe yang kehabisan waktu dilaporkan sebagai "bukan audio" (400) alih-alih 500.
- Versi skema (`PRAGMA user_version`) agar backup dari versi lebih baru ditolak.
- Batas panjang antrean pemutar; jeda singkat sebelum mpv dicoba ulang.
- Disk penuh → pesan khusus (507); arsip `*-sebelum-restore-*` belum dipangkas otomatis.
- Penulisan data selama unggah restore (sebelum disiapkan) belum diblokir.
- `ubahJenisBel`, re-render TTS, dan simpan pengaturan tidak atomik (jendela kecil).
- `Tick` penjadwal melakukan I/O database sambil memegang kunci.

## Frontend
- Restore: jika boot awal gagal dibaca, halaman bisa memuat ulang terlalu cepat.
- Navigasi tab dengan panah, `document.title` per halaman, fokus ke judul saat pindah halaman.
- Komponen input bersama (kelas input masih disalin di beberapa halaman); helper "hari ini" bersama.
- Jam di Beranda sedikit melompat setiap penyegaran status.

## Deploy
- Instal ulang tanpa internet berhenti di `apt-get update`.

## Tahap 2 (Rencana 6–7) — temuan kecil yang ditunda

- `formatjadwal.Baca`: error tipe JSON salah tidak menyebut bel mana; data setelah objek JSON pertama diabaikan.
- `ImporJadwal` memvalidasi nama di file sebelum nama pengganti dipakai (file bernama kosong ditolak walau nama baru valid).
- `/api/status` mengirim `profil.jumlahBel` = 0 (tidak dipakai UI).
- Respons `PUT /api/pengaturan` tidak atomik (volume/profil tersimpan walau penyimpanan zona gagal).
- Zona OS dipakai untuk log & nama folder restore sebelum `time.Local` diatur saat start.
- Bel darurat/stop ikut ditolak 503 selama ±2,5 dtk restart setelah ganti zona (sama seperti restore).
- Jadwal: 5–7 kolom sempit di lebar 768–1000px; profil kosong tanpa ajakan "tambah bel".
- Tab hari (Jadwal) & tab sumber (Suara) belum mendukung navigasi tombol panah; overlay restart belum mengunci Tab.
- Laci menu HP tanpa focus trap; pil status tidak punya keadaan "tidak terhubung".
- Template: tanggal "Dibuat" diambil dari string UTC; layout dua kolom (lg) belum diuji otomatis.
- Beberapa celah tes: batas parameter pembuat (kegiatan Jumat, ujian, 23:59), file >256 KB di periksa, ganti nama template ke nama yang sudah ada.
