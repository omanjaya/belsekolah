package store

import (
	"errors"
	"reflect"
	"testing"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/schedule"
)

// isiProfil membuat profil "Normal" (default) dengan 3 bel memakai jenis "Masuk".
func isiProfil(t *testing.T, s *Store) (profil, jenis int64) {
	t.Helper()
	profil, jenis, _ = dataDasar(t, s)
	for _, b := range []schedule.Jadwal{
		{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, Label: "Masuk"},
		{ProfilID: profil, Hari: 1, Jam: "12:00", JenisBelID: jenis, Label: "Pulang"},
		{ProfilID: profil, Hari: 5, Jam: "07:00", JenisBelID: jenis, Label: ""},
	} {
		harus(s.CreateJadwal(ctx, b))(t)
	}
	return
}

func TestJumlahBelPerProfil(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	kosong := harus(s.CreateProfil(ctx, "Kosong"))(t)
	n := harus(s.JumlahBelPerProfil(ctx))(t)
	if n[profil] != 3 || n[kosong] != 0 {
		t.Fatalf("jumlah = %v", n)
	}
}

func TestDuplikatProfil(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	baru := harus(s.DuplikatProfil(ctx, profil, "Normal (salinan)"))(t)
	asal := harus(s.ListJadwal(ctx, profil))(t)
	salin := harus(s.ListJadwal(ctx, baru))(t)
	if len(salin) != len(asal) {
		t.Fatalf("salinan %d bel, asal %d", len(salin), len(asal))
	}
	for i := range asal {
		if salin[i].Hari != asal[i].Hari || salin[i].Jam != asal[i].Jam || salin[i].JenisBelID != asal[i].JenisBelID || salin[i].Label != asal[i].Label {
			t.Fatalf("bel %d berbeda: %+v vs %+v", i, salin[i], asal[i])
		}
	}
	if _, err := s.DuplikatProfil(ctx, profil, "Normal"); !errors.Is(err, ErrDuplikat) {
		t.Fatalf("nama bentrok: err = %v", err)
	}
	if _, err := s.DuplikatProfil(ctx, 999, "X"); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("profil tak ada: err = %v", err)
	}
	if n := harus(s.CountProfil(ctx))(t); n != 2 {
		t.Fatalf("profil = %d setelah duplikat gagal, mau 2", n)
	}
}

func TestEksporImpor_RoundTrip(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s)
	d := harus(s.EksporProfil(ctx, profil))(t)
	if d.Format != formatjadwal.Format || d.Nama != "Normal" || len(d.Jadwal) != 3 || len(d.JenisBel) != 1 {
		t.Fatalf("ekspor = %+v", d)
	}
	if tak := harus(s.JenisBelTakDikenal(ctx, d))(t); len(tak) != 0 {
		t.Fatalf("tak dikenal = %v", tak)
	}
	nama := harus(s.NamaProfilTersedia(ctx, d.Nama))(t)
	if nama != "Normal (impor)" {
		t.Fatalf("nama tersedia = %q", nama)
	}
	baru := harus(s.ImporJadwal(ctx, d, nama, nil))(t)
	asal := harus(s.ListJadwal(ctx, profil))(t)
	hasil := harus(s.ListJadwal(ctx, baru))(t)
	for i := range asal {
		asal[i].ID, asal[i].ProfilID, hasil[i].ID, hasil[i].ProfilID = 0, 0, 0, 0
	}
	if !reflect.DeepEqual(asal, hasil) {
		t.Fatalf("round-trip berbeda:\n%+v\n%+v", asal, hasil)
	}
	if hasil[0].JenisBelID != jenis {
		t.Fatal("jenis bel yang namanya sama harus dipakai ulang")
	}
	if nama := harus(s.NamaProfilTersedia(ctx, "Normal"))(t); nama != "Normal (impor 2)" {
		t.Fatalf("nama tersedia kedua = %q", nama)
	}
}

func dataAsing() formatjadwal.Data {
	d := formatjadwal.Baru("Asing")
	d.JenisBel = []formatjadwal.JenisBel{{Nama: "MASUK", Warna: "#111111"}, {Nama: "Bel Literasi", Warna: "#222222"}, {Nama: "Bel Doa", Warna: "#333333"}}
	d.Jadwal = []formatjadwal.Bel{
		{Hari: 1, Jam: "07:00", JenisBel: "MASUK"},
		{Hari: 1, Jam: "07:15", JenisBel: "Bel Literasi", Label: "Literasi"},
		{Hari: 1, Jam: "07:30", JenisBel: "Bel Doa", Label: "Doa"},
	}
	return d
}

func TestImpor_Pemetaan(t *testing.T) {
	s := bukaStore(t)
	_, jenis := isiProfil(t, s)
	d := dataAsing()
	tak := harus(s.JenisBelTakDikenal(ctx, d))(t)
	if !reflect.DeepEqual(tak, []string{"Bel Literasi", "Bel Doa"}) {
		t.Fatalf("tak dikenal = %v (MASUK harus cocok dengan Masuk)", tak)
	}
	// Pemetaan kurang → ditolak, tidak ada sisa.
	_, err := s.ImporJadwal(ctx, d, "Asing", map[string]PemetaanJenis{"Bel Literasi": {Baru: true}})
	harapValidasi(t, err)
	if n := harus(s.CountProfil(ctx))(t); n != 1 {
		t.Fatalf("profil = %d setelah impor gagal", n)
	}
	if js := harus(s.ListJenisBel(ctx))(t); len(js) != 1 {
		t.Fatalf("jenis bel = %d setelah impor gagal (transaksi harus dibatalkan)", len(js))
	}
	// Jenis tujuan tidak ada → ditolak.
	_, err = s.ImporJadwal(ctx, d, "Asing", map[string]PemetaanJenis{"Bel Literasi": {Baru: true}, "Bel Doa": {JenisBelID: 999}})
	harapValidasi(t, err)

	id := harus(s.ImporJadwal(ctx, d, "Asing", map[string]PemetaanJenis{
		"Bel Literasi": {Baru: true}, "Bel Doa": {JenisBelID: jenis},
	}))(t)
	js := harus(s.ListJadwal(ctx, id))(t)
	if len(js) != 3 || js[0].JenisBelID != jenis || js[2].JenisBelID != jenis {
		t.Fatalf("hasil impor = %+v", js)
	}
	semua := harus(s.ListJenisBel(ctx))(t)
	if len(semua) != 2 || semua[1].Nama != "Bel Literasi" || semua[1].Warna != "#222222" {
		t.Fatalf("jenis bel baru = %+v", semua)
	}
	urutan := harus(s.UrutanBel(ctx, semua[1].ID))(t)
	if len(urutan) != 1 {
		t.Fatalf("jenis bel baru harus punya 1 suara, ada %d", len(urutan))
	}
	if _, err := s.ImporJadwal(ctx, d, "Asing", map[string]PemetaanJenis{"Bel Literasi": {Baru: true}, "Bel Doa": {Baru: true}}); !errors.Is(err, ErrDuplikat) {
		t.Fatalf("nama profil bentrok: err = %v", err)
	}
}

func TestImpor_DataTidakValid(t *testing.T) {
	s := bukaStore(t)
	isiProfil(t, s)
	d := dataAsing()
	d.Jadwal[0].Hari = 9
	_, err := s.ImporJadwal(ctx, d, "X", nil)
	harapValidasi(t, err)
}
