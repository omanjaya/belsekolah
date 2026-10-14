package api

import (
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"belsekolah/internal/auth"
)

func alamatIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Password string `json:"password"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	token, err := s.Auth.Login(r.Context(), alamatIP(r), p.Password)
	switch {
	case errors.Is(err, auth.ErrTerkunci):
		tulisError(w, http.StatusTooManyRequests, err.Error())
		return
	case errors.Is(err, auth.ErrPasswordSalah):
		tulisError(w, http.StatusUnauthorized, "password salah")
		return
	case err != nil:
		s.tulisErrorData(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: namaCookie, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
		MaxAge: int(auth.UmurSesi.Seconds()),
	})
	wajib, err := s.Auth.WajibGanti(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, map[string]bool{"wajibGantiPassword": wajib})
}

type sesiDTO struct {
	Masuk              bool   `json:"masuk"`
	WajibGantiPassword bool   `json:"wajibGantiPassword"`
	Boot               string `json:"boot"`
}

func (s *server) sesi(w http.ResponseWriter, r *http.Request) {
	masuk := false
	if c, err := r.Cookie(namaCookie); err == nil {
		masuk = s.Auth.Valid(c.Value)
	}
	wajib := false
	if masuk {
		var err error
		if wajib, err = s.Auth.WajibGanti(r.Context()); err != nil {
			s.tulisErrorData(w, err)
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	tulisJSON(w, http.StatusOK, sesiDTO{Masuk: masuk, WajibGantiPassword: wajib, Boot: s.boot})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(namaCookie); err == nil {
		s.Auth.Logout(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: namaCookie, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode, MaxAge: -1})
	tulisPesan(w, http.StatusOK, "berhasil keluar")
}

func (s *server) gantiPassword(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Lama string `json:"lama"`
		Baru string `json:"baru"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	token := ""
	if c, err := r.Cookie(namaCookie); err == nil {
		token = c.Value
	}
	err := s.Auth.GantiPassword(r.Context(), token, p.Lama, p.Baru)
	switch {
	case errors.Is(err, auth.ErrPasswordSalah):
		tulisError(w, http.StatusBadRequest, "password lama salah")
		return
	case errors.Is(err, auth.ErrPasswordPendek), errors.Is(err, auth.ErrPasswordPanjang), errors.Is(err, auth.ErrPasswordSama):
		tulisError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		s.tulisErrorData(w, err)
		return
	}
	if err := os.Remove(filepath.Join(s.DataDir, auth.NamaFilePasswordAwal)); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.Log.Warn("gagal menghapus file password awal", "err", err)
	}
	tulisPesan(w, http.StatusOK, "password berhasil diganti")
}
