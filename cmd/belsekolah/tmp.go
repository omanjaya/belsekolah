package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// siapkanTmp mengarahkan file sementara ke <data>/tmp.
//
// Di Debian 13 /tmp adalah tmpfs (RAM): unggahan audio/restore berukuran besar
// akan memakan memori dan bisa gagal di mini PC kecil. Folder ini dikosongkan
// dan dibuat ulang setiap start, sehingga sisa file dari proses yang mati
// mendadak tidak menumpuk.
func siapkanTmp(dataDir string) error {
	tmp := filepath.Join(dataDir, "tmp")
	if err := os.RemoveAll(tmp); err != nil {
		return fmt.Errorf("bersihkan folder sementara: %w", err)
	}
	if err := os.MkdirAll(tmp, 0o700); err != nil {
		return fmt.Errorf("buat folder sementara: %w", err)
	}
	return os.Setenv("TMPDIR", tmp)
}

// sapuSisaAudio menghapus file sementara unggahan/TTS (.unggah-*, .tts-*) yang
// tertinggal di folder audio akibat proses berhenti mendadak. Mengembalikan jumlah yang dihapus.
func sapuSisaAudio(audioDir string) int {
	n := 0
	for _, pola := range []string{".unggah-*", ".tts-*"} {
		cocok, _ := filepath.Glob(filepath.Join(audioDir, pola))
		for _, f := range cocok {
			if os.Remove(f) == nil {
				n++
			}
		}
	}
	return n
}
