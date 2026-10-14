package api

import (
	"errors"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"belsekolah/internal/store"
	"belsekolah/internal/zona"
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
	p.ZonaWaktu = zona.Kode(s.zonaAktif())
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
	var ianaBaru string
	if p.ZonaWaktu != "" {
		iana, ok := zona.IANA(p.ZonaWaktu)
		if !ok {
			tulisError(w, http.StatusBadRequest, "zona waktu harus WIB, WITA, atau WIT")
			return
		}
		if iana != s.zonaAktif() {
			if s.restore.Load() != restoreIdle {
				tulisError(w, http.StatusServiceUnavailable, "restore sedang berjalan, coba lagi nanti")
				return
			}
			ianaBaru = iana
		}
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
	if ianaBaru != "" {
		if err := s.Store.SetZonaWaktu(r.Context(), ianaBaru); err != nil {
			s.tulisErrorData(w, err)
			return
		}
		s.akanMulaiUlang.Store(true)
		s.Log.Warn("zona waktu diganti, aplikasi akan dimulai ulang", "zona", ianaBaru)
		out, err := s.bacaPengaturan(r)
		if err != nil {
			s.tulisErrorData(w, err)
			return
		}
		out.ZonaWaktu = zona.Kode(ianaBaru)
		out.MulaiUlang = true
		tulisJSON(w, http.StatusOK, out)
		if s.MulaiUlang != nil {
			go func() {
				time.Sleep(500 * time.Millisecond) // beri waktu respons terkirim
				s.MulaiUlang()
			}()
		}
		return
	}
	s.ambilPengaturan(w, r)
}

func (s *server) sistem(w http.ResponseWriter, r *http.Request) {
	now := s.Now()
	out := sistemDTO{
		Versi:     s.Versi,
		ZonaWaktu: zona.Kode(s.zonaAktif()),
		ZonaIana:  s.zonaAktif(),
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
	out.Boot = s.boot
	lp := s.LookPath
	if lp == nil {
		lp = exec.LookPath
	}
	ada := func(nama string) bool { _, err := lp(nama); return err == nil }
	out.Alat = alatDTO{Mpv: ada("mpv"), Ffprobe: ada("ffprobe"), EdgeTTS: ada("edge-tts")}
	out.RuangKosongMb = ruangKosongMB(s.DataDir)
	out.BackupTerakhir = backupTerakhir(s.DataDir)
	tulisJSON(w, http.StatusOK, out)
}

// ruangKosongMB mengembalikan ruang kosong (MB) pada volume folder data; -1 bila tidak terbaca.
func ruangKosongMB(dir string) int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return -1
	}
	return int64(uint64(st.Bavail) * uint64(st.Bsize) / (1 << 20))
}

// backupTerakhir mengembalikan tanggal (YYYY-MM-DD) backup harian terbaru, atau kosong.
func backupTerakhir(dataDir string) string {
	nama, err := filepath.Glob(filepath.Join(dataDir, "backup", "bel-????-??-??.db"))
	if err != nil || len(nama) == 0 {
		return ""
	}
	sort.Strings(nama)
	b := filepath.Base(nama[len(nama)-1])
	return strings.TrimSuffix(strings.TrimPrefix(b, "bel-"), ".db")
}
