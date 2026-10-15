package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"belsekolah/internal/formatjadwal"
)

// dariFormat mengubah error validasi format file menjadi error validasi store.
func dariFormat(err error) error {
	var v *formatjadwal.ErrorValidasi
	if errors.As(err, &v) {
		return validasi("%s", v.Pesan)
	}
	return err
}

// JumlahBelPerProfil mengembalikan jumlah bel per ID profil (profil tanpa bel tidak ada di peta).
func (s *Store) JumlahBelPerProfil(ctx context.Context) (map[int64]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT profil_id, COUNT(*) FROM jadwal GROUP BY profil_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]int)
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// DuplikatProfil membuat profil baru berisi salinan seluruh bel profil id.
func (s *Store) DuplikatProfil(ctx context.Context, id int64, nama string) (int64, error) {
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	if _, err := s.GetProfil(ctx, id); err != nil {
		return 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO profil (nama) VALUES (?)`, nama)
	if err != nil {
		return 0, terjemahkan(err, nil)
	}
	baru, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label, pengumuman_audio_id)
		 SELECT ?, hari, jam, jenis_bel_id, label, pengumuman_audio_id FROM jadwal WHERE profil_id = ?`, baru, id); err != nil {
		return 0, err
	}
	return baru, tx.Commit()
}

// EksporProfil menyusun file jadwal dari satu profil.
func (s *Store) EksporProfil(ctx context.Context, id int64) (formatjadwal.Data, error) {
	p, err := s.GetProfil(ctx, id)
	if err != nil {
		return formatjadwal.Data{}, err
	}
	jadwal, err := s.ListJadwal(ctx, id)
	if err != nil {
		return formatjadwal.Data{}, err
	}
	jenis, err := s.ListJenisBel(ctx)
	if err != nil {
		return formatjadwal.Data{}, err
	}
	peta := make(map[int64]JenisBel, len(jenis))
	for _, j := range jenis {
		peta[j.ID] = j
	}
	d := formatjadwal.Baru(p.Nama)
	masuk := make(map[int64]bool)
	for _, b := range jadwal {
		j, ok := peta[b.JenisBelID]
		if !ok {
			return formatjadwal.Data{}, fmt.Errorf("jenis bel %d untuk jadwal %d tidak ada", b.JenisBelID, b.ID)
		}
		if !masuk[j.ID] {
			masuk[j.ID] = true
			d.JenisBel = append(d.JenisBel, formatjadwal.JenisBel{Nama: j.Nama, Warna: j.Warna})
		}
		d.Jadwal = append(d.Jadwal, formatjadwal.Bel{Hari: b.Hari, Jam: b.Jam, JenisBel: j.Nama, Label: b.Label})
	}
	// File hasil ekspor harus bisa diimpor kembali; periksa dengan aturan yang sama.
	if len(d.Jadwal) > formatjadwal.MaksBel {
		return formatjadwal.Data{}, validasi("profil melebihi %d bel, tidak bisa diekspor", formatjadwal.MaksBel)
	}
	asal := make(map[string]string, len(d.JenisBel))
	for _, j := range d.JenisBel {
		k := kunciNama(j.Nama)
		if lain, ada := asal[k]; ada {
			return formatjadwal.Data{}, validasi("jenis bel %q dan %q hanya beda huruf besar/kecil; ganti salah satu nama dahulu", lain, j.Nama)
		}
		asal[k] = j.Nama
	}
	if err := d.Validasi(); err != nil {
		return formatjadwal.Data{}, err
	}
	return d, nil
}

func kunciNama(n string) string { return strings.ToLower(strings.TrimSpace(n)) }

// JenisBelTakDikenal mengembalikan nama jenis bel yang dipakai d tetapi belum ada di database.
func (s *Store) JenisBelTakDikenal(ctx context.Context, d formatjadwal.Data) ([]string, error) {
	peta, err := s.bacaPetaJenis(ctx)
	if err != nil {
		return nil, err
	}
	return peta.takDikenal(d), nil
}

// NamaProfilTersedia mengembalikan nama, atau nama + " (impor)", " (impor 2)", ... yang belum dipakai.
func (s *Store) NamaProfilTersedia(ctx context.Context, nama string) (string, error) {
	nama = strings.TrimSpace(nama)
	if r := []rune(nama); len(r) > 85 {
		nama = string(r[:85])
	}
	ps, err := s.ListProfil(ctx)
	if err != nil {
		return "", err
	}
	dipakai := make(map[string]bool, len(ps))
	for _, p := range ps {
		dipakai[p.Nama] = true
	}
	if !dipakai[nama] {
		return nama, nil
	}
	for i := 1; ; i++ {
		calon := nama + " (impor)"
		if i > 1 {
			calon = fmt.Sprintf("%s (impor %d)", nama, i)
		}
		if !dipakai[calon] && utf8.RuneCountInString(calon) <= 100 {
			return calon, nil
		}
	}
}

// PemetaanJenis menentukan jenis bel tujuan untuk satu nama jenis bel yang tidak dikenal.
type PemetaanJenis struct {
	JenisBelID int64 `json:"jenisBelId"`
	Baru       bool  `json:"baru"`
}

// suaraJenisBaru memilih suara untuk jenis bel baru hasil impor: "Ding Dong" bawaan, atau suara pertama.
func (s *Store) suaraJenisBaru(ctx context.Context) (int64, error) {
	a, err := s.FindAudio(ctx, "bawaan", "Ding Dong")
	if err == nil {
		return a.ID, nil
	}
	if !errors.Is(err, ErrTidakAda) {
		return 0, err
	}
	semua, err := s.ListAudio(ctx)
	if err != nil {
		return 0, err
	}
	if len(semua) == 0 {
		return 0, validasi("belum ada suara untuk jenis bel baru; tambahkan suara dahulu")
	}
	return semua[0].ID, nil
}

// ImporJadwal membuat profil baru dari d dalam satu transaksi. Jenis bel yang namanya sudah ada
// (tanpa membedakan huruf besar/kecil) dipakai langsung; sisanya harus ada di pemetaan.
func (s *Store) ImporJadwal(ctx context.Context, d formatjadwal.Data, nama string, pemetaan map[string]PemetaanJenis) (int64, error) {
	if err := d.Validasi(); err != nil {
		return 0, dariFormat(err)
	}
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	// Semua pembacaan sebelum transaksi: database hanya punya satu koneksi.
	peta, err := s.bacaPetaJenis(ctx)
	if err != nil {
		return 0, err
	}
	perluSuara, err := peta.periksaPemetaan(d, pemetaan)
	if err != nil {
		return 0, err
	}
	var suara int64
	if perluSuara {
		if suara, err = s.suaraJenisBaru(ctx); err != nil {
			return 0, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO profil (nama) VALUES (?)`, nama)
	if err != nil {
		return 0, terjemahkan(err, nil)
	}
	profil, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	idJenis, err := peta.buatJenis(ctx, tx, d, pemetaan, suara)
	if err != nil {
		return 0, err
	}
	for _, b := range d.Jadwal {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label) VALUES (?, ?, ?, ?, ?)`,
			profil, b.Hari, b.Jam, idJenis[kunciNama(b.JenisBel)], strings.TrimSpace(b.Label)); err != nil {
			return 0, terjemahkan(err, errRefJadwal)
		}
	}
	return profil, tx.Commit()
}
