package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Audio adalah satu file suara. File relatif terhadap folder audio.
type Audio struct {
	ID       int64
	Nama     string
	Sumber   string // bawaan, upload, tts
	File     string
	DurasiMS int64
	TeksTTS  string
	SuaraTTS string
	Dibuat   time.Time
}

type JenisBel struct {
	ID    int64
	Nama  string
	Warna string
}

const kolomAudio = `a.id, a.nama, a.sumber, a.file, a.durasi_ms, a.teks_tts, a.suara_tts, a.dibuat`

type pemindai interface {
	Scan(dest ...any) error
}

func scanAudio(r pemindai) (Audio, error) {
	var a Audio
	var dibuat string
	err := r.Scan(&a.ID, &a.Nama, &a.Sumber, &a.File, &a.DurasiMS, &a.TeksTTS, &a.SuaraTTS, &dibuat)
	if errors.Is(err, sql.ErrNoRows) {
		return Audio{}, ErrTidakAda
	}
	if err != nil {
		return Audio{}, err
	}
	if a.Dibuat, err = time.Parse(time.RFC3339, dibuat); err != nil {
		return Audio{}, fmt.Errorf("waktu dibuat audio %d tidak valid: %w", a.ID, err)
	}
	return a, nil
}

func (s *Store) CreateAudio(ctx context.Context, a Audio) (int64, error) {
	return s.insert(ctx,
		`INSERT INTO audio (nama, sumber, file, durasi_ms, teks_tts, suara_tts, dibuat)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.Nama, a.Sumber, a.File, a.DurasiMS, a.TeksTTS, a.SuaraTTS, a.Dibuat.Format(time.RFC3339))
}

func (s *Store) GetAudio(ctx context.Context, id int64) (Audio, error) {
	return scanAudio(s.db.QueryRowContext(ctx, `SELECT `+kolomAudio+` FROM audio a WHERE a.id = ?`, id))
}

func (s *Store) FindAudio(ctx context.Context, sumber, nama string) (Audio, error) {
	return scanAudio(s.db.QueryRowContext(ctx,
		`SELECT `+kolomAudio+` FROM audio a WHERE a.sumber = ? AND a.nama = ? ORDER BY a.id LIMIT 1`, sumber, nama))
}

func (s *Store) CreateJenisBel(ctx context.Context, nama, warna string) (int64, error) {
	return s.insert(ctx, `INSERT INTO jenis_bel (nama, warna) VALUES (?, ?)`, nama, warna)
}

func (s *Store) ListJenisBel(ctx context.Context) ([]JenisBel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, nama, warna FROM jenis_bel ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JenisBel
	for rows.Next() {
		var j JenisBel
		if err := rows.Scan(&j.ID, &j.Nama, &j.Warna); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// SetUrutanBel mengganti seluruh urutan audio milik sebuah jenis bel.
func (s *Store) SetUrutanBel(ctx context.Context, jenisBelID int64, audioIDs []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM urutan_bel WHERE jenis_bel_id = ?`, jenisBelID); err != nil {
		return err
	}
	for i, audioID := range audioIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO urutan_bel (jenis_bel_id, urutan, audio_id) VALUES (?, ?, ?)`,
			jenisBelID, i+1, audioID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// UrutanBel mengembalikan audio milik jenis bel sesuai urutan putar.
func (s *Store) UrutanBel(ctx context.Context, jenisBelID int64) ([]Audio, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+kolomAudio+` FROM urutan_bel u JOIN audio a ON a.id = u.audio_id
		 WHERE u.jenis_bel_id = ? ORDER BY u.urutan`, jenisBelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Audio
	for rows.Next() {
		a, err := scanAudio(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
