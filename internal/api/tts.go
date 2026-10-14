package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"belsekolah/internal/store"
	"belsekolah/internal/tts"
)

const batasTeksTTS = 1000

type permintaanTTS struct {
	Nama  string `json:"nama"`
	Teks  string `json:"teks"`
	Suara string `json:"suara"`
}

func (s *server) listSuaraTTS(w http.ResponseWriter, r *http.Request) {
	tulisJSON(w, http.StatusOK, tts.DaftarSuara)
}

// bacaPermintaanTTS membaca dan memvalidasi body; menulis 400 dan mengembalikan false jika tidak valid.
func bacaPermintaanTTS(w http.ResponseWriter, r *http.Request) (permintaanTTS, bool) {
	var p permintaanTTS
	if !bacaJSON(w, r, &p) {
		return p, false
	}
	p.Teks = strings.TrimSpace(p.Teks)
	switch {
	case p.Teks == "":
		tulisError(w, http.StatusBadRequest, "teks tidak boleh kosong")
		return p, false
	case utf8.RuneCountInString(p.Teks) > batasTeksTTS:
		tulisError(w, http.StatusBadRequest, "teks maksimal 1000 karakter")
		return p, false
	case !tts.SuaraDikenal(p.Suara):
		tulisError(w, http.StatusBadRequest, "suara tidak dikenal")
		return p, false
	}
	return p, true
}

// buatSuara memanggil TTS; menulis respons error dan mengembalikan false jika gagal.
func (s *server) buatSuara(w http.ResponseWriter, r *http.Request, p permintaanTTS) ([]byte, bool) {
	if s.TTS == nil {
		tulisError(w, http.StatusServiceUnavailable, tts.ErrTidakTerpasang.Error())
		return nil, false
	}
	b, err := s.TTS.Buat(r.Context(), p.Teks, p.Suara)
	switch {
	case err == nil:
		return b, true
	case errors.Is(err, tts.ErrTidakTerpasang):
		tulisError(w, http.StatusServiceUnavailable, tts.ErrTidakTerpasang.Error())
	case errors.Is(err, tts.ErrGagal):
		s.Log.Warn("TTS gagal", "err", err)
		tulisError(w, http.StatusServiceUnavailable, tts.ErrGagal.Error())
	case errors.Is(err, tts.ErrSuaraTidakDikenal):
		tulisError(w, http.StatusBadRequest, "suara tidak dikenal")
	default:
		s.tulisErrorData(w, err)
	}
	return nil, false
}

func (s *server) pratinjauTTS(w http.ResponseWriter, r *http.Request) {
	p, ok := bacaPermintaanTTS(w, r)
	if !ok {
		return
	}
	b, ok := s.buatSuara(w, r, p)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(b)
}

// simpanSementara menulis data ke file sementara di folder audio dan mengukur durasinya.
// Pemanggil wajib me-rename atau menghapus file yang dikembalikan.
func (s *server) simpanSementara(r *http.Request, data []byte) (string, int64, error) {
	tmp, err := os.CreateTemp(s.AudioDir, ".tts-*")
	if err != nil {
		return "", 0, err
	}
	_, err = tmp.Write(data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp.Name())
		return "", 0, err
	}
	durasi, err := s.DurasiAudio(r.Context(), tmp.Name())
	if err != nil {
		os.Remove(tmp.Name())
		return "", 0, err
	}
	return tmp.Name(), durasi, nil
}

func namaFileTTS() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return "tts-" + hex.EncodeToString(b) + ".mp3"
}

func (s *server) buatAudioTTS(w http.ResponseWriter, r *http.Request) {
	p, ok := bacaPermintaanTTS(w, r)
	if !ok {
		return
	}
	nama := strings.TrimSpace(p.Nama)
	if nama == "" {
		nama = p.Teks
		if rs := []rune(nama); len(rs) > 40 {
			nama = string(rs[:40])
		}
	}
	data, ok := s.buatSuara(w, r, p)
	if !ok {
		return
	}
	tmp, durasi, err := s.simpanSementara(r, data)
	if err != nil {
		s.tulisErrorAudio(w, err)
		return
	}
	namaFile := namaFileTTS()
	tujuan := filepath.Join(s.AudioDir, namaFile)
	if err := os.Rename(tmp, tujuan); err != nil {
		os.Remove(tmp)
		s.tulisErrorData(w, err)
		return
	}
	id, err := s.Store.CreateAudio(r.Context(), store.Audio{
		Nama: nama, Sumber: "tts", File: namaFile, DurasiMS: durasi,
		TeksTTS: p.Teks, SuaraTTS: p.Suara, Dibuat: s.Now(),
	})
	if err != nil {
		os.Remove(tujuan)
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimAudio(w, r, id, http.StatusCreated)
}

func (s *server) ubahAudioTTS(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	a, err := s.Store.GetAudio(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if a.Sumber != "tts" {
		tulisError(w, http.StatusConflict, "hanya suara TTS yang bisa dibuat ulang dari teks")
		return
	}
	p, ok := bacaPermintaanTTS(w, r)
	if !ok {
		return
	}
	data, ok := s.buatSuara(w, r, p)
	if !ok {
		return
	}
	tujuan, err := s.pathAudio(a)
	if err != nil {
		tulisError(w, http.StatusNotFound, "data tidak ditemukan")
		return
	}
	tmp, durasi, err := s.simpanSementara(r, data)
	if err != nil {
		s.tulisErrorAudio(w, err)
		return
	}
	if err := os.Rename(tmp, tujuan); err != nil {
		os.Remove(tmp)
		s.tulisErrorData(w, err)
		return
	}
	if err := s.Store.UpdateAudioTTS(r.Context(), id, p.Teks, p.Suara, durasi); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimAudio(w, r, id, http.StatusOK)
}
