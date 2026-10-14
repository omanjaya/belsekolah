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
