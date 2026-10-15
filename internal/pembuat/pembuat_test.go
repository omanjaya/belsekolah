package pembuat

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"belsekolah/internal/formatjadwal"
)

func smpNormal() Parameter {
	return Parameter{
		Jenjang: "SMP", Varian: "normal", HariSekolah: 5,
		JamMasuk: "07:00", LamaJP: 40, JumlahJP: 7, JumlahJPJumat: 5, JumlahJPSabtu: 5,
		Istirahat:    []Istirahat{{SetelahJP: 3, Menit: 15}, {SetelahJP: 6, Menit: 15}},
		UpacaraSenin: Upacara{Aktif: true, Menit: 40},
	}
}

// ringkas menulis bel satu hari sebagai "HH:MM Jenis|Label" dipisah koma.
func ringkas(d formatjadwal.Data, hari int) string {
	var s []string
	for _, b := range d.Jadwal {
		if b.Hari == hari {
			s = append(s, fmt.Sprintf("%s %s|%s", b.Jam, b.JenisBel, b.Label))
		}
	}
	return strings.Join(s, ", ")
}

func TestBuat_SMPNormal(t *testing.T) {
	d, err := Buat(smpNormal())
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "SMP Normal 5 hari" {
		t.Errorf("nama = %q", d.Nama)
	}
	senin := "07:00 Masuk|Upacara, 07:40 Ganti Pelajaran|Jam ke-1, 08:20 Ganti Pelajaran|Jam ke-2, " +
		"09:00 Ganti Pelajaran|Jam ke-3, 09:40 Istirahat|Istirahat, 09:55 Masuk|Masuk setelah istirahat, " +
		"10:35 Ganti Pelajaran|Jam ke-5, 11:15 Ganti Pelajaran|Jam ke-6, 11:55 Istirahat|Istirahat, " +
		"12:10 Masuk|Masuk setelah istirahat, 12:50 Pulang|Pulang"
	selasa := "07:00 Masuk|Masuk, 07:40 Ganti Pelajaran|Jam ke-2, 08:20 Ganti Pelajaran|Jam ke-3, " +
		"09:00 Istirahat|Istirahat, 09:15 Masuk|Masuk setelah istirahat, 09:55 Ganti Pelajaran|Jam ke-5, " +
		"10:35 Ganti Pelajaran|Jam ke-6, 11:15 Istirahat|Istirahat, 11:30 Masuk|Masuk setelah istirahat, 12:10 Pulang|Pulang"
	jumat := "07:00 Masuk|Masuk, 07:40 Ganti Pelajaran|Jam ke-2, 08:20 Ganti Pelajaran|Jam ke-3, " +
		"09:00 Istirahat|Istirahat, 09:15 Masuk|Masuk setelah istirahat, 09:55 Ganti Pelajaran|Jam ke-5, 10:35 Pulang|Pulang"
	for _, c := range []struct {
		hari int
		mau  string
	}{{1, senin}, {2, selasa}, {3, selasa}, {4, selasa}, {5, jumat}, {6, ""}, {7, ""}} {
		if got := ringkas(d, c.hari); got != c.mau {
			t.Errorf("hari %d:\n got %s\nmau %s", c.hari, got, c.mau)
		}
	}
	if err := d.Validasi(); err != nil {
		t.Fatalf("hasil tidak lolos validasi format: %v", err)
	}
	if len(d.JenisBel) != 4 {
		t.Fatalf("jenis bel = %+v", d.JenisBel)
	}
}

