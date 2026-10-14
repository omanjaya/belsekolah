package scheduler

import (
	"testing"

	"belsekolah/internal/player"
	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

func TestStatusSebelumDanSesudahMuat(t *testing.T) {
	s, clock, _, _ := siapkan(dataSenin())
	if st := s.Status(); st.Siap {
		t.Fatal("Status harus belum siap sebelum Tick pertama")
	}
	clock.set("2026-10-12", 7, 0, 5)
	tick(t, s)
	st := s.Status()
	if !st.Siap || st.Tanggal != "2026-10-12" || st.ProfilID != 1 || st.Libur != nil {
		t.Fatalf("status = %+v", st)
	}
	if len(st.Bel) != 1 || st.Bel[0].Label != "Masuk" || !st.Bel[0].Sudah || st.Bel[0].JenisBelID != 10 {
		t.Fatalf("bel = %+v", st.Bel)
	}
}

func TestStatusLiburDisalin(t *testing.T) {
	d := dataSenin()
	d.Libur = []schedule.Libur{{ID: 7, Mulai: "2026-10-12", Selesai: "2026-10-12", Keterangan: "Libur"}}
	s, clock, _, _ := siapkan(d)
	clock.set("2026-10-12", 6, 0, 0)
	tick(t, s)
	st := s.Status()
	if st.Libur == nil || st.Libur.ID != 7 {
		t.Fatalf("libur = %+v", st.Libur)
	}
	st.Libur.Keterangan = "diubah"
	if s.Status().Libur.Keterangan != "Libur" {
		t.Fatal("Status harus mengembalikan salinan, bukan pointer ke data internal")
	}
}

func TestBunyikanManual(t *testing.T) {
	s, clock, sumber, pemutar := siapkan(dataSenin())
	clock.set("2026-10-12", 10, 15, 0)
	s.BunyikanManual(ctx, 10, "Manual: Masuk", false)
	s.BunyikanManual(ctx, 10, "Darurat: Masuk", true)
	if len(pemutar.diputar) != 2 {
		t.Fatalf("diputar = %d, mau 2", len(pemutar.diputar))
	}
	if pemutar.diputar[0].Prioritas != player.Biasa || pemutar.diputar[1].Prioritas != player.Darurat {
		t.Errorf("prioritas = %v, %v", pemutar.diputar[0].Prioritas, pemutar.diputar[1].Prioritas)
	}
	pemutar.diputar[0].Selesai(player.Berhasil, "")
	pemutar.diputar[1].Selesai(player.Dihentikan, "dihentikan oleh admin")
	r := sumber.semuaRiwayat()
	if len(r) != 2 {
		t.Fatalf("riwayat = %+v", r)
	}
	if r[0].Sumber != store.SumberManual || r[0].JadwalID != 0 || r[0].Label != "Manual: Masuk" ||
		r[0].Status != "berhasil" || !r[0].Waktu.Equal(clock.t) {
		t.Errorf("riwayat manual = %+v", r[0])
	}
	if r[1].Sumber != store.SumberDarurat || r[1].Status != "dihentikan" || r[1].Pesan == "" {
		t.Errorf("riwayat darurat = %+v", r[1])
	}
}

func TestFileUrutanMembuangPath(t *testing.T) {
	s, _, sumber, _ := siapkan(dataSenin())
	sumber.urutan = []store.Audio{{File: "../../etc/passwd"}, {File: "a.wav"}}
	files := s.FileUrutan(ctx, 10)
	if len(files) != 2 || files[0] != "/data/audio/passwd" || files[1] != "/data/audio/a.wav" {
		t.Fatalf("files = %v", files)
	}
}
