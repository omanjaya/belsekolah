package scheduler

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"belsekolah/internal/player"
	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

var (
	ctx = context.Background()
	wib = time.FixedZone("WIB", 7*3600)
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }

// set mengatur jam pada tanggal (YYYY-MM-DD) dan jam:menit:detik WIB.
func (c *fakeClock) set(tanggal string, h, m, s int) {
	d, err := time.ParseInLocation(schedule.FormatTanggal, tanggal, wib)
	if err != nil {
		panic(err)
	}
	c.t = time.Date(d.Year(), d.Month(), d.Day(), h, m, s, 0, wib)
}

type fakeSumber struct {
	mu       sync.Mutex
	data     schedule.Data
	tercatat map[string]map[int64]bool
	riwayat  []store.Riwayat
	urutan   []store.Audio
	audio    map[int64]store.Audio // untuk GetAudio; id yang tak ada -> error
}

func (f *fakeSumber) GetAudio(_ context.Context, id int64) (store.Audio, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if a, ok := f.audio[id]; ok {
		return a, nil
	}
	return store.Audio{}, store.ErrTidakAda
}

func (f *fakeSumber) LoadDataJadwal(context.Context) (schedule.Data, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data, nil
}

func (f *fakeSumber) JadwalTercatat(_ context.Context, tanggal string) (map[int64]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[int64]bool{}
	for id := range f.tercatat[tanggal] {
		out[id] = true
	}
	return out, nil
}

func (f *fakeSumber) InsertRiwayat(_ context.Context, r store.Riwayat) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.riwayat = append(f.riwayat, r)
	return nil
}

func (f *fakeSumber) UrutanBel(_ context.Context, jenisBelID int64) ([]store.Audio, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.urutan != nil {
		return f.urutan, nil
	}
	return []store.Audio{{ID: 1, File: "ding.wav"}}, nil
}

func (f *fakeSumber) semuaRiwayat() []store.Riwayat {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]store.Riwayat(nil), f.riwayat...)
}

type fakePemutar struct{ diputar []player.Permintaan }

func (f *fakePemutar) Play(req player.Permintaan) { f.diputar = append(f.diputar, req) }

// Senin 2026-10-12 dan Selasa 2026-10-13.
func dataSenin() schedule.Data {
	return schedule.Data{
		ProfilDefaultID: 1,
		Jadwal: []schedule.Jadwal{
			{ID: 1, ProfilID: 1, Hari: 1, Jam: "07:00", JenisBelID: 10, Label: "Masuk"},
			{ID: 2, ProfilID: 1, Hari: 2, Jam: "07:00", JenisBelID: 10, Label: "Masuk Selasa"},
		},
	}
}

func siapkan(data schedule.Data) (*Scheduler, *fakeClock, *fakeSumber, *fakePemutar) {
	clock := &fakeClock{}
	sumber := &fakeSumber{data: data, tercatat: map[string]map[int64]bool{}}
	pemutar := &fakePemutar{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(clock, sumber, pemutar, "/data/audio", log), clock, sumber, pemutar
}

func tick(t *testing.T, s *Scheduler) {
	t.Helper()
	if err := s.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}
}

func TestBelTepatWaktu(t *testing.T) {
	s, clock, _, pemutar := siapkan(dataSenin())
	clock.set("2026-10-12", 6, 59, 59)
	tick(t, s)
	if len(pemutar.diputar) != 0 {
		t.Fatal("bel tidak boleh berbunyi sebelum waktunya")
	}
	clock.set("2026-10-12", 7, 0, 0)
	tick(t, s)
	clock.set("2026-10-12", 7, 0, 1)
	tick(t, s)
	if len(pemutar.diputar) != 1 {
		t.Fatalf("bel diputar %d kali, mau 1", len(pemutar.diputar))
	}
	req := pemutar.diputar[0]
	if len(req.Files) != 1 || req.Files[0] != "/data/audio/ding.wav" || req.Prioritas != player.Biasa {
		t.Fatalf("permintaan = %+v", req)
	}
}

func TestSelesaiMencatatRiwayat(t *testing.T) {
	s, clock, sumber, pemutar := siapkan(dataSenin())
	clock.set("2026-10-12", 7, 0, 0)
	tick(t, s)
	pemutar.diputar[0].Selesai(player.Berhasil, "")
	r := sumber.semuaRiwayat()
	if len(r) != 1 {
		t.Fatalf("riwayat = %+v", r)
	}
	if r[0].JadwalID != 1 || r[0].JenisBelID != 10 || r[0].Label != "Masuk" ||
		r[0].Sumber != store.SumberJadwal || r[0].Status != "berhasil" ||
		!r[0].Waktu.Equal(time.Date(2026, 10, 12, 7, 0, 0, 0, wib)) {
		t.Fatalf("riwayat = %+v", r[0])
	}
}

