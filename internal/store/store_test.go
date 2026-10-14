package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"belsekolah/internal/schedule"
)

var ctx = context.Background()

func bukaStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "bel.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// harus dipakai sebagai harus(f())(t): Go tidak mengizinkan f(t, g()) untuk g yang mengembalikan dua nilai.
func harus[T any](v T, err error) func(*testing.T) T {
	return func(t *testing.T) T {
		t.Helper()
		if err != nil {
			t.Fatalf("error tak terduga: %v", err)
		}
		return v
	}
}

func TestOpen_BisaDibukaUlang(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bel.db")
	s := harus(Open(path))(t)
	harus(s.CreateProfil(ctx, "Normal"))(t)
	s.Close()
	s2 := harus(Open(path))(t)
	defer s2.Close()
	if n := harus(s2.CountProfil(ctx))(t); n != 1 {
		t.Fatalf("CountProfil = %d, mau 1", n)
	}
}

func TestPengaturan(t *testing.T) {
	s := bukaStore(t)
	if _, err := s.GetSetting(ctx, "volume"); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v, mau ErrTidakAda", err)
	}
	if err := s.SetSetting(ctx, "volume", "80"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(ctx, "volume", "90"); err != nil {
		t.Fatal(err)
	}
	if v := harus(s.GetSetting(ctx, "volume"))(t); v != "90" {
		t.Fatalf("volume = %q, mau 90", v)
	}
}

func TestAudio(t *testing.T) {
	s := bukaStore(t)
	dibuat := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	id := harus(s.CreateAudio(ctx, Audio{Nama: "Ding", Sumber: "bawaan", File: "ding.wav", DurasiMS: 1800, Dibuat: dibuat}))(t)
	a := harus(s.GetAudio(ctx, id))(t)
	if a.Nama != "Ding" || a.File != "ding.wav" || a.DurasiMS != 1800 || !a.Dibuat.Equal(dibuat) {
		t.Fatalf("audio = %+v", a)
	}
	if f := harus(s.FindAudio(ctx, "bawaan", "Ding"))(t); f.ID != id {
		t.Fatalf("FindAudio ID = %d, mau %d", f.ID, id)
	}
	if _, err := s.FindAudio(ctx, "upload", "Ding"); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v, mau ErrTidakAda", err)
	}
	if _, err := s.GetAudio(ctx, 999); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v, mau ErrTidakAda", err)
	}
}

func TestUrutanBel(t *testing.T) {
	s := bukaStore(t)
	now := time.Now()
	a1 := harus(s.CreateAudio(ctx, Audio{Nama: "A", Sumber: "bawaan", File: "a.wav", Dibuat: now}))(t)
	a2 := harus(s.CreateAudio(ctx, Audio{Nama: "B", Sumber: "bawaan", File: "b.wav", Dibuat: now}))(t)
	j := harus(s.CreateJenisBel(ctx, "Istirahat", "#16a34a"))(t)

	if err := s.SetUrutanBel(ctx, j, []int64{a2, a1, a2}); err != nil {
		t.Fatal(err)
	}
	urutan := harus(s.UrutanBel(ctx, j))(t)
	if len(urutan) != 3 || urutan[0].ID != a2 || urutan[1].ID != a1 || urutan[2].ID != a2 {
		t.Fatalf("urutan = %+v", urutan)
	}

	if err := s.SetUrutanBel(ctx, j, []int64{a1}); err != nil {
		t.Fatal(err)
	}
	if urutan := harus(s.UrutanBel(ctx, j))(t); len(urutan) != 1 || urutan[0].ID != a1 {
		t.Fatalf("urutan setelah diganti = %+v", urutan)
	}

	daftar := harus(s.ListJenisBel(ctx))(t)
	if len(daftar) != 1 || daftar[0].Nama != "Istirahat" || daftar[0].Warna != "#16a34a" {
		t.Fatalf("ListJenisBel = %+v", daftar)
	}
}

func TestCreateJadwal_Validasi(t *testing.T) {
	s := bukaStore(t)
	p := harus(s.CreateProfil(ctx, "Normal"))(t)
	jb := harus(s.CreateJenisBel(ctx, "Masuk", "#2563eb"))(t)
	if _, err := s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: p, Hari: 1, Jam: "7:00", JenisBelID: jb}); err == nil {
		t.Error("jam tidak valid seharusnya ditolak")
	}
	if _, err := s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: p, Hari: 8, Jam: "07:00", JenisBelID: jb}); err == nil {
		t.Error("hari 8 seharusnya ditolak")
	}
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: p, Hari: 1, Jam: "07:00", JenisBelID: jb}))(t)
	if _, err := s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: p, Hari: 1, Jam: "07:00", JenisBelID: jb}); err == nil {
		t.Error("jadwal ganda pada profil, hari, dan jam yang sama seharusnya ditolak")
	}
}

