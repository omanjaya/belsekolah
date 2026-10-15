// Package pembuat menyusun jadwal bel dari parameter sekolah (template bawaan).
package pembuat

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/schedule"
)

// Nama jenis bel; sama dengan jenis bel awal di paket bawaan.
const (
	jenisMasuk     = "Masuk"
	jenisGanti     = "Ganti Pelajaran"
	jenisIstirahat = "Istirahat"
	jenisPulang    = "Pulang"
)

// Warna sama dengan jenisBelAwal di paket bawaan.
var jenisBel = []formatjadwal.JenisBel{
	{Nama: jenisMasuk, Warna: "#2563eb"},
	{Nama: jenisGanti, Warna: "#0d9488"},
	{Nama: jenisIstirahat, Warna: "#16a34a"},
	{Nama: jenisPulang, Warna: "#9333ea"},
}

type Istirahat struct {
	SetelahJP int `json:"setelahJP"`
	Menit     int `json:"menit"`
}

type Upacara struct {
	Aktif bool `json:"aktif"`
	Menit int  `json:"menit"`
}

type Kegiatan struct {
	Aktif bool   `json:"aktif"`
	Menit int    `json:"menit"`
	Label string `json:"label"`
}

type Ujian struct {
	JamMulai   string `json:"jamMulai"`
	JumlahSesi int    `json:"jumlahSesi"`
	LamaSesi   int    `json:"lamaSesi"`
	Jeda       int    `json:"jeda"`
}

type Parameter struct {
	Jenjang       string      `json:"jenjang"` // SD, SMP, SMA
	Varian        string      `json:"varian"`  // normal, ramadan, ujian
	HariSekolah   int         `json:"hariSekolah"`
	JamMasuk      string      `json:"jamMasuk"`
	LamaJP        int         `json:"lamaJP"`
	JumlahJP      int         `json:"jumlahJP"`
	JumlahJPJumat int         `json:"jumlahJPJumat"`
	JumlahJPSabtu int         `json:"jumlahJPSabtu"`
	Istirahat     []Istirahat `json:"istirahat"`
	UpacaraSenin  Upacara     `json:"upacaraSenin"`
	KegiatanJumat Kegiatan    `json:"kegiatanJumat"`
	Ujian         Ujian       `json:"ujian"`
}

var namaJenjang = map[string]string{"SD": "SD", "SMP": "SMP", "SMA": "SMA/SMK"}
var namaVarian = map[string]string{"normal": "Normal", "ramadan": "Ramadan", "ujian": "Ujian"}

func salah(format string, a ...any) error {
	return &formatjadwal.ErrorValidasi{Pesan: fmt.Sprintf(format, a...)}
}

func diLuar(v, min, maks int) bool { return v < min || v > maks }

func menitDari(jam, nama string) (int, error) {
	h, m, err := schedule.ParseJam(jam)
	if err != nil {
		return 0, salah("%s tidak valid, gunakan format HH:MM", nama)
	}
	return h*60 + m, nil
}

func (p *Parameter) validasi() error {
	if _, ok := namaJenjang[p.Jenjang]; !ok {
		return salah("jenjang harus SD, SMP, atau SMA")
	}
	if _, ok := namaVarian[p.Varian]; !ok {
		return salah("varian harus normal, ramadan, atau ujian")
	}
	if p.HariSekolah != 5 && p.HariSekolah != 6 {
		return salah("hari sekolah harus 5 atau 6")
	}
	if p.Varian == "ujian" {
		u := p.Ujian
		if _, err := menitDari(u.JamMulai, "jam mulai ujian"); err != nil {
			return err
		}
		switch {
		case diLuar(u.JumlahSesi, 1, 4):
			return salah("jumlah sesi ujian harus 1 sampai 4")
		case diLuar(u.LamaSesi, 30, 180):
			return salah("lama sesi ujian harus 30 sampai 180 menit")
		case diLuar(u.Jeda, 5, 60):
			return salah("jeda antarsesi harus 5 sampai 60 menit")
		}
		return nil
	}
	if _, err := menitDari(p.JamMasuk, "jam masuk"); err != nil {
		return err
	}
	switch {
	case diLuar(p.LamaJP, 10, 90):
		return salah("lama jam pelajaran harus 10 sampai 90 menit")
	case diLuar(p.JumlahJP, 1, 12):
		return salah("jumlah jam pelajaran harus 1 sampai 12")
	case diLuar(p.JumlahJPJumat, 1, 12):
		return salah("jumlah jam pelajaran hari Jumat harus 1 sampai 12")
	case p.HariSekolah == 6 && diLuar(p.JumlahJPSabtu, 1, 12):
		return salah("jumlah jam pelajaran hari Sabtu harus 1 sampai 12")
	case len(p.Istirahat) > 2:
		return salah("maksimal 2 kali istirahat")
	}
	sebelum := 0
	for i, s := range p.Istirahat {
		if s.SetelahJP <= sebelum || s.SetelahJP >= p.JumlahJP {
			return salah("istirahat ke-%d harus setelah jam pelajaran yang urut dan sebelum jam terakhir", i+1)
		}
		if diLuar(s.Menit, 5, 60) {
			return salah("lama istirahat ke-%d harus 5 sampai 60 menit", i+1)
		}
		sebelum = s.SetelahJP
	}
	if p.UpacaraSenin.Aktif && diLuar(p.UpacaraSenin.Menit, 10, 90) {
		return salah("lama upacara harus 10 sampai 90 menit")
	}
	if p.KegiatanJumat.Aktif {
		if diLuar(p.KegiatanJumat.Menit, 10, 90) {
			return salah("lama kegiatan Jumat harus 10 sampai 90 menit")
		}
		if utf8.RuneCountInString(strings.TrimSpace(p.KegiatanJumat.Label)) > 100 {
			return salah("nama kegiatan Jumat maksimal 100 karakter")
		}
	}
	return nil
}

