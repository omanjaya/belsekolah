package formatjadwal

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func contoh() Data {
	d := Baru("Normal")
	d.JenisBel = []JenisBel{{"Masuk", "#2563eb"}, {"Pulang", "#9333ea"}, {"Tak Dipakai", "#000000"}}
	d.Jadwal = []Bel{
		{Hari: 1, Jam: "07:00", JenisBel: "Masuk", Label: "Masuk"},
		{Hari: 1, Jam: "13:00", JenisBel: "Pulang", Label: "Pulang"},
		{Hari: 5, Jam: "07:00", JenisBel: "Masuk", Label: ""},
	}
	return d
}

func TestPengumuman_BacaTulisOmitempty(t *testing.T) {
	d := contoh()
	d.Jadwal[0].Pengumuman = "Jam ke-1 dimulai"
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `"pengumuman":"Jam ke-1 dimulai"`) {
		t.Fatalf("pengumuman tidak tertulis: %s", s)
	}
	if strings.Count(s, `"pengumuman"`) != 1 {
		t.Fatalf("omitempty gagal: %s", s)
	}
	got, err := Baca(b)
	if err != nil || got.Jadwal[0].Pengumuman != "Jam ke-1 dimulai" || got.Jadwal[1].Pengumuman != "" {
		t.Fatalf("Baca = %+v err=%v", got.Jadwal, err)
	}
}

func TestPengumuman_TerlaluPanjang(t *testing.T) {
	d := contoh()
	d.Jadwal[1].Pengumuman = strings.Repeat("é", 101)
	harapTidakValid(t, d.Validasi(), "bel ke-2: pengumuman maksimal 100 karakter")
	d.Jadwal[1].Pengumuman = "  " + strings.Repeat("é", 100) + "  "
	if err := d.Validasi(); err != nil {
		t.Fatal(err)
	}
}

func harapTidakValid(t *testing.T, err error, sub string) {
	t.Helper()
	var v *ErrorValidasi
	if !errors.As(err, &v) {
		t.Fatalf("err = %v (%T), mau *ErrorValidasi", err, err)
	}
	if !strings.Contains(v.Pesan, sub) {
		t.Fatalf("pesan %q tidak memuat %q", v.Pesan, sub)
	}
}

func TestBaca_RoundTrip(t *testing.T) {
	b, err := json.Marshal(contoh())
	if err != nil {
		t.Fatal(err)
	}
	d, err := Baca(b)
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "Normal" || len(d.Jadwal) != 3 || d.Jadwal[1].Jam != "13:00" {
		t.Fatalf("hasil baca salah: %+v", d)
	}
}

func TestBaca_Ditolak(t *testing.T) {
	besar := `{"format":"belsekolah-jadwal","versi":1,"nama":"` + strings.Repeat("a", MaksUkuran) + `"}`
	cases := []struct{ nama, isi, sub string }{
		{"bukan json", `ini bukan json`, "bukan file jadwal"},
		{"format lain", `{"format":"lain","versi":1,"nama":"x","jenisBel":[],"jadwal":[]}`, "bukan file jadwal"},
		{"versi baru", `{"format":"belsekolah-jadwal","versi":2,"nama":"x","jenisBel":[],"jadwal":[]}`, "versi"},
		{"terlalu besar", besar, "terlalu besar"},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			_, err := Baca([]byte(c.isi))
			harapTidakValid(t, err, c.sub)
		})
	}
}

func TestValidasi(t *testing.T) {
	cases := []struct {
		nama string
		ubah func(*Data)
		sub  string
	}{
		{"nama kosong", func(d *Data) { d.Nama = " " }, "nama"},
		{"hari salah", func(d *Data) { d.Jadwal[0].Hari = 8 }, "bel ke-1"},
		{"jam salah", func(d *Data) { d.Jadwal[1].Jam = "7:00" }, "bel ke-2"},
		{"jenis tak ada", func(d *Data) { d.Jadwal[2].JenisBel = "Hilang" }, "Hilang"},
		{"bel ganda", func(d *Data) { d.Jadwal[1].Jam = "07:00" }, "ganda"},
		{"label panjang", func(d *Data) { d.Jadwal[0].Label = strings.Repeat("x", 101) }, "label"},
		{"jenis ganda beda huruf", func(d *Data) { d.JenisBel[1].Nama = "masuk" }, "ganda"},
		{"warna salah", func(d *Data) { d.JenisBel[0].Warna = "biru" }, "warna"},
		{"terlalu banyak", func(d *Data) {
			d.Jadwal = nil
			for i := 0; i <= MaksBel; i++ {
				d.Jadwal = append(d.Jadwal, Bel{Hari: 1 + i%7, Jam: "07:00", JenisBel: "Masuk"})
			}
		}, "maksimal"},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			d := contoh()
			c.ubah(&d)
			harapTidakValid(t, d.Validasi(), c.sub)
		})
	}
	if err := contoh().Validasi(); err != nil {
		t.Fatalf("contoh valid ditolak: %v", err)
	}
}

func TestJenisDipakaiDanHariBerisi(t *testing.T) {
	d := contoh()
	j := d.JenisDipakai()
	if len(j) != 2 || j[0].Nama != "Masuk" || j[1].Nama != "Pulang" {
		t.Fatalf("JenisDipakai = %+v", j)
	}
	h := d.HariBerisi()
	if len(h) != 2 || h[0] != 1 || h[1] != 5 {
		t.Fatalf("HariBerisi = %v", h)
	}
}
