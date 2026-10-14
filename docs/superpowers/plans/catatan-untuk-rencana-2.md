# Catatan dari Rencana 1 untuk Rencana 2+

Temuan review Rencana 1 yang sengaja ditunda karena baru relevan setelah API ada.

Diselesaikan di Rencana 2: sanitasi path audio, rate-limit log, validasi input API, -tz kosong ditolak.

- `Scheduler.Tick` melakukan I/O database sambil memegang `s.mu`; `Reload()` dari handler HTTP ikut menunggu.
- Callback `player.Permintaan.Selesai` tidak boleh panic (panic menghentikan loop pemutar); `Play` tidak boleh dipanggil setelah shutdown dimulai.
- Pertimbangkan: kegagalan `bawaan.Pasang` pada instalasi yang sudah berisi jadwal cukup jadi WARN, bukan berhenti.
- Pertimbangkan: start pertama pada DB tanpa riwayat tidak perlu mencatat `terlewat` untuk bel pagi.
- Retry mpv tanpa jeda; pertimbangkan jeda 200–500 ms.
- Uji dengar manual (mpv + speaker) dan cek bunyi "klik" di akhir nada bawaan.
