package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"belsekolah/internal/store"
)

var ctx = context.Background()

func siapkanData(t *testing.T) (dataDir string, st *store.Store) {
	t.Helper()
	dataDir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "audio"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dataDir, "bel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := st.CreateProfil(ctx, "Normal"); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.wav", "b.mp3", ".unggah-123"} {
		if err := os.WriteFile(filepath.Join(dataDir, "audio", n), []byte("isi "+n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dataDir, st
}

func namaDiZip(t *testing.T, b []byte) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range zr.File {
		out = append(out, f.Name)
	}
	sort.Strings(out)
	return out
}

func buatZip(t *testing.T, path string, isi map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for nama, data := range isi {
		w, err := zw.Create(nama)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}
	zw.Close()
	f.Close()
}

type entri struct {
	nama string
	isi  []byte
}

func buatZipUrut(t *testing.T, path string, isi []entri) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, e := range isi {
		w, err := zw.Create(e.nama)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(e.isi)
	}
	zw.Close()
	f.Close()
}

func TestHarianSekaliPerHariDanDipangkas(t *testing.T) {
	dataDir, st := siapkanData(t)
	dir := filepath.Join(dataDir, "backup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tgl := range []string{"2026-10-01", "2026-10-02", "2026-10-03", "2026-10-04", "2026-10-05", "2026-10-06", "2026-10-07"} {
		os.WriteFile(filepath.Join(dir, "bel-"+tgl+".db"), []byte("lama"), 0o644)
	}
	now := time.Date(2026, 10, 15, 2, 0, 0, 0, time.Local)
	if err := Harian(ctx, st, dir, now, 7); err != nil {
		t.Fatal(err)
	}
	if err := Harian(ctx, st, dir, now.Add(time.Hour), 7); err != nil {
		t.Fatalf("panggilan kedua di hari yang sama harus aman: %v", err)
	}
	ents, _ := os.ReadDir(dir)
	var nama []string
	for _, e := range ents {
		nama = append(nama, e.Name())
	}
	if len(nama) != 7 || nama[0] != "bel-2026-10-02.db" || nama[6] != "bel-2026-10-15.db" {
		t.Fatalf("isi folder backup = %v", nama)
	}
	baru, err := store.Open(filepath.Join(dir, "bel-2026-10-15.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer baru.Close()
	if n, _ := baru.CountProfil(ctx); n != 1 {
		t.Errorf("backup harian berisi %d profil, mau 1", n)
	}
}

func TestTulisZip(t *testing.T) {
	dataDir, st := siapkanData(t)
	var buf bytes.Buffer
	if err := TulisZip(ctx, &buf, st, filepath.Join(dataDir, "audio")); err != nil {
		t.Fatal(err)
	}
	got := namaDiZip(t, buf.Bytes())
	want := []string{"audio/a.wav", "audio/b.mp3", "bel.db"}
	if len(got) != len(want) {
		t.Fatalf("isi zip = %v, mau %v (file berawalan titik dilewati)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("isi zip = %v, mau %v", got, want)
		}
	}
}

func TestSiapkanDanTerapkan(t *testing.T) {
	asal, st := siapkanData(t)
	zipPath := filepath.Join(t.TempDir(), "backup.zip")
	f, _ := os.Create(zipPath)
	if err := TulisZip(ctx, f, st, filepath.Join(asal, "audio")); err != nil {
		t.Fatal(err)
	}
	f.Close()

	tujuan := t.TempDir()
	os.MkdirAll(filepath.Join(tujuan, "audio"), 0o755)
	os.WriteFile(filepath.Join(tujuan, "audio", "lama.wav"), []byte("lama"), 0o644)
	stLama, _ := store.Open(filepath.Join(tujuan, "bel.db"))
	stLama.CreateProfil(ctx, "Lama")
	stLama.CreateProfil(ctx, "Lama 2")
	stLama.Close()

	if err := Siapkan(ctx, zipPath, tujuan); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 15, 9, 30, 0, 0, time.Local)
	diterapkan, err := TerapkanTertunda(tujuan, now)
	if err != nil || !diterapkan {
		t.Fatalf("TerapkanTertunda = %v, %v", diterapkan, err)
	}
	if _, err := os.Stat(filepath.Join(tujuan, "restore-tertunda")); !os.IsNotExist(err) {
		t.Error("folder tertunda harus dihapus setelah diterapkan")
	}
	hasil, err := store.Open(filepath.Join(tujuan, "bel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer hasil.Close()
	if n, _ := hasil.CountProfil(ctx); n != 1 {
		t.Errorf("profil setelah restore = %d, mau 1 (dari backup)", n)
	}
	if _, err := os.Stat(filepath.Join(tujuan, "audio", "a.wav")); err != nil {
		t.Error("audio dari backup harus ada")
	}
	if _, err := os.Stat(filepath.Join(tujuan, "audio", "lama.wav")); !os.IsNotExist(err) {
		t.Error("audio lama harus dipindahkan")
	}
	if _, err := os.Stat(filepath.Join(tujuan, "backup", "bel-sebelum-restore-20261015-093000.db")); err != nil {
		t.Errorf("database lama harus disimpan: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tujuan, "backup", "audio-sebelum-restore-20261015-093000", "lama.wav")); err != nil {
		t.Errorf("audio lama harus disimpan: %v", err)
	}

	again, err := TerapkanTertunda(tujuan, now)
	if err != nil || again {
		t.Errorf("tanpa restore tertunda: %v, %v", again, err)
	}
}

func TestSiapkanMenolakIsiTidakValid(t *testing.T) {
	_, st := siapkanData(t)
	snap := filepath.Join(t.TempDir(), "snap.db")
	if err := st.BackupKe(ctx, snap); err != nil {
		t.Fatal(err)
	}
	db, _ := os.ReadFile(snap)
	kosong := filepath.Join(t.TempDir(), "kosong.db")
	stKosong, _ := store.Open(kosong)
	stKosong.Close()
	dbKosong, _ := os.ReadFile(kosong)

	kasus := map[string]map[string][]byte{
		"tanpa bel.db":         {"audio/a.wav": []byte("x")},
		"path traversal":       {"bel.db": db, "audio/../evil": []byte("x")},
		"folder bertingkat":    {"bel.db": db, "audio/sub/a.wav": []byte("x")},
		"file asing":           {"bel.db": db, "etc/passwd": []byte("x")},
		"db rusak":             {"bel.db": []byte("bukan sqlite")},
		"db tanpa profil":      {"bel.db": dbKosong},
		"nama berawalan titik": {"bel.db": db, "audio/.x": []byte("x")},
	}
	for nama, isi := range kasus {
		t.Run(nama, func(t *testing.T) {
			dataDir := t.TempDir()
			zipPath := filepath.Join(t.TempDir(), "x.zip")
			buatZip(t, zipPath, isi)
			err := Siapkan(ctx, zipPath, dataDir)
			if !errors.Is(err, ErrBackupTidakValid) {
				t.Fatalf("err = %v, mau ErrBackupTidakValid", err)
			}
			if _, err := os.Stat(filepath.Join(dataDir, "restore-tertunda")); !os.IsNotExist(err) {
				t.Error("restore tidak valid tidak boleh meninggalkan folder tertunda")
			}
		})
	}
	bukanZip := filepath.Join(t.TempDir(), "x.zip")
	os.WriteFile(bukanZip, []byte("bukan zip"), 0o644)
	if err := Siapkan(ctx, bukanZip, t.TempDir()); !errors.Is(err, ErrBackupTidakValid) {
		t.Errorf("bukan zip: %v", err)
	}
}

func TestSiapkanMenolakIsiGanda(t *testing.T) {
	_, st := siapkanData(t)
	snap := filepath.Join(t.TempDir(), "snap.db")
	if err := st.BackupKe(ctx, snap); err != nil {
		t.Fatal(err)
	}
	db, _ := os.ReadFile(snap)
	kasus := map[string][]entri{
		"bel.db ganda":      {{"bel.db", db}, {"bel.db", db}},
		"audio/a.wav ganda": {{"bel.db", db}, {"audio/a.wav", []byte("x")}, {"audio/a.wav", []byte("y")}},
	}
	for nama, isi := range kasus {
		t.Run(nama, func(t *testing.T) {
			dataDir := t.TempDir()
			zipPath := filepath.Join(t.TempDir(), "x.zip")
			buatZipUrut(t, zipPath, isi)
			err := Siapkan(ctx, zipPath, dataDir)
			if !errors.Is(err, ErrBackupTidakValid) {
				t.Fatalf("err = %v, mau ErrBackupTidakValid", err)
			}
			if _, err := os.Stat(filepath.Join(dataDir, "restore-tertunda")); !os.IsNotExist(err) {
				t.Error("tidak boleh meninggalkan folder tertunda")
			}
		})
	}
}

func TestTerapkanTertundaSebagian(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 30, 0, 0, time.Local)
	tulis := func(p, isi string) {
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(isi), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	baca := func(p string) string { b, _ := os.ReadFile(p); return string(b) }
	ada := func(p string) bool { _, err := os.Stat(p); return err == nil }

	t.Run("hanya bel.db", func(t *testing.T) {
		d := t.TempDir()
		tulis(filepath.Join(d, "bel.db"), "lama")
		tulis(filepath.Join(d, "audio", "baru.wav"), "baru")
		tulis(filepath.Join(d, "restore-tertunda", "bel.db"), "baru")
		ok, err := TerapkanTertunda(d, now)
		if err != nil || !ok {
			t.Fatalf("= %v, %v", ok, err)
		}
		if baca(filepath.Join(d, "bel.db")) != "baru" {
			t.Error("db harus terpasang")
		}
		if ada(filepath.Join(d, "backup", "audio-sebelum-restore-20261015-093000")) {
			t.Error("audio saat ini tidak boleh diarsipkan")
		}
		if !ada(filepath.Join(d, "audio", "baru.wav")) {
			t.Error("audio harus tetap ada")
		}
		if ada(filepath.Join(d, "restore-tertunda")) {
			t.Error("folder tertunda harus dihapus")
		}
	})
	t.Run("hanya audio", func(t *testing.T) {
		d := t.TempDir()
		tulis(filepath.Join(d, "bel.db"), "hidup")
		tulis(filepath.Join(d, "audio", "lama.wav"), "lama")
		tulis(filepath.Join(d, "restore-tertunda", "audio", "baru.wav"), "baru")
		ok, err := TerapkanTertunda(d, now)
		if err != nil || !ok {
			t.Fatalf("= %v, %v", ok, err)
		}
		if baca(filepath.Join(d, "bel.db")) != "hidup" {
			t.Error("db tidak boleh disentuh")
		}
		if !ada(filepath.Join(d, "audio", "baru.wav")) || ada(filepath.Join(d, "audio", "lama.wav")) {
			t.Error("audio harus diganti")
		}
		if ada(filepath.Join(d, "backup", "bel-sebelum-restore-20261015-093000.db")) {
			t.Error("db tidak boleh diarsipkan")
		}
	})
	t.Run("kosong", func(t *testing.T) {
		d := t.TempDir()
		os.MkdirAll(filepath.Join(d, "restore-tertunda"), 0o755)
		ok, err := TerapkanTertunda(d, now)
		if err != nil || !ok {
			t.Fatalf("= %v, %v", ok, err)
		}
		if ada(filepath.Join(d, "restore-tertunda")) {
			t.Error("folder tertunda harus dihapus")
		}
	})
}
