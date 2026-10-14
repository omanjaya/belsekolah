package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"belsekolah/internal/backup"
)

// unduhBackup membuat zip di file sementara lebih dulu, agar kegagalan bisa dilaporkan sebagai 500.
func (s *server) unduhBackup(w http.ResponseWriter, r *http.Request) {
	tmp, err := os.CreateTemp("", "belsekolah-*.zip")
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if err := backup.TulisZip(r.Context(), tmp, s.Store, s.AudioDir); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	nama := fmt.Sprintf("bel-sekolah-%s.zip", s.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nama+`"`)
	http.ServeContent(w, r, nama, s.Now(), tmp)
}

func (s *server) pulihkan(w http.ResponseWriter, r *http.Request) {
	if !s.restore.CompareAndSwap(restoreIdle, restoreProses) {
		tulisError(w, http.StatusConflict, "restore sedang diproses")
		return
	}
	disiapkan := false
	defer func() {
		if !disiapkan {
			s.restore.Store(restoreIdle)
		}
	}()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		tulisError(w, http.StatusBadRequest, "file backup terlalu besar atau form tidak valid")
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile("file")
	if err != nil {
		tulisError(w, http.StatusBadRequest, "pilih file backup (.zip)")
		return
	}
	defer f.Close()

	tmp, err := os.CreateTemp("", "belsekolah-restore-*.zip")
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	defer os.Remove(tmp.Name())
	_, err = io.Copy(tmp, f)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}

	siapkan := backup.Siapkan
	if s.SiapkanRestore != nil {
		siapkan = s.SiapkanRestore
	}
	if err := siapkan(r.Context(), tmp.Name(), s.DataDir); err != nil {
		if errors.Is(err, backup.ErrBackupTidakValid) {
			tulisError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.tulisErrorData(w, err)
		return
	}
	disiapkan = true
	s.restore.Store(restoreSiap)
	s.Log.Warn("restore disiapkan, aplikasi akan dimulai ulang")
	tulisPesan(w, http.StatusAccepted, "backup diterima, aplikasi dimulai ulang dalam beberapa detik")
	if s.MulaiUlang != nil {
		go func() {
			time.Sleep(500 * time.Millisecond) // beri waktu respons terkirim
			s.MulaiUlang()
		}()
	}
}
