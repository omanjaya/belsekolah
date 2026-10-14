// Package player memutar urutan file audio satu per satu dengan antrean.
// Bel darurat memotong suara yang sedang diputar dan mengosongkan antrean.
package player

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Status string

// Nilai Status sama dengan status pada riwayat bel.
const (
	Berhasil   Status = "berhasil"
	Cadangan   Status = "cadangan"
	Gagal      Status = "gagal"
	Dihentikan Status = "dihentikan"
)

type Prioritas int

const (
	Biasa Prioritas = iota
	Darurat
	// Tes adalah pemutaran uji dari panel admin. Tidak pernah memotong apa pun,
	// dan dibuang bila bel Biasa/Darurat datang agar bel asli tidak tertunda.
	Tes
)

// Permintaan adalah satu bel: beberapa file diputar berurutan.
// Selesai (boleh nil) dipanggil tepat sekali: dari goroutine player, atau
// secara sinkron dari goroutine yang memanggil Play/Stop bila permintaan
// dibuang dari antrean (juga saat Jalankan berhenti).
type Permintaan struct {
	Files     []string
	Prioritas Prioritas
	Selesai   func(status Status, pesan string)
}

// Runner memutar satu file sampai selesai, atau berhenti saat ctx dibatalkan.
type Runner interface {
	Run(ctx context.Context, file string, volume int) error
}

type Player struct {
	runner   Runner
	cadangan string
	sinyal   chan struct{}

	mu      sync.Mutex
	antrean []Permintaan
	batal   context.CancelFunc // membatalkan permintaan yang sedang diputar
	sedang  Prioritas          // prioritas permintaan yang sedang diputar (valid bila batal != nil)
	volume  int
}

func New(runner Runner, fileCadangan string) *Player {
	return &Player{
		runner:   runner,
		cadangan: fileCadangan,
		sinyal:   make(chan struct{}, 1),
		volume:   100,
	}
}

// SetVolume mengatur volume 0–100 untuk permintaan berikutnya.
func (p *Player) SetVolume(v int) {
	v = max(0, min(100, v))
	p.mu.Lock()
	p.volume = v
	p.mu.Unlock()
}

func (p *Player) Play(req Permintaan) {
	p.mu.Lock()
	var dibuang []Permintaan
	alasan := "dihentikan oleh bel darurat"
	if req.Prioritas == Darurat {
		dibuang = p.antrean
		p.antrean = []Permintaan{req}
		if p.batal != nil {
			p.batal()
		}
	} else {
		if req.Prioritas == Biasa {
			// Tes tidak boleh menunda bel asli.
			alasan = "dihentikan oleh bel"
			sisa := p.antrean[:0:0]
			for _, q := range p.antrean {
				if q.Prioritas == Tes {
					dibuang = append(dibuang, q)
				} else {
					sisa = append(sisa, q)
				}
			}
			p.antrean = sisa
			if p.batal != nil && p.sedang == Tes {
				p.batal()
			}
		}
		p.antrean = append(p.antrean, req)
	}
	p.mu.Unlock()
	for _, d := range dibuang {
		selesai(d, Dihentikan, alasan)
	}
	p.bangunkan()
}

// Stop menghentikan suara yang sedang diputar dan mengosongkan antrean.
func (p *Player) Stop() {
	p.mu.Lock()
	dibuang := p.antrean
	p.antrean = nil
	if p.batal != nil {
		p.batal()
	}
	p.mu.Unlock()
	for _, d := range dibuang {
		selesai(d, Dihentikan, "dihentikan oleh admin")
	}
}

// Jalankan memproses antrean sampai ctx selesai.
func (p *Player) Jalankan(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			p.kosongkan()
			return
		}
		p.mu.Lock()
		if len(p.antrean) == 0 {
			p.mu.Unlock()
			select {
			case <-ctx.Done():
				p.kosongkan()
				return
			case <-p.sinyal:
				continue
			}
		}
		req := p.antrean[0]
		p.antrean = p.antrean[1:]
		reqCtx, batal := context.WithCancel(ctx)
		p.batal = batal
		p.sedang = req.Prioritas
		volume := p.volume
		p.mu.Unlock()

		status, pesan := p.putar(reqCtx, req.Files, volume)

		p.mu.Lock()
		p.batal = nil
		p.mu.Unlock()
		batal()
		selesai(req, status, pesan)
	}
}

// kosongkan membuang seluruh antrean saat aplikasi berhenti.
func (p *Player) kosongkan() {
	p.mu.Lock()
	dibuang := p.antrean
	p.antrean = nil
	p.mu.Unlock()
	for _, d := range dibuang {
		selesai(d, Dihentikan, "aplikasi berhenti")
	}
}

func (p *Player) bangunkan() {
	select {
	case p.sinyal <- struct{}{}:
	default:
	}
}

func selesai(req Permintaan, status Status, pesan string) {
	if req.Selesai != nil {
		req.Selesai(status, pesan)
	}
}

var peringkat = map[Status]int{Berhasil: 0, Cadangan: 1, Gagal: 2, Dihentikan: 3}

func lebihBuruk(a, b Status) Status {
	if peringkat[b] > peringkat[a] {
		return b
	}
	return a
}

func (p *Player) putar(ctx context.Context, files []string, volume int) (Status, string) {
	if len(files) == 0 {
		return p.putarCadangan(ctx, volume, "jenis bel belum memiliki suara")
	}
	status := Berhasil
	var pesan []string
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			st, ps := p.putarCadangan(ctx, volume, fmt.Sprintf("file %s tidak bisa dibaca", filepath.Base(f)))
			status = lebihBuruk(status, st)
			pesan = append(pesan, ps)
			if st == Dihentikan {
				return Dihentikan, strings.Join(pesan, "; ")
			}
			continue
		}
		err := p.runner.Run(ctx, f, volume)
		if err != nil && ctx.Err() == nil {
			err = p.runner.Run(ctx, f, volume) // coba ulang sekali
		}
		if ctx.Err() != nil {
			return Dihentikan, "pemutaran dihentikan"
		}
		if err != nil {
			st, ps := p.putarCadangan(ctx, volume, fmt.Sprintf("gagal memutar %s: %v", filepath.Base(f), err))
			status = lebihBuruk(status, st)
			pesan = append(pesan, ps)
			if st == Dihentikan {
				return Dihentikan, strings.Join(pesan, "; ")
			}
		}
	}
	return status, strings.Join(pesan, "; ")
}

func (p *Player) putarCadangan(ctx context.Context, volume int, alasan string) (Status, string) {
	err := p.runner.Run(ctx, p.cadangan, volume)
	if ctx.Err() != nil {
		return Dihentikan, alasan
	}
	if err != nil {
		return Gagal, alasan + "; nada cadangan juga gagal: " + err.Error()
	}
	return Cadangan, alasan + ", diganti nada cadangan"
}
