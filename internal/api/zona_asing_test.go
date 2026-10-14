package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"belsekolah/internal/auth"
)

// siapkanZona membuat server uji kedua di atas lingkungan l dengan zona aktif tertentu.
func siapkanZona(t *testing.T, l *lingkungan, iana string) {
	t.Helper()
	au := auth.New(l.st, func() time.Time { return l.now })
	h := New(Deps{
		Store: l.st, Auth: au, Penjadwal: l.penjadwal, Pemutar: l.pemutar,
		DataDir: l.dataDir, AudioDir: l.audioDir, Versi: "uji", Zona: iana,
		Web:        fstest.MapFS{"index.html": {Data: []byte("x")}},
		Log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:        func() time.Time { return l.now },
		MulaiUlang: func() { l.mulaiUlang.Add(1) },
		TTS:        l.tts,
	})
	l.srv.Close()
	l.srv = httptest.NewServer(h)
	t.Cleanup(l.srv.Close)
}

func TestZona_DiLuarWibWitaWitTidakBerubah(t *testing.T) {
	l := siapkan(t)
	siapkanZona(t, l, "Asia/Pontianak")
	c := l.masuk(t)
	p := pengaturanSekarang(t, l, c)
	if p["zonaWaktu"] != "Asia/Pontianak" {
		t.Fatalf("zonaWaktu = %v", p["zonaWaktu"])
	}
	p["volume"] = float64(42)
	resp := kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", p)
	harapKode(t, resp, 200)
	var hasil map[string]any
	bacaBody(t, resp, &hasil)
	if hasil["mulaiUlang"] == true {
		t.Fatal("zona tidak berubah, tidak boleh mulai ulang")
	}
	if v, _ := l.st.Volume(ctx); v != 42 {
		t.Fatalf("volume = %d", v)
	}
	time.Sleep(700 * time.Millisecond)
	if l.mulaiUlang.Load() != 0 {
		t.Fatal("MulaiUlang terpanggil")
	}

	// Mengganti ke WITA tetap memulai ulang.
	p["zonaWaktu"] = "WITA"
	resp = kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", p)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &hasil)
	if hasil["mulaiUlang"] != true {
		t.Fatal("ganti ke WITA harus mulaiUlang")
	}
	tungguSampai(t, func() bool { return l.mulaiUlang.Load() == 1 })
}

func TestServer_ZonaKosongTidakLocal(t *testing.T) {
	l := siapkan(t)
	siapkanZona(t, l, "")
	c := l.masuk(t)
	var si struct {
		ZonaIana string `json:"zonaIana"`
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/sistem", nil)
	harapKode(t, resp, http.StatusOK)
	bacaBody(t, resp, &si)
	if si.ZonaIana == "" || si.ZonaIana == "Local" {
		t.Fatalf("zonaIana = %q", si.ZonaIana)
	}
}