func TestDalamToleransiTetapBerbunyi(t *testing.T) {
	s, clock, _, pemutar := siapkan(dataSenin())
	clock.set("2026-10-12", 7, 0, 59)
	tick(t, s)
	if len(pemutar.diputar) != 1 {
		t.Fatalf("bel diputar %d kali, mau 1", len(pemutar.diputar))
	}
}

func TestTerlewatTidakDisusulkan(t *testing.T) {
	s, clock, sumber, pemutar := siapkan(dataSenin())
	clock.set("2026-10-12", 7, 1, 0) // tepat 60 detik setelah bel
	tick(t, s)
	if len(pemutar.diputar) != 0 {
		t.Fatal("bel terlewat tidak boleh disusulkan")
	}
	r := sumber.semuaRiwayat()
	if len(r) != 1 || r[0].Status != "terlewat" || r[0].JadwalID != 1 {
		t.Fatalf("riwayat = %+v", r)
	}
	clock.set("2026-10-12", 7, 2, 0)
	tick(t, s)
	if len(sumber.semuaRiwayat()) != 1 {
		t.Fatal("bel terlewat hanya dicatat sekali")
	}
}

func TestJamMundurTidakBerbunyiDuaKali(t *testing.T) {
	s, clock, _, pemutar := siapkan(dataSenin())
	clock.set("2026-10-12", 7, 0, 5)
	tick(t, s)
	clock.set("2026-10-12", 6, 59, 0)
	tick(t, s)
	clock.set("2026-10-12", 7, 0, 10)
	tick(t, s)
	if len(pemutar.diputar) != 1 {
		t.Fatalf("bel diputar %d kali, mau 1", len(pemutar.diputar))
	}
}

func TestRestartTidakMengulang(t *testing.T) {
	s, clock, sumber, pemutar := siapkan(dataSenin())
	sumber.tercatat["2026-10-12"] = map[int64]bool{1: true}
	clock.set("2026-10-12", 7, 0, 10)
	tick(t, s)
	if len(pemutar.diputar) != 0 || len(sumber.semuaRiwayat()) != 0 {
		t.Fatal("bel yang sudah tercatat tidak boleh berbunyi atau dicatat lagi")
	}
}

func TestGantiHari(t *testing.T) {
	s, clock, _, pemutar := siapkan(dataSenin())
	clock.set("2026-10-12", 7, 0, 1)
	tick(t, s)
	clock.set("2026-10-13", 6, 0, 0)
	tick(t, s)
	clock.set("2026-10-13", 7, 0, 1)
	tick(t, s)
	if len(pemutar.diputar) != 2 {
		t.Fatalf("bel diputar %d kali, mau 2 (Senin dan Selasa)", len(pemutar.diputar))
	}
}

func TestReloadSetelahAdminMengubahJadwal(t *testing.T) {
	s, clock, sumber, pemutar := siapkan(dataSenin())
	clock.set("2026-10-12", 7, 59, 0)
	tick(t, s)
	before := len(sumber.semuaRiwayat()) // bel 07:00 tercatat terlewat saat start

	sumber.mu.Lock()
	sumber.data.Jadwal = append(sumber.data.Jadwal,
		schedule.Jadwal{ID: 3, ProfilID: 1, Hari: 1, Jam: "06:30", JenisBelID: 10, Label: "Baru lewat"},
		schedule.Jadwal{ID: 4, ProfilID: 1, Hari: 1, Jam: "08:00", JenisBelID: 10, Label: "Baru nanti"},
	)
	sumber.mu.Unlock()
	s.Reload()

	clock.set("2026-10-12", 8, 0, 10)
	tick(t, s)
	if len(pemutar.diputar) != 1 {
		t.Fatalf("bel diputar %d kali, mau 1 (hanya 08:00)", len(pemutar.diputar))
	}
	if after := len(sumber.semuaRiwayat()); after != before {
		t.Fatalf("riwayat bertambah %d, bel yang baru dibuat setelah waktunya tidak boleh dicatat terlewat", after-before)
	}
}

func TestLiburTidakBerbunyi(t *testing.T) {
	d := dataSenin()
	d.Libur = []schedule.Libur{{ID: 1, Mulai: "2026-10-12", Selesai: "2026-10-12", Keterangan: "Libur"}}
	s, clock, sumber, pemutar := siapkan(d)
	clock.set("2026-10-12", 7, 0, 0)
	tick(t, s)
	if len(pemutar.diputar) != 0 || len(sumber.semuaRiwayat()) != 0 {
		t.Fatal("tidak boleh ada bel atau riwayat di hari libur")
	}
}
