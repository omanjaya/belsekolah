// Command belsekolah menjalankan bel sekolah otomatis beserta antarmuka webnya.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata" // zona waktu tertanam, agar Debian minimal tanpa tzdata tetap jalan

	"belsekolah/internal/api"
	"belsekolah/internal/auth"
	"belsekolah/internal/backup"
	"belsekolah/internal/bawaan"
	"belsekolah/internal/nada"
	"belsekolah/internal/player"
	"belsekolah/internal/scheduler"
	"belsekolah/internal/sistem"
	"belsekolah/internal/store"
	"belsekolah/internal/tts"
	"belsekolah/internal/webui"
	"belsekolah/internal/zona"
)

// versi diisi saat build: -ldflags "-X main.versi=..."
var versi = "dev"

var _ scheduler.Sumber = (*store.Store)(nil)

// errMulaiUlang membuat proses keluar dengan kode 3 agar systemd/Docker memulainya lagi.
var errMulaiUlang = errors.New("mulai ulang atas permintaan (restore atau zona waktu)")

func main() {
	dataDir := flag.String("data", "/var/lib/belsekolah", "folder data (database dan audio)")
	tz := flag.String("tz", "Asia/Jakarta", "zona waktu awal; pilihan di web lebih diutamakan")
	addr := flag.String("addr", ":80", "alamat web server")
	reset := flag.Bool("reset-password", false, "buat password admin acak baru lalu keluar")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if *reset {
		if err := resetPassword(*dataDir); err != nil {
			log.Error("reset password gagal", "err", err)
			os.Exit(1)
		}
		return
	}
	err := jalankan(*dataDir, *addr, *tz, log)
	switch {
	case errors.Is(err, errMulaiUlang):
		log.Info("keluar untuk dimulai ulang; layanan akan berjalan lagi")
		os.Exit(3)
	case err != nil:
		log.Error("aplikasi berhenti", "err", err)
		os.Exit(1)
	}
}

func tulisPasswordAwal(dataDir, pw string) error {
	path := filepath.Join(dataDir, auth.NamaFilePasswordAwal)
	if err := os.WriteFile(path, []byte(pw+"\n"), 0o600); err != nil {
		return err
	}
	// WriteFile tidak mengubah mode file yang sudah ada.
	return os.Chmod(path, 0o600)
}

func resetPassword(dataDir string) error {
	st, err := store.Open(filepath.Join(dataDir, "bel.db"))
	if err != nil {
		return err
	}
	defer st.Close()
	pw, err := auth.New(st, time.Now).ResetPassword(context.Background())
	if err != nil {
		return err
	}
	if err := tulisPasswordAwal(dataDir, pw); err != nil {
		return err
	}
	fmt.Printf("Password admin baru: %s\nMulai ulang layanan agar sesi lama berakhir (systemctl restart belsekolah).\n", pw)
	return nil
}

func jalankan(dataDir, addr, tz string, log *slog.Logger) error {
	sinyal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, batal := context.WithCancelCause(sinyal)
	defer batal(nil)

	// Sebelum apa pun memakai file sementara (restore, unggahan, zip backup).
	if err := siapkanTmp(dataDir); err != nil {
		return err
	}

	if diterapkan, err := backup.TerapkanTertunda(dataDir, time.Now()); err != nil {
		return fmt.Errorf("terapkan restore: %w", err)
	} else if diterapkan {
		log.Warn("restore dari backup diterapkan; data lama disimpan di folder backup")
	}

	audioDir := filepath.Join(dataDir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return err
	}
	if n := sapuSisaAudio(audioDir); n > 0 {
		log.Warn("sisa file sementara di folder audio dihapus", "jumlah", n)
	}
	// Nada cadangan ditulis ulang setiap start, terpisah dari folder audio.
	cadangan := filepath.Join(dataDir, "cadangan.wav")
	if err := os.WriteFile(cadangan, nada.Cadangan(), 0o644); err != nil {
		return fmt.Errorf("tulis nada cadangan: %w", err)
	}

	st, err := store.Open(filepath.Join(dataDir, "bel.db"))
	if err != nil {
		return err
	}
	defer st.Close()

	tersimpan, err := st.ZonaWaktu(ctx)
	if err != nil {
		return fmt.Errorf("baca zona waktu: %w", err)
	}
	loc, peringatan := zona.Pilih(tersimpan, tz)
	for _, p := range peringatan {
		log.Warn(p)
	}
	// Belum ada goroutine yang membaca jam, jadi aman mengganti time.Local di sini.
	time.Local = loc

	if err := bawaan.Pasang(ctx, st, audioDir, time.Now()); err != nil {
		return fmt.Errorf("pasang data awal: %w", err)
	}

	au := auth.New(st, time.Now)
	if pw, dibuat, err := au.SiapkanPasswordAwal(ctx); err != nil {
		return fmt.Errorf("siapkan password awal: %w", err)
	} else if dibuat {
		if err := tulisPasswordAwal(dataDir, pw); err != nil {
			return err
		}
		log.Warn("password admin awal dibuat; baca file ini lalu ganti setelah login pertama",
			"file", filepath.Join(dataDir, auth.NamaFilePasswordAwal))
	}

	pemutar := player.New(player.Mpv{}, cadangan)
	if v, err := st.Volume(ctx); err == nil {
		pemutar.SetVolume(v)
	} else {
		log.Warn("volume tersimpan tidak terbaca, memakai 100", "err", err)
	}
	selesai := make(chan struct{})
	go func() {
		defer close(selesai)
		pemutar.Jalankan(ctx)
	}()
	// Didaftarkan setelah defer st.Close(), jadi berjalan lebih dulu: pemutar harus
	// selesai (Selesai -> InsertRiwayat) sebelum database ditutup.
	defer func() { <-selesai }()

	sch := scheduler.New(scheduler.JamSistem{}, st, pemutar, audioDir, log)
	selesaiBackup := make(chan struct{})
	go func() {
		defer close(selesaiBackup)
		backup.JalankanHarian(ctx, st, filepath.Join(dataDir, "backup"), log)
	}()
	defer func() { <-selesaiBackup }() // harus selesai sebelum database ditutup

	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		Handler: api.New(api.Deps{
			Store: st, Auth: au, Penjadwal: sch, Pemutar: pemutar,
			DataDir: dataDir, AudioDir: audioDir, Versi: versi, Zona: loc.String(),
			Web: webui.FS(), Log: log, Now: time.Now,
			CekNTP: sistem.CekNTP(), DurasiAudio: sistem.DurasiAudio,
			MulaiUlang: func() { batal(errMulaiUlang) },
			TTS:        tts.EdgeTTS{},
		}),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			batal(fmt.Errorf("web server: %w", err))
		}
	}()

	log.Info("bel sekolah berjalan", "versi", versi, "alamat", addr, "data", dataDir,
		"zona_waktu", loc.String())
	sch.Run(ctx)

	tutup, batalTutup := context.WithTimeout(context.Background(), 5*time.Second)
	defer batalTutup()
	if err := srv.Shutdown(tutup); err != nil {
		log.Warn("web server tidak berhenti rapi", "err", err)
		srv.Close()
	}
	log.Info("bel sekolah berhenti")

	if sebab := context.Cause(ctx); sebab != nil && !errors.Is(sebab, context.Canceled) {
		return sebab
	}
	return nil
}
