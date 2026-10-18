package store

import (
	"errors"
	"testing"
	"time"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/gabung"
	"belsekolah/internal/schedule"
)

func buatSuara(t *testing.T, s *Store, nama string) int64 {
	t.Helper()
	return harus(s.CreateAudio(ctx, Audio{Nama: nama, Sumber: "upload", File: "x.mp3", Dibuat: time.Now()}))(t)
}

func TestCocokkanPengumuman(t *testing.T) {
	s := bukaStore(t)
	id := buatSuara(t, s, "Jam ke-1 dimulai")
	r := harus(s.CocokkanPengumuman(ctx, "Jam ke-1"))(t)
	if r == nil || r.ID != id || r.Nama != "Jam ke-1 dimulai" {
		t.Fatalf("hasil = %+v", r)
	}
	if r := harus(s.CocokkanPengumuman(ctx, "Istirahat"))(t); r != nil {
		t.Fatalf("seharusnya nil: %+v", r)
	}
}

func TestSaranDanTerapkanPengumuman(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	satu := buatSuara(t, s, "Jam ke-1 dimulai")
	dua := buatSuara(t, s, "Jam ke-2 dimulai")
	tambah := func(p int64, hari int, jam, label string, peng int64) int64 {
		return harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: p, Hari: hari, Jam: jam, JenisBelID: jenis, Label: label, PengumumanID: peng}))(t)
	}
	b2 := tambah(profil, 2, "08:00", "Jam ke-2", 0)
	b1 := tambah(profil, 1, "07:00", "Jam ke-1", 0)
	terisi := tambah(profil, 1, "09:00", "Jam ke-1", satu)
	tambah(profil, 1, "10:00", "Tidak ada kecocokan", 0)
	lain := harus(s.CreateProfil(ctx, "Lain"))(t)
	bl := tambah(lain, 1, "07:00", "Jam ke-1", 0)

	saran := harus(s.SaranPengumuman(ctx, profil))(t)
	if len(saran) != 2 {
		t.Fatalf("saran = %+v", saran)
	}
	if saran[0] != (SaranPengumuman{JadwalID: b1, Hari: 1, Jam: "07:00", Label: "Jam ke-1", Suara: SuaraRingkas{ID: satu, Nama: "Jam ke-1 dimulai"}}) {
		t.Fatalf("saran[0] = %+v", saran[0])
	}
	if saran[1].JadwalID != b2 || saran[1].Suara.ID != dua {
		t.Fatalf("saran[1] = %+v", saran[1])
	}
	if _, err := s.SaranPengumuman(ctx, 9999); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v", err)
	}
	if k := harus(s.SaranPengumuman(ctx, harus(s.CreateProfil(ctx, "Kosong"))(t)))(t); k == nil || len(k) != 0 {
		t.Fatalf("kosong = %#v", k)
	}

	// Terapkan: b1 (kosong) berubah, baris terisi tidak ditimpa, baris profil lain diabaikan.
	n := harus(s.TerapkanPengumuman(ctx, profil, []IsiPengumuman{
		{JadwalID: b1, AudioID: dua}, {JadwalID: terisi, AudioID: dua}, {JadwalID: bl, AudioID: dua}}))(t)
	if n != 1 {
		t.Fatalf("n = %d", n)
	}
	if j := harus(s.GetJadwal(ctx, b1))(t); j.PengumumanID != dua {
		t.Fatalf("b1 = %+v", j)
	}
	if j := harus(s.GetJadwal(ctx, terisi))(t); j.PengumumanID != satu {
		t.Fatalf("terisi tertimpa: %+v", j)
	}
	if j := harus(s.GetJadwal(ctx, bl))(t); j.PengumumanID != 0 {
		t.Fatalf("profil lain berubah: %+v", j)
	}

	_, err := s.TerapkanPengumuman(ctx, profil, []IsiPengumuman{{JadwalID: b2, AudioID: 9999}})
	var v *ErrorValidasi
	if !errors.As(err, &v) {
		t.Fatalf("err = %v", err)
	}
	if j := harus(s.GetJadwal(ctx, b2))(t); j.PengumumanID != 0 {
		t.Fatal("b2 berubah padahal galat")
	}
	if _, err := s.TerapkanPengumuman(ctx, 9999, nil); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v", err)
	}
}

func TestEksporImpor_Pengumuman(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	satu := buatSuara(t, s, "Jam ke-1 dimulai")
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, Label: "X", PengumumanID: satu}))(t)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "08:00", JenisBelID: jenis, Label: "Y"}))(t)
	d := harus(s.EksporProfil(ctx, profil))(t)
	if d.Jadwal[0].Pengumuman != "Jam ke-1 dimulai" || d.Jadwal[1].Pengumuman != "" {
		t.Fatalf("ekspor = %+v", d.Jadwal)
	}
	baru := harus(s.ImporJadwal(ctx, d, "Salinan", nil))(t)
	l := harus(s.ListJadwal(ctx, baru))(t)
	if l[0].PengumumanID != satu || l[1].PengumumanID != 0 {
		t.Fatalf("impor = %+v", l)
	}
}

