package bawaan

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"belsekolah/internal/nada"
	"belsekolah/internal/store"
)

func TestPasang_Idempoten(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	audioDir := filepath.Join(dir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dir, "bel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		if err := Pasang(ctx, st, audioDir, now); err != nil {
			t.Fatalf("Pasang ke-%d: %v", i+1, err)
		}
	}

	for _, b := range nada.Daftar {
		if _, err := os.Stat(filepath.Join(audioDir, b.File)); err != nil {
			t.Errorf("file %s belum dibuat: %v", b.File, err)
		}
		a, err := st.FindAudio(ctx, "bawaan", b.Nama)
		if err != nil {
			t.Errorf("audio %s tidak ada: %v", b.Nama, err)
		} else if a.DurasiMS <= 0 {
			t.Errorf("durasi %s = %d", b.Nama, a.DurasiMS)
		}
	}

	if n, _ := st.CountProfil(ctx); n != 1 {
		t.Errorf("jumlah profil = %d, mau 1", n)
	}
	def, err := st.GetSetting(ctx, store.KunciProfilDefault)
	if err != nil {
		t.Fatalf("profil default belum diatur: %v", err)
	}
	if _, err := strconv.ParseInt(def, 10, 64); err != nil {
		t.Errorf("profil default tidak valid: %q", def)
	}

	jenis, err := st.ListJenisBel(ctx)
	if err != nil {
		t.Fatal(err)
	}
	mau := []string{"Masuk", "Ganti Pelajaran", "Istirahat", "Pulang", "Darurat"}
	if len(jenis) != len(mau) {
		t.Fatalf("jenis bel = %+v", jenis)
	}
	for i, j := range jenis {
		if j.Nama != mau[i] {
			t.Errorf("jenis bel ke-%d = %s, mau %s", i, j.Nama, mau[i])
		}
		urutan, err := st.UrutanBel(ctx, j.ID)
		if err != nil || len(urutan) == 0 {
			t.Errorf("jenis bel %s tidak punya suara (err %v)", j.Nama, err)
		}
	}
}

func TestPasang_FileHilangDitulisUlang(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "bel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := Pasang(ctx, st, dir, time.Now()); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, nada.Daftar[0].File)
	os.Remove(f)
	if err := Pasang(ctx, st, dir, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f); err != nil {
		t.Fatalf("file bawaan yang terhapus seharusnya ditulis ulang: %v", err)
	}
}
