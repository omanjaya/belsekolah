package store

import (
	"errors"
	"testing"
	"time"

	"belsekolah/internal/schedule"
)

func harapValidasi(t *testing.T, err error) {
	t.Helper()
	var v *ErrorValidasi
	if !errors.As(err, &v) {
		t.Fatalf("err = %v (%T), mau *ErrorValidasi", err, err)
	}
	if v.Pesan == "" {
		t.Fatal("pesan validasi kosong")
	}
}

func dataDasar(t *testing.T, s *Store) (profil, jenis, audio int64) {
	t.Helper()
	profil = harus(s.CreateProfil(ctx, "Normal"))(t)
	if err := s.SetProfilDefault(ctx, profil); err != nil {
		t.Fatal(err)
	}
	audio = harus(s.CreateAudio(ctx, Audio{Nama: "Ding", Sumber: "bawaan", File: "ding.wav", Dibuat: time.Now()}))(t)
	jenis = harus(s.CreateJenisBel(ctx, "Masuk", "#2563eb"))(t)
	if err := s.SetUrutanBel(ctx, jenis, []int64{audio}); err != nil {
		t.Fatal(err)
	}
	return
}

func TestValidasiNamaDanWarna(t *testing.T) {
	s := bukaStore(t)
	_, err := s.CreateProfil(ctx, "   ")
	harapValidasi(t, err)
	_, err = s.CreateJenisBel(ctx, "Masuk", "biru")
	harapValidasi(t, err)
	_, err = s.CreateJenisBel(ctx, string(make([]rune, 101)), "#000000")
	harapValidasi(t, err)
	id := harus(s.CreateProfil(ctx, "  Ujian  "))(t)
	if p := harus(s.GetProfil(ctx, id))(t); p.Nama != "Ujian" {
		t.Errorf("nama = %q, mau di-trim", p.Nama)
	}
	if _, err := s.CreateProfil(ctx, "Ujian"); !errors.Is(err, ErrDuplikat) {
		t.Errorf("err = %v, mau ErrDuplikat", err)
	}
}

func TestProfilCRUD(t *testing.T) {
	s := bukaStore(t)
	normal, _, _ := dataDasar(t, s)
	ujian := harus(s.CreateProfil(ctx, "Ujian"))(t)

	if err := s.UpdateProfil(ctx, ujian, "Ujian Akhir"); err != nil {
		t.Fatal(err)
	}
	daftar := harus(s.ListProfil(ctx))(t)
	if len(daftar) != 2 || daftar[1].Nama != "Ujian Akhir" {
		t.Fatalf("ListProfil = %+v", daftar)
	}
	if err := s.UpdateProfil(ctx, 999, "X"); !errors.Is(err, ErrTidakAda) {
		t.Errorf("update ID tak ada: %v", err)
	}
	if err := s.DeleteProfil(ctx, normal); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus profil default: %v, mau ErrDipakai", err)
	}
	harus(s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-10-19", Selesai: "2026-10-23", ProfilID: ujian}))(t)
	if err := s.DeleteProfil(ctx, ujian); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus profil yang dipakai periode: %v, mau ErrDipakai", err)
	}
	if def := harus(s.ProfilDefault(ctx))(t); def != normal {
		t.Errorf("ProfilDefault = %d, mau %d", def, normal)
	}
	if err := s.SetProfilDefault(ctx, 999); !errors.Is(err, ErrTidakAda) {
		t.Errorf("SetProfilDefault tak ada: %v", err)
	}
	if err := s.DeleteProfil(ctx, 999); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus ID tak ada: %v", err)
	}
}

func TestProfilDefaultBelumDiatur(t *testing.T) {
	s := bukaStore(t)
	if _, err := s.ProfilDefault(ctx); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v, mau ErrTidakAda", err)
	}
}

