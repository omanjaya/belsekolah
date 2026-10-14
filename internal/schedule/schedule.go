// Package schedule menentukan bel apa saja yang berbunyi pada suatu tanggal.
// Paket ini murni: tidak membaca jam sistem, database, maupun file.
package schedule

import (
	"fmt"
	"sort"
	"time"
)

// FormatTanggal adalah format tanggal yang dipakai di seluruh aplikasi.
const FormatTanggal = "2006-01-02"

// Jadwal adalah satu bel pada hari tertentu dalam sebuah profil.
type Jadwal struct {
	ID         int64
	ProfilID   int64
	Hari       int    // 1=Senin ... 7=Minggu
	Jam        string // "HH:MM"
	JenisBelID int64
	Label      string
}

// Periode membuat rentang tanggal memakai profil tertentu.
type Periode struct {
	ID         int64
	Mulai      string // YYYY-MM-DD, inklusif
	Selesai    string // YYYY-MM-DD, inklusif
	ProfilID   int64
	Keterangan string
}

// Libur adalah rentang tanggal tanpa bel terjadwal.
type Libur struct {
	ID         int64
	Mulai      string // YYYY-MM-DD, inklusif
	Selesai    string // YYYY-MM-DD, inklusif
	Keterangan string
}

// Data adalah seluruh data yang dibutuhkan untuk menentukan bel.
type Data struct {
	ProfilDefaultID int64
	Periode         []Periode
	Libur           []Libur
	Jadwal          []Jadwal
}

// Bel adalah satu bel yang harus berbunyi pada waktu tertentu.
type Bel struct {
	JadwalID   int64
	Waktu      time.Time
	JenisBelID int64
	Label      string
}

// Hasil adalah bel pada suatu tanggal beserta profil yang dipakai.
type Hasil struct {
	ProfilID int64  // 0 jika libur
	Libur    *Libur // nil jika bukan hari libur
	Bel      []Bel  // terurut berdasarkan waktu
}

// ParseJam mengurai jam berformat HH:MM.
func ParseJam(s string) (jam, menit int, err error) {
	t, err := time.Parse("15:04", s)
	if err != nil || len(s) != 5 {
		return 0, 0, fmt.Errorf("jam tidak valid %q, gunakan format HH:MM", s)
	}
	return t.Hour(), t.Minute(), nil
}

// Hari mengembalikan 1=Senin ... 7=Minggu.
func Hari(t time.Time) int {
	h := int(t.Weekday())
	if h == 0 {
		return 7
	}
	return h
}

func dalamRentang(tgl, mulai, selesai string) bool {
	return mulai <= tgl && tgl <= selesai
}

// BelPadaTanggal menerapkan aturan: libur → periode khusus → profil default.
// Waktu bel memakai zona waktu dari tgl.
func BelPadaTanggal(tgl time.Time, d Data) Hasil {
	kunci := tgl.Format(FormatTanggal)
	for i := range d.Libur {
		if dalamRentang(kunci, d.Libur[i].Mulai, d.Libur[i].Selesai) {
			l := d.Libur[i]
			return Hasil{Libur: &l}
		}
	}

	profil := d.ProfilDefaultID
	for _, p := range d.Periode {
		if dalamRentang(kunci, p.Mulai, p.Selesai) {
			profil = p.ProfilID
			break
		}
	}

	hari := Hari(tgl)
	var bel []Bel
	for _, j := range d.Jadwal {
		if j.ProfilID != profil || j.Hari != hari {
			continue
		}
		h, m, err := ParseJam(j.Jam)
		if err != nil {
			continue
		}
		bel = append(bel, Bel{
			JadwalID:   j.ID,
			Waktu:      time.Date(tgl.Year(), tgl.Month(), tgl.Day(), h, m, 0, 0, tgl.Location()),
			JenisBelID: j.JenisBelID,
			Label:      j.Label,
		})
	}
	sort.Slice(bel, func(a, b int) bool {
		if !bel[a].Waktu.Equal(bel[b].Waktu) {
			return bel[a].Waktu.Before(bel[b].Waktu)
		}
		return bel[a].JadwalID < bel[b].JadwalID
	})
	return Hasil{ProfilID: profil, Bel: bel}
}