// penyusun menampung bel satu hari dan mencatat bila jam melewati tengah malam.
type penyusun struct {
	hari    int
	bel     []formatjadwal.Bel
	lewatan bool
}

func (s *penyusun) tambah(menit int, jenis, label string) {
	if menit >= 24*60 {
		s.lewatan = true
		return
	}
	s.bel = append(s.bel, formatjadwal.Bel{
		Hari: s.hari, Jam: fmt.Sprintf("%02d:%02d", menit/60, menit%60), JenisBel: jenis, Label: label,
	})
}

func (p *Parameter) jumlahJPHari(hari int) int {
	switch hari {
	case 5:
		return p.JumlahJPJumat
	case 6:
		return p.JumlahJPSabtu
	}
	return p.JumlahJP
}

func (p *Parameter) hariPelajaran(s *penyusun) {
	t, _ := menitDari(p.JamMasuk, "jam masuk")
	label, tambahan := "Masuk", 0
	switch {
	case s.hari == 1 && p.UpacaraSenin.Aktif:
		label, tambahan = "Upacara", p.UpacaraSenin.Menit
	case s.hari == 5 && p.KegiatanJumat.Aktif:
		label, tambahan = strings.TrimSpace(p.KegiatanJumat.Label), p.KegiatanJumat.Menit
		if label == "" {
			label = "Kegiatan Jumat"
		}
	}
	s.tambah(t, jenisMasuk, label)
	if tambahan > 0 {
		t += tambahan
		s.tambah(t, jenisGanti, "Jam ke-1")
	}
	istirahat := make(map[int]int, len(p.Istirahat))
	for _, i := range p.Istirahat {
		istirahat[i.SetelahJP] = i.Menit
	}
	jumlah := p.jumlahJPHari(s.hari)
	for k := 1; k <= jumlah; k++ {
		t += p.LamaJP // akhir JP k
		if k == jumlah {
			s.tambah(t, jenisPulang, "Pulang")
			break
		}
		if m, ok := istirahat[k]; ok {
			s.tambah(t, jenisIstirahat, "Istirahat")
			t += m
			s.tambah(t, jenisMasuk, "Masuk setelah istirahat")
		} else {
			s.tambah(t, jenisGanti, fmt.Sprintf("Jam ke-%d", k+1))
		}
	}
}

func (p *Parameter) hariUjian(s *penyusun) {
	t, _ := menitDari(p.Ujian.JamMulai, "jam mulai ujian")
	for i := 1; i <= p.Ujian.JumlahSesi; i++ {
		s.tambah(t, jenisMasuk, fmt.Sprintf("Sesi %d dimulai", i))
		t += p.Ujian.LamaSesi
		if i == p.Ujian.JumlahSesi {
			s.tambah(t, jenisPulang, "Ujian selesai")
			break
		}
		s.tambah(t, jenisIstirahat, fmt.Sprintf("Sesi %d selesai", i))
		t += p.Ujian.Jeda
	}
}

// Buat menyusun jadwal Senin sampai hari sekolah terakhir dari parameter.
func Buat(p Parameter) (formatjadwal.Data, error) {
	if err := p.validasi(); err != nil {
		return formatjadwal.Data{}, err
	}
	d := formatjadwal.Baru(fmt.Sprintf("%s %s %d hari", namaJenjang[p.Jenjang], namaVarian[p.Varian], p.HariSekolah))
	d.JenisBel = append([]formatjadwal.JenisBel(nil), jenisBel...)
	for hari := 1; hari <= p.HariSekolah; hari++ {
		s := &penyusun{hari: hari}
		if p.Varian == "ujian" {
			p.hariUjian(s)
		} else {
			p.hariPelajaran(s)
		}
		if s.lewatan {
			return formatjadwal.Data{}, salah("jadwal melewati tengah malam; kurangi jumlah atau lama jam pelajaran")
		}
		d.Jadwal = append(d.Jadwal, s.bel...)
	}
	if err := d.Validasi(); err != nil {
		return formatjadwal.Data{}, err
	}
	return d, nil
}
