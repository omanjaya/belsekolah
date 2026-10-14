package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"belsekolah/internal/nada"
	"belsekolah/internal/sistem"
	"belsekolah/internal/store"
)

func ambilBoot(t *testing.T, l *lingkungan) string {
	t.Helper()
	resp := kirim(t, l.klien(), "GET", l.srv.URL+"/api/sesi", nil)
	harapKode(t, resp, 200)
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, mau no-store", cc)
	}
	var h struct {
		Boot string `json:"boot"`
	}
	bacaBody(t, resp, &h)
	if h.Boot == "" {
		t.Fatal("boot kosong")
	}
	return h.Boot
}

func TestBootID(t *testing.T) {
	a, b := siapkan(t), siapkan(t)
	a1, a2 := ambilBoot(t, a), ambilBoot(t, a)
	if a1 != a2 {
		t.Errorf("boot tidak stabil: %s vs %s", a1, a2)
	}
	if b1 := ambilBoot(t, b); b1 == a1 {
		t.Errorf("dua instance punya boot sama: %s", b1)
	}
	c := a.masuk(t)
	resp := kirim(t, c, "GET", a.srv.URL+"/api/sistem", nil)
	harapKode(t, resp, 200)
	var sis struct {
		Boot string `json:"boot"`
		Alat *struct {
			Mpv     *bool `json:"mpv"`
			Ffprobe *bool `json:"ffprobe"`
			EdgeTTS *bool `json:"edgeTts"`
		} `json:"alat"`
		Ruang  *int64  `json:"ruangKosongMb"`
		Backup *string `json:"backupTerakhir"`
	}
	bacaBody(t, resp, &sis)
	if sis.Boot != a1 {
		t.Errorf("boot /api/sistem = %q, mau %q", sis.Boot, a1)
	}
	if sis.Alat == nil || sis.Alat.Mpv == nil || sis.Alat.Ffprobe == nil || sis.Alat.EdgeTTS == nil {
		t.Errorf("alat tidak lengkap: %+v", sis.Alat)
	}
	if sis.Ruang == nil || *sis.Ruang <= 0 {
		t.Errorf("ruangKosongMb = %v", sis.Ruang)
	}
	if sis.Backup == nil {
		t.Error("backupTerakhir tidak ada")
	}
}

func TestBackupTerakhir(t *testing.T) {
	dir := t.TempDir()
	if got := backupTerakhir(dir); got != "" {
		t.Errorf("tanpa backup = %q", got)
	}
	os.MkdirAll(filepath.Join(dir, "backup"), 0o755)
	for _, n := range []string{"bel-2026-10-01.db", "bel-2026-10-09.db", "bel-sebelum-restore-x.db"} {
		os.WriteFile(filepath.Join(dir, "backup", n), nil, 0o644)
	}
	if got := backupTerakhir(dir); got != "2026-10-09" {
		t.Errorf("backupTerakhir = %q", got)
	}
}

func TestHeaderAman(t *testing.T) {
	l := siapkan(t)
	for _, p := range []string{"/", "/api/sesi", "/api/tidak-ada"} {
		resp := kirim(t, l.klien(), "GET", l.srv.URL+p, nil)
		resp.Body.Close()
		if resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("X-Frame-Options") != "DENY" {
			t.Errorf("%s: header pengaman tidak ada", p)
		}
	}
}

func TestRestoreMengunciPerubahan(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	resp := unggah(t, c, l.srv.URL+"/api/restore", "", "b.zip", []byte("x"))
	harapKode(t, resp, 400) // backup tidak valid; kunci harus dilepas lagi
	resp.Body.Close()
	l.restoreHook = func(context.Context, string, string) error { return nil }
	resp = unggah(t, c, l.srv.URL+"/api/restore", "", "b.zip", []byte("x"))
	harapKode(t, resp, 202)
	resp.Body.Close()

	for _, m := range [][2]string{{"POST", "/api/bel/stop"}, {"PUT", "/api/pengaturan"}, {"POST", "/api/restore"}, {"POST", "/api/password"}} {
		resp := kirim(t, c, m[0], l.srv.URL+m[1], map[string]string{})
		harapKode(t, resp, 503)
		if pesan := pesanError(t, resp); pesan != "aplikasi sedang dimulai ulang untuk menerapkan restore" {
			t.Errorf("pesan = %q", pesan)
		}
	}
	resp = kirim(t, c, "GET", l.srv.URL+"/api/status", nil)
	harapKode(t, resp, 200)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/logout", nil)
	harapKode(t, resp, 200)
	resp.Body.Close()
	resp = kirim(t, l.klien(), "POST", l.srv.URL+"/api/login", map[string]string{"password": l.password})
	harapKode(t, resp, 200)
	resp.Body.Close()
}