func dataPengumuman(peng, label string) formatjadwal.Data {
	d := formatjadwal.Baru("Impor")
	d.JenisBel = []formatjadwal.JenisBel{{Nama: "Masuk", Warna: "#2563eb"}}
	d.Jadwal = []formatjadwal.Bel{{Hari: 1, Jam: "07:00", JenisBel: "Masuk", Label: label, Pengumuman: peng}}
	return d
}

func TestImpor_PengumumanTakDikenalDanOtomatis(t *testing.T) {
	s := bukaStore(t)
	dataDasar(t, s)
	satu := buatSuara(t, s, "Jam ke-1 dimulai")

	d := dataPengumuman("Suara Hilang", "Jam ke-1")
	if h := harus(s.PengumumanTakDikenal(ctx, d))(t); len(h) != 1 || h[0] != "Suara Hilang" {
		t.Fatalf("tak dikenal = %v", h)
	}
	id := harus(s.ImporJadwal(ctx, d, "A", nil))(t)
	if j := harus(s.ListJadwal(ctx, id))(t)[0]; j.PengumumanID != 0 {
		t.Fatalf("seharusnya NULL: %+v", j)
	}

	d = dataPengumuman("", "Jam ke-1")
	if h := harus(s.PengumumanTakDikenal(ctx, d))(t); h == nil || len(h) != 0 {
		t.Fatalf("tak dikenal = %#v", h)
	}
	id = harus(s.ImporJadwal(ctx, d, "B", nil))(t)
	if j := harus(s.ListJadwal(ctx, id))(t)[0]; j.PengumumanID != satu {
		t.Fatalf("otomatis gagal: %+v", j)
	}
}

func TestGabung_Pengumuman(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	satu := buatSuara(t, s, "Jam ke-1 dimulai")
	dua := buatSuara(t, s, "Jam ke-2 dimulai")
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, Label: "Lama", PengumumanID: satu}))(t)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "08:00", JenisBelID: jenis, Label: "Sama"}))(t)

	d := formatjadwal.Baru("G")
	d.JenisBel = []formatjadwal.JenisBel{{Nama: "Masuk", Warna: "#2563eb"}}
	d.Jadwal = []formatjadwal.Bel{
		{Hari: 1, Jam: "07:00", JenisBel: "Masuk", Label: "Baru", Pengumuman: "Jam ke-2 dimulai"}, // timpa
		{Hari: 1, Jam: "08:00", JenisBel: "Masuk", Label: "Sama", Pengumuman: "Jam ke-2 dimulai"}, // kembar
		{Hari: 2, Jam: "07:00", JenisBel: "Masuk", Label: "Jam ke-1"},                             // baru, otomatis
		{Hari: 2, Jam: "08:00", JenisBel: "Masuk", Label: "Z", Pengumuman: "Tidak Ada"},           // baru, tak ditemukan
		{Hari: 2, Jam: "09:00", JenisBel: "Masuk", Label: "Z2", Pengumuman: "tidak ada"},          // nama sama (unik)
	}
	h := harus(s.PeriksaGabung(ctx, profil, d, nil))(t)
	if len(h.PengumumanTakDitemukan) != 1 || h.PengumumanTakDitemukan[0] != "Tidak Ada" {
		t.Fatalf("tak ditemukan = %#v", h.PengumumanTakDitemukan)
	}
	if k := harus(s.PeriksaGabung(ctx, profil, dataGabung(), nil))(t); k.PengumumanTakDitemukan == nil {
		t.Fatal("harus non-nil")
	}

	harus(s.GabungJadwal(ctx, profil, d, nil, []gabung.Slot{{Hari: 1, Jam: "07:00"}}))(t)
	got := map[string]schedule.Jadwal{}
	for _, j := range harus(s.ListJadwal(ctx, profil))(t) {
		got[j.Jam+string(rune('0'+j.Hari))] = j
	}
	if got["07:001"].PengumumanID != dua || got["07:001"].Label != "Baru" {
		t.Fatalf("timpa = %+v", got["07:001"])
	}
	if got["08:001"].PengumumanID != 0 {
		t.Fatalf("kembar berubah: %+v", got["08:001"])
	}
	if got["07:002"].PengumumanID != satu {
		t.Fatalf("otomatis = %+v", got["07:002"])
	}
	if got["08:002"].PengumumanID != 0 || got["09:002"].PengumumanID != 0 {
		t.Fatalf("tak ditemukan harus NULL: %+v %+v", got["08:002"], got["09:002"])
	}
}
