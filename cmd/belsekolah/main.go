// Command belsekolah menjalankan bel sekolah otomatis.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"belsekolah/internal/bawaan"
	"belsekolah/internal/nada"
	"belsekolah/internal/player"
	"belsekolah/internal/scheduler"
	"belsekolah/internal/store"
)

var _ scheduler.Sumber = (*store.Store)(nil)

func main() {
	dataDir := flag.String("data", "/var/lib/belsekolah", "folder data (database dan audio)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := jalankan(*dataDir, log); err != nil {
		log.Error("aplikasi berhenti", "err", err)
		os.Exit(1)
	}
}

func jalankan(dataDir string, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	audioDir := filepath.Join(dataDir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return err
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

	if err := bawaan.Pasang(ctx, st, audioDir, time.Now()); err != nil {
		return fmt.Errorf("pasang data awal: %w", err)
	}

	pemutar := player.New(player.Mpv{}, cadangan)
	selesai := make(chan struct{})
	go func() {
		defer close(selesai)
		pemutar.Jalankan(ctx)
	}()
	// Didaftarkan setelah defer st.Close(), jadi berjalan lebih dulu: pemutar harus
	// selesai (Selesai -> InsertRiwayat) sebelum database ditutup.
	defer func() { <-selesai }()

	sch := scheduler.New(scheduler.JamSistem{}, st, pemutar, audioDir, log)
	log.Info("bel sekolah berjalan", "data", dataDir, "zona_waktu", time.Now().Format("MST -07:00"))
	sch.Run(ctx)
	log.Info("bel sekolah berhenti")
	return nil
}
