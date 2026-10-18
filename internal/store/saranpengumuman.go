package store

import (
	"context"
	"errors"
	"sort"
	"strings"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/pengumuman"
)

// SuaraRingkas adalah suara dari pustaka (id dan nama) untuk ditampilkan sebagai saran.
type SuaraRingkas struct {
	ID   int64  `json:"id"`
	Nama string `json:"nama"`
}

// SaranPengumuman: bel yang pengumumannya masih kosong dan punya suara yang cocok dengan labelnya.
type SaranPengumuman struct {
	JadwalID int64        `json:"jadwalId"`
	Hari     int          `json:"hari"`
	Jam      string       `json:"jam"`
	Label    string       `json:"label"`
	Suara    SuaraRingkas `json:"suara"`
}

// IsiPengumuman menetapkan suara pengumuman untuk satu bel.
type IsiPengumuman struct {
	JadwalID int64 `json:"jadwalId"`
	AudioID  int64 `json:"audioId"`
}

// petaSuara memotret pustaka suara untuk mengisi pengumuman.
type petaSuara struct {
	kandidat []pengumuman.Kandidat
	idNama   map[string]int64 // kunciNama(nama) -> id; nama ganda: id terkecil
}

// bacaPetaSuara membaca pustaka suara. Panggil di luar transaksi.
func (s *Store) bacaPetaSuara(ctx context.Context) (petaSuara, error) {
	semua, err := s.ListAudio(ctx) // urut id naik
	if err != nil {
		return petaSuara{}, err
	}
	p := petaSuara{kandidat: make([]pengumuman.Kandidat, 0, len(semua)), idNama: make(map[string]int64, len(semua))}
	for _, a := range semua {
		p.kandidat = append(p.kandidat, pengumuman.Kandidat{ID: a.ID, Nama: a.Nama})
		k := kunciNama(a.Nama)
		if id, ada := p.idNama[k]; !ada || a.ID < id {
			p.idNama[k] = a.ID
		}
	}
	return p, nil
}

// resolusi menentukan suara pengumuman: nama dari file bila ada (tidak ditemukan -> 0, true),
// selain itu dicocokkan dari label.
func (p petaSuara) resolusi(namaFile, label string) (id int64, takDitemukan bool) {
	if strings.TrimSpace(namaFile) != "" {
		if id, ok := p.idNama[kunciNama(namaFile)]; ok {
			return id, false
		}
		return 0, true
	}
	id, _ = pengumuman.Cocokkan(label, p.kandidat)
	return id, false
}

// CocokkanPengumuman mencari suara yang cocok dengan keterangan; nil bila tidak ada kecocokan.
func (s *Store) CocokkanPengumuman(ctx context.Context, keterangan string) (*SuaraRingkas, error) {
	p, err := s.bacaPetaSuara(ctx)
	if err != nil {
		return nil, err
	}
	id, ok := pengumuman.Cocokkan(keterangan, p.kandidat)
	if !ok {
		return nil, nil
	}
	for _, k := range p.kandidat {
		if k.ID == id {
			return &SuaraRingkas{ID: k.ID, Nama: k.Nama}, nil
		}
	}
	return nil, nil
}

// PengumumanTakDikenal mengembalikan nama pengumuman di d yang tidak ada di pustaka suara
// (unik, urut kemunculan pertama).
func (s *Store) PengumumanTakDikenal(ctx context.Context, d formatjadwal.Data) ([]string, error) {
	p, err := s.bacaPetaSuara(ctx)
	if err != nil {
		return nil, err
	}
	out := []string{}
	lihat := map[string]bool{}
	for _, b := range d.Jadwal {
		if _, tak := p.resolusi(b.Pengumuman, b.Label); !tak {
			continue
		}
		k := kunciNama(b.Pengumuman)
		if !lihat[k] {
			lihat[k] = true
			out = append(out, strings.TrimSpace(b.Pengumuman))
		}
	}
	return out, nil
}

// SaranPengumuman mengembalikan saran suara untuk bel profil yang pengumumannya masih kosong,
// urut hari lalu jam. Bel tanpa kecocokan dilewati.
func (s *Store) SaranPengumuman(ctx context.Context, profilID int64) ([]SaranPengumuman, error) {
	if _, err := s.GetProfil(ctx, profilID); err != nil {
		return nil, err
	}
	p, err := s.bacaPetaSuara(ctx)
	if err != nil {
		return nil, err
	}
	nama := make(map[int64]string, len(p.kandidat))
	for _, k := range p.kandidat {
		nama[k.ID] = k.Nama
	}
	jadwal, err := s.ListJadwal(ctx, profilID)
	if err != nil {
		return nil, err
	}
	out := []SaranPengumuman{}
	for _, j := range jadwal {
		if j.PengumumanID != 0 {
			continue
		}
		id, ok := pengumuman.Cocokkan(j.Label, p.kandidat)
		if !ok {
			continue
		}
		out = append(out, SaranPengumuman{JadwalID: j.ID, Hari: j.Hari, Jam: j.Jam, Label: j.Label,
			Suara: SuaraRingkas{ID: id, Nama: nama[id]}})
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].Hari != out[b].Hari {
			return out[a].Hari < out[b].Hari
		}
		return out[a].Jam < out[b].Jam
	})
	return out, nil
}

// TerapkanPengumuman mengisi pengumuman bel profil yang masih kosong, dalam satu transaksi.
// Bel yang sudah terisi atau milik profil lain dilewati. Mengembalikan jumlah bel yang berubah.
func (s *Store) TerapkanPengumuman(ctx context.Context, profilID int64, isi []IsiPengumuman) (int, error) {
	// Semua pembacaan sebelum transaksi: database hanya punya satu koneksi.
	if _, err := s.GetProfil(ctx, profilID); err != nil {
		return 0, err
	}
	diperiksa := map[int64]bool{}
	for _, i := range isi {
		if diperiksa[i.AudioID] {
			continue
		}
		if _, err := s.GetAudio(ctx, i.AudioID); err != nil {
			if errors.Is(err, ErrTidakAda) {
				return 0, validasi("suara pengumuman tidak ditemukan")
			}
			return 0, err
		}
		diperiksa[i.AudioID] = true
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	n := 0
	for _, i := range isi {
		res, err := tx.ExecContext(ctx,
			`UPDATE jadwal SET pengumuman_audio_id = ? WHERE id = ? AND profil_id = ? AND pengumuman_audio_id IS NULL`,
			i.AudioID, i.JadwalID, profilID)
		if err != nil {
			return 0, terjemahkan(err, errRefJadwal)
		}
		c, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		n += int(c)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}
