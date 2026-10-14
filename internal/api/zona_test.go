package api

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestZona_GetPengaturanStatusSistem(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)

	var p struct {
		ZonaWaktu string `json:"zonaWaktu"`
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/pengaturan", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &p)
	if p.ZonaWaktu != "WIB" {
		t.Fatalf("pengaturan.zonaWaktu = %q, mau WIB", p.ZonaWaktu)
	}

	var st struct {
		ZonaWaktu string `json:"zonaWaktu"`
	}
	resp = kirim(t, c, "GET", l.srv.URL+"/api/status", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &st)
	if st.ZonaWaktu != "WIB" {
		t.Fatalf("status.zonaWaktu = %q, mau WIB", st.ZonaWaktu)
	}

	var si struct {
		ZonaWaktu string `json:"zonaWaktu"`
		ZonaIana  string `json:"zonaIana"`
	}
	resp = kirim(t, c, "GET", l.srv.URL+"/api/sistem", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &si)
	if si.ZonaWaktu != "WIB" || si.ZonaIana != "Asia/Jakarta" {
		t.Fatalf("sistem zona = %q/%q", si.ZonaWaktu, si.ZonaIana)
	}
}

func pengaturanSekarang(t *testing.T, l *lingkungan, c *http.Client) map[string]any {
	t.Helper()
	var p map[string]any
	resp := kirim(t, c, "GET", l.srv.URL+"/api/pengaturan", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &p)
	return p
}

func TestZona_GantiMemulaiUlang(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	p := pengaturanSekarang(t, l, c)
	p["zonaWaktu"] = "WITA"
	resp := kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", p)
	harapKode(t, resp, 200)
	var hasil struct {
		ZonaWaktu  string `json:"zonaWaktu"`
		MulaiUlang bool   `json:"mulaiUlang"`
	}
	bacaBody(t, resp, &hasil)
	if !hasil.MulaiUlang {
		t.Fatal("mengubah zona harus membalas mulaiUlang=true")
	}
	if z, _ := l.st.ZonaWaktu(ctx); z != "Asia/Makassar" {
		t.Fatalf("zona tersimpan = %q", z)
	}
	tungguSampai(t, func() bool { return l.mulaiUlang.Load() == 1 })

	// Setelah dijadwalkan mulai ulang, perubahan data ditolak 503.
	resp = kirim(t, c, "POST", l.srv.URL+"/api/profil", map[string]string{"nama": "Baru"})
	harapKode(t, resp, http.StatusServiceUnavailable)
	resp.Body.Close()
}

func TestZona_SamaTidakMulaiUlang(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	p := pengaturanSekarang(t, l, c)
	p["zonaWaktu"] = "WIB"
	p["volume"] = float64(70)
	resp := kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", p)
	harapKode(t, resp, 200)
	var hasil map[string]any
	bacaBody(t, resp, &hasil)
	if hasil["mulaiUlang"] == true {
		t.Fatal("zona sama tidak boleh memulai ulang")
	}
	time.Sleep(700 * time.Millisecond)
	if l.mulaiUlang.Load() != 0 {
		t.Fatal("MulaiUlang terpanggil padahal zona tidak berubah")
	}
	if z, _ := l.st.ZonaWaktu(ctx); z != "" {
		t.Fatalf("zona sama tidak perlu disimpan, tersimpan %q", z)
	}
}

func TestZona_TanpaFieldTetapJalan(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	def, _ := l.st.ProfilDefault(ctx)
	resp := kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", map[string]any{"volume": 50, "profilDefaultId": def})
	harapKode(t, resp, 200)
	resp.Body.Close()
}

func TestZona_KodeTidakValid(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	p := pengaturanSekarang(t, l, c)
	p["zonaWaktu"] = "Asia/Makassar"
	p["volume"] = float64(10)
	resp := kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", p)
	harapKode(t, resp, 400)
	if m := pesanError(t, resp); !berisi(m, "WIB") {
		t.Fatalf("pesan = %q, harus menyebut pilihan WIB/WITA/WIT", m)
	}
	if v, _ := l.st.Volume(ctx); v == 10 {
		t.Fatal("permintaan yang ditolak tidak boleh mengubah volume")
	}
}

func TestZona_DitolakSaatRestoreProses(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	mulai := make(chan struct{})
	lanjut := make(chan struct{})
	l.restoreHook = func(_ context.Context, _, _ string) error {
		close(mulai)
		<-lanjut
		return errors.New("batal")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		resp := unggahRestore(t, l, c, []byte("PK"))
		resp.Body.Close()
	}()
	<-mulai
	p := pengaturanSekarang(t, l, c)
	p["zonaWaktu"] = "WIT"
	resp := kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", p)
	harapKode(t, resp, http.StatusServiceUnavailable)
	resp.Body.Close()
	close(lanjut)
	<-done
	if z, _ := l.st.ZonaWaktu(ctx); z != "" {
		t.Fatalf("zona tidak boleh tersimpan saat restore berlangsung, tersimpan %q", z)
	}
}
