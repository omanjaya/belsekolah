package sistem

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"belsekolah/internal/nada"
)

func TestDurasiAudio(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe tidak terpasang")
	}
	dir := t.TempDir()
	wav := filepath.Join(dir, "a.wav")
	os.WriteFile(wav, nada.DingDong(), 0o644)
	d, err := DurasiAudio(context.Background(), wav)
	if err != nil {
		t.Fatal(err)
	}
	if d < 1700 || d > 1900 {
		t.Errorf("durasi = %d ms, mau sekitar 1800", d)
	}

	palsu := filepath.Join(dir, "palsu.mp3")
	os.WriteFile(palsu, []byte("ini bukan audio"), 0o644)
	if _, err := DurasiAudio(context.Background(), palsu); !errors.Is(err, ErrBukanAudio) {
		t.Errorf("file palsu: %v, mau ErrBukanAudio", err)
	}
}

func TestCekNTPTidakPanik(t *testing.T) {
	cek := CekNTP()
	_ = cek() // nil di macOS (tanpa timedatectl), bool di Debian
	_ = cek() // panggilan kedua memakai cache
}
