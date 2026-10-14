package api

import (
	"encoding/json"
	"net/http"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/pembuat"
)

func (s *server) listTemplate(w http.ResponseWriter, r *http.Request) {
	ts, err := s.Store.ListTemplate(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]templateDTO, 0, len(ts))
	for _, t := range ts {
		out = append(out, keTemplateDTO(t, false))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimTemplate(w http.ResponseWriter, r *http.Request, id int64, kode int, denganData bool) {
	t, err := s.Store.GetTemplate(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, keTemplateDTO(t, denganData))
}

type buatTemplateDTO struct {
	Nama     string          `json:"nama"`
	ProfilID int64           `json:"profilId"`
	Data     json.RawMessage `json:"data"`
}

func (s *server) buatTemplate(w http.ResponseWriter, r *http.Request) {
	var in buatTemplateDTO
	if !bacaJSON(w, r, &in) {
		return
	}
	var d formatjadwal.Data
	var err error
	switch {
	case in.ProfilID > 0:
		d, err = s.Store.EksporProfil(r.Context(), in.ProfilID)
	case len(in.Data) > 0:
		d, err = formatjadwal.Baca(in.Data)
	default:
		tulisError(w, http.StatusBadRequest, "pilih profil atau file jadwal untuk template")
		return
	}
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	id, err := s.Store.CreateTemplate(r.Context(), in.Nama, d, s.Now())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.kirimTemplate(w, r, id, http.StatusCreated, false)
}

func (s *server) ambilTemplate(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	s.kirimTemplate(w, r, id, http.StatusOK, true)
}

func (s *server) ubahTemplate(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var in struct {
		Nama string `json:"nama"`
	}
	if !bacaJSON(w, r, &in) {
		return
	}
	if err := s.Store.UpdateNamaTemplate(r.Context(), id, in.Nama); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.kirimTemplate(w, r, id, http.StatusOK, false)
}

func (s *server) hapusTemplate(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteTemplate(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) eksporTemplate(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	t, err := s.Store.GetTemplate(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	kirimFileJadwal(w, t.Data)
}

// buatDariParameter menyusun jadwal dari parameter pembuat tanpa menyimpan apa pun.
func (s *server) buatDariParameter(w http.ResponseWriter, r *http.Request) {
	var p pembuat.Parameter
	if !bacaJSON(w, r, &p) {
		return
	}
	d, err := pembuat.Buat(p)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, d)
}
