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
	GetAudio(ctx context.Context, id int64) (store.Audio, error)
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

// StatusBel adalah satu bel hari ini beserta apakah sudah dibunyikan atau dicatat.
type StatusBel struct {
	JadwalID   int64
	Waktu      time.Time
	JenisBelID int64
	Label      string
	Sudah      bool
}

// Status adalah gambaran jadwal hari ini menurut pemeriksaan terakhir.
type Status struct {
	Siap     bool // false sebelum data dimuat pertama kali
	Tanggal  string
	ProfilID int64
	Libur    *schedule.Libur
	Bel      []StatusBel
}

const statusTerlewat = "terlewat"

// Status mengembalikan salinan keadaan jadwal hari ini.
func (s *Scheduler) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{Siap: s.tanggal != "", Tanggal: s.tanggal, ProfilID: s.hasil.ProfilID}
	if l := s.hasil.Libur; l != nil {
		salinan := *l
		st.Libur = &salinan
	}
	for _, b := range s.hasil.Bel {
		st.Bel = append(st.Bel, StatusBel{
			JadwalID: b.JadwalID, Waktu: b.Waktu, JenisBelID: b.JenisBelID,
			Label: b.Label, Sudah: s.sudah[b.JadwalID],
		})
	}
	return st
}

// Run memanggil Tick setiap detik sampai ctx selesai. Error yang sama dicatat paling sering sekali per menit.
func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	var errTerakhir string
	var dicatat time.Time
	for {
		if err := s.Tick(ctx); err != nil {
			if err.Error() != errTerakhir || time.Since(dicatat) >= time.Minute {
				s.log.Error("pemeriksaan jadwal gagal", "err", err)
				errTerakhir, dicatat = err.Error(), time.Now()
			}
		} else if errTerakhir != "" {
			s.log.Info("pemeriksaan jadwal pulih")
			errTerakhir = ""
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
			s.catat(ctx, store.Riwayat{
				Waktu: b.Waktu, JadwalID: b.JadwalID, JenisBelID: b.JenisBelID, Label: b.Label,
				Sumber: store.SumberJadwal, Status: statusTerlewat,
				Pesan: fmt.Sprintf("terlewat %s", telat.Round(time.Second)),
			})
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
	s.log.Info("bel berbunyi", "label", b.Label, "waktu", b.Waktu.Format("15:04"))
	s.putar(ctx, store.Riwayat{
		Waktu: b.Waktu, JadwalID: b.JadwalID, JenisBelID: b.JenisBelID,
		Label: b.Label, Sumber: store.SumberJadwal,
	}, player.Biasa, b.PengumumanID)
}

// BunyikanManual membunyikan jenis bel di luar jadwal dan mencatatnya di riwayat.
// Aman dipanggil dari goroutine mana pun; tidak memakai kunci scheduler.
func (s *Scheduler) BunyikanManual(ctx context.Context, jenisBelID int64, label string, darurat bool) {
	sumber, prioritas := store.SumberManual, player.Biasa
	if darurat {
		sumber, prioritas = store.SumberDarurat, player.Darurat
	}
	s.log.Info("bel manual", "label", label, "darurat", darurat)
	s.putar(ctx, store.Riwayat{
		Waktu: s.clock.Now(), JenisBelID: jenisBelID, Label: label, Sumber: sumber,
	}, prioritas, 0)
}

func (s *Scheduler) putar(ctx context.Context, r store.Riwayat, prioritas player.Prioritas, pengumumanID int64) {
	files := s.FileUrutan(ctx, r.JenisBelID)
	if pengumumanID != 0 {
		if a, err := s.sumber.GetAudio(ctx, pengumumanID); err != nil {
			s.log.Error("muat suara pengumuman gagal", "audio", pengumumanID, "err", err)
		} else {
			files = append(files, filepath.Join(s.audioDir, filepath.Base(a.File)))
		}
	}
	s.pemutar.Play(player.Permintaan{
		Files:     files,
		Prioritas: prioritas,
		Selesai: func(status player.Status, pesan string) {
			r.Status, r.Pesan = string(status), pesan
			s.catat(context.Background(), r)
		},
	})
}

// FileUrutan mengembalikan path file audio sebuah jenis bel. Jika gagal dimuat,
// hasilnya kosong dan pemutar akan memakai nada cadangan.
func (s *Scheduler) FileUrutan(ctx context.Context, jenisBelID int64) []string {
	audio, err := s.sumber.UrutanBel(ctx, jenisBelID)
	if err != nil {
		s.log.Error("muat urutan bel gagal", "jenis_bel", jenisBelID, "err", err)
	}
	var files []string
	for _, a := range audio {
		files = append(files, filepath.Join(s.audioDir, filepath.Base(a.File)))
	}
	return files
}

func (s *Scheduler) catat(ctx context.Context, r store.Riwayat) {
	if err := s.sumber.InsertRiwayat(ctx, r); err != nil {
		s.log.Error("simpan riwayat gagal", "label", r.Label, "err", err)
	}
}
