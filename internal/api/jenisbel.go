package api

import (
	"net/http"

	"belsekolah/internal/player"
)

func (s *server) keJenisBelDTO(r *http.Request, id int64) (jenisBelDTO, error) {
	j, err := s.Store.GetJenisBel(r.Context(), id)
	if err != nil {
		return jenisBelDTO{}, err
	}
	urutan, err := s.Store.UrutanBel(r.Context(), id)
	if err != nil {
		return jenisBelDTO{}, err
	}
	out := jenisBelDTO{ID: j.ID, Nama: j.Nama, Warna: j.Warna, AudioIDs: make([]int64, 0, len(urutan))}
	for _, a := range urutan {
		out.AudioIDs = append(out.AudioIDs, a.ID)
	}
	return out, nil
}

func (s *server) listJenisBel(w http.ResponseWriter, r *http.Request) {
	js, err := s.Store.ListJenisBel(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]jenisBelDTO, 0, len(js))
	for _, j := range js {
		d, err := s.keJenisBelDTO(r, j.ID)
		if err != nil {
			s.tulisErrorData(w, err)
			return
		}
		out = append(out, d)
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimJenisBel(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	d, err := s.keJenisBelDTO(r, id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, d)
}

func (s *server) buatJenisBel(w http.ResponseWriter, r *http.Request) {
	var d jenisBelDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	id, err := s.Store.CreateJenisBel(r.Context(), d.Nama, d.Warna)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if err := s.Store.SetUrutanBel(r.Context(), id, d.AudioIDs); err != nil {
		if errHapus := s.Store.DeleteJenisBel(r.Context(), id); errHapus != nil {
			s.Log.Error("gagal membersihkan jenis bel", "id", id, "err", errHapus)
		}
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimJenisBel(w, r, id, http.StatusCreated)
}

func (s *server) ubahJenisBel(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var d jenisBelDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	if err := s.Store.UpdateJenisBel(r.Context(), id, d.Nama, d.Warna); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if err := s.Store.SetUrutanBel(r.Context(), id, d.AudioIDs); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimJenisBel(w, r, id, http.StatusOK)
}

func (s *server) hapusJenisBel(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteJenisBel(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}

// tesJenisBel memutar jenis bel di speaker tanpa mencatat riwayat.
func (s *server) tesJenisBel(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if _, err := s.Store.GetJenisBel(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.Pemutar.Play(player.Permintaan{Files: s.Penjadwal.FileUrutan(r.Context(), id)})
	tulisPesan(w, http.StatusAccepted, "suara diputar")
}
