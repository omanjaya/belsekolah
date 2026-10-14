package player

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	mu        sync.Mutex
	dipanggil []string
	volume    []int
	gagal     map[string]int // file -> sisa jumlah kegagalan
	blokFile  string         // Run untuk file ini menunggu sampai ctx dibatalkan
}

func (f *fakeRunner) Run(ctx context.Context, file string, volume int) error {
	f.mu.Lock()
	f.dipanggil = append(f.dipanggil, filepath.Base(file))
	f.volume = append(f.volume, volume)
	n := f.gagal[file]
	if n > 0 {
		f.gagal[file] = n - 1
	}
	blok := file == f.blokFile
	f.mu.Unlock()
	if blok {
		<-ctx.Done()
		return ctx.Err()
	}
	if n > 0 {
		return errors.New("mpv keluar dengan kode 2")
	}
	return nil
}

func (f *fakeRunner) panggilan() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.dipanggil...)
}

type hasil struct {
	status Status
	pesan  string
}

func buatFile(t *testing.T, dir string, nama ...string) []string {
	t.Helper()
	var out []string
	for _, n := range nama {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func mulai(t *testing.T, r *fakeRunner) (*Player, string) {
	t.Helper()
	dir := t.TempDir()
	cadangan := buatFile(t, dir, "cadangan.wav")[0]
	p := New(r, cadangan)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go p.Jalankan(ctx)
	return p, dir
}

func putar(p *Player, files []string, prio Prioritas) chan hasil {
	ch := make(chan hasil, 1)
	p.Play(Permintaan{Files: files, Prioritas: prio, Selesai: func(s Status, pesan string) {
		ch <- hasil{s, pesan}
	}})
	return ch
}

func tunggu(t *testing.T, ch chan hasil) hasil {
	t.Helper()
	select {
	case h := <-ch:
		return h
	case <-time.After(2 * time.Second):
		t.Fatal("timeout menunggu permintaan selesai")
		return hasil{}
	}
}

func tungguDipanggil(t *testing.T, r *fakeRunner, nama string) {
	t.Helper()
	batas := time.Now().Add(2 * time.Second)
	for time.Now().Before(batas) {
		for _, p := range r.panggilan() {
			if p == nama {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("runner tidak pernah memutar %s; panggilan: %v", nama, r.panggilan())
}

func sama(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPutarBerurutan(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	p.SetVolume(70)
	files := buatFile(t, dir, "a.wav", "b.wav")
	h := tunggu(t, putar(p, files, Biasa))
	if h.status != Berhasil {
		t.Fatalf("status = %s (%s), mau berhasil", h.status, h.pesan)
	}
	if !sama(r.panggilan(), []string{"a.wav", "b.wav"}) {
		t.Fatalf("panggilan = %v", r.panggilan())
	}
	if r.volume[0] != 70 {
		t.Errorf("volume = %d, mau 70", r.volume[0])
	}
}

func TestAntreanBerurutan(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	a := buatFile(t, dir, "a.wav")
	b := buatFile(t, dir, "b.wav")
	ch1 := putar(p, a, Biasa)
	ch2 := putar(p, b, Biasa)
	tunggu(t, ch1)
	tunggu(t, ch2)
	if !sama(r.panggilan(), []string{"a.wav", "b.wav"}) {
		t.Fatalf("panggilan = %v", r.panggilan())
	}
}

func TestFileHilangPakaiCadangan(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	h := tunggu(t, putar(p, []string{filepath.Join(dir, "hilang.mp3")}, Biasa))
	if h.status != Cadangan {
		t.Fatalf("status = %s, mau cadangan", h.status)
	}
	if !sama(r.panggilan(), []string{"cadangan.wav"}) {
		t.Fatalf("panggilan = %v", r.panggilan())
	}
	if h.pesan == "" {
		t.Error("pesan seharusnya menjelaskan file yang hilang")
	}
}

func TestTanpaFilePakaiCadangan(t *testing.T) {
	r := &fakeRunner{}
	p, _ := mulai(t, r)
	h := tunggu(t, putar(p, nil, Biasa))
	if h.status != Cadangan || !sama(r.panggilan(), []string{"cadangan.wav"}) {
		t.Fatalf("status = %s, panggilan = %v", h.status, r.panggilan())
	}
}

func TestGagalSekaliDicobaUlang(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	files := buatFile(t, dir, "a.wav")
	r.gagal = map[string]int{files[0]: 1}
	h := tunggu(t, putar(p, files, Biasa))
	if h.status != Berhasil || !sama(r.panggilan(), []string{"a.wav", "a.wav"}) {
		t.Fatalf("status = %s, panggilan = %v", h.status, r.panggilan())
	}
}

func TestGagalDuaKaliPakaiCadangan(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	files := buatFile(t, dir, "a.wav", "b.wav")
	r.gagal = map[string]int{files[0]: 2}
	h := tunggu(t, putar(p, files, Biasa))
	if h.status != Cadangan {
		t.Fatalf("status = %s (%s), mau cadangan", h.status, h.pesan)
	}
	if !strings.Contains(h.pesan, "mpv keluar") || !strings.Contains(h.pesan, "cadangan") {
		t.Errorf("pesan = %q, harus menyebut error mpv dan nada cadangan", h.pesan)
	}
	// File berikutnya tetap diputar.
	if !sama(r.panggilan(), []string{"a.wav", "a.wav", "cadangan.wav", "b.wav"}) {
		t.Fatalf("panggilan = %v", r.panggilan())
	}
}

func TestCadanganJugaGagal(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	files := buatFile(t, dir, "a.wav")
	cad := filepath.Join(dir, "cadangan.wav")
	r.gagal = map[string]int{files[0]: 2, cad: 1}
	h := tunggu(t, putar(p, files, Biasa))
	if h.status != Gagal {
		t.Fatalf("status = %s (%s), mau gagal", h.status, h.pesan)
	}
	if !sama(r.panggilan(), []string{"a.wav", "a.wav", "cadangan.wav"}) {
		t.Fatalf("panggilan = %v", r.panggilan())
	}
}

func TestSelesaiMemanggilPlayLagi(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	a := buatFile(t, dir, "a.wav")
	b := buatFile(t, dir, "b.wav")
	ch1 := make(chan hasil, 1)
	ch2 := make(chan hasil, 1)
	p.Play(Permintaan{Files: a, Selesai: func(s Status, pesan string) {
		p.Play(Permintaan{Files: b, Selesai: func(s Status, pesan string) { ch2 <- hasil{s, pesan} }})
		ch1 <- hasil{s, pesan}
	}})
	tunggu(t, ch1)
	if h := tunggu(t, ch2); h.status != Berhasil {
		t.Fatalf("status = %s, mau berhasil", h.status)
	}
}

func TestCtxBerhentiMembuangAntrean(t *testing.T) {
	r := &fakeRunner{}
	dir := t.TempDir()
	cad := buatFile(t, dir, "cadangan.wav")[0]
	a := buatFile(t, dir, "a.wav")
	b := buatFile(t, dir, "b.wav")
	r.blokFile = a[0]
	p := New(r, cad)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	berhenti := make(chan struct{})
	go func() { p.Jalankan(ctx); close(berhenti) }()

	chA := putar(p, a, Biasa)
	tungguDipanggil(t, r, "a.wav")
	chB := putar(p, b, Biasa)
	cancel()

	if h := tunggu(t, chA); h.status != Dihentikan {
		t.Errorf("A status = %s, mau dihentikan", h.status)
	}
	if h := tunggu(t, chB); h.status != Dihentikan {
		t.Errorf("B status = %s, mau dihentikan", h.status)
	}
	select {
	case <-berhenti:
	case <-time.After(2 * time.Second):
		t.Fatal("Jalankan tidak kembali")
	}
	if !sama(r.panggilan(), []string{"a.wav"}) {
		t.Errorf("panggilan = %v, b.wav tidak boleh diputar", r.panggilan())
	}
}

func TestDaruratMemotong(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	a := buatFile(t, dir, "a.wav")
	b := buatFile(t, dir, "b.wav")
	d := buatFile(t, dir, "darurat.wav")
	r.blokFile = a[0]

	chA := putar(p, a, Biasa)
	tungguDipanggil(t, r, "a.wav")
	chB := putar(p, b, Biasa)
	chD := putar(p, d, Darurat)

	if h := tunggu(t, chA); h.status != Dihentikan {
		t.Errorf("A status = %s, mau dihentikan", h.status)
	}
	if h := tunggu(t, chB); h.status != Dihentikan {
		t.Errorf("B status = %s, mau dihentikan", h.status)
	}
	if h := tunggu(t, chD); h.status != Berhasil {
		t.Errorf("darurat status = %s, mau berhasil", h.status)
	}
	if !sama(r.panggilan(), []string{"a.wav", "darurat.wav"}) {
		t.Errorf("panggilan = %v, b.wav tidak boleh diputar", r.panggilan())
	}
}

func TestStop(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	a := buatFile(t, dir, "a.wav")
	b := buatFile(t, dir, "b.wav")
	r.blokFile = a[0]

	chA := putar(p, a, Biasa)
	tungguDipanggil(t, r, "a.wav")
	chB := putar(p, b, Biasa)
	p.Stop()

	if h := tunggu(t, chA); h.status != Dihentikan {
		t.Errorf("A status = %s, mau dihentikan", h.status)
	}
	if h := tunggu(t, chB); h.status != Dihentikan {
		t.Errorf("B status = %s, mau dihentikan", h.status)
	}

	// Player tetap bisa dipakai setelah Stop.
	r.mu.Lock()
	r.blokFile = ""
	r.mu.Unlock()
	if h := tunggu(t, putar(p, b, Biasa)); h.status != Berhasil {
		t.Errorf("setelah Stop status = %s, mau berhasil", h.status)
	}
}

func TestSetVolumeDibatasi(t *testing.T) {
	r := &fakeRunner{}
	p, dir := mulai(t, r)
	p.SetVolume(150)
	tunggu(t, putar(p, buatFile(t, dir, "a.wav"), Biasa))
	p.SetVolume(-5)
	tunggu(t, putar(p, buatFile(t, dir, "b.wav"), Biasa))
	if r.volume[0] != 100 || r.volume[1] != 0 {
		t.Fatalf("volume = %v, mau [100 0]", r.volume)
	}
}
