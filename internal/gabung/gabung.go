// Package gabung mengelompokkan bel masukan terhadap jadwal sebuah profil (baru/kembar/bentrok)
// dan menyusun rencana penggabungan. Murni, tanpa database.
package gabung

import (
	"sort"
	"strings"
)

type Status string

const (
	Baru    Status = "baru"
	Kembar  Status = "kembar"
	Bentrok Status = "bentrok"
)

// Isi adalah isi satu slot. JenisBelID 0 berarti jenis belum dikenal (belum dipetakan atau akan dibuat baru).
type Isi struct {
	JenisBelID int64
	Label      string
}

// Lama adalah bel yang sudah ada di profil tujuan.
type Lama struct {
	JadwalID int64
	Hari     int
	Jam      string
	Isi
}

// Masuk adalah bel masukan; slot (Hari, Jam) dianggap unik di antara masukan.
type Masuk struct {
	Hari int
	Jam  string
	Isi
}

// Slot menunjuk hari dan jam di profil tujuan.
type Slot struct {
	Hari int    `json:"hari"`
	Jam  string `json:"jam"`
}

// Baris adalah hasil klasifikasi satu bel masukan; Indeks menunjuk posisinya di slice masukan.
type Baris struct {
	Indeks int
	Hari   int
	Jam    string
	Status Status
	Lama   *Lama
	Baru   Masuk
}

func sama(a, b Isi) bool {
	return a.JenisBelID != 0 && a.JenisBelID == b.JenisBelID &&
		strings.TrimSpace(a.Label) == strings.TrimSpace(b.Label)
}

// Klasifikasi mengelompokkan setiap bel masukan, diurutkan menurut hari lalu jam.
func Klasifikasi(lama []Lama, masuk []Masuk) []Baris {
	ada := make(map[Slot]*Lama, len(lama))
	for i := range lama {
		ada[Slot{lama[i].Hari, lama[i].Jam}] = &lama[i]
	}
	out := make([]Baris, 0, len(masuk))
	for i, m := range masuk {
		b := Baris{Indeks: i, Hari: m.Hari, Jam: m.Jam, Baru: m, Status: Baru}
		if l, ok := ada[Slot{m.Hari, m.Jam}]; ok {
			salinan := *l
			b.Lama = &salinan
			if sama(m.Isi, l.Isi) {
				b.Status = Kembar
			} else {
				b.Status = Bentrok
			}
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Hari != out[j].Hari {
			return out[i].Hari < out[j].Hari
		}
		return out[i].Jam < out[j].Jam
	})
	return out
}

// Rencana adalah tindakan penggabungan. Dilewati = kembar + bentrok yang dipertahankan.
type Rencana struct {
	Tambah   []Baris
	Timpa    []Baris
	Dilewati int
}

// Rencanakan: baru ditambah, kembar dilewati, bentrok ditimpa hanya bila slotnya ada di timpa.
// Slot di timpa yang bukan bentrok diabaikan.
func Rencanakan(baris []Baris, timpa []Slot) Rencana {
	pilih := make(map[Slot]bool, len(timpa))
	for _, s := range timpa {
		pilih[s] = true
	}
	r := Rencana{Tambah: []Baris{}, Timpa: []Baris{}}
	for _, b := range baris {
		switch {
		case b.Status == Baru:
			r.Tambah = append(r.Tambah, b)
		case b.Status == Bentrok && pilih[Slot{b.Hari, b.Jam}]:
			r.Timpa = append(r.Timpa, b)
		default:
			r.Dilewati++
		}
	}
	return r
}