func TestJadwalCRUDDanSalin(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	senin1 := harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, Label: " Jam ke-1 "}))(t)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "09:30", JenisBelID: jenis, Label: "Istirahat"}))(t)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 2, Jam: "10:00", JenisBelID: jenis, Label: "Lama"}))(t)

	j := harus(s.GetJadwal(ctx, senin1))(t)
	if j.Label != "Jam ke-1" {
		t.Errorf("label = %q, mau di-trim", j.Label)
	}
	j.Jam = "07:15"
	if err := s.UpdateJadwal(ctx, j); err != nil {
		t.Fatal(err)
	}
	j.Jam = "9:00"
	harapValidasi(t, s.UpdateJadwal(ctx, j))
	j.Jam, j.JenisBelID = "07:15", 999
	harapValidasi(t, s.UpdateJadwal(ctx, j))

	_, err := s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "09:30", JenisBelID: jenis})
	if !errors.Is(err, ErrDuplikat) {
		t.Errorf("jadwal ganda: %v, mau ErrDuplikat", err)
	}
	_, err = s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: 999, Hari: 1, Jam: "11:00", JenisBelID: jenis})
	harapValidasi(t, err)

	if err := s.SalinHari(ctx, profil, 1, 2); err != nil {
		t.Fatal(err)
	}
	var selasa []string
	for _, x := range harus(s.ListJadwal(ctx, profil))(t) {
		if x.Hari == 2 {
			selasa = append(selasa, x.Jam+" "+x.Label)
		}
	}
	if len(selasa) != 2 || selasa[0] != "07:15 Jam ke-1" || selasa[1] != "09:30 Istirahat" {
		t.Fatalf("Selasa setelah salin = %v (jadwal lama harus diganti)", selasa)
	}
	harapValidasi(t, s.SalinHari(ctx, profil, 1, 1))
	harapValidasi(t, s.SalinHari(ctx, profil, 0, 2))
	if err := s.SalinHari(ctx, 999, 1, 2); !errors.Is(err, ErrTidakAda) {
		t.Errorf("salin profil tak ada: %v", err)
	}

	if err := s.DeleteJadwal(ctx, senin1); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteJadwal(ctx, senin1); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus dua kali: %v", err)
	}
	if err := s.DeleteJenisBel(ctx, jenis); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus jenis bel yang dipakai jadwal: %v, mau ErrDipakai", err)
	}
}