func TestBuat_EnamHariDanKegiatanJumat(t *testing.T) {
	p := smpNormal()
	p.HariSekolah = 6
	p.JumlahJPSabtu = 2
	p.UpacaraSenin.Aktif = false
	p.KegiatanJumat = Kegiatan{Aktif: true, Menit: 30, Label: ""}
	d, err := Buat(p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "SMP Normal 6 hari" {
		t.Errorf("nama = %q", d.Nama)
	}
	if got := ringkas(d, 1); !strings.HasPrefix(got, "07:00 Masuk|Masuk, 07:40 Ganti Pelajaran|Jam ke-2") {
		t.Errorf("Senin tanpa upacara = %s", got)
	}
	if got := ringkas(d, 5); !strings.HasPrefix(got, "07:00 Masuk|Kegiatan Jumat, 07:30 Ganti Pelajaran|Jam ke-1, 08:10 Ganti Pelajaran|Jam ke-2") {
		t.Errorf("Jumat dengan kegiatan = %s", got)
	}
	if got := ringkas(d, 6); got != "07:00 Masuk|Masuk, 07:40 Ganti Pelajaran|Jam ke-2, 08:20 Pulang|Pulang" {
		t.Errorf("Sabtu = %s", got)
	}
}

func TestBuat_Ujian(t *testing.T) {
	p := Parameter{Jenjang: "SMA", Varian: "ujian", HariSekolah: 5,
		Ujian: Ujian{JamMulai: "07:30", JumlahSesi: 2, LamaSesi: 120, Jeda: 30}}
	d, err := Buat(p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "SMA/SMK Ujian 5 hari" {
		t.Errorf("nama = %q", d.Nama)
	}
	mau := "07:30 Masuk|Sesi 1 dimulai, 09:30 Istirahat|Sesi 1 selesai, 10:00 Masuk|Sesi 2 dimulai, 12:00 Pulang|Ujian selesai"
	for h := 1; h <= 5; h++ {
		if got := ringkas(d, h); got != mau {
			t.Errorf("hari %d = %s", h, got)
		}
	}
}

func TestBuat_Ramadan(t *testing.T) {
	p := smpNormal()
	p.Varian = "ramadan"
	p.LamaJP = 30
	d, err := Buat(p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "SMP Ramadan 5 hari" {
		t.Errorf("nama = %q", d.Nama)
	}
	if got := ringkas(d, 2); !strings.HasPrefix(got, "07:00 Masuk|Masuk, 07:30 Ganti Pelajaran|Jam ke-2") {
		t.Errorf("Selasa Ramadan = %s", got)
	}
}

func TestBuat_Ditolak(t *testing.T) {
	cases := []struct {
		nama string
		ubah func(*Parameter)
		sub  string
	}{
		{"jenjang", func(p *Parameter) { p.Jenjang = "TK" }, "jenjang"},
		{"varian", func(p *Parameter) { p.Varian = "libur" }, "varian"},
		{"hari", func(p *Parameter) { p.HariSekolah = 4 }, "hari sekolah"},
		{"jam masuk", func(p *Parameter) { p.JamMasuk = "7.00" }, "jam masuk"},
		{"lama jp", func(p *Parameter) { p.LamaJP = 5 }, "lama jam pelajaran"},
		{"jumlah jp", func(p *Parameter) { p.JumlahJP = 13 }, "jumlah jam pelajaran"},
		{"jumlah jumat", func(p *Parameter) { p.JumlahJPJumat = 0 }, "Jumat"},
		{"istirahat banyak", func(p *Parameter) {
			p.Istirahat = append(p.Istirahat, Istirahat{SetelahJP: 1, Menit: 10})
		}, "istirahat"},
		{"istirahat tidak naik", func(p *Parameter) { p.Istirahat[1].SetelahJP = 3 }, "istirahat"},
		{"istirahat di akhir", func(p *Parameter) { p.Istirahat[1].SetelahJP = 7 }, "istirahat"},
		{"istirahat menit", func(p *Parameter) { p.Istirahat[0].Menit = 61 }, "istirahat"},
		{"upacara menit", func(p *Parameter) { p.UpacaraSenin.Menit = 5 }, "upacara"},
		{"tengah malam", func(p *Parameter) { p.JamMasuk = "22:00"; p.LamaJP = 90; p.JumlahJP = 12 }, "tengah malam"},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			p := smpNormal()
			c.ubah(&p)
			_, err := Buat(p)
			var v *formatjadwal.ErrorValidasi
			if !errors.As(err, &v) || !strings.Contains(v.Pesan, c.sub) {
				t.Fatalf("err = %v, mau ErrorValidasi memuat %q", err, c.sub)
			}
		})
	}
	u := Parameter{Jenjang: "SD", Varian: "ujian", HariSekolah: 5, Ujian: Ujian{JamMulai: "07:30", JumlahSesi: 5, LamaSesi: 60, Jeda: 10}}
	if _, err := Buat(u); err == nil || !strings.Contains(err.Error(), "sesi") {
		t.Fatalf("jumlah sesi 5 harus ditolak, err = %v", err)
	}
}
