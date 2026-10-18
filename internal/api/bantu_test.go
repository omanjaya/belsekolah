package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"golang.org/x/crypto/bcrypt"

	"belsekolah/internal/auth"
	"belsekolah/internal/backup"
	"belsekolah/internal/bawaan"
	"belsekolah/internal/nada"
	"belsekolah/internal/player"
	"belsekolah/internal/scheduler"
	"belsekolah/internal/sistem"
	"belsekolah/internal/store"
)

type panggilManual struct {
	JenisBelID int64
	Label      string
	Darurat    bool
}

type fakePenjadwal struct {
	mu     sync.Mutex
	status scheduler.Status
	reload int
	manual []panggilManual
	files  map[int64][]string
}

func (f *fakePenjadwal) Status() scheduler.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakePenjadwal) Reload() {
	f.mu.Lock()
	f.reload++
	f.mu.Unlock()
}

func (f *fakePenjadwal) BunyikanManual(_ context.Context, id int64, label string, darurat bool) {
	f.mu.Lock()
	f.manual = append(f.manual, panggilManual{id, label, darurat})
	f.mu.Unlock()
}

func (f *fakePenjadwal) FileUrutan(_ context.Context, id int64) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.files[id]
}

func (f *fakePenjadwal) jumlahReload() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reload
}

type fakePemutar struct {
	mu      sync.Mutex
	diputar []player.Permintaan
	stop    int
	stopTes int
	volume  int
}

func (f *fakePemutar) StopTes() {
	f.mu.Lock()
	f.stopTes++
	f.mu.Unlock()
}

func (f *fakePemutar) Play(r player.Permintaan) {
	f.mu.Lock()
	f.diputar = append(f.diputar, r)
	f.mu.Unlock()
}

func (f *fakePemutar) Stop() {
	f.mu.Lock()
	f.stop++
	f.mu.Unlock()
}

func (f *fakePemutar) SetVolume(v int) {
	f.mu.Lock()
	f.volume = v
	f.mu.Unlock()
}

type lingkungan struct {
	srv        *httptest.Server
	st         *store.Store
	penjadwal  *fakePenjadwal
	pemutar    *fakePemutar
	dataDir    string
	audioDir   string
	password   string
	now        time.Time
	ntp        *bool
	tts        *fakeTTS
	mulaiUlang atomic.Int32

	// Hook uji; diatur setelah siapkan() dan sebelum permintaan dikirim.
	durasiHook  func(ctx context.Context, path string) (int64, error)
	restoreHook func(ctx context.Context, zipPath, dataDir string) error
}

var ctx = context.Background()

// durasiPalsu menganggap file berawalan "RIFF" sebagai audio 1500 ms.
func durasiPalsu(_ context.Context, path string) (int64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if bytes.HasPrefix(b, []byte("RIFF")) {
		return 1500, nil
	}
	return 0, sistem.ErrBukanAudio
}

