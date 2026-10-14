package api

import "net/http"

func (s *server) listProfil(w http.ResponseWriter, r *http.Request) {
	ps, err := s.Store.ListProfil(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	jumlah, err := s.Store.JumlahBelPerProfil(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]profilDTO, 0, len(ps))
	for _, p := range ps {
		d := keProfilDTO(p)
		d.JumlahBel = jumlah[p.ID]
		out = append(out, d)
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimProfil(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	p, err := s.Store.GetProfil(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	jumlah, err := s.Store.JumlahBelPerProfil(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	d := keProfilDTO(p)
	d.JumlahBel = jumlah[p.ID]
	tulisJSON(w, kode, d)
}

func (s *server) buatProfil(w http.ResponseWriter, r *http.Request) {
	var p profilDTO
	if !bacaJSON(w, r, &p) {
		return
	}
	id, err := s.Store.CreateProfil(r.Context(), p.Nama)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimProfil(w, r, id, http.StatusCreated)
}

func (s *server) ubahProfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var p profilDTO
	if !bacaJSON(w, r, &p) {
		return
	}
	if err := s.Store.UpdateProfil(r.Context(), id, p.Nama); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimProfil(w, r, id, http.StatusOK)
}

func (s *server) hapusProfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteProfil(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}
