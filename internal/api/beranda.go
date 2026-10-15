package api

import (
	"fmt"
	"net/http"

	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
	"belsekolah/internal/zona"
)

func (s *server) status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := s.Now()
	st := s.Penjadwal.Status()
	out := statusDTO{
		Sekarang: now, Tanggal: st.Tanggal, Hari: schedule.Hari(now),
		Bel: []belHariIniDTO{}, Peringatan: []string{},
		ZonaWaktu: zona.Kode(s.zonaAktif()),
	}
	if !st.Siap {
		out.Tanggal = now.Format(schedule.FormatTanggal)
		out.Peringatan = append(out.Peringatan, "Penjadwal belum siap, tunggu beberapa detik")
	}
	if st.ProfilID != 0 {
		if p, err := s.Store.GetProfil(ctx, st.ProfilID); err == nil {
			d := keProfilDTO(p)
			out.Profil = &d
		}
	}
	if st.Libur != nil {
		d := keLiburDTO(*st.Libur)
		out.Libur = &d
	}

	jenis, err := s.Store.ListJenisBel(ctx)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	peta := make(map[int64]jenisBelRingkasDTO, len(jenis))
	for _, j := range jenis {
		peta[j.ID] = jenisBelRingkasDTO{ID: j.ID, Nama: j.Nama, Warna: j.Warna}
	}
	for _, b := range st.Bel {
		item := belHariIniDTO{JadwalID: b.JadwalID, Jam: b.Waktu.Format("15:04"), Waktu: b.Waktu, Label: b.Label, Sudah: b.Sudah}
		if j, ok := peta[b.JenisBelID]; ok {
			item.JenisBel = &j
		}
		if b.PengumumanID != 0 {
			if a, err := s.Store.GetAudio(ctx, b.PengumumanID); err == nil {
				item.Pengumuman = &store.SuaraRingkas{ID: a.ID, Nama: a.Nama}
			}
		}
		out.Bel = append(out.Bel, item)
		if out.Berikutnya == nil && !b.Sudah && b.Waktu.After(now) {
			salinan := item
			out.Berikutnya = &salinan
		}
	}

	riwayat, err := s.Store.CariRiwayat(ctx, store.FilterRiwayat{Tanggal: out.Tanggal, Batas: 200})
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	for i := len(riwayat) - 1; i >= 0; i-- { // urut dari yang paling awal
		rw := riwayat[i]
		switch rw.Status {
		case "gagal":
			out.Peringatan = append(out.Peringatan, fmt.Sprintf("Bel %q pukul %s gagal berbunyi: %s", rw.Label, rw.Waktu.Format("15:04"), rw.Pesan))
		case "cadangan":
			out.Peringatan = append(out.Peringatan, fmt.Sprintf("Bel %q pukul %s memakai nada cadangan: %s", rw.Label, rw.Waktu.Format("15:04"), rw.Pesan))
		}
	}
	if s.CekNTP != nil {
		if ok := s.CekNTP(); ok != nil && !*ok {
			out.Peringatan = append(out.Peringatan, "Jam belum tersinkron dengan internet (NTP); periksa koneksi atau atur jam manual")
		}
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) belManual(w http.ResponseWriter, r *http.Request)  { s.bel(w, r, false) }
func (s *server) belDarurat(w http.ResponseWriter, r *http.Request) { s.bel(w, r, true) }

func (s *server) bel(w http.ResponseWriter, r *http.Request, darurat bool) {
	var p struct {
		JenisBelID int64 `json:"jenisBelId"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	j, err := s.Store.GetJenisBel(r.Context(), p.JenisBelID)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	label := "Manual: " + j.Nama
	if darurat {
		label = "Darurat: " + j.Nama
	}
	s.Penjadwal.BunyikanManual(r.Context(), j.ID, label, darurat)
	tulisPesan(w, http.StatusAccepted, "bel dibunyikan")
}

func (s *server) belStop(w http.ResponseWriter, r *http.Request) {
	s.Pemutar.Stop()
	tulisPesan(w, http.StatusOK, "suara dihentikan")
}
