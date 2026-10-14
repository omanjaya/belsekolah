package api

import (
	"errors"
	"net"
	"net/http"

	"belsekolah/internal/store"
)

func (s *server) bacaPengaturan(r *http.Request) (pengaturanDTO, error) {
	var p pengaturanDTO
	v, err := s.Store.Volume(r.Context())
	if err != nil {
		return p, err
	}
	p.Volume = v
	def, err := s.Store.ProfilDefault(r.Context())
	if err != nil && !errors.Is(err, store.ErrTidakAda) {
		return p, err
	}
	p.ProfilDefaultID = def
	return p, nil
}

func (s *server) ambilPengaturan(w http.ResponseWriter, r *http.Request) {
	p, err := s.bacaPengaturan(r)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, p)
}

func (s *server) ubahPengaturan(w http.ResponseWriter, r *http.Request) {
	var p pengaturanDTO
	if !bacaJSON(w, r, &p) {
		return
	}
	if p.Volume < 0 || p.Volume > 100 {
		tulisError(w, http.StatusBadRequest, "volume harus 0 sampai 100")
		return
	}
	if err := s.Store.SetProfilDefault(r.Context(), p.ProfilDefaultID); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if err := s.Store.SetVolume(r.Context(), p.Volume); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.Pemutar.SetVolume(p.Volume)
	s.ubahData()
	s.ambilPengaturan(w, r)
}

func (s *server) sistem(w http.ResponseWriter, r *http.Request) {
	now := s.Now()
	out := sistemDTO{
		Versi:     s.Versi,
		ZonaWaktu: now.Location().String() + " (" + now.Format("MST -07:00") + ")",
		Sekarang:  now,
		IP:        []string{},
	}
	if alamat, err := net.InterfaceAddrs(); err == nil {
		for _, a := range alamat {
			if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() && ipn.IP.To4() != nil {
				out.IP = append(out.IP, ipn.IP.String())
			}
		}
	}
	if s.CekNTP != nil {
		out.NTP = s.CekNTP()
	}
	tulisJSON(w, http.StatusOK, out)
}
