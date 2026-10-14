package schedule

import (
	"testing"
	"time"
)

var wib = time.FixedZone("WIB", 7*3600)

func tgl(s string) time.Time {
	t, err := time.ParseInLocation(FormatTanggal, s, wib)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseJam(t *testing.T) {
	kasus := []struct {
		masukan    string
		jam, menit int
		harusGagal bool
	}{
		{"07:00", 7, 0, false},
		{"23:59", 23, 59, false},
		{"00:00", 0, 0, false},
		{"7:00", 0, 0, true},
		{"24:00", 0, 0, true},
		{"07:60", 0, 0, true},
		{"0700", 0, 0, true},
		{"", 0, 0, true},
	}
	for _, k := range kasus {
		j, m, err := ParseJam(k.masukan)
		if k.harusGagal {
			if err == nil {
				t.Errorf("ParseJam(%q) seharusnya gagal", k.masukan)
			}
			continue
		}
		if err != nil || j != k.jam || m != k.menit {
			t.Errorf("ParseJam(%q) = %d,%d,%v; mau %d,%d", k.masukan, j, m, err, k.jam, k.menit)
		}
	}
}

func TestHari(t *testing.T) {
	// 2026-10-12 adalah Senin, 2026-10-18 adalah Minggu.
	if h := Hari(tgl("2026-10-12")); h != 1 {
		t.Errorf("Senin = %d, mau 1", h)
	}
	if h := Hari(tgl("2026-10-18")); h != 7 {
		t.Errorf("Minggu = %d, mau 7", h)
	}
}

func dataContoh() Data {
	return Data{
		ProfilDefaultID: 1,
		Jadwal: []Jadwal{
			{ID: 3, ProfilID: 1, Hari: 1, Jam: "12:00", JenisBelID: 30, Label: "Pulang"},
			{ID: 1, ProfilID: 1, Hari: 1, Jam: "07:00", JenisBelID: 10, Label: "Masuk"},
			{ID: 2, ProfilID: 1, Hari: 1, Jam: "09:30", JenisBelID: 20, Label: "Istirahat"},
			{ID: 4, ProfilID: 1, Hari: 2, Jam: "07:15", JenisBelID: 10, Label: "Masuk Selasa"},
			{ID: 5, ProfilID: 2, Hari: 1, Jam: "08:00", JenisBelID: 10, Label: "Masuk Ujian"},
			{ID: 6, ProfilID: 1, Hari: 1, Jam: "rusak", JenisBelID: 10, Label: "Jam rusak"},
		},
		Periode: []Periode{
			{ID: 1, Mulai: "2026-10-19", Selesai: "2026-10-23", ProfilID: 2, Keterangan: "Ujian"},
		},
		Libur: []Libur{
			{ID: 1, Mulai: "2026-10-19", Selesai: "2026-10-19", Keterangan: "Libur tabrak ujian"},
			{ID: 2, Mulai: "2026-12-29", Selesai: "2027-01-02", Keterangan: "Libur akhir tahun"},
		},
	}
}

func labels(h Hasil) []string {
	var out []string
	for _, b := range h.Bel {
		out = append(out, b.Label)
	}
	return out
}

func samaString(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBelPadaTanggal_ProfilDefaultTerurut(t *testing.T) {
	h := BelPadaTanggal(tgl("2026-10-12"), dataContoh()) // Senin
	if h.ProfilID != 1 || h.Libur != nil {
		t.Fatalf("profil=%d libur=%v, mau profil 1 tanpa libur", h.ProfilID, h.Libur)
	}
	mau := []string{"Masuk", "Istirahat", "Pulang"}
	if !samaString(labels(h), mau) {
		t.Fatalf("label = %v, mau %v", labels(h), mau)
	}
	b := h.Bel[0]
	waktuMau := time.Date(2026, 10, 12, 7, 0, 0, 0, wib)
	if !b.Waktu.Equal(waktuMau) || b.Waktu.Location() != wib {
		t.Errorf("waktu = %v, mau %v di zona WIB", b.Waktu, waktuMau)
	}
	if b.JadwalID != 1 || b.JenisBelID != 10 {
		t.Errorf("bel pertama = %+v", b)
	}
}

func TestBelPadaTanggal_HariLain(t *testing.T) {
	h := BelPadaTanggal(tgl("2026-10-13"), dataContoh()) // Selasa
	if !samaString(labels(h), []string{"Masuk Selasa"}) {
		t.Fatalf("label = %v", labels(h))
	}
}

func TestBelPadaTanggal_HariTanpaJadwal(t *testing.T) {
	h := BelPadaTanggal(tgl("2026-10-18"), dataContoh()) // Minggu
	if h.ProfilID != 1 || len(h.Bel) != 0 || h.Libur != nil {
		t.Fatalf("hasil = %+v, mau profil 1 tanpa bel", h)
	}
}

func TestBelPadaTanggal_LiburMengalahkanPeriode(t *testing.T) {
	h := BelPadaTanggal(tgl("2026-10-19"), dataContoh())
	if h.Libur == nil || h.Libur.ID != 1 || len(h.Bel) != 0 || h.ProfilID != 0 {
		t.Fatalf("hasil = %+v, mau libur ID 1 tanpa bel", h)
	}
}

func TestBelPadaTanggal_PeriodeKhususInklusif(t *testing.T) {
	// 2026-10-26 Senin (setelah periode), 2026-10-19..23 periode ujian.
	// Senin dalam periode tertutup libur, jadi uji batas akhir dengan data tanpa libur.
	d := dataContoh()
	d.Libur = nil
	for _, s := range []string{"2026-10-19", "2026-10-23"} {
		h := BelPadaTanggal(tgl(s), d)
		if h.ProfilID != 2 {
			t.Errorf("%s profil = %d, mau 2", s, h.ProfilID)
		}
	}
	if h := BelPadaTanggal(tgl("2026-10-19"), d); !samaString(labels(h), []string{"Masuk Ujian"}) {
		t.Errorf("Senin ujian label = %v", labels(h))
	}
	if h := BelPadaTanggal(tgl("2026-10-24"), d); h.ProfilID != 1 {
		t.Errorf("setelah periode profil = %d, mau 1", h.ProfilID)
	}
}

func TestBelPadaTanggal_LiburLintasTahun(t *testing.T) {
	for _, s := range []string{"2026-12-29", "2027-01-01", "2027-01-02"} {
		if h := BelPadaTanggal(tgl(s), dataContoh()); h.Libur == nil || h.Libur.ID != 2 {
			t.Errorf("%s seharusnya libur akhir tahun, dapat %+v", s, h)
		}
	}
	if h := BelPadaTanggal(tgl("2027-01-03"), dataContoh()); h.Libur != nil {
		t.Errorf("2027-01-03 seharusnya bukan libur")
	}
}
