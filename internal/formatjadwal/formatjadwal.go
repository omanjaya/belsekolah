// Package formatjadwal mendefinisikan file jadwal (ekspor/impor profil dan template).
package formatjadwal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"belsekolah/internal/schedule"
)

const (
	Format     = "belsekolah-jadwal"
	Versi      = 1
	MaksBel    = 1000
	MaksUkuran = 256 << 10
)

type JenisBel struct {
	Nama  string `json:"nama"`
	Warna string `json:"warna"`
}

type Bel struct {
	Hari     int    `json:"hari"`
	Jam      string `json:"jam"`
	JenisBel string `json:"jenisBel"`
	Label    string `json:"label"`
}

type Data struct {
	Format   string     `json:"format"`
	Versi    int        `json:"versi"`
	Nama     string     `json:"nama"`
	JenisBel []JenisBel `json:"jenisBel"`
	Jadwal   []Bel      `json:"jadwal"`
}

// ErrorValidasi menandai file atau isi yang tidak valid; Pesan aman ditampilkan ke pengguna.
type ErrorValidasi struct{ Pesan string }

func (e *ErrorValidasi) Error() string { return e.Pesan }

func salah(format string, a ...any) error {
	return &ErrorValidasi{Pesan: fmt.Sprintf(format, a...)}
}

// Baru membuat Data kosong berformat versi terbaru.
func Baru(nama string) Data {
	return Data{Format: Format, Versi: Versi, Nama: nama, JenisBel: []JenisBel{}, Jadwal: []Bel{}}
}

// Baca mengurai dan memvalidasi isi file jadwal.
func Baca(b []byte) (Data, error) {
	var d Data
	if len(b) > MaksUkuran {
		return d, salah("file jadwal terlalu besar (maksimal %d KB)", MaksUkuran>>10)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&d); err != nil || d.Format != Format {
		return Data{}, salah("bukan file jadwal bel sekolah")
	}
	if d.Versi != Versi {
		return Data{}, salah("versi file jadwal %d tidak didukung (aplikasi ini memakai versi %d)", d.Versi, Versi)
	}
	if err := d.Validasi(); err != nil {
		return Data{}, err
	}
	return d, nil
}

var polaWarna = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Validasi memeriksa isi Data dengan aturan yang sama seperti input manual.
func (d Data) Validasi() error {
	nama := strings.TrimSpace(d.Nama)
	if nama == "" || utf8.RuneCountInString(nama) > 100 {
		return salah("nama jadwal wajib diisi, maksimal 100 karakter")
	}
	if len(d.Jadwal) > MaksBel {
		return salah("maksimal %d bel dalam satu file jadwal", MaksBel)
	}
	jenis := make(map[string]bool, len(d.JenisBel))
	for _, j := range d.JenisBel {
		n := strings.TrimSpace(j.Nama)
		if n == "" || utf8.RuneCountInString(n) > 100 {
			return salah("nama jenis bel wajib diisi, maksimal 100 karakter")
		}
		if !polaWarna.MatchString(j.Warna) {
			return salah("warna jenis bel %q harus berformat #RRGGBB", j.Nama)
		}
		k := strings.ToLower(n)
		if jenis[k] {
			return salah("jenis bel %q ganda", j.Nama)
		}
		jenis[k] = true
	}
	waktu := make(map[string]bool, len(d.Jadwal))
	for i, b := range d.Jadwal {
		if b.Hari < 1 || b.Hari > 7 {
			return salah("bel ke-%d: hari harus 1 (Senin) sampai 7 (Minggu)", i+1)
		}
		if _, _, err := schedule.ParseJam(b.Jam); err != nil {
			return salah("bel ke-%d: %s", i+1, err.Error())
		}
		if utf8.RuneCountInString(strings.TrimSpace(b.Label)) > 100 {
			return salah("bel ke-%d: label maksimal 100 karakter", i+1)
		}
		if !jenis[strings.ToLower(strings.TrimSpace(b.JenisBel))] {
			return salah("bel ke-%d: jenis bel %q tidak ada di daftar jenis bel", i+1, b.JenisBel)
		}
		k := fmt.Sprintf("%d %s", b.Hari, b.Jam)
		if waktu[k] {
			return salah("bel ke-%d: bel ganda di hari dan jam yang sama (%s)", i+1, b.Jam)
		}
		waktu[k] = true
	}
	return nil
}

// JenisDipakai mengembalikan jenis bel yang dirujuk jadwal, sesuai urutan di d.JenisBel.
func (d Data) JenisDipakai() []JenisBel {
	dipakai := make(map[string]bool)
	for _, b := range d.Jadwal {
		dipakai[strings.ToLower(strings.TrimSpace(b.JenisBel))] = true
	}
	var out []JenisBel
	for _, j := range d.JenisBel {
		if dipakai[strings.ToLower(strings.TrimSpace(j.Nama))] {
			out = append(out, j)
		}
	}
	return out
}

// HariBerisi mengembalikan hari (1–7, naik) yang memiliki bel.
func (d Data) HariBerisi() []int {
	ada := make(map[int]bool)
	for _, b := range d.Jadwal {
		ada[b.Hari] = true
	}
	out := make([]int, 0, len(ada))
	for h := range ada {
		out = append(out, h)
	}
	sort.Ints(out)
	return out
}
