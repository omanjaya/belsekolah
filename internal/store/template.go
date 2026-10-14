package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"belsekolah/internal/formatjadwal"
)

// Template adalah jadwal tersimpan yang bisa dipakai untuk membuat profil baru.
type Template struct {
	ID     int64
	Nama   string
	Data   formatjadwal.Data
	Dibuat time.Time
}

func (s *Store) CreateTemplate(ctx context.Context, nama string, d formatjadwal.Data, now time.Time) (int64, error) {
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	d.Nama = nama
	if err := d.Validasi(); err != nil {
		return 0, dariFormat(err)
	}
	b, err := json.Marshal(d)
	if err != nil {
		return 0, err
	}
	id, err := s.insert(ctx, `INSERT INTO template (nama, data, dibuat) VALUES (?, ?, ?)`,
		nama, string(b), now.Format(time.RFC3339))
	return id, terjemahkan(err, nil)
}

func scanTemplate(r pemindai) (Template, error) {
	var t Template
	var data, dibuat string
	err := r.Scan(&t.ID, &t.Nama, &data, &dibuat)
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrTidakAda
	}
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal([]byte(data), &t.Data); err != nil {
		return t, fmt.Errorf("data template %d rusak: %w", t.ID, err)
	}
	if t.Dibuat, err = time.Parse(time.RFC3339, dibuat); err != nil {
		return t, fmt.Errorf("tanggal template %d rusak: %w", t.ID, err)
	}
	return t, nil
}

func (s *Store) ListTemplate(ctx context.Context) ([]Template, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, nama, data, dibuat FROM template ORDER BY nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTemplate(ctx context.Context, id int64) (Template, error) {
	return scanTemplate(s.db.QueryRowContext(ctx, `SELECT id, nama, data, dibuat FROM template WHERE id = ?`, id))
}

// UpdateNamaTemplate mengganti nama template sekaligus nama di dalam datanya.
func (s *Store) UpdateNamaTemplate(ctx context.Context, id int64, nama string) error {
	nama, err := validasiNama(nama)
	if err != nil {
		return err
	}
	t, err := s.GetTemplate(ctx, id)
	if err != nil {
		return err
	}
	t.Data.Nama = nama
	b, err := json.Marshal(t.Data)
	if err != nil {
		return err
	}
	return terjemahkan(cekDiubah(s.db.ExecContext(ctx,
		`UPDATE template SET nama = ?, data = ? WHERE id = ?`, nama, string(b), id)), nil)
}

func (s *Store) DeleteTemplate(ctx context.Context, id int64) error {
	return cekDiubah(s.db.ExecContext(ctx, `DELETE FROM template WHERE id = ?`, id))
}
