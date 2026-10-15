package pengumuman

import "testing"

func TestNormal(t *testing.T) {
	for in, mau := range map[string]string{
		"Jam ke-3":                    "jam ke 3",
		"  Akhir pelajaran (pulang) ": "akhir pelajaran pulang",
		"Sesi 1: Dimulai/IND":         "sesi 1 dimulai ind",
		"a__b..c":                     "a b c",
	} {
		if got := Normal(in); got != mau {
			t.Errorf("Normal(%q) = %q, mau %q", in, got, mau)
		}
	}
}

func TestCocokkan(t *testing.T) {
	suara := []Kandidat{
		{1, "Jam ke-1 dimulai"}, {10, "Jam ke-10 dimulai"}, {2, "Jam ke-2 dimulai"},
		{20, "Istirahat"}, {21, "Istirahat pertama"}, {22, "Istirahat kedua"},
		{30, "Akhir pelajaran (pulang)"}, {40, "Upacara bendera dimulai"},
		{50, "Bel masuk sekolah"}, {60, "Ujian (Waktu Telah Habis)"},
		{70, "Kegiatan Pramuka 5 menit"}, {71, "Kegiatan Pramuka akhir"},
	}
	kasus := []struct {
		ket string
		id  int64
		ok  bool
	}{
		{"Jam ke-1", 1, true}, // tingkat 2, bukan "jam ke 10 dimulai" (batas kata)
		{"Jam ke-10", 10, true},
		{"jam ke 2", 2, true},
		{"Istirahat", 20, true}, // tingkat 1 mengalahkan tingkat 3
		{"Pulang", 30, true},    // padanan → tingkat 3
		{"Upacara", 40, true},   // padanan → tingkat 2
		{"Masuk", 50, true},     // padanan → tingkat 1
		{"Ujian selesai", 60, true},
		{"Kegiatan Pramuka", 0, false}, // dua kandidat di tingkat 3 → ambigu
		{"Jam ke-7", 0, false},
		{"", 0, false},
		{"   ", 0, false},
	}
	for _, k := range kasus {
		id, ok := Cocokkan(k.ket, suara)
		if id != k.id || ok != k.ok {
			t.Errorf("Cocokkan(%q) = (%d, %v), mau (%d, %v)", k.ket, id, ok, k.id, k.ok)
		}
	}
}

func TestCocokkan_AmbiguTingkatSama(t *testing.T) {
	_, ok := Cocokkan("Istirahat", []Kandidat{{1, "Istirahat"}, {2, "ISTIRAHAT"}})
	if ok {
		t.Fatal("dua kandidat tingkat 1 harus dianggap ambigu")
	}
}
