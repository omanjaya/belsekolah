package api

import (
	"encoding/json"
	"net/http"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/gabung"
	"belsekolah/internal/store"
)

type gabungDTO struct {
	Data     json.RawMessage                `json:"data"`
	Pemetaan map[string]store.PemetaanJenis `json:"pemetaan"`
	Timpa    []gabung.Slot                  `json:"timpa"`
}

// bacaGabung membaca id profil dan badan permintaan gabung. Menulis error dan mengembalikan false jika gagal.
func bacaGabung(w http.ResponseWriter, r *http.Request) (int64, gabungDTO, formatjadwal.Data, bool) {
	var in gabungDTO
	id, ok := idDariPath(w, r)
	if !ok || !bacaJSON(w, r, &in) {
		return 0, in, formatjadwal.Data{}, false
	}
	if len(in.Data) == 0 {
		tulisError(w, http.StatusBadRequest, "data jadwal wajib diisi")
		return 0, in, formatjadwal.Data{}, false
	}
	d, err := formatjadwal.Baca(in.Data)
	if err != nil {
		tulisError(w, http.StatusBadRequest, err.Error())
		return 0, in, formatjadwal.Data{}, false
	}
	return id, in, d, true
}

func (s *server) periksaGabung(w http.ResponseWriter, r *http.Request) {
	id, in, d, ok := bacaGabung(w, r)
	if !ok {
		return
	}
	h, err := s.Store.PeriksaGabung(r.Context(), id, d, in.Pemetaan)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, h)
}

func (s *server) gabungJadwal(w http.ResponseWriter, r *http.Request) {
	id, in, d, ok := bacaGabung(w, r)
	if !ok {
		return
	}
	h, err := s.Store.GabungJadwal(r.Context(), id, d, in.Pemetaan, in.Timpa)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	tulisJSON(w, http.StatusOK, h)
}
