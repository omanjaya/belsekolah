package api

import (
	"net/http"
	"strconv"

	"belsekolah/internal/store"
)

func (s *server) listRiwayat(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.FilterRiwayat{Tanggal: q.Get("tanggal"), Status: q.Get("status")}
	if b := q.Get("batas"); b != "" {
		n, err := strconv.Atoi(b)
		if err != nil {
			tulisError(w, http.StatusBadRequest, "parameter batas harus angka")
			return
		}
		f.Batas = n
	}
	rs, err := s.Store.CariRiwayat(r.Context(), f)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]riwayatDTO, 0, len(rs))
	for _, rw := range rs {
		out = append(out, keRiwayatDTO(rw))
	}
	tulisJSON(w, http.StatusOK, out)
}
