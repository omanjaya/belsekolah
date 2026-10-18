package store

import (
	"context"
	"database/sql"
	"strings"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/gabung"
)

type IsiGabung struct {
	JenisBel string `json:"jenisBel"`
	Label    string `json:"label"`
}

type BarisGabung struct {
	Hari   int           `json:"hari"`
	Jam    string        `json:"jam"`
	Status gabung.Status `json:"status"`
	Lama   *IsiGabung    `json:"lama"`
	Baru   IsiGabung     `json:"baru"`
}

type RingkasanGabung struct {
	Baru    int `json:"baru"`
	Kembar  int `json:"kembar"`
	Bentrok int `json:"bentrok"`
}

type HasilPeriksaGabung struct {
	JenisBelTakDikenal []string        `json:"jenisBelTakDikenal"`
	Ringkasan          RingkasanGabung `json:"ringkasan"`
	Baris              []BarisGabung   `json:"baris"`
	// PengumumanTakDitemukan: nama suara pengumuman di file yang tidak ada di pustaka.
	PengumumanTakDitemukan []string `json:"pengumumanTakDitemukan"`
}

type HasilGabung struct {
	Ditambah int `json:"ditambah"`
	Ditimpa  int `json:"ditimpa"`
	Dilewati int `json:"dilewati"`
}

// masukDari mengubah bel d menjadi masukan gabung dengan id jenis dari idJenis (kunciNama -> id).
func masukDari(d formatjadwal.Data, idJenis map[string]int64) []gabung.Masuk {
	out := make([]gabung.Masuk, len(d.Jadwal))
	for i, b := range d.Jadwal {
		out[i] = gabung.Masuk{Hari: b.Hari, Jam: b.Jam,
			Isi: gabung.Isi{JenisBelID: idJenis[kunciNama(b.JenisBel)], Label: strings.TrimSpace(b.Label)}}
	}
	return out
}

func jadwalLama(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, profilID int64) ([]gabung.Lama, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, hari, jam, jenis_bel_id, label FROM jadwal WHERE profil_id = ?`, profilID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gabung.Lama
	for rows.Next() {
		var l gabung.Lama
		if err := rows.Scan(&l.JadwalID, &l.Hari, &l.Jam, &l.JenisBelID, &l.Label); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// PeriksaGabung mengklasifikasi bel d terhadap jadwal profil tanpa mengubah apa pun.
// Jenis yang belum dipetakan (atau dipetakan ke jenis baru) dianggap berbeda dari semua jenis yang ada.
func (s *Store) PeriksaGabung(ctx context.Context, profilID int64, d formatjadwal.Data, pemetaan map[string]PemetaanJenis) (HasilPeriksaGabung, error) {
	if err := d.Validasi(); err != nil {
		return HasilPeriksaGabung{}, dariFormat(err)
	}
	if _, err := s.GetProfil(ctx, profilID); err != nil {
		return HasilPeriksaGabung{}, err
	}
	peta, err := s.bacaPetaJenis(ctx)
	if err != nil {
		return HasilPeriksaGabung{}, err
	}
	lama, err := jadwalLama(ctx, s.db, profilID)
	if err != nil {
		return HasilPeriksaGabung{}, err
	}
	tak, err := s.PengumumanTakDikenal(ctx, d)
	if err != nil {
		return HasilPeriksaGabung{}, err
	}
	h := HasilPeriksaGabung{JenisBelTakDikenal: peta.takDikenal(d), Baris: []BarisGabung{}, PengumumanTakDitemukan: tak}
	for _, b := range gabung.Klasifikasi(lama, masukDari(d, peta.idSementara(d, pemetaan))) {
		asal := d.Jadwal[b.Indeks]
		bg := BarisGabung{Hari: b.Hari, Jam: b.Jam, Status: b.Status,
			Baru: IsiGabung{JenisBel: strings.TrimSpace(asal.JenisBel), Label: strings.TrimSpace(asal.Label)}}
		if b.Lama != nil {
			bg.Lama = &IsiGabung{JenisBel: peta.namaID[b.Lama.JenisBelID], Label: b.Lama.Label}
		}
		switch b.Status {
		case gabung.Baru:
			h.Ringkasan.Baru++
		case gabung.Kembar:
			h.Ringkasan.Kembar++
		case gabung.Bentrok:
			h.Ringkasan.Bentrok++
		}
		h.Baris = append(h.Baris, bg)
	}
	return h, nil
}

// GabungJadwal menambah bel baru dan menimpa bentrok yang dipilih dalam satu transaksi.
// Klasifikasi dihitung ulang dari database; bel lama di luar masukan tidak disentuh.
func (s *Store) GabungJadwal(ctx context.Context, profilID int64, d formatjadwal.Data, pemetaan map[string]PemetaanJenis, timpa []gabung.Slot) (HasilGabung, error) {
	if err := d.Validasi(); err != nil {
		return HasilGabung{}, dariFormat(err)
	}
	// Semua pembacaan lewat s.db sebelum transaksi: database hanya punya satu koneksi.
	if _, err := s.GetProfil(ctx, profilID); err != nil {
		return HasilGabung{}, err
	}
	peta, err := s.bacaPetaJenis(ctx)
	if err != nil {
		return HasilGabung{}, err
	}
	suaraPeng, err := s.bacaPetaSuara(ctx)
	if err != nil {
		return HasilGabung{}, err
	}
	perluSuara, err := peta.periksaPemetaan(d, pemetaan)
	if err != nil {
		return HasilGabung{}, err
	}
	var suara int64
	if perluSuara {
		if suara, err = s.suaraJenisBaru(ctx); err != nil {
			return HasilGabung{}, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return HasilGabung{}, err
	}
	defer tx.Rollback()
	idJenis, err := peta.buatJenis(ctx, tx, d, pemetaan, suara)
	if err != nil {
		return HasilGabung{}, err
	}
	lama, err := jadwalLama(ctx, tx, profilID)
	if err != nil {
		return HasilGabung{}, err
	}
	r := gabung.Rencanakan(gabung.Klasifikasi(lama, masukDari(d, idJenis)), timpa)
	if len(lama)+len(r.Tambah) > formatjadwal.MaksBel {
		return HasilGabung{}, validasi("profil akan berisi %d bel, melebihi batas %d bel", len(lama)+len(r.Tambah), formatjadwal.MaksBel)
	}
	for _, b := range r.Tambah {
		asal := d.Jadwal[b.Indeks]
		peng, _ := suaraPeng.resolusi(asal.Pengumuman, asal.Label)
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label, pengumuman_audio_id) VALUES (?, ?, ?, ?, ?, ?)`,
			profilID, b.Hari, b.Jam, b.Baru.JenisBelID, b.Baru.Label, nullID(peng)); err != nil {
			return HasilGabung{}, terjemahkan(err, errRefJadwal)
		}
	}
	for _, b := range r.Timpa {
		asal := d.Jadwal[b.Indeks]
		peng, _ := suaraPeng.resolusi(asal.Pengumuman, asal.Label)
		if _, err := tx.ExecContext(ctx, `UPDATE jadwal SET jenis_bel_id = ?, label = ?, pengumuman_audio_id = ? WHERE id = ?`,
			b.Baru.JenisBelID, b.Baru.Label, nullID(peng), b.Lama.JadwalID); err != nil {
			return HasilGabung{}, terjemahkan(err, errRefJadwal)
		}
	}
	if err := tx.Commit(); err != nil {
		return HasilGabung{}, err
	}
	return HasilGabung{Ditambah: len(r.Tambah), Ditimpa: len(r.Timpa), Dilewati: r.Dilewati}, nil
}
