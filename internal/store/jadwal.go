package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"belsekolah/internal/schedule"
)

func (s *Store) CountProfil(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM profil`).Scan(&n)
	return n, err
}

type Profil struct {
	ID   int64
	Nama string
}

func (s *Store) CreateProfil(ctx context.Context, nama string) (int64, error) {
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	id, err := s.insert(ctx, `INSERT INTO profil (nama) VALUES (?)`, nama)
	return id, terjemahkan(err, nil)
}

func (s *Store) ListProfil(ctx context.Context) ([]Profil, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, nama FROM profil ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Profil
	for rows.Next() {
		var p Profil
		if err := rows.Scan(&p.ID, &p.Nama); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProfil(ctx context.Context, id int64) (Profil, error) {
	var p Profil
	err := s.db.QueryRowContext(ctx, `SELECT id, nama FROM profil WHERE id = ?`, id).Scan(&p.ID, &p.Nama)
	if errors.Is(err, sql.ErrNoRows) {
		return Profil{}, ErrTidakAda
	}
	return p, err
}

func (s *Store) UpdateProfil(ctx context.Context, id int64, nama string) error {
	nama, err := validasiNama(nama)
	if err != nil {
		return err
	}
	return terjemahkan(cekDiubah(s.db.ExecContext(ctx, `UPDATE profil SET nama = ? WHERE id = ?`, nama, id)), nil)
}

// DeleteProfil menolak profil default dan profil yang dipakai periode khusus.
// Jadwal milik profil ikut terhapus.
func (s *Store) DeleteProfil(ctx context.Context, id int64) error {
	def, err := s.ProfilDefault(ctx)
	if err != nil && !errors.Is(err, ErrTidakAda) {
		return err
	}
	if def == id {
		return dipakai("profil default tidak bisa dihapus")
	}
	return terjemahkan(
		cekDiubah(s.db.ExecContext(ctx, `DELETE FROM profil WHERE id = ?`, id)),
		dipakai("profil masih dipakai oleh periode khusus"))
}

// ProfilDefault mengembalikan ErrTidakAda jika belum diatur.
func (s *Store) ProfilDefault(ctx context.Context) (int64, error) {
	nilai, err := s.GetSetting(ctx, KunciProfilDefault)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(nilai, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("profil default tidak valid %q: %w", nilai, err)
	}
	return id, nil
}

func (s *Store) SetProfilDefault(ctx context.Context, id int64) error {
	if _, err := s.GetProfil(ctx, id); err != nil {
		return err
	}
	return s.SetSetting(ctx, KunciProfilDefault, strconv.FormatInt(id, 10))
}

func validasiJadwal(j schedule.Jadwal) (schedule.Jadwal, error) {
	if _, _, err := schedule.ParseJam(j.Jam); err != nil {
		return j, validasi("%s", err.Error())
	}
	if j.Hari < 1 || j.Hari > 7 {
		return j, validasi("hari harus 1 (Senin) sampai 7 (Minggu), bukan %d", j.Hari)
	}
	label, err := validasiTeks(j.Label, "label", 100, false)
	if err != nil {
		return j, err
	}
	j.Label = label
	return j, nil
}

var errRefJadwal = validasi("profil atau jenis bel tidak ditemukan")

func (s *Store) CreateJadwal(ctx context.Context, j schedule.Jadwal) (int64, error) {
	j, err := validasiJadwal(j)
	if err != nil {
		return 0, err
	}
	if err := s.cekPengumuman(ctx, j.PengumumanID); err != nil {
		return 0, err
	}
	id, err := s.insert(ctx,
		`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label, pengumuman_audio_id) VALUES (?, ?, ?, ?, ?, ?)`,
		j.ProfilID, j.Hari, j.Jam, j.JenisBelID, j.Label, nullID(j.PengumumanID))
	return id, terjemahkan(err, errRefJadwal)
}

const kolomJadwal = `id, profil_id, hari, jam, jenis_bel_id, label, pengumuman_audio_id`

func scanJadwal(r pemindai) (schedule.Jadwal, error) {
	var j schedule.Jadwal
	var png sql.NullInt64
	err := r.Scan(&j.ID, &j.ProfilID, &j.Hari, &j.Jam, &j.JenisBelID, &j.Label, &png)
	j.PengumumanID = png.Int64
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrTidakAda
	}
	return j, err
}

func (s *Store) ListJadwal(ctx context.Context, profilID int64) ([]schedule.Jadwal, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+kolomJadwal+` FROM jadwal WHERE profil_id = ? ORDER BY hari, jam`, profilID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schedule.Jadwal
	for rows.Next() {
		j, err := scanJadwal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) GetJadwal(ctx context.Context, id int64) (schedule.Jadwal, error) {
	return scanJadwal(s.db.QueryRowContext(ctx, `SELECT `+kolomJadwal+` FROM jadwal WHERE id = ?`, id))
}

func (s *Store) UpdateJadwal(ctx context.Context, j schedule.Jadwal) error {
	j, err := validasiJadwal(j)
	if err != nil {
		return err
	}
	if err := s.cekPengumuman(ctx, j.PengumumanID); err != nil {
		return err
	}
	return terjemahkan(cekDiubah(s.db.ExecContext(ctx,
		`UPDATE jadwal SET profil_id = ?, hari = ?, jam = ?, jenis_bel_id = ?, label = ?, pengumuman_audio_id = ? WHERE id = ?`,
		j.ProfilID, j.Hari, j.Jam, j.JenisBelID, j.Label, nullID(j.PengumumanID), j.ID)), errRefJadwal)
}

// cekPengumuman memastikan suara pengumuman (bila ada) benar-benar ada.
// Dipanggil di luar transaksi.
func (s *Store) cekPengumuman(ctx context.Context, id int64) error {
	if id == 0 {
		return nil
	}
	if _, err := s.GetAudio(ctx, id); err != nil {
		if errors.Is(err, ErrTidakAda) {
			return validasi("suara pengumuman tidak ditemukan")
		}
		return err
	}
	return nil
}

func (s *Store) DeleteJadwal(ctx context.Context, id int64) error {
	return cekDiubah(s.db.ExecContext(ctx, `DELETE FROM jadwal WHERE id = ?`, id))
}

// SalinHari mengganti seluruh jadwal hari `ke` dengan salinan jadwal hari `dari` pada profil yang sama.
func (s *Store) SalinHari(ctx context.Context, profilID int64, dari, ke int) error {
	if dari < 1 || dari > 7 || ke < 1 || ke > 7 {
		return validasi("hari harus 1 (Senin) sampai 7 (Minggu)")
	}
	if dari == ke {
		return validasi("hari asal dan hari tujuan tidak boleh sama")
	}
	if _, err := s.GetProfil(ctx, profilID); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM jadwal WHERE profil_id = ? AND hari = ?`, profilID, ke); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label, pengumuman_audio_id)
		 SELECT profil_id, ?, jam, jenis_bel_id, label, pengumuman_audio_id FROM jadwal WHERE profil_id = ? AND hari = ?`,
		ke, profilID, dari); err != nil {
		return err
	}
	return tx.Commit()
}

func validasiRentang(mulai, selesai string) error {
	a, err := time.Parse(schedule.FormatTanggal, mulai)
	if err != nil {
		return validasi("tanggal mulai tidak valid %q, gunakan format YYYY-MM-DD", mulai)
	}
	b, err := time.Parse(schedule.FormatTanggal, selesai)
	if err != nil {
		return validasi("tanggal selesai tidak valid %q, gunakan format YYYY-MM-DD", selesai)
	}
	if b.Before(a) {
		return validasi("tanggal selesai tidak boleh sebelum tanggal mulai")
	}
	return nil
}

func validasiKeterangan(k string) (string, error) {
	return validasiTeks(k, "keterangan", 200, false)
}

var errRefProfil = validasi("profil tidak ditemukan")

// simpanPeriode menjalankan insert (id == 0) atau update dalam satu transaksi bersama cek tumpang tindih.
func (s *Store) simpanPeriode(ctx context.Context, p schedule.Periode) (int64, error) {
	if err := validasiRentang(p.Mulai, p.Selesai); err != nil {
		return 0, err
	}
	ket, err := validasiKeterangan(p.Keterangan)
	if err != nil {
		return 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM periode_khusus WHERE tgl_mulai <= ? AND tgl_selesai >= ? AND id != ?`,
		p.Selesai, p.Mulai, p.ID).Scan(&n); err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, ErrPeriodeTumpangTindih
	}
	id := p.ID
	if id == 0 {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO periode_khusus (tgl_mulai, tgl_selesai, profil_id, keterangan) VALUES (?, ?, ?, ?)`,
			p.Mulai, p.Selesai, p.ProfilID, ket)
		if err != nil {
			return 0, terjemahkan(err, errRefProfil)
		}
		if id, err = res.LastInsertId(); err != nil {
			return 0, err
		}
	} else {
		err := cekDiubah(tx.ExecContext(ctx,
			`UPDATE periode_khusus SET tgl_mulai = ?, tgl_selesai = ?, profil_id = ?, keterangan = ? WHERE id = ?`,
			p.Mulai, p.Selesai, p.ProfilID, ket, p.ID))
		if err != nil {
			return 0, terjemahkan(err, errRefProfil)
		}
	}
	return id, tx.Commit()
}

// CreatePeriode menolak periode yang bertabrakan dengan periode lain.
func (s *Store) CreatePeriode(ctx context.Context, p schedule.Periode) (int64, error) {
	p.ID = 0
	return s.simpanPeriode(ctx, p)
}

// UpdatePeriode menolak perubahan yang membuat periode bertabrakan dengan periode lain.
func (s *Store) UpdatePeriode(ctx context.Context, p schedule.Periode) error {
	if p.ID == 0 {
		return ErrTidakAda
	}
	_, err := s.simpanPeriode(ctx, p)
	return err
}

func (s *Store) GetPeriode(ctx context.Context, id int64) (schedule.Periode, error) {
	var p schedule.Periode
	err := s.db.QueryRowContext(ctx,
		`SELECT id, tgl_mulai, tgl_selesai, profil_id, keterangan FROM periode_khusus WHERE id = ?`, id).
		Scan(&p.ID, &p.Mulai, &p.Selesai, &p.ProfilID, &p.Keterangan)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrTidakAda
	}
	return p, err
}

func (s *Store) ListPeriode(ctx context.Context) ([]schedule.Periode, error) {
	return s.semuaPeriode(ctx)
}

func (s *Store) DeletePeriode(ctx context.Context, id int64) error {
	return cekDiubah(s.db.ExecContext(ctx, `DELETE FROM periode_khusus WHERE id = ?`, id))
}

func (s *Store) CreateLibur(ctx context.Context, l schedule.Libur) (int64, error) {
	if err := validasiRentang(l.Mulai, l.Selesai); err != nil {
		return 0, err
	}
	ket, err := validasiKeterangan(l.Keterangan)
	if err != nil {
		return 0, err
	}
	return s.insert(ctx,
		`INSERT INTO libur (tgl_mulai, tgl_selesai, keterangan) VALUES (?, ?, ?)`,
		l.Mulai, l.Selesai, ket)
}

func (s *Store) GetLibur(ctx context.Context, id int64) (schedule.Libur, error) {
	var l schedule.Libur
	err := s.db.QueryRowContext(ctx,
		`SELECT id, tgl_mulai, tgl_selesai, keterangan FROM libur WHERE id = ?`, id).
		Scan(&l.ID, &l.Mulai, &l.Selesai, &l.Keterangan)
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrTidakAda
	}
	return l, err
}

func (s *Store) ListLibur(ctx context.Context) ([]schedule.Libur, error) {
	return s.semuaLibur(ctx)
}

func (s *Store) UpdateLibur(ctx context.Context, l schedule.Libur) error {
	if err := validasiRentang(l.Mulai, l.Selesai); err != nil {
		return err
	}
	ket, err := validasiKeterangan(l.Keterangan)
	if err != nil {
		return err
	}
	return cekDiubah(s.db.ExecContext(ctx,
		`UPDATE libur SET tgl_mulai = ?, tgl_selesai = ?, keterangan = ? WHERE id = ?`,
		l.Mulai, l.Selesai, ket, l.ID))
}

func (s *Store) DeleteLibur(ctx context.Context, id int64) error {
	return cekDiubah(s.db.ExecContext(ctx, `DELETE FROM libur WHERE id = ?`, id))
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
		`SELECT `+kolomJadwal+` FROM jadwal ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schedule.Jadwal
	for rows.Next() {
		j, err := scanJadwal(rows)
		if err != nil {
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
