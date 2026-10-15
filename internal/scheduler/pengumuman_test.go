package scheduler

import (
	"testing"

	"belsekolah/internal/store"
)

func TestPengumumanDiputarSetelahUrutanJenis(t *testing.T) {
	data := dataSenin()
	data.Jadwal[0].PengumumanID = 7
	s, clock, sumber, pemutar := siapkan(data)
	sumber.audio = map[int64]store.Audio{7: {ID: 7, File: "sub/umum.mp3"}}
	clock.set("2026-10-12", 7, 0, 0)
	tick(t, s)
	if len(pemutar.diputar) != 1 {
		t.Fatalf("diputar %d kali", len(pemutar.diputar))
	}
	f := pemutar.diputar[0].Files
	if len(f) != 2 || f[0] != "/data/audio/ding.wav" || f[1] != "/data/audio/umum.mp3" {
		t.Fatalf("files = %v", f)
	}
}

func TestPengumumanGagalMuatHanyaUrutanJenis(t *testing.T) {
	data := dataSenin()
	data.Jadwal[0].PengumumanID = 99
	s, clock, _, pemutar := siapkan(data)
	clock.set("2026-10-12", 7, 0, 0)
	tick(t, s)
	f := pemutar.diputar[0].Files
	if len(f) != 1 || f[0] != "/data/audio/ding.wav" {
		t.Fatalf("files = %v", f)
	}
}
