// Package sistem membungkus program sistem: ffprobe dan timedatectl.
package sistem

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os/exec"
	"strconv"
	"time"
)

var (
	ErrBukanAudio      = errors.New("file bukan audio yang valid")
	ErrFfprobeTidakAda = errors.New("ffprobe belum terpasang")
)

const batasWaktuFfprobe = 15 * time.Second

// DurasiAudio mengembalikan durasi file audio dalam milidetik menggunakan ffprobe.
func DurasiAudio(ctx context.Context, path string) (int64, error) {
	ctx, batal := context.WithTimeout(ctx, batasWaktuFfprobe)
	defer batal()
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error",
		"-show_entries", "format=duration:stream=codec_type",
		"-of", "json", path).Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return 0, ErrBukanAudio
		}
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) {
			return 0, ErrFfprobeTidakAda
		}
		return 0, err
	}
	var hasil struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &hasil); err != nil {
		return 0, ErrBukanAudio
	}
	adaAudio := false
	for _, s := range hasil.Streams {
		if s.CodecType == "audio" {
			adaAudio = true
		}
	}
	detik, err := strconv.ParseFloat(hasil.Format.Duration, 64)
	if !adaAudio || err != nil || detik <= 0 {
		return 0, ErrBukanAudio
	}
	return int64(detik * 1000), nil
}