func TestCreatePeriode_Validasi(t *testing.T) {
	s := bukaStore(t)
	p := harus(s.CreateProfil(ctx, "Ujian"))(t)
	harus(s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-10-19", Selesai: "2026-10-23", ProfilID: p}))(t)

	if _, err := s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-10-23", Selesai: "2026-10-25", ProfilID: p}); !errors.Is(err, ErrPeriodeTumpangTindih) {
		t.Errorf("tumpang tindih di batas: err = %v", err)
	}
	if _, err := s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-10-10", Selesai: "2026-10-30", ProfilID: p}); !errors.Is(err, ErrPeriodeTumpangTindih) {
		t.Errorf("periode yang melingkupi: err = %v", err)
	}
	harus(s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-10-24", Selesai: "2026-10-24", ProfilID: p}))(t)

	if _, err := s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-11-05", Selesai: "2026-11-01", ProfilID: p}); err == nil {
		t.Error("selesai sebelum mulai seharusnya ditolak")
	}
	if _, err := s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-13-01", Selesai: "2026-13-02", ProfilID: p}); err == nil {
		t.Error("tanggal tidak valid seharusnya ditolak")
	}
}

func TestLoadDataJadwal(t *testing.T) {
	s := bukaStore(t)
	normal := harus(s.CreateProfil(ctx, "Normal"))(t)
	ujian := harus(s.CreateProfil(ctx, "Ujian"))(t)
	if err := s.SetSetting(ctx, KunciProfilDefault, "1"); err != nil {
		t.Fatal(err)
	}
	jb := harus(s.CreateJenisBel(ctx, "Masuk", "#2563eb"))(t)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: normal, Hari: 1, Jam: "07:00", JenisBelID: jb, Label: "Jam ke-1"}))(t)
	harus(s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-10-19", Selesai: "2026-10-23", ProfilID: ujian, Keterangan: "PTS"}))(t)
	harus(s.CreateLibur(ctx, schedule.Libur{Mulai: "2026-12-25", Selesai: "2026-12-25", Keterangan: "Natal"}))(t)

	d := harus(s.LoadDataJadwal(ctx))(t)
	if d.ProfilDefaultID != normal {
		t.Errorf("ProfilDefaultID = %d, mau %d", d.ProfilDefaultID, normal)
	}
	if len(d.Jadwal) != 1 || d.Jadwal[0].Label != "Jam ke-1" || d.Jadwal[0].Jam != "07:00" || d.Jadwal[0].JenisBelID != jb {
		t.Errorf("Jadwal = %+v", d.Jadwal)
	}
	if len(d.Periode) != 1 || d.Periode[0].ProfilID != ujian || d.Periode[0].Keterangan != "PTS" {
		t.Errorf("Periode = %+v", d.Periode)
	}
	if len(d.Libur) != 1 || d.Libur[0].Keterangan != "Natal" {
		t.Errorf("Libur = %+v", d.Libur)
	}
}

func TestLoadDataJadwal_TanpaDefault(t *testing.T) {
	s := bukaStore(t)
	d := harus(s.LoadDataJadwal(ctx))(t)
	if d.ProfilDefaultID != 0 {
		t.Fatalf("ProfilDefaultID = %d, mau 0", d.ProfilDefaultID)
	}
}

func TestRiwayat(t *testing.T) {
	s := bukaStore(t)
	wib := time.FixedZone("WIB", 7*3600)
	w1 := time.Date(2026, 10, 12, 7, 0, 0, 0, wib)
	w2 := time.Date(2026, 10, 12, 9, 30, 0, 0, wib)
	w3 := time.Date(2026, 10, 13, 7, 0, 0, 0, wib)
	for _, r := range []Riwayat{
		{Waktu: w1, JadwalID: 1, JenisBelID: 10, Label: "Masuk", Sumber: SumberJadwal, Status: "berhasil"},
		{Waktu: w2, JadwalID: 0, JenisBelID: 20, Label: "Manual", Sumber: SumberManual, Status: "berhasil"},
		{Waktu: w3, JadwalID: 2, JenisBelID: 10, Label: "Masuk", Sumber: SumberJadwal, Status: "terlewat", Pesan: "mati listrik"},
	} {
		if err := s.InsertRiwayat(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	tercatat := harus(s.JadwalTercatat(ctx, "2026-10-12"))(t)
	if len(tercatat) != 1 || !tercatat[1] {
		t.Errorf("JadwalTercatat 12 Okt = %v, mau {1:true}", tercatat)
	}
	if kosong := harus(s.JadwalTercatat(ctx, "2026-10-20"))(t); kosong == nil || len(kosong) != 0 {
		t.Errorf("JadwalTercatat tanggal kosong = %v, mau map kosong non-nil", kosong)
	}

	terbaru := harus(s.RiwayatTerbaru(ctx, 2))(t)
	if len(terbaru) != 2 || terbaru[0].Status != "terlewat" || terbaru[0].Pesan != "mati listrik" || !terbaru[0].Waktu.Equal(w3) {
		t.Fatalf("RiwayatTerbaru = %+v", terbaru)
	}
	if terbaru[1].JadwalID != 0 || terbaru[1].Sumber != SumberManual {
		t.Errorf("riwayat manual = %+v", terbaru[1])
	}
}

func TestRiwayat_StatusTidakValidDitolak(t *testing.T) {
	s := bukaStore(t)
	err := s.InsertRiwayat(ctx, Riwayat{Waktu: time.Now(), Sumber: SumberJadwal, Status: "aneh"})
	if err == nil {
		t.Fatal("status tidak valid seharusnya ditolak")
	}
}
