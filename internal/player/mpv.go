package player

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Mpv memutar file lewat program mpv. Output audio dipilih otomatis oleh mpv
// (ALSA pada Debian tanpa PulseAudio).
type Mpv struct {
	Perintah string // kosong berarti "mpv"
}

func (m Mpv) Run(ctx context.Context, file string, volume int) error {
	perintah := m.Perintah
	if perintah == "" {
		perintah = "mpv"
	}
	cmd := exec.CommandContext(ctx, perintah,
		"--no-config", "--no-video", "--no-terminal", "--really-quiet",
		fmt.Sprintf("--volume=%d", volume),
		"--", file)
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
