# Catatan dari Rencana 1 untuk Rencana 2+

Temuan review Rencana 1 yang sengaja ditunda karena baru relevan setelah API ada.

- Nama file audio upload/TTS wajib dibuat server (ID acak + ekstensi); loader audio menolak `..` dan pemisah path (`scheduler` menggabungkan `audioDir` + `audio.File` dari DB).
- Validasi input di batas API: jam `HH:MM`, tanggal `YYYY-MM-DD` (`schedule` mengandalkan format ini dan melewati jam tidak valid tanpa log).
- `Scheduler.Tick` melakukan I/O database sambil memegang `s.mu`; `Reload()` dari handler HTTP ikut menunggu.
- Error muat jadwal tercatat setiap detik; beri rate-limit atau catat hanya saat status berubah.
- Callback `player.Permintaan.Selesai` tidak boleh panic (panic menghentikan loop pemutar); `Play` tidak boleh dipanggil setelah shutdown dimulai.
- Pertimbangkan: kegagalan `bawaan.Pasang` pada instalasi yang sudah berisi jadwal cukup jadi WARN, bukan berhenti.
- Pertimbangkan: start pertama pada DB tanpa riwayat tidak perlu mencatat `terlewat` untuk bel pagi.
- Retry mpv tanpa jeda; pertimbangkan jeda 200–500 ms.
- `-tz ""` berarti UTC; validasi di skrip deploy (Rencana 5).
- Uji dengar manual (mpv + speaker) dan cek bunyi "klik" di akhir nada bawaan.
