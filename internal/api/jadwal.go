package api

import (
	"net/http"
	"strconv"
)

func (s *server) listJadwal(w http.ResponseWriter, r *http.Request) {
	profil, err := strconv.ParseInt(r.URL.Query().Get("profil"), 10, 64)
	if err != nil {
		tulisError(w, http.StatusBadRequest, "parameter profil wajib diisi")
		return
	}
	js, err := s.Store.ListJadwal(r.Context(), profil)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]jadwalDTO, 0, len(js))
	for _, j := range js {
		out = append(out, keJadwalDTO(j))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimJadwal(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	j, err := s.Store.GetJadwal(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, keJadwalDTO(j))
}

func (s *server) buatJadwal(w http.ResponseWriter, r *http.Request) {
	var d jadwalDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	id, err := s.Store.CreateJadwal(r.Context(), d.keJadwal())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimJadwal(w, r, id, http.StatusCreated)
}

func (s *server) ubahJadwal(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var d jadwalDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	d.ID = id
	if err := s.Store.UpdateJadwal(r.Context(), d.keJadwal()); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimJadwal(w, r, id, http.StatusOK)
}

func (s *server) hapusJadwal(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteJadwal(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) salinJadwal(w http.ResponseWriter, r *http.Request) {
	var p struct {
		ProfilID int64 `json:"profilId"`
		DariHari int   `json:"dariHari"`
		KeHari   int   `json:"keHari"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	if err := s.Store.SalinHari(r.Context(), p.ProfilID, p.DariHari, p.KeHari); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	tulisPesan(w, http.StatusOK, "jadwal berhasil disalin")
}

func (s *server) listPeriode(w http.ResponseWriter, r *http.Request) {
	ps, err := s.Store.ListPeriode(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]periodeDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, kePeriodeDTO(p))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimPeriode(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	p, err := s.Store.GetPeriode(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, kePeriodeDTO(p))
}

func (s *server) buatPeriode(w http.ResponseWriter, r *http.Request) {
	var d periodeDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	id, err := s.Store.CreatePeriode(r.Context(), d.kePeriode())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimPeriode(w, r, id, http.StatusCreated)
}

func (s *server) ubahPeriode(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var d periodeDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	d.ID = id
	if err := s.Store.UpdatePeriode(r.Context(), d.kePeriode()); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimPeriode(w, r, id, http.StatusOK)
}

func (s *server) hapusPeriode(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeletePeriode(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) listLibur(w http.ResponseWriter, r *http.Request) {
	ls, err := s.Store.ListLibur(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]liburDTO, 0, len(ls))
	for _, l := range ls {
		out = append(out, keLiburDTO(l))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimLibur(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	l, err := s.Store.GetLibur(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, keLiburDTO(l))
}

func (s *server) buatLibur(w http.ResponseWriter, r *http.Request) {
	var d liburDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	id, err := s.Store.CreateLibur(r.Context(), d.keLibur())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimLibur(w, r, id, http.StatusCreated)
}

func (s *server) ubahLibur(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var d liburDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	d.ID = id
	if err := s.Store.UpdateLibur(r.Context(), d.keLibur()); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimLibur(w, r, id, http.StatusOK)
}

func (s *server) hapusLibur(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteLibur(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}
