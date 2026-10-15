package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/store"
)

func tulisJSON(w http.ResponseWriter, kode int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(kode)
	json.NewEncoder(w).Encode(v)
}

func tulisError(w http.ResponseWriter, kode int, pesan string) {
	tulisJSON(w, kode, map[string]string{"error": pesan})
}

func tulisPesan(w http.ResponseWriter, kode int, pesan string) {
	tulisJSON(w, kode, map[string]string{"pesan": pesan})
}

// bacaJSON membaca body JSON (maks 1 MB, field tak dikenal ditolak). Menulis 400 dan mengembalikan false jika gagal.
func bacaJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		tulisError(w, http.StatusBadRequest, "format data tidak valid")
		return false
	}
	return true
}

// tulisErrorData memetakan error store ke kode HTTP.
func (s *server) tulisErrorData(w http.ResponseWriter, err error) {
	var v *store.ErrorValidasi
	var f *formatjadwal.ErrorValidasi
	switch {
	case errors.As(err, &v):
		tulisError(w, http.StatusBadRequest, v.Pesan)
	case errors.As(err, &f):
		tulisError(w, http.StatusBadRequest, f.Pesan)
	case errors.Is(err, store.ErrTidakAda):
		tulisError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, store.ErrDipakai), errors.Is(err, store.ErrDuplikat), errors.Is(err, store.ErrPeriodeTumpangTindih):
		tulisError(w, http.StatusConflict, err.Error())
	default:
		s.Log.Error("kesalahan server", "err", err)
		tulisError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

// idDariPath membaca {id}; menulis 404 dan mengembalikan false jika bukan angka.
func idDariPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		tulisError(w, http.StatusNotFound, "data tidak ditemukan")
		return 0, false
	}
	return id, true
}
