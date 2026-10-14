// Package backup membuat backup harian, file zip backup, dan menerapkan restore saat start.
package backup

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

// ErrBackupTidakValid membungkus semua kesalahan isi file backup.
var ErrBackupTidakValid = errors.New("file backup tidak valid")

const (
	SimpanHarian   = 7
	folderTertunda = "restore-tertunda"
	namaDB         = "bel.db"
	awalanAudio    = "audio/"
	batasEkstrak   = 1 << 30 // total 1 GB, pelindung zip bomb
)

func tidakValid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrBackupTidakValid, fmt.Sprintf(format, a...))
}

// Harian membuat backup database untuk tanggal `now` jika belum ada, lalu menyisakan `simpan` file terbaru.
func Harian(ctx context.Context, st *store.Store, dir string, now time.Time, simpan int) error {
	if simpan < 1 {
		simpan = 1
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tgl := now.Format(schedule.FormatTanggal)
	tujuan := filepath.Join(dir, "bel-"+tgl+".db")
	if _, err := os.Stat(tujuan); errors.Is(err, os.ErrNotExist) {
		tmp := filepath.Join(dir, ".bel-"+tgl+".db.tmp")
		os.Remove(tmp)
		if err := st.BackupKe(ctx, tmp); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("backup harian: %w", err)
		}
		if err := os.Rename(tmp, tujuan); err != nil {
			os.Remove(tmp)
			return fmt.Errorf("backup harian: %w", err)
		}
	} else if err != nil {
		return err
	}
	lama, err := filepath.Glob(filepath.Join(dir, "bel-????-??-??.db"))
	if err != nil {
		return err
	}
	sort.Strings(lama)
	for len(lama) > simpan {
		if err := os.Remove(lama[0]); err != nil {
			return err
		}
		lama = lama[1:]
	}
	return nil
}

