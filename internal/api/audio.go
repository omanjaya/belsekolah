package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"belsekolah/internal/player"
	"belsekolah/internal/store"
)

const batasUnggahAudio = 20 << 20

func (s *server) pathAudio(a store.Audio) string {
	return filepath.Join(s.AudioDir, filepath.Base(a.File))
}

func (s *server) listAudio(w http.ResponseWriter, r *http.Request) {
	as, err := s.Store.ListAudio(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]audioDTO, 0, len(as))
	for _, a := range as {
		out = append(out, keAudioDTO(a))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimAudio(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	a, err := s.Store.GetAudio(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, keAudioDTO(a))
}

func namaAcak(ext string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return "upload-" + hex.EncodeToString(b) + ext
}

func (s *server) unggahAudio(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, batasUnggahAudio+(1<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		tulisError(w, http.StatusBadRequest, "file terlalu besar (maksimal 20 MB) atau form tidak valid")
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, hdr, err := r.FormFile("file")
	if err != nil {
		tulisError(w, http.StatusBadRequest, "pilih file audio")
		return
	}
	defer f.Close()
	if hdr.Size > batasUnggahAudio {
		tulisError(w, http.StatusBadRequest, "file terlalu besar (maksimal 20 MB)")
		return
	}
	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	if ext != ".mp3" && ext != ".wav" {
		tulisError(w, http.StatusBadRequest, "hanya file MP3 atau WAV yang diterima")
		return
	}
	nama := strings.TrimSpace(r.FormValue("nama"))
	if nama == "" {
		nama = strings.TrimSuffix(filepath.Base(hdr.Filename), filepath.Ext(hdr.Filename))
	}

	tmp, err := os.CreateTemp(s.AudioDir, ".unggah-*")
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // tidak berpengaruh setelah berhasil di-rename
	_, err = io.Copy(tmp, f)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}

	durasi, err := s.DurasiAudio(r.Context(), tmpPath)
	if err != nil {
		tulisError(w, http.StatusBadRequest, "file bukan audio yang valid")
		return
	}
	namaFile := namaAcak(ext)
	tujuan := filepath.Join(s.AudioDir, namaFile)
	if err := os.Rename(tmpPath, tujuan); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	id, err := s.Store.CreateAudio(r.Context(), store.Audio{
		Nama: nama, Sumber: "upload", File: namaFile, DurasiMS: durasi, Dibuat: s.Now(),
	})
	if err != nil {
		os.Remove(tujuan)
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimAudio(w, r, id, http.StatusCreated)
}

func (s *server) ubahAudio(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var p struct {
		Nama string `json:"nama"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	if err := s.Store.UpdateNamaAudio(r.Context(), id, p.Nama); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimAudio(w, r, id, http.StatusOK)
}

func (s *server) hapusAudio(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	a, err := s.Store.DeleteAudio(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if err := os.Remove(s.pathAudio(a)); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.Log.Warn("gagal menghapus file audio", "file", a.File, "err", err)
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) fileAudio(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	a, err := s.Store.GetAudio(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	http.ServeFile(w, r, s.pathAudio(a))
}

// tesAudio memutar satu suara di speaker tanpa mencatat riwayat.
func (s *server) tesAudio(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	a, err := s.Store.GetAudio(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.Pemutar.Play(player.Permintaan{Files: []string{s.pathAudio(a)}, Prioritas: player.Tes})
	tulisPesan(w, http.StatusAccepted, "suara diputar")
}