func TestPeriodeDanLiburCRUD(t *testing.T) {
	s := bukaStore(t)
	profil, _, _ := dataDasar(t, s)
	a := harus(s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-10-19", Selesai: "2026-10-23", ProfilID: profil, Keterangan: "PTS"}))(t)
	b := harus(s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-11-02", Selesai: "2026-11-06", ProfilID: profil}))(t)

	p := harus(s.GetPeriode(ctx, a))(t)
	p.Selesai = "2026-10-24" // memperpanjang diri sendiri tidak dianggap tumpang tindih
	if err := s.UpdatePeriode(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.Selesai = "2026-11-03"
	if err := s.UpdatePeriode(ctx, p); !errors.Is(err, ErrPeriodeTumpangTindih) {
		t.Errorf("update menabrak periode lain: %v", err)
	}
	_, err := s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-12-01", Selesai: "2026-12-02", ProfilID: 999})
	harapValidasi(t, err)
	if err := s.DeletePeriode(ctx, b); err != nil {
		t.Fatal(err)
	}
	if n := len(harus(s.ListPeriode(ctx))(t)); n != 1 {
		t.Errorf("jumlah periode = %d, mau 1", n)
	}

	l := harus(s.CreateLibur(ctx, schedule.Libur{Mulai: "2026-12-25", Selesai: "2026-12-25", Keterangan: " Natal "}))(t)
	libur := harus(s.GetLibur(ctx, l))(t)
	if libur.Keterangan != "Natal" {
		t.Errorf("keterangan = %q", libur.Keterangan)
	}
	libur.Selesai = "2026-12-26"
	if err := s.UpdateLibur(ctx, libur); err != nil {
		t.Fatal(err)
	}
	libur.Mulai = "2026-12-30"
	harapValidasi(t, s.UpdateLibur(ctx, libur))
	if err := s.DeleteLibur(ctx, l); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteLibur(ctx, l); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus libur dua kali: %v", err)
	}
	_, err = s.CreateLibur(ctx, schedule.Libur{Mulai: "2026-12-25", Selesai: "2026-12-25", Keterangan: string(make([]rune, 201))})
	harapValidasi(t, err)
}

func TestAudioDanJenisBelCRUD(t *testing.T) {
	s := bukaStore(t)
	_, jenis, bawaan := dataDasar(t, s)
	up := harus(s.CreateAudio(ctx, Audio{Nama: "Lagu", Sumber: "upload", File: "upload-1.mp3", Dibuat: time.Now()}))(t)
	_, err := s.CreateAudio(ctx, Audio{Nama: " ", Sumber: "upload", File: "x.mp3", Dibuat: time.Now()})
	harapValidasi(t, err)

	if err := s.UpdateNamaAudio(ctx, up, "Lagu Wajib"); err != nil {
		t.Fatal(err)
	}
	if n := len(harus(s.ListAudio(ctx))(t)); n != 2 {
		t.Errorf("ListAudio = %d, mau 2", n)
	}
	if _, err := s.DeleteAudio(ctx, bawaan); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus audio bawaan: %v, mau ErrDipakai", err)
	}
	if err := s.SetUrutanBel(ctx, jenis, []int64{up}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DeleteAudio(ctx, up); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus audio yang dipakai: %v, mau ErrDipakai", err)
	}
	harapValidasi(t, s.SetUrutanBel(ctx, jenis, nil))
	harapValidasi(t, s.SetUrutanBel(ctx, jenis, []int64{999}))
	harapValidasi(t, s.SetUrutanBel(ctx, jenis, make([]int64, 11)))
	if err := s.SetUrutanBel(ctx, jenis, []int64{bawaan}); err != nil {
		t.Fatal(err)
	}
	a := harus(s.DeleteAudio(ctx, up))(t)
	if a.File != "upload-1.mp3" {
		t.Errorf("DeleteAudio mengembalikan %+v", a)
	}

	if err := s.UpdateJenisBel(ctx, jenis, "Masuk Kelas", "#ff0000"); err != nil {
		t.Fatal(err)
	}
	j := harus(s.GetJenisBel(ctx, jenis))(t)
	if j.Nama != "Masuk Kelas" || j.Warna != "#ff0000" {
		t.Errorf("jenis bel = %+v", j)
	}
	harapValidasi(t, s.UpdateJenisBel(ctx, jenis, "Masuk", "merah"))
	if _, err := s.GetJenisBel(ctx, 999); !errors.Is(err, ErrTidakAda) {
		t.Errorf("GetJenisBel tak ada: %v", err)
	}
	baru := harus(s.CreateJenisBel(ctx, "Pulang", "#9333ea"))(t)
	if err := s.DeleteJenisBel(ctx, baru); err != nil {
		t.Fatal(err)
	}
}

func TestVolume(t *testing.T) {
	s := bukaStore(t)
	if v := harus(s.Volume(ctx))(t); v != 100 {
		t.Errorf("volume default = %d, mau 100", v)
	}
	if err := s.SetVolume(ctx, 65); err != nil {
		t.Fatal(err)
	}
	if v := harus(s.Volume(ctx))(t); v != 65 {
		t.Errorf("volume = %d, mau 65", v)
	}
	harapValidasi(t, s.SetVolume(ctx, 101))
	harapValidasi(t, s.SetVolume(ctx, -1))
}

func TestCariRiwayat(t *testing.T) {
	s := bukaStore(t)
	wib := time.FixedZone("WIB", 7*3600)
	for i, st := range []string{"berhasil", "gagal", "berhasil"} {
		w := time.Date(2026, 10, 12+i/2, 7+i, 0, 0, 0, wib)
		if err := s.InsertRiwayat(ctx, Riwayat{Waktu: w, Sumber: SumberManual, Status: st, Label: st}); err != nil {
			t.Fatal(err)
		}
	}
	semua := harus(s.CariRiwayat(ctx, FilterRiwayat{}))(t)
	if len(semua) != 3 {
		t.Fatalf("semua = %d, mau 3", len(semua))
	}
	tgl := harus(s.CariRiwayat(ctx, FilterRiwayat{Tanggal: "2026-10-12"}))(t)
	if len(tgl) != 2 {
		t.Errorf("tanggal 12 = %d, mau 2", len(tgl))
	}
	gagal := harus(s.CariRiwayat(ctx, FilterRiwayat{Status: "gagal"}))(t)
	if len(gagal) != 1 || gagal[0].Status != "gagal" {
		t.Errorf("status gagal = %+v", gagal)
	}
	if n := len(harus(s.CariRiwayat(ctx, FilterRiwayat{Batas: 1}))(t)); n != 1 {
		t.Errorf("batas 1 = %d", n)
	}
	_, err := s.CariRiwayat(ctx, FilterRiwayat{Tanggal: "12-10-2026"})
	harapValidasi(t, err)
	_, err = s.CariRiwayat(ctx, FilterRiwayat{Status: "aneh"})
	harapValidasi(t, err)
}
