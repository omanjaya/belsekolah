package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"belsekolah/internal/player"
	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

// Toleransi: bel yang terlambat lebih dari ini tidak dibunyikan, hanya dicatat terlewat.
const Toleransi = 60 * time.Second

type Clock interface {
	Now() time.Time
}

type JamSistem struct{}

func (JamSistem) Now() time.Time { return time.Now() }

type Sumber interface {
	LoadDataJadwal(ctx context.Context) (schedule.Data, error)
	JadwalTercatat(ctx context.Context, tanggal string) (map[int64]bool, error)
	InsertRiwayat(ctx context.Context, r store.Riwayat) error
	UrutanBel(ctx context.Context, jenisBelID int64) ([]store.Audio, error)
}

type Pemutar interface {
	Play(req player.Permintaan)
}

type Scheduler struct {
	clock    Clock
	sumber   Sumber
	pemutar  Pemutar
	audioDir string
	log      *slog.Logger

	mu        sync.Mutex
	tanggal   string
	hasil     schedule.Hasil
	sudah     map[int64]bool // jadwal hari ini yang sudah dibunyikan/dicatat
	perluMuat bool
	editan    bool // muat ulang karena admin mengubah data
}

func New(clock Clock, sumber Sumber, pemutar Pemutar, audioDir string, log *slog.Logger) *Scheduler {
	return &Scheduler{
		clock:     clock,
		sumber:    sumber,
		pemutar:   pemutar,
		audioDir:  audioDir,
		log:       log,
		perluMuat: true,
	}
}

// Reload meminta data dimuat ulang pada detik berikutnya (dipanggil setelah admin mengubah data).
func (s *Scheduler) Reload() {
	s.mu.Lock()
	s.perluMuat = true
	s.editan = true
	s.mu.Unlock()
}

// Run memanggil Tick setiap detik sampai ctx selesai.
func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		if err := s.Tick(ctx); err != nil {
			s.log.Error("pemeriksaan jadwal gagal", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick memeriksa sekali apakah ada bel yang jatuh tempo.
func (s *Scheduler) Tick(ctx context.Context) error {
	now := s.clock.Now()
	tanggal := now.Format(schedule.FormatTanggal)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.perluMuat || tanggal != s.tanggal {
		gantiHari := tanggal != s.tanggal
		if err := s.muat(ctx, now, tanggal); err != nil {
			return err
		}
		if s.editan && !gantiHari {
			// Bel yang baru dibuat setelah waktunya lewat tidak dianggap terlewat.
			for _, b := range s.hasil.Bel {
				if now.Sub(b.Waktu) >= Toleransi {
					s.sudah[b.JadwalID] = true
				}
			}
		}
		s.perluMuat, s.editan = false, false
	}

	for _, b := range s.hasil.Bel {
		if s.sudah[b.JadwalID] || now.Before(b.Waktu) {
			continue
		}
		s.sudah[b.JadwalID] = true
		if telat := now.Sub(b.Waktu); telat >= Toleransi {
			s.catat(ctx, b, "terlewat", fmt.Sprintf("terlewat %s", telat.Round(time.Second)))
			continue
		}
		s.bunyikan(ctx, b)
	}
	return nil
}

func (s *Scheduler) muat(ctx context.Context, now time.Time, tanggal string) error {
	data, err := s.sumber.LoadDataJadwal(ctx)
	if err != nil {
		return fmt.Errorf("muat jadwal: %w", err)
	}
	tercatat, err := s.sumber.JadwalTercatat(ctx, tanggal)
	if err != nil {
		return fmt.Errorf("muat riwayat: %w", err)
	}
	if tanggal == s.tanggal {
		// Bel yang sedang diputar belum punya riwayat; pertahankan catatan di memori.
		for id := range s.sudah {
			tercatat[id] = true
		}
	}
	s.tanggal = tanggal
	s.hasil = schedule.BelPadaTanggal(now, data)
	s.sudah = tercatat
	return nil
}

func (s *Scheduler) bunyikan(ctx context.Context, b schedule.Bel) {
	var files []string
	audio, err := s.sumber.UrutanBel(ctx, b.JenisBelID)
	if err != nil {
		// Tanpa file, player memutar nada cadangan.
		s.log.Error("muat urutan bel gagal", "jenis_bel", b.JenisBelID, "err", err)
	}
	for _, a := range audio {
		files = append(files, filepath.Join(s.audioDir, a.File))
	}
	s.log.Info("bel berbunyi", "label", b.Label, "waktu", b.Waktu.Format("15:04"))
	s.pemutar.Play(player.Permintaan{
		Files:     files,
		Prioritas: player.Biasa,
		Selesai: func(status player.Status, pesan string) {
			s.catat(context.Background(), b, string(status), pesan)
		},
	})
}

func (s *Scheduler) catat(ctx context.Context, b schedule.Bel, status, pesan string) {
	err := s.sumber.InsertRiwayat(ctx, store.Riwayat{
		Waktu:      b.Waktu,
		JadwalID:   b.JadwalID,
		JenisBelID: b.JenisBelID,
		Label:      b.Label,
		Sumber:     store.SumberJadwal,
		Status:     status,
		Pesan:      pesan,
	})
	if err != nil {
		s.log.Error("simpan riwayat gagal", "label", b.Label, "err", err)
	}
}
