package sistem

import (
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CekNTP mengembalikan fungsi yang melaporkan apakah jam sistem tersinkron NTP.
// Hasil di-cache 60 detik; nil berarti tidak diketahui (timedatectl tidak tersedia).
func CekNTP() func() *bool {
	var mu sync.Mutex
	var nilai *bool
	var kapan time.Time
	return func() *bool {
		mu.Lock()
		defer mu.Unlock()
		if !kapan.IsZero() && time.Since(kapan) < time.Minute {
			return nilai
		}
		kapan = time.Now()
		out, err := exec.Command("timedatectl", "show", "-p", "NTPSynchronized", "--value").Output()
		if err != nil {
			nilai = nil
			return nil
		}
		v := strings.TrimSpace(string(out)) == "yes"
		nilai = &v
		return nilai
	}
}
