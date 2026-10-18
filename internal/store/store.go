// Package store menyimpan seluruh data aplikasi di SQLite.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var skema string

var (
	ErrTidakAda             = errors.New("data tidak ditemukan")
	ErrPeriodeTumpangTindih = errors.New("periode khusus bertabrakan dengan periode lain")
)

// KunciProfilDefault adalah kunci pengaturan berisi ID profil default.
const KunciProfilDefault = "profil_default_id"

// Sumber bel pada riwayat.
const (
	SumberJadwal  = "jadwal"
	SumberManual  = "manual"
	SumberDarurat = "darurat"
)

type Store struct {
	db *sql.DB
}

// Open membuka (atau membuat) database dan menjalankan migrasi skema.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, err
	}
	// Satu koneksi: SQLite hanya punya satu penulis, dan ini menghindari SQLITE_BUSY.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(skema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrasi skema: %w", err)
	}
	if err := migrasi(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrasi skema: %w", err)
	}
	return &Store{db: db}, nil
}

// migrasi menambah kolom yang belum ada pada database lama. Idempoten.
func migrasi(db *sql.DB) error {
	ada, err := adaKolom(db, "jadwal", "pengumuman_audio_id")
	if err != nil || ada {
		return err
	}
	_, err = db.Exec(`ALTER TABLE jadwal ADD COLUMN pengumuman_audio_id INTEGER REFERENCES audio(id) ON DELETE RESTRICT`)
	return err
}

func adaKolom(db *sql.DB, tabel, kolom string) (bool, error) {
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, tabel)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return false, err
		}
		if n == kolom {
			return true, nil
		}
	}
	return false, rows.Err()
}

// dsn menyusun DSN SQLite; path di-escape agar karakter seperti ?, #, % dan spasi aman.
func dsn(path string) string {
	// Path relatif dijadikan absolut: url.URL salah menafsirkannya sebagai host.
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	u := url.URL{
		Scheme:   "file",
		Path:     path,
		RawQuery: "_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)",
	}
	return u.String()
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) insert(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetSetting mengembalikan ErrTidakAda jika kunci belum pernah disimpan.
func (s *Store) GetSetting(ctx context.Context, kunci string) (string, error) {
	var nilai string
	err := s.db.QueryRowContext(ctx, `SELECT nilai FROM pengaturan WHERE kunci = ?`, kunci).Scan(&nilai)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrTidakAda
	}
	return nilai, err
}

func (s *Store) SetSetting(ctx context.Context, kunci, nilai string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO pengaturan (kunci, nilai) VALUES (?, ?)
		 ON CONFLICT (kunci) DO UPDATE SET nilai = excluded.nilai`, kunci, nilai)
	return err
}
