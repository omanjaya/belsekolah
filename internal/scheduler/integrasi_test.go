package scheduler

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"belsekolah/internal/player"
	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

// siapkanStoreNyata membuat store sungguhan berisi profil default, satu jenis bel
// berikut audio, dan jadwal Senin 07:00.
func siapkanStoreNyata(t *testing.T) (*store.Store, int64) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "bel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	must := func(id int64, err error) int64 {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	profil := must(st.CreateProfil(ctx, "Reguler"))
	if err := st.SetSetting(ctx, store.KunciProfilDefault, "1"); err != nil {
		t.Fatal(err)
	}
	jenis := must(st.CreateJenisBel(ctx, "Masuk", "#2563eb"))
	audio := must(st.CreateAudio(ctx, store.Audio{Nama: "Ding", Sumber: "bawaan", File: "ding.wav", Dibuat: time.Now()}))
	if err := st.SetUrutanBel(ctx, jenis, []int64{audio}); err != nil {
		t.Fatal(err)
	}
	jadwal := must(st.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, Label: "Masuk"}))
	return st, jadwal
}

func schedulerBaru(st *store.Store, clock Clock) (*Scheduler, *fakePemutar) {
	pemutar := &fakePemutar{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(clock, st, pemutar, "/data/audio", log), pemutar
}

func TestIntegrasiStoreNyata(t *testing.T) {
	st, jadwalID := siapkanStoreNyata(t)
	clock := &fakeClock{}

	// (a) bel berbunyi tepat sekali dan riwayat tercatat berhasil.
	clock.set("2026-10-12", 7, 0, 5)
	s1, p1 := schedulerBaru(st, clock)
	tick(t, s1)
	if len(p1.diputar) != 1 {
		t.Fatalf("bel diputar %d kali, mau 1", len(p1.diputar))
	}
	req := p1.diputar[0]
	if len(req.Files) != 1 || req.Files[0] != filepath.Join("/data/audio", "ding.wav") {
		t.Fatalf("file = %v", req.Files)
	}
	req.Selesai(player.Berhasil, "")
	r, err := st.RiwayatTerbaru(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 1 || r[0].Status != "berhasil" || r[0].JadwalID != jadwalID {
		t.Fatalf("riwayat = %+v", r)
	}

	// (b) restart: scheduler baru pada store yang sama tidak mengulang.
	clock.set("2026-10-12", 7, 0, 30)
	s2, p2 := schedulerBaru(st, clock)
	tick(t, s2)
	if len(p2.diputar) != 0 {
		t.Fatalf("restart mengulang bel: %d kali", len(p2.diputar))
	}

	// (c) start terlambat dengan database baru: terlewat, tidak diputar.
	st3, _ := siapkanStoreNyata(t)
	clock.set("2026-10-12", 7, 5, 0)
	s3, p3 := schedulerBaru(st3, clock)
	tick(t, s3)
	if len(p3.diputar) != 0 {
		t.Fatalf("bel terlewat diputar: %d kali", len(p3.diputar))
	}
	r3, err := st3.RiwayatTerbaru(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(r3) != 1 || r3[0].Status != "terlewat" {
		t.Fatalf("riwayat terlewat = %+v", r3)
	}
}
