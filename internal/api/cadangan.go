package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/store"
)

var polaBukanSlug = regexp.MustCompile(`[^a-z0-9]+`)

// namaFileJadwal membentuk "jadwal-<slug>.json" yang aman untuk header.
func namaFileJadwal(nama string) string {
	slug := strings.Trim(polaBukanSlug.ReplaceAllString(strings.ToLower(nama), "-"), "-")
	if slug == "" {
		slug = "profil"
	}
	return "jadwal-" + slug + ".json"
}

func kirimFileJadwal(w http.ResponseWriter, d formatjadwal.Data) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, namaFileJadwal(d.Nama)))
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(d)
}

func (s *server) duplikatProfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var p profilDTO
	if !bacaJSON(w, r, &p) {
		return
	}
	baru, err := s.Store.DuplikatProfil(r.Context(), id, p.Nama)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimProfil(w, r, baru, http.StatusCreated)
}

func (s *server) eksporProfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	d, err := s.Store.EksporProfil(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	kirimFileJadwal(w, d)
}

// bacaFileJadwal membaca body mentah sebagai file jadwal. Menulis 400 dan mengembalikan false jika gagal.
func bacaFileJadwal(w http.ResponseWriter, r *http.Request) (formatjadwal.Data, bool) {
	b, err := io.ReadAll(io.LimitReader(r.Body, formatjadwal.MaksUkuran+1))
	if err != nil {
		tulisError(w, http.StatusBadRequest, "file jadwal tidak terbaca")
		return formatjadwal.Data{}, false
	}
	d, err := formatjadwal.Baca(b)
	if err != nil {
		tulisError(w, http.StatusBadRequest, err.Error())
		return formatjadwal.Data{}, false
	}
	return d, true
}

type periksaDTO struct {
	Nama               string   `json:"nama"`
	JumlahBel          int      `json:"jumlahBel"`
	JenisBelTakDikenal []string `json:"jenisBelTakDikenal"`
	NamaTersedia       string   `json:"namaTersedia"`

	PengumumanTakDitemukan []string `json:"pengumumanTakDitemukan"`
}

func (s *server) periksaImpor(w http.ResponseWriter, r *http.Request) {
	d, ok := bacaFileJadwal(w, r)
	if !ok {
		return
	}
	tak, err := s.Store.JenisBelTakDikenal(r.Context(), d)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	png, err := s.Store.PengumumanTakDikenal(r.Context(), d)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if png == nil {
		png = []string{}
	}
	nama, err := s.Store.NamaProfilTersedia(r.Context(), d.Nama)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, periksaDTO{Nama: d.Nama, JumlahBel: len(d.Jadwal), JenisBelTakDikenal: tak, NamaTersedia: nama, PengumumanTakDitemukan: png})
}

type imporDTO struct {
	Data     json.RawMessage                `json:"data"`
	Nama     string                         `json:"nama"`
	Pemetaan map[string]store.PemetaanJenis `json:"pemetaan"`
}

func (s *server) impor(w http.ResponseWriter, r *http.Request) {
	var in imporDTO
	if !bacaJSON(w, r, &in) {
		return
	}
	d, err := formatjadwal.Baca(in.Data)
	if err != nil {
		tulisError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := s.Store.ImporJadwal(r.Context(), d, in.Nama, in.Pemetaan)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	tulisJSON(w, http.StatusCreated, map[string]int64{"profilId": id})
}
