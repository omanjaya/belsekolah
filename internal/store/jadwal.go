package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"belsekolah/internal/schedule"
)

func (s *Store) CreateProfil(ctx context.Context, nama string) (int64, error) {
	return s.insert(ctx, `INSERT INTO profil (nama) VALUES (?)`, nama)
}

func (s *Store) CountProfil(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profil`).Scan(&n)
	return n, err
}

func (s *Store) CreateJadwal(ctx context.Context, j schedule.Jadwal) (int64, error) {
	if _, _, err := schedule.ParseJam(j.Jam); err != nil {
		return 0, err
	}
	if j.Hari < 1 || j.Hari > 7 {
		return 0, fmt.Errorf("hari harus 1 (Senin) sampai 7 (Minggu), bukan %d", j.Hari)
	}
	return s.insert(ctx,
		`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label) VALUES (?, ?, ?, ?, ?)`,
		j.ProfilID, j.Hari, j.Jam, j.JenisBelID, j.Label)
}

func validasiRentang(mulai, selesai string) error {
	a, err := time.Parse(schedule.FormatTanggal, mulai)
	if err != nil {
		return fmt.Errorf("tanggal mulai tidak valid %q, gunakan format YYYY-MM-DD", mulai)
	}
	b, err := time.Parse(schedule.FormatTanggal, selesai)
	if err != nil {
		return fmt.Errorf("tanggal selesai tidak valid %q, gunakan format YYYY-MM-DD", selesai)
	}
	if b.Before(a) {
		return errors.New("tanggal selesai tidak boleh sebelum tanggal mulai")
	}
	return nil
}

// CreatePeriode menolak periode yang bertabrakan dengan periode lain.
func (s *Store) CreatePeriode(ctx context.Context, p schedule.Periode) (int64, error) {
	if err := validasiRentang(p.Mulai, p.Selesai); err != nil {
		return 0, err
	}
	// Cek tumpang tindih dan INSERT dalam satu transaksi agar pemanggil bersamaan tidak lolos bersama.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM periode_khusus WHERE tgl_mulai <= ? AND tgl_selesai >= ?`,
		p.Selesai, p.Mulai).Scan(&n); err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, ErrPeriodeTumpangTindih
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO periode_khusus (tgl_mulai, tgl_selesai, profil_id, keterangan) VALUES (?, ?, ?, ?)`,
		p.Mulai, p.Selesai, p.ProfilID, p.Keterangan)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s *Store) CreateLibur(ctx context.Context, l schedule.Libur) (int64, error) {
	if err := validasiRentang(l.Mulai, l.Selesai); err != nil {
		return 0, err
	}
	return s.insert(ctx,
		`INSERT INTO libur (tgl_mulai, tgl_selesai, keterangan) VALUES (?, ?, ?)`,
		l.Mulai, l.Selesai, l.Keterangan)
}

// LoadDataJadwal memuat semua data yang dibutuhkan schedule.BelPadaTanggal.
func (s *Store) LoadDataJadwal(ctx context.Context) (schedule.Data, error) {
	var d schedule.Data
	nilai, err := s.GetSetting(ctx, KunciProfilDefault)
	switch {
	case errors.Is(err, ErrTidakAda):
	case err != nil:
		return d, err
	default:
		if d.ProfilDefaultID, err = strconv.ParseInt(nilai, 10, 64); err != nil {
			return d, fmt.Errorf("profil default tidak valid %q: %w", nilai, err)
		}
	}
	if d.Jadwal, err = s.semuaJadwal(ctx); err != nil {
		return d, err
	}
	if d.Periode, err = s.semuaPeriode(ctx); err != nil {
		return d, err
	}
	if d.Libur, err = s.semuaLibur(ctx); err != nil {
		return d, err
	}
	return d, nil
}

func (s *Store) semuaJadwal(ctx context.Context) ([]schedule.Jadwal, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, profil_id, hari, jam, jenis_bel_id, label FROM jadwal ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schedule.Jadwal
	for rows.Next() {
		var j schedule.Jadwal
		if err := rows.Scan(&j.ID, &j.ProfilID, &j.Hari, &j.Jam, &j.JenisBelID, &j.Label); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) semuaPeriode(ctx context.Context) ([]schedule.Periode, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, tgl_mulai, tgl_selesai, profil_id, keterangan FROM periode_khusus ORDER BY tgl_mulai`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schedule.Periode
	for rows.Next() {
		var p schedule.Periode
		if err := rows.Scan(&p.ID, &p.Mulai, &p.Selesai, &p.ProfilID, &p.Keterangan); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) semuaLibur(ctx context.Context) ([]schedule.Libur, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, tgl_mulai, tgl_selesai, keterangan FROM libur ORDER BY tgl_mulai`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schedule.Libur
	for rows.Next() {
		var l schedule.Libur
		if err := rows.Scan(&l.ID, &l.Mulai, &l.Selesai, &l.Keterangan); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