func TestRestoreParalelDitolak409(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	masuk, lepas := make(chan struct{}), make(chan struct{})
	l.restoreHook = func(context.Context, string, string) error {
		close(masuk)
		<-lepas
		return nil
	}
	hasil := make(chan int, 1)
	go func() {
		resp := unggah(t, c, l.srv.URL+"/api/restore", "", "b.zip", []byte("x"))
		resp.Body.Close()
		hasil <- resp.StatusCode
	}()
	<-masuk
	resp := unggah(t, c, l.srv.URL+"/api/restore", "", "b.zip", []byte("x"))
	harapKode(t, resp, 409)
	if pesan := pesanError(t, resp); pesan != "restore sedang diproses" {
		t.Errorf("pesan = %q", pesan)
	}
	close(lepas)
	if k := <-hasil; k != 202 {
		t.Errorf("restore pertama = %d, mau 202", k)
	}
}

func TestRestoreGagalMelepasKunci(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	l.restoreHook = func(context.Context, string, string) error { return errors.New("disk penuh") }
	resp := unggah(t, c, l.srv.URL+"/api/restore", "", "b.zip", []byte("x"))
	harapKode(t, resp, 500)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/bel/stop", nil)
	harapKode(t, resp, 200)
	resp.Body.Close()
	l.restoreHook = func(context.Context, string, string) error { return nil }
	resp = unggah(t, c, l.srv.URL+"/api/restore", "", "b.zip", []byte("x"))
	harapKode(t, resp, 202)
	resp.Body.Close()
}

func TestErrorFfprobe(t *testing.T) {
	kasus := []struct {
		err  error
		kode int
		isi  string
	}{
		{sistem.ErrBukanAudio, 400, "file bukan audio yang valid"},
		{sistem.ErrFfprobeTidakAda, 503, "program ffprobe belum terpasang"},
		{errors.New("meledak"), 500, "terjadi kesalahan pada server"},
	}
	for _, k := range kasus {
		l := siapkan(t)
		c := l.masuk(t)
		l.durasiHook = func(context.Context, string) (int64, error) { return 0, k.err }

		resp := unggah(t, c, l.srv.URL+"/api/audio", "", "a.wav", nada.DingDong())
		harapKode(t, resp, k.kode)
		if p := pesanError(t, resp); p != k.isi {
			t.Errorf("unggah %v: pesan %q", k.err, p)
		}
		resp = kirim(t, c, "POST", l.srv.URL+"/api/audio/tts", map[string]string{"teks": "halo", "suara": "id-ID-GadisNeural"})
		harapKode(t, resp, k.kode)
		if p := pesanError(t, resp); p != k.isi {
			t.Errorf("tts %v: pesan %q", k.err, p)
		}
	}
}

func TestPathAudioMenolakNamaBerbahaya(t *testing.T) {
	s := &server{Deps: Deps{AudioDir: "/data/audio"}}
	for _, nama := range []string{"", ".", "..", "/", "a/.."} {
		if _, err := s.pathAudio(audioUji(nama)); err == nil {
			t.Errorf("nama %q seharusnya ditolak", nama)
		}
	}
	p, err := s.pathAudio(audioUji("../../etc/x.mp3"))
	if err != nil || p != "/data/audio/x.mp3" {
		t.Errorf("p = %q, err = %v", p, err)
	}
}

func TestFileAudioNamaRusak404(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	id, err := l.st.CreateAudio(ctx, audioUji(".."))
	if err != nil {
		t.Fatal(err)
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/audio/"+strconv.FormatInt(id, 10)+"/file", nil)
	harapKode(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

func audioUji(file string) store.Audio {
	return store.Audio{Nama: "Uji", Sumber: "upload", File: file, DurasiMS: 1000}
}