// JalankanHarian memeriksa backup harian saat start lalu setiap jam sampai ctx selesai.
func JalankanHarian(ctx context.Context, st *store.Store, dir string, log *slog.Logger) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if err := Harian(ctx, st, dir, time.Now(), SimpanHarian); err != nil {
			log.Error("backup harian gagal", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// TulisZip menulis snapshot database dan seluruh file audio ke w.
func TulisZip(ctx context.Context, w io.Writer, st *store.Store, audioDir string) error {
	tmp, err := os.MkdirTemp("", "belsekolah-backup-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	snap := filepath.Join(tmp, namaDB)
	if err := st.BackupKe(ctx, snap); err != nil {
		return fmt.Errorf("snapshot database: %w", err)
	}

	zw := zip.NewWriter(w)
	if err := tambahFile(zw, namaDB, snap); err != nil {
		return err
	}
	ents, err := os.ReadDir(audioDir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if !e.Type().IsRegular() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if err := tambahFile(zw, awalanAudio+e.Name(), filepath.Join(audioDir, e.Name())); err != nil {
			return err
		}
	}
	return zw.Close()
}

func tambahFile(zw *zip.Writer, nama, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w, err := zw.Create(nama)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// Siapkan memvalidasi zip backup lalu mengekstraknya ke folder tertunda.
// Restore baru diterapkan oleh TerapkanTertunda saat aplikasi start berikutnya.
func Siapkan(ctx context.Context, zipPath, dataDir string) (err error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return tidakValid("bukan file zip")
	}
	defer zr.Close()

	staging := filepath.Join(dataDir, folderTertunda+".baru")
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(staging)
		}
	}()
	if err := os.MkdirAll(filepath.Join(staging, "audio"), 0o755); err != nil {
		return err
	}

	adaDB := false
	var total int64
	terlihat := map[string]bool{}
	for _, f := range zr.File {
		if terlihat[f.Name] {
			return tidakValid("isi ganda %q", f.Name)
		}
		terlihat[f.Name] = true
		switch {
		case f.Name == namaDB:
			adaDB = true
			if err := ekstrak(f, filepath.Join(staging, namaDB), &total); err != nil {
				return err
			}
		case f.Name == awalanAudio:
			continue
		case strings.HasPrefix(f.Name, awalanAudio):
			nama := strings.TrimPrefix(f.Name, awalanAudio)
			if nama == "" || nama != filepath.Base(nama) || strings.HasPrefix(nama, ".") ||
				strings.ContainsAny(nama, `/\`) || !f.Mode().IsRegular() {
				return tidakValid("isi tidak dikenal %q", f.Name)
			}
			if err := ekstrak(f, filepath.Join(staging, "audio", nama), &total); err != nil {
				return err
			}
		default:
			return tidakValid("isi tidak dikenal %q", f.Name)
		}
	}
	if !adaDB {
		return tidakValid("tidak berisi %s", namaDB)
	}
	if err := periksaDB(ctx, filepath.Join(staging, namaDB)); err != nil {
		return err
	}

	final := filepath.Join(dataDir, folderTertunda)
	if err := os.RemoveAll(final); err != nil {
		return err
	}
	return os.Rename(staging, final)
}

func ekstrak(f *zip.File, tujuan string, total *int64) error {
	r, err := f.Open()
	if err != nil {
		return tidakValid("tidak bisa membaca %q", f.Name)
	}
	defer r.Close()
	w, err := os.OpenFile(tujuan, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return tidakValid("tidak bisa menulis %q", f.Name)
	}
	n, err := io.Copy(w, io.LimitReader(r, batasEkstrak-*total+1))
	cerr := w.Close()
	*total += n
	if *total > batasEkstrak {
		return tidakValid("isi terlalu besar")
	}
	if err != nil {
		return tidakValid("tidak bisa membaca %q", f.Name)
	}
	return cerr
}

func periksaDB(ctx context.Context, path string) error {
	st, err := store.Open(path)
	if err != nil {
		return tidakValid("database rusak")
	}
	defer st.Close()
	if _, err := st.LoadDataJadwal(ctx); err != nil {
		return tidakValid("database rusak")
	}
	n, err := st.CountProfil(ctx)
	if err != nil || n == 0 {
		return tidakValid("database tidak berisi profil jadwal")
	}
	return nil
}

// TerapkanTertunda menerapkan restore yang disiapkan, sebelum database dibuka.
// Data lama dipindahkan ke <data>/backup dengan akhiran waktu.
//
// Urutan aman diulang: audio lebih dulu, lalu database. Setiap bagian hanya
// dijalankan jika sumbernya masih ada di folder tertunda, dan sumber itu dipindah
// (rename) sebagai langkah terakhir bagian tersebut. Jadi bagian yang sudah
// selesai tidak diulang pada percobaan berikutnya, database baru tidak pernah
// diarsipkan, dan database lama baru diarsipkan tepat sebelum penggantinya
// dipasang sehingga folder data tidak pernah tanpa database.
func TerapkanTertunda(dataDir string, now time.Time) (bool, error) {
	tertunda := filepath.Join(dataDir, folderTertunda)
	if _, err := os.Stat(tertunda); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("terapkan restore: %w", err)
	}
	gagal := func(err error) (bool, error) { return false, fmt.Errorf("terapkan restore: %w", err) }
	ada := func(p string) bool { _, err := os.Stat(p); return err == nil }

	arsip := filepath.Join(dataDir, "backup")
	stempel := now.Format("20060102-150405")
	audio := filepath.Join(dataDir, "audio")
	if ada(filepath.Join(tertunda, "audio")) {
		if err := os.MkdirAll(arsip, 0o755); err != nil {
			return gagal(err)
		}
		if ada(audio) {
			if err := os.Rename(audio, filepath.Join(arsip, "audio-sebelum-restore-"+stempel)); err != nil {
				return gagal(err)
			}
		}
		if err := os.Rename(filepath.Join(tertunda, "audio"), audio); err != nil {
			return gagal(err)
		}
	}
	if ada(filepath.Join(tertunda, namaDB)) {
		if err := os.MkdirAll(arsip, 0o755); err != nil {
			return gagal(err)
		}
		db := filepath.Join(dataDir, namaDB)
		for _, akhiran := range []string{"", "-wal", "-shm"} {
			if ada(db + akhiran) {
				if err := os.Rename(db+akhiran, filepath.Join(arsip, "bel-sebelum-restore-"+stempel+".db"+akhiran)); err != nil {
					return gagal(err)
				}
			}
		}
		if err := os.Rename(filepath.Join(tertunda, namaDB), db); err != nil {
			return gagal(err)
		}
	}
	if err := os.RemoveAll(tertunda); err != nil {
		return gagal(err)
	}
	return true, nil
}
