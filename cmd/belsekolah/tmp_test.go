package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSiapkanTmp(t *testing.T) {
	dir := t.TempDir()
	lama := filepath.Join(dir, "tmp", "sisa.zip")
	os.MkdirAll(filepath.Dir(lama), 0o700)
	os.WriteFile(lama, []byte("x"), 0o600)
	t.Setenv("TMPDIR", "")

	if err := siapkanTmp(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(lama); !os.IsNotExist(err) {
		t.Error("sisa lama harus terhapus")
	}
	if got, want := os.Getenv("TMPDIR"), filepath.Join(dir, "tmp"); got != want {
		t.Errorf("TMPDIR = %q, mau %q", got, want)
	}
	info, err := os.Stat(filepath.Join(dir, "tmp"))
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("folder tmp: %v %v", info, err)
	}
}

func TestSapuSisaAudio(t *testing.T) {
	dir := t.TempDir()
	hapus := []string{".unggah-123", ".tts-abc"}
	simpan := []string{"upload-1.mp3", "tts-1.mp3", "bel.wav"}
	for _, n := range append(append([]string{}, hapus...), simpan...) {
		os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644)
	}
	if n := sapuSisaAudio(dir); n != 2 {
		t.Errorf("dihapus = %d, mau 2", n)
	}
	for _, n := range hapus {
		if _, err := os.Stat(filepath.Join(dir, n)); !os.IsNotExist(err) {
			t.Errorf("%s harus terhapus", n)
		}
	}
	for _, n := range simpan {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s harus tetap ada", n)
		}
	}
}
