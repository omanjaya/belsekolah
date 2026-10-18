package store

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"belsekolah/internal/schedule"
)

func suaraUmum(t *testing.T, s *Store) int64 {
	t.Helper()
	return harus(s.CreateAudio(ctx, Audio{Nama: "Umum", Sumber: "upload", File: "umum.mp3", Dibuat: time.Now()}))(t)
}

func TestMigrasi_KolomPengumumanDitambahkanIdempoten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lama.db")
	// Database lama: skema sekarang tanpa kolom pengumuman_audio_id.
	lama := strings.Replace(skema, "  pengumuman_audio_id INTEGER REFERENCES audio(id) ON DELETE RESTRICT,\n", "", 1)
	if lama == skema {
		t.Fatal("baris kolom tidak ditemukan di skema")
	}
	raw, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.Exec(lama); err != nil {
		t.Fatal(err)
	}
	if ada, _ := adaKolom(raw, "jadwal", "pengumuman_audio_id"); ada {
		t.Fatal("db lama seharusnya belum punya kolom")
	}
	raw.Close()

	for i := 0; i < 2; i++ {
		s, err := Open(path)
		if err != nil {
			t.Fatalf("Open ke-%d: %v", i+1, err)
		}
		if ada, err := adaKolom(s.db, "jadwal", "pengumuman_audio_id"); err != nil || !ada {
			t.Fatalf("kolom ada=%v err=%v", ada, err)
		}
		s.Close()
	}
}

func TestJadwal_PengumumanCRUD(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	umum := suaraUmum(t, s)

	id := harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, PengumumanID: umum}))(t)
	if j := harus(s.GetJadwal(ctx, id))(t); j.PengumumanID != umum {
		t.Fatalf("Get PengumumanID = %d", j.PengumumanID)
	}
	if l := harus(s.ListJadwal(ctx, profil))(t); len(l) != 1 || l[0].PengumumanID != umum {
		t.Fatalf("List = %+v", l)
	}
	if d := harus(s.LoadDataJadwal(ctx))(t); len(d.Jadwal) != 1 || d.Jadwal[0].PengumumanID != umum {
		t.Fatalf("LoadDataJadwal = %+v", d.Jadwal)
	}

	// id 0 -> NULL
	if err := s.UpdateJadwal(ctx, schedule.Jadwal{ID: id, ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis}); err != nil {
		t.Fatal(err)
	}
	var png sql.NullInt64
	if err := s.db.QueryRow(`SELECT pengumuman_audio_id FROM jadwal WHERE id = ?`, id).Scan(&png); err != nil || png.Valid {
		t.Fatalf("seharusnya NULL: %+v err=%v", png, err)
	}
	// Update mengisi kembali
	if err := s.UpdateJadwal(ctx, schedule.Jadwal{ID: id, ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, PengumumanID: umum}); err != nil {
		t.Fatal(err)
	}
	if j := harus(s.GetJadwal(ctx, id))(t); j.PengumumanID != umum {
		t.Fatalf("setelah update PengumumanID = %d", j.PengumumanID)
	}

	// id tak ada -> validasi
	_, err := s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 2, Jam: "07:00", JenisBelID: jenis, PengumumanID: 9999})
	harapValidasi(t, err)
	harapValidasi(t, s.UpdateJadwal(ctx, schedule.Jadwal{ID: id, ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, PengumumanID: 9999}))
}

func TestPengumuman_SalinHariDanDuplikatProfil(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	umum := suaraUmum(t, s)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, PengumumanID: umum}))(t)

	if err := s.SalinHari(ctx, profil, 1, 2); err != nil {
		t.Fatal(err)
	}
	var hari2 []schedule.Jadwal
	for _, j := range harus(s.ListJadwal(ctx, profil))(t) {
		if j.Hari == 2 {
			hari2 = append(hari2, j)
		}
	}
	if len(hari2) != 1 || hari2[0].PengumumanID != umum {
		t.Fatalf("salin hari = %+v", hari2)
	}

	baru := harus(s.DuplikatProfil(ctx, profil, "Salinan"))(t)
	l := harus(s.ListJadwal(ctx, baru))(t)
	if len(l) != 2 {
		t.Fatalf("duplikat punya %d bel", len(l))
	}
	for _, j := range l {
		if j.PengumumanID != umum {
			t.Fatalf("duplikat tidak membawa pengumuman: %+v", j)
		}
	}
}

func TestDeleteAudio_DipakaiPengumuman(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	umum := suaraUmum(t, s)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, PengumumanID: umum}))(t)
	_, err := s.DeleteAudio(ctx, umum)
	if !errors.Is(err, ErrDipakai) || !strings.Contains(err.Error(), "1 bel") {
		t.Fatalf("err = %v", err)
	}
}
