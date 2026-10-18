package api

import (
	"net/http"

	"belsekolah/internal/store"
)

func (s *server) cocokPengumuman(w http.ResponseWriter, r *http.Request) {
	suara, err := s.Store.CocokkanPengumuman(r.Context(), r.URL.Query().Get("keterangan"))
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, struct {
		Suara *store.SuaraRingkas `json:"suara"`
	}{suara})
}

func (s *server) saranPengumuman(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	saran, err := s.Store.SaranPengumuman(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if saran == nil {
		saran = []store.SaranPengumuman{}
	}
	tulisJSON(w, http.StatusOK, saran)
}

func (s *server) terapkanPengumuman(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var in struct {
		Isi []store.IsiPengumuman `json:"isi"`
	}
	if !bacaJSON(w, r, &in) {
		return
	}
	n, err := s.Store.TerapkanPengumuman(r.Context(), id, in.Isi)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	tulisJSON(w, http.StatusOK, struct {
		Diisi int `json:"diisi"`
	}{n})
}
