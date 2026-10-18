package api

import (
	"time"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

type profilDTO struct {
	ID        int64  `json:"id"`
	Nama      string `json:"nama"`
	JumlahBel int    `json:"jumlahBel"`
}

type jenisBelRingkasDTO struct {
	ID    int64  `json:"id"`
	Nama  string `json:"nama"`
	Warna string `json:"warna"`
}

type jenisBelDTO struct {
	ID       int64   `json:"id"`
	Nama     string  `json:"nama"`
	Warna    string  `json:"warna"`
	AudioIDs []int64 `json:"audioIds"`
}

type audioDTO struct {
	ID       int64     `json:"id"`
	Nama     string    `json:"nama"`
	Sumber   string    `json:"sumber"`
	DurasiMS int64     `json:"durasiMs"`
	TeksTTS  string    `json:"teksTts,omitempty"`
	SuaraTTS string    `json:"suaraTts,omitempty"`
	Dibuat   time.Time `json:"dibuat"`
}

type jadwalDTO struct {
	ID         int64  `json:"id"`
	ProfilID   int64  `json:"profilId"`
	Hari       int    `json:"hari"`
	Jam        string `json:"jam"`
	JenisBelID int64  `json:"jenisBelId"`
	Label      string `json:"label"`
	// PengumumanID: null berarti tanpa pengumuman (0 di dalam).
	PengumumanID *int64 `json:"pengumumanId"`
}

type periodeDTO struct {
	ID         int64  `json:"id"`
	Mulai      string `json:"mulai"`
	Selesai    string `json:"selesai"`
	ProfilID   int64  `json:"profilId"`
	Keterangan string `json:"keterangan"`
}

type liburDTO struct {
	ID         int64  `json:"id"`
	Mulai      string `json:"mulai"`
	Selesai    string `json:"selesai"`
	Keterangan string `json:"keterangan"`
}

type riwayatDTO struct {
	ID         int64     `json:"id"`
	Waktu      time.Time `json:"waktu"`
	JadwalID   int64     `json:"jadwalId,omitempty"`
	JenisBelID int64     `json:"jenisBelId,omitempty"`
	Label      string    `json:"label"`
	Sumber     string    `json:"sumber"`
	Status     string    `json:"status"`
	Pesan      string    `json:"pesan,omitempty"`
}

type belHariIniDTO struct {
	JadwalID int64               `json:"jadwalId"`
	Jam      string              `json:"jam"`
	Waktu    time.Time           `json:"waktu"`
	Label    string              `json:"label"`
	JenisBel *jenisBelRingkasDTO `json:"jenisBel"`
	Sudah    bool                `json:"sudah"`
	// Pengumuman: suara pengumuman bel ini, null bila tidak ada.
	Pengumuman *store.SuaraRingkas `json:"pengumuman"`
}

type statusDTO struct {
	Sekarang   time.Time       `json:"sekarang"`
	Tanggal    string          `json:"tanggal"`
	Hari       int             `json:"hari"`
	Profil     *profilDTO      `json:"profil"`
	Libur      *liburDTO       `json:"libur"`
	Bel        []belHariIniDTO `json:"bel"`
	Berikutnya *belHariIniDTO  `json:"berikutnya"`
	Peringatan []string        `json:"peringatan"`
	ZonaWaktu  string          `json:"zonaWaktu"`
}

type pengaturanDTO struct {
	Volume          int    `json:"volume"`
	ProfilDefaultID int64  `json:"profilDefaultId"`
	ZonaWaktu       string `json:"zonaWaktu"`
	MulaiUlang      bool   `json:"mulaiUlang,omitempty"`
}

type sistemDTO struct {
	Versi     string    `json:"versi"`
	ZonaWaktu string    `json:"zonaWaktu"`
	ZonaIana  string    `json:"zonaIana"`
	Sekarang  time.Time `json:"sekarang"`
	IP        []string  `json:"ip"`
	NTP       *bool     `json:"ntp"`

	Boot           string  `json:"boot"`
	Alat           alatDTO `json:"alat"`
	RuangKosongMb  int64   `json:"ruangKosongMb"`
	BackupTerakhir string  `json:"backupTerakhir"` // tanggal YYYY-MM-DD, kosong bila belum ada
}

type alatDTO struct {
	Mpv     bool `json:"mpv"`
	Ffprobe bool `json:"ffprobe"`
	EdgeTTS bool `json:"edgeTts"`
}

func keProfilDTO(p store.Profil) profilDTO { return profilDTO{ID: p.ID, Nama: p.Nama} }

func keAudioDTO(a store.Audio) audioDTO {
	return audioDTO{ID: a.ID, Nama: a.Nama, Sumber: a.Sumber, DurasiMS: a.DurasiMS,
		TeksTTS: a.TeksTTS, SuaraTTS: a.SuaraTTS, Dibuat: a.Dibuat}
}

func keJadwalDTO(j schedule.Jadwal) jadwalDTO {
	d := jadwalDTO{ID: j.ID, ProfilID: j.ProfilID, Hari: j.Hari, Jam: j.Jam, JenisBelID: j.JenisBelID, Label: j.Label}
	if j.PengumumanID != 0 {
		id := j.PengumumanID
		d.PengumumanID = &id
	}
	return d
}

func (d jadwalDTO) keJadwal() schedule.Jadwal {
	j := schedule.Jadwal{ID: d.ID, ProfilID: d.ProfilID, Hari: d.Hari, Jam: d.Jam, JenisBelID: d.JenisBelID, Label: d.Label}
	if d.PengumumanID != nil {
		j.PengumumanID = *d.PengumumanID
	}
	return j
}

func kePeriodeDTO(p schedule.Periode) periodeDTO {
	return periodeDTO{ID: p.ID, Mulai: p.Mulai, Selesai: p.Selesai, ProfilID: p.ProfilID, Keterangan: p.Keterangan}
}

func (d periodeDTO) kePeriode() schedule.Periode {
	return schedule.Periode{ID: d.ID, Mulai: d.Mulai, Selesai: d.Selesai, ProfilID: d.ProfilID, Keterangan: d.Keterangan}
}

func keLiburDTO(l schedule.Libur) liburDTO {
	return liburDTO{ID: l.ID, Mulai: l.Mulai, Selesai: l.Selesai, Keterangan: l.Keterangan}
}

func (d liburDTO) keLibur() schedule.Libur {
	return schedule.Libur{ID: d.ID, Mulai: d.Mulai, Selesai: d.Selesai, Keterangan: d.Keterangan}
}

func keRiwayatDTO(r store.Riwayat) riwayatDTO {
	return riwayatDTO{ID: r.ID, Waktu: r.Waktu, JadwalID: r.JadwalID, JenisBelID: r.JenisBelID,
		Label: r.Label, Sumber: r.Sumber, Status: r.Status, Pesan: r.Pesan}
}

type templateDTO struct {
	ID        int64              `json:"id"`
	Nama      string             `json:"nama"`
	JumlahBel int                `json:"jumlahBel"`
	Hari      []int              `json:"hari"`
	Dibuat    time.Time          `json:"dibuat"`
	Data      *formatjadwal.Data `json:"data,omitempty"`
}

func keTemplateDTO(t store.Template, denganData bool) templateDTO {
	out := templateDTO{ID: t.ID, Nama: t.Nama, JumlahBel: len(t.Data.Jadwal), Hari: t.Data.HariBerisi(), Dibuat: t.Dibuat}
	if denganData {
		d := t.Data
		out.Data = &d
	}
	return out
}
