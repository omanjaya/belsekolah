// Package tts mengubah teks menjadi MP3 dengan memanggil program edge-tts.
// CLI resmi dipakai (bukan protokol sendiri) karena Microsoft sering mengubah
// token DRM-nya; cukup `pipx upgrade edge-tts` saat itu terjadi.
package tts

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Suara struct {
	ID   string `json:"id"`
	Nama string `json:"nama"`
}

var DaftarSuara = []Suara{
	{ID: "id-ID-GadisNeural", Nama: "Gadis (perempuan)"},
	{ID: "id-ID-ArdiNeural", Nama: "Ardi (laki-laki)"},
}

var (
	ErrSuaraTidakDikenal = errors.New("suara tidak dikenal")
	ErrTidakTerpasang    = errors.New("program edge-tts belum terpasang")
	ErrGagal             = errors.New("gagal membuat suara TTS; pastikan mini PC terhubung ke internet")
)

func SuaraDikenal(id string) bool {
	for _, s := range DaftarSuara {
		if s.ID == id {
			return true
		}
	}
	return false
}

// EdgeTTS menjalankan CLI edge-tts. Nilai kosong memakai "edge-tts" dan batas 60 detik.
type EdgeTTS struct {
	Perintah string
	Batas    time.Duration
}

func (e EdgeTTS) Buat(ctx context.Context, teks, suara string) ([]byte, error) {
	if !SuaraDikenal(suara) {
		return nil, ErrSuaraTidakDikenal
	}
	perintah := e.Perintah
	if perintah == "" {
		perintah = "edge-tts"
	}
	batas := e.Batas
	if batas == 0 {
		batas = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, batas)
	defer cancel()

	dir, err := os.MkdirTemp("", "belsekolah-tts-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	fileTeks := filepath.Join(dir, "teks.txt")
	if err := os.WriteFile(fileTeks, []byte(teks), 0o600); err != nil {
		return nil, err
	}
	fileMP3 := filepath.Join(dir, "suara.mp3")

	cmd := exec.CommandContext(ctx, perintah, "--voice", suara, "--file", fileTeks, "--write-media", fileMP3)
	cmd.WaitDelay = 2 * time.Second
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			return nil, ErrTidakTerpasang
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: melewati batas waktu %s", ErrGagal, batas)
		}
		return nil, fmt.Errorf("%w: %s", ErrGagal, ringkas(out))
	}
	b, err := os.ReadFile(fileMP3)
	if err != nil || len(b) == 0 {
		return nil, fmt.Errorf("%w: edge-tts tidak menghasilkan suara", ErrGagal)
	}
	return b, nil
}

// ringkas mengambil baris terakhir output yang tidak kosong (biasanya pesan error Python).
func ringkas(out []byte) string {
	baris := strings.Split(strings.TrimSpace(string(out)), "\n")
	s := strings.TrimSpace(baris[len(baris)-1])
	if len(s) > 200 {
		s = s[len(s)-200:]
	}
	return s
}
