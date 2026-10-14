package player

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Mpv memutar file lewat program mpv. Output audio dipilih otomatis oleh mpv
// (ALSA pada Debian tanpa PulseAudio).
type Mpv struct {
	Perintah string // kosong berarti "mpv"
}

// EnvAO adalah variabel lingkungan opsional untuk memaksa output audio mpv
// (mis. "null"). Dipakai di container development: perangkat ALSA "null" tidak
// punya jam sehingga mpv selesai seketika, sedangkan --ao=null berjalan real-time.
const EnvAO = "BELSEKOLAH_MPV_AO"

func argumenMpv(file string, volume int, ao string) []string {
	arg := []string{"--no-config", "--no-video", "--no-terminal", "--really-quiet",
		fmt.Sprintf("--volume=%d", volume)}
	if ao != "" {
		arg = append(arg, "--ao="+ao)
	}
	return append(arg, "--", file)
}

func (m Mpv) Run(ctx context.Context, file string, volume int) error {
	perintah := m.Perintah
	if perintah == "" {
		perintah = "mpv"
	}
	cmd := exec.CommandContext(ctx, perintah, argumenMpv(file, volume, os.Getenv(EnvAO))...)
	cmd.WaitDelay = 2 * time.Second // cegah hang pada pipe yang diwarisi proses anak
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
