package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"belsekolah/internal/schedule"
)

// Riwayat adalah satu catatan bel. JadwalID/JenisBelID 0 berarti kosong.
type Riwayat struct {
	ID         int64
	Waktu      time.Time
	JadwalID   int64
	JenisBelID int64
	Label      string
	Sumber     string
	Status     string
	Pesan      string
}

func nullID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// InsertRiwayat menyimpan riwayat; kolom tanggal diambil dari Waktu (zona waktu Waktu).
func (s *Store) InsertRiwayat(ctx context.Context, r Riwayat) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO riwayat_bel (waktu, tanggal, jadwal_id, jenis_bel_id, label, sumber, status, pesan)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Waktu.UTC().Format(time.RFC3339), r.Waktu.Format(schedule.FormatTanggal),
		nullID(r.JadwalID), nullID(r.JenisBelID), r.Label, r.Sumber, r.Status, r.Pesan)
	return err
}

// JadwalTercatat mengembalikan ID jadwal yang sudah punya riwayat pada tanggal itu.
func (s *Store) JadwalTercatat(ctx context.Context, tanggal string) (map[int64]bool, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT jadwal_id FROM riwayat_bel WHERE tanggal = ? AND jadwal_id IS NOT NULL`, tanggal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// FilterRiwayat menyaring riwayat; nilai kosong berarti tanpa filter. Batas default 200, maks 1000.
type FilterRiwayat struct {
	Tanggal string
	Status  string
	Batas   int
}

var statusRiwayat = map[string]bool{
	"berhasil": true, "gagal": true, "terlewat": true, "cadangan": true, "dihentikan": true,
}

func (s *Store) CariRiwayat(ctx context.Context, f FilterRiwayat) ([]Riwayat, error) {
	if f.Batas <= 0 || f.Batas > 1000 {
		f.Batas = 200
	}
	q := `SELECT id, waktu, jadwal_id, jenis_bel_id, label, sumber, status, pesan FROM riwayat_bel WHERE 1 = 1`
	var args []any
	if f.Tanggal != "" {
		if _, err := time.Parse(schedule.FormatTanggal, f.Tanggal); err != nil {
			return nil, validasi("tanggal tidak valid %q, gunakan format YYYY-MM-DD", f.Tanggal)
		}
		q += ` AND tanggal = ?`
		args = append(args, f.Tanggal)
	}
	if f.Status != "" {
		if !statusRiwayat[f.Status] {
			return nil, validasi("status riwayat tidak dikenal %q", f.Status)
		}
		q += ` AND status = ?`
		args = append(args, f.Status)
	}
	q += ` ORDER BY waktu DESC, id DESC LIMIT ?`
	args = append(args, f.Batas)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Riwayat
	for rows.Next() {
		r, err := scanRiwayat(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RiwayatTerbaru mengembalikan riwayat terbaru lebih dulu.
func (s *Store) RiwayatTerbaru(ctx context.Context, batas int) ([]Riwayat, error) {
	return s.CariRiwayat(ctx, FilterRiwayat{Batas: batas})
}

func scanRiwayat(rows *sql.Rows) (Riwayat, error) {
	var r Riwayat
	var waktu string
	var jadwalID, jenisBelID sql.NullInt64
	if err := rows.Scan(&r.ID, &waktu, &jadwalID, &jenisBelID, &r.Label, &r.Sumber, &r.Status, &r.Pesan); err != nil {
		return r, err
	}
	t, err := time.Parse(time.RFC3339, waktu)
	if err != nil {
		return r, fmt.Errorf("waktu riwayat %d tidak valid: %w", r.ID, err)
	}
	r.Waktu = t.Local()
	r.JadwalID, r.JenisBelID = jadwalID.Int64, jenisBelID.Int64
	return r, nil
}
