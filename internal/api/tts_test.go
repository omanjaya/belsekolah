package api

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"belsekolah/internal/nada"
	"belsekolah/internal/tts"
)

func TestDaftarSuaraTTS(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	var daftar []tts.Suara
	resp := kirim(t, c, "GET", l.srv.URL+"/api/tts/suara", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &daftar)
	if len(daftar) != 2 || daftar[0].ID != "id-ID-GadisNeural" {
		t.Fatalf("suara = %+v", daftar)
	}
}

func TestPratinjauTTS(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	u := l.srv.URL + "/api/tts/pratinjau"

	resp := kirim(t, c, "POST", u, map[string]string{"teks": "  Selamat pagi  ", "suara": "id-ID-GadisNeural"})
	harapKode(t, resp, 200)
	if ct := resp.Header.Get("Content-Type"); ct != "audio/mpeg" {
		t.Errorf("content-type = %q", ct)
	}
	isi, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Equal(isi, nada.DingDong()) {
		t.Error("isi pratinjau berbeda")
	}
	if l.tts.panggilan[0] != "id-ID-GadisNeural|Selamat pagi" {
		t.Errorf("panggilan = %v (teks harus di-trim)", l.tts.panggilan)
	}

	for nama, body := range map[string]map[string]string{
		"teks kosong":  {"teks": "   ", "suara": "id-ID-GadisNeural"},
		"teks panjang": {"teks": strings.Repeat("a", 1001), "suara": "id-ID-GadisNeural"},
		"suara asing":  {"teks": "halo", "suara": "en-US-AriaNeural"},
	} {
		resp := kirim(t, c, "POST", u, body)
		if resp.StatusCode != 400 {
			t.Errorf("%s: kode %d, mau 400", nama, resp.StatusCode)
		}
		resp.Body.Close()
	}

	l.tts.err = fmt.Errorf("%w: Cannot connect to host", tts.ErrGagal)
	resp = kirim(t, c, "POST", u, map[string]string{"teks": "halo", "suara": "id-ID-GadisNeural"})
	harapKode(t, resp, 503)
	if p := pesanError(t, resp); p != tts.ErrGagal.Error() {
		t.Errorf("pesan = %q (detail teknis tidak boleh bocor)", p)
	}
	l.tts.err = tts.ErrTidakTerpasang
	resp = kirim(t, c, "POST", u, map[string]string{"teks": "halo", "suara": "id-ID-GadisNeural"})
	harapKode(t, resp, 503)
	if p := pesanError(t, resp); p != tts.ErrTidakTerpasang.Error() {
		t.Errorf("pesan = %q", p)
	}
}

func TestBuatDanUbahAudioTTS(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)

	var a audioDTO
	teks := "Perhatian, waktu istirahat telah tiba. Silakan keluar kelas dengan tertib dan jaga kebersihan."
	resp := kirim(t, c, "POST", l.srv.URL+"/api/audio/tts", map[string]string{"teks": teks, "suara": "id-ID-ArdiNeural"})
	harapKode(t, resp, 201)
	bacaBody(t, resp, &a)
	if a.Sumber != "tts" || a.TeksTTS != teks || a.SuaraTTS != "id-ID-ArdiNeural" || a.DurasiMS != 1500 {
		t.Fatalf("audio = %+v", a)
	}
	if a.Nama != string([]rune(teks)[:40]) {
		t.Errorf("nama default = %q, mau 40 karakter pertama teks", a.Nama)
	}
	tersimpan, _ := l.st.GetAudio(ctx, a.ID)
	if !strings.HasPrefix(tersimpan.File, "tts-") || filepath.Ext(tersimpan.File) != ".mp3" {
		t.Errorf("nama file = %q", tersimpan.File)
	}
	path := filepath.Join(l.audioDir, tersimpan.File)
	if isi, _ := os.ReadFile(path); !bytes.Equal(isi, nada.DingDong()) {
		t.Error("file TTS tidak tersimpan")
	}

	var b audioDTO
	resp = kirim(t, c, "POST", l.srv.URL+"/api/audio/tts", map[string]string{"nama": "Pulang", "teks": "Waktunya pulang", "suara": "id-ID-GadisNeural"})
	harapKode(t, resp, 201)
	bacaBody(t, resp, &b)
	if b.Nama != "Pulang" {
		t.Errorf("nama = %q", b.Nama)
	}

	l.tts.hasil = nada.Sirene()
	resp = kirim(t, c, "PUT", fmt.Sprintf("%s/api/audio/%d/tts", l.srv.URL, a.ID), map[string]string{"teks": "Teks baru", "suara": "id-ID-GadisNeural"})
	harapKode(t, resp, 200)
	bacaBody(t, resp, &a)
	if a.TeksTTS != "Teks baru" || a.SuaraTTS != "id-ID-GadisNeural" {
		t.Fatalf("audio diubah = %+v", a)
	}
	if isi, _ := os.ReadFile(path); !bytes.Equal(isi, nada.Sirene()) {
		t.Error("file harus ditimpa dengan suara baru di path yang sama")
	}

	daftar, _ := l.st.ListAudio(ctx)
	resp = kirim(t, c, "PUT", fmt.Sprintf("%s/api/audio/%d/tts", l.srv.URL, daftar[0].ID), map[string]string{"teks": "x", "suara": "id-ID-GadisNeural"})
	harapKode(t, resp, 409) // audio bawaan bukan TTS
	resp.Body.Close()
	resp = kirim(t, c, "PUT", l.srv.URL+"/api/audio/999/tts", map[string]string{"teks": "x", "suara": "id-ID-GadisNeural"})
	harapKode(t, resp, 404)
	resp.Body.Close()

	l.tts.err = tts.ErrGagal
	resp = kirim(t, c, "POST", l.srv.URL+"/api/audio/tts", map[string]string{"teks": "gagal", "suara": "id-ID-GadisNeural"})
	harapKode(t, resp, 503)
	resp.Body.Close()
	ents, _ := os.ReadDir(l.audioDir)
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".") {
			t.Errorf("file sementara tertinggal: %s", e.Name())
		}
	}
}
