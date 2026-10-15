package store

import (
	"context"
	"database/sql"
	"strings"

	"belsekolah/internal/formatjadwal"
)

// petaJenis memotret jenis bel di database untuk mencocokkan nama di file jadwal.
type petaJenis struct {
	idAda  map[string]int64 // kunciNama -> id
	namaID map[int64]string // id -> nama di database
}

func (s *Store) bacaPetaJenis(ctx context.Context) (petaJenis, error) {
	ada, err := s.ListJenisBel(ctx)
	if err != nil {
		return petaJenis{}, err
	}
	p := petaJenis{idAda: make(map[string]int64, len(ada)), namaID: make(map[int64]string, len(ada))}
	for _, j := range ada {
		p.idAda[kunciNama(j.Nama)] = j.ID
		p.namaID[j.ID] = j.Nama
	}
	return p, nil
}

// takDikenal mengembalikan nama jenis yang dipakai d tetapi tidak ada di database (tanpa memandang pemetaan).
func (p petaJenis) takDikenal(d formatjadwal.Data) []string {
	out := []string{}
	for _, j := range d.JenisDipakai() {
		if _, ok := p.idAda[kunciNama(j.Nama)]; !ok {
			out = append(out, j.Nama)
		}
	}
	return out
}

// idSementara: kunciNama -> id jenis untuk setiap jenis yang dipakai d, tanpa membuat apa pun.
// 0 bila belum dipetakan atau dipetakan ke jenis baru.
func (p petaJenis) idSementara(d formatjadwal.Data, pemetaan map[string]PemetaanJenis) map[string]int64 {
	out := make(map[string]int64)
	for _, j := range d.JenisDipakai() {
		k := kunciNama(j.Nama)
		if id, ok := p.idAda[k]; ok {
			out[k] = id
			continue
		}
		if m, ok := pemetaan[j.Nama]; ok && !m.Baru {
			if _, sah := p.namaID[m.JenisBelID]; sah {
				out[k] = m.JenisBelID
				continue
			}
		}
		out[k] = 0
	}
	return out
}

// periksaPemetaan memastikan setiap jenis tak dikenal dipetakan ke jenis yang ada atau ke jenis baru.
// perluSuara true bila ada jenis baru yang akan dibuat.
func (p petaJenis) periksaPemetaan(d formatjadwal.Data, pemetaan map[string]PemetaanJenis) (perluSuara bool, err error) {
	for _, j := range d.JenisDipakai() {
		if _, ok := p.idAda[kunciNama(j.Nama)]; ok {
			continue
		}
		m, ok := pemetaan[j.Nama]
		switch {
		case !ok || (!m.Baru && m.JenisBelID == 0):
			return false, validasi("jenis bel %q belum dipetakan", j.Nama)
		case m.Baru:
			perluSuara = true
		case p.namaID[m.JenisBelID] == "":
			return false, validasi("jenis bel tujuan untuk %q tidak ditemukan", j.Nama)
		}
	}
	return perluSuara, nil
}

// buatJenis (di dalam tx) mengembalikan kunciNama -> id untuk setiap jenis yang dipakai d,
// membuat jenis baru (dengan satu suara) sesuai pemetaan. Panggil periksaPemetaan dahulu.
func (p petaJenis) buatJenis(ctx context.Context, tx *sql.Tx, d formatjadwal.Data, pemetaan map[string]PemetaanJenis, suara int64) (map[string]int64, error) {
	dipakai := d.JenisDipakai()
	idJenis := make(map[string]int64, len(dipakai))
	for _, j := range dipakai {
		k := kunciNama(j.Nama)
		if id, ok := p.idAda[k]; ok {
			idJenis[k] = id
			continue
		}
		m := pemetaan[j.Nama]
		if !m.Baru {
			idJenis[k] = m.JenisBelID
			continue
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO jenis_bel (nama, warna) VALUES (?, ?)`, strings.TrimSpace(j.Nama), j.Warna)
		if err != nil {
			return nil, terjemahkan(err, nil)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO urutan_bel (jenis_bel_id, urutan, audio_id) VALUES (?, 1, ?)`, id, suara); err != nil {
			return nil, err
		}
		idJenis[k] = id
	}
	return idJenis, nil
}
