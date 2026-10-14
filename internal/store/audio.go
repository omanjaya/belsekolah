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

func (s *Store) GetAudio(ctx context.Context, id int64) (Audio, error) {
	return scanAudio(s.db.QueryRowContext(ctx, `SELECT `+kolomAudio+` FROM audio a WHERE a.id = ?`, id))
}

func (s *Store) FindAudio(ctx context.Context, sumber, nama string) (Audio, error) {
	return scanAudio(s.db.QueryRowContext(ctx,
		`SELECT `+kolomAudio+` FROM audio a WHERE a.sumber = ? AND a.nama = ? ORDER BY a.id LIMIT 1`, sumber, nama))
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

func (s *Store) CreateAudio(ctx context.Context, a Audio) (int64, error) {
	nama, err := validasiNama(a.Nama)
	if err != nil {
		return 0, err
	}
	return s.insert(ctx,
		`INSERT INTO audio (nama, sumber, file, durasi_ms, teks_tts, suara_tts, dibuat)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		nama, a.Sumber, a.File, a.DurasiMS, a.TeksTTS, a.SuaraTTS, a.Dibuat.Format(time.RFC3339))
}

func (s *Store) ListAudio(ctx context.Context) ([]Audio, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+kolomAudio+` FROM audio a ORDER BY a.id`)
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

func (s *Store) UpdateNamaAudio(ctx context.Context, id int64, nama string) error {
	nama, err := validasiNama(nama)
	if err != nil {
		return err
	}
	return cekDiubah(s.db.ExecContext(ctx, `UPDATE audio SET nama = ? WHERE id = ?`, nama, id))
}

// DeleteAudio menghapus baris audio dan mengembalikannya agar pemanggil bisa menghapus filenya.
func (s *Store) DeleteAudio(ctx context.Context, id int64) (Audio, error) {
	a, err := s.GetAudio(ctx, id)
	if err != nil {
		return Audio{}, err
	}
	if a.Sumber == "bawaan" {
		return Audio{}, dipakai("suara bawaan tidak bisa dihapus")
	}
	err = terjemahkan(
		cekDiubah(s.db.ExecContext(ctx, `DELETE FROM audio WHERE id = ?`, id)),
		dipakai("suara masih dipakai oleh jenis bel"))
	if err != nil {
		return Audio{}, err
	}
	return a, nil
}

func (s *Store) CreateJenisBel(ctx context.Context, nama, warna string) (int64, error) {
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	if err := validasiWarna(warna); err != nil {
		return 0, err
	}
	id, err := s.insert(ctx, `INSERT INTO jenis_bel (nama, warna) VALUES (?, ?)`, nama, warna)
	return id, terjemahkan(err, nil)
}

func (s *Store) GetJenisBel(ctx context.Context, id int64) (JenisBel, error) {
	var j JenisBel
	err := s.db.QueryRowContext(ctx, `SELECT id, nama, warna FROM jenis_bel WHERE id = ?`, id).
		Scan(&j.ID, &j.Nama, &j.Warna)
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrTidakAda
	}
	return j, err
}

func (s *Store) UpdateJenisBel(ctx context.Context, id int64, nama, warna string) error {
	nama, err := validasiNama(nama)
	if err != nil {
		return err
	}
	if err := validasiWarna(warna); err != nil {
		return err
	}
	return terjemahkan(cekDiubah(s.db.ExecContext(ctx,
		`UPDATE jenis_bel SET nama = ?, warna = ? WHERE id = ?`, nama, warna, id)), nil)
}

func (s *Store) DeleteJenisBel(ctx context.Context, id int64) error {
	return terjemahkan(
		cekDiubah(s.db.ExecContext(ctx, `DELETE FROM jenis_bel WHERE id = ?`, id)),
		dipakai("jenis bel masih dipakai di jadwal"))
}

// SetUrutanBel mengganti seluruh urutan audio milik sebuah jenis bel (1–10 audio).
func (s *Store) SetUrutanBel(ctx context.Context, jenisBelID int64, audioIDs []int64) error {
	if len(audioIDs) == 0 {
		return validasi("pilih minimal satu suara")
	}
	if len(audioIDs) > 10 {
		return validasi("maksimal 10 suara dalam satu jenis bel")
	}
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
			return terjemahkan(err, validasi("suara atau jenis bel tidak ditemukan"))
		}
	}
	return tx.Commit()
}