func siapkan(t *testing.T) *lingkungan {
	t.Helper()
	auth.BiayaHash = bcrypt.MinCost
	l := &lingkungan{penjadwal: &fakePenjadwal{}, pemutar: &fakePemutar{}, tts: &fakeTTS{}}
	l.now = time.Date(2026, 10, 12, 8, 0, 0, 0, time.Local) // Senin
	l.dataDir = t.TempDir()
	l.audioDir = filepath.Join(l.dataDir, "audio")
	if err := os.MkdirAll(l.audioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(l.dataDir, "bel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	l.st = st
	if err := bawaan.Pasang(ctx, st, l.audioDir, l.now); err != nil {
		t.Fatal(err)
	}
	au := auth.New(st, func() time.Time { return l.now })
	pw, _, err := au.SiapkanPasswordAwal(ctx)
	if err != nil {
		t.Fatal(err)
	}
	l.password = pw
	os.WriteFile(filepath.Join(l.dataDir, auth.NamaFilePasswordAwal), []byte(pw), 0o600)

	h := New(Deps{
		Store: st, Auth: au, Penjadwal: l.penjadwal, Pemutar: l.pemutar,
		DataDir: l.dataDir, AudioDir: l.audioDir, Versi: "uji", Zona: "Asia/Jakarta",
		Web:    fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>Bel</title>")}, "app.js": {Data: []byte("js")}},
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:    func() time.Time { return l.now },
		CekNTP: func() *bool { return l.ntp },
		DurasiAudio: func(c context.Context, p string) (int64, error) {
			if l.durasiHook != nil {
				return l.durasiHook(c, p)
			}
			return durasiPalsu(c, p)
		},
		SiapkanRestore: func(c context.Context, z, d string) error {
			if l.restoreHook != nil {
				return l.restoreHook(c, z, d)
			}
			return backup.Siapkan(c, z, d)
		},
		MulaiUlang: func() { l.mulaiUlang.Add(1) },
		TTS:        l.tts,
	})
	l.srv = httptest.NewServer(h)
	t.Cleanup(l.srv.Close)
	return l
}

func (l *lingkungan) klien() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// masuk login dan (jika perlu) mengganti password awal, lalu mengembalikan klien yang sudah login.
func (l *lingkungan) masuk(t *testing.T) *http.Client {
	t.Helper()
	c := l.klien()
	resp := kirim(t, c, "POST", l.srv.URL+"/api/login", map[string]string{"password": l.password})
	harapKode(t, resp, 200)
	var hasil struct {
		WajibGantiPassword bool `json:"wajibGantiPassword"`
	}
	bacaBody(t, resp, &hasil)
	if hasil.WajibGantiPassword {
		baru := "rahasia-baru-123"
		resp := kirim(t, c, "POST", l.srv.URL+"/api/password", map[string]string{"lama": l.password, "baru": baru})
		harapKode(t, resp, 200)
		resp.Body.Close()
		l.password = baru
	}
	return c
}

func kirim(t *testing.T, c *http.Client, metode, url string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(metode, url, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func harapKode(t *testing.T, resp *http.Response, kode int) {
	t.Helper()
	if resp.StatusCode != kode {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("%s %s: kode %d, mau %d; body: %s", resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, kode, b)
	}
}

func bacaBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func pesanError(t *testing.T, resp *http.Response) string {
	t.Helper()
	var e struct {
		Error string `json:"error"`
	}
	bacaBody(t, resp, &e)
	if e.Error == "" {
		t.Fatal("respons error tanpa field error")
	}
	return e.Error
}

func berisi(s, sub string) bool { return strings.Contains(s, sub) }

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// unggahRestore mengirim isi sebagai file backup.zip ke POST /api/restore.
func unggahRestore(t *testing.T, l *lingkungan, c *http.Client, isi []byte) *http.Response {
	t.Helper()
	return unggah(t, c, l.srv.URL+"/api/restore", "", "backup.zip", isi)
}

func storeRiwayat(l *lingkungan, status string) store.Riwayat {
	return store.Riwayat{Waktu: l.now, Label: "Uji", Sumber: store.SumberManual, Status: status}
}

func tungguSampai(t *testing.T, kondisi func() bool) {
	t.Helper()
	batas := time.Now().Add(3 * time.Second)
	for time.Now().Before(batas) {
		if kondisi() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("kondisi tidak terpenuhi dalam 3 detik")
}

type fakeTTS struct {
	mu        sync.Mutex
	err       error
	hasil     []byte
	panggilan []string // "suara|teks"
}

func (f *fakeTTS) Buat(_ context.Context, teks, suara string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.panggilan = append(f.panggilan, suara+"|"+teks)
	if f.err != nil {
		return nil, f.err
	}
	if f.hasil != nil {
		return f.hasil, nil
	}
	return nada.DingDong(), nil // berawalan RIFF sehingga durasiPalsu menganggapnya audio
}
