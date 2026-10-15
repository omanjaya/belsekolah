package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"belsekolah/internal/schedule"
)

type ringkasProfil struct {
	ID        int64  `json:"id"`
	Nama      string `json:"nama"`
	JumlahBel int    `json:"jumlahBel"`
}

// profilBerisi membuat 2 bel di profil default dan mengembalikan ID-nya.
func profilBerisi(t *testing.T, l *lingkungan) int64 {
	t.Helper()
	profil, err := l.st.ProfilDefault(ctx)
	if err != nil {
		t.Fatal(err)
	}
	js, _ := l.st.ListJenisBel(ctx)
	for _, jam := range []string{"07:00", "12:00"} {
		if _, err := l.st.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: jam, JenisBelID: js[0].ID, Label: "Uji"}); err != nil {
			t.Fatal(err)
		}
	}
	return profil
}

func kirimMentah(t *testing.T, c *http.Client, metode, url string, isi []byte) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(metode, url, bytes.NewReader(isi))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestProfil_JumlahBelDanDuplikat(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil := profilBerisi(t, l)

	var daftar []ringkasProfil
	resp := kirim(t, c, "GET", l.srv.URL+"/api/profil", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &daftar)
	if len(daftar) != 1 || daftar[0].JumlahBel != 2 {
		t.Fatalf("profil = %+v", daftar)
	}

	reloadSebelum := l.penjadwal.jumlahReload()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/profil/"+itoa(profil)+"/duplikat", map[string]string{"nama": "Normal (salinan)"})
	harapKode(t, resp, 201)
	var baru ringkasProfil
	bacaBody(t, resp, &baru)
	if baru.Nama != "Normal (salinan)" || baru.JumlahBel != 2 {
		t.Fatalf("duplikat = %+v", baru)
	}
	if l.penjadwal.jumlahReload() == reloadSebelum {
		t.Fatal("duplikat harus memuat ulang penjadwal")
	}
	resp = kirim(t, c, "POST", l.srv.URL+"/api/profil/"+itoa(profil)+"/duplikat", map[string]string{"nama": "Normal"})
	harapKode(t, resp, 409)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/profil/9999/duplikat", map[string]string{"nama": "X"})
	harapKode(t, resp, 404)
	resp.Body.Close()
}

func TestEksporPeriksaImpor(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil := profilBerisi(t, l)

	resp := kirim(t, c, "GET", l.srv.URL+"/api/profil/"+itoa(profil)+"/ekspor", nil)
	harapKode(t, resp, 200)
	if cd := resp.Header.Get("Content-Disposition"); cd != `attachment; filename="jadwal-normal.json"` {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	file, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	resp = kirimMentah(t, c, "POST", l.srv.URL+"/api/impor/periksa", file)
	harapKode(t, resp, 200)
	var periksa struct {
		Nama               string   `json:"nama"`
		JumlahBel          int      `json:"jumlahBel"`
		JenisBelTakDikenal []string `json:"jenisBelTakDikenal"`
		NamaTersedia       string   `json:"namaTersedia"`
	}
	bacaBody(t, resp, &periksa)
	if periksa.Nama != "Normal" || periksa.JumlahBel != 2 || len(periksa.JenisBelTakDikenal) != 0 || periksa.NamaTersedia != "Normal (impor)" {
		t.Fatalf("periksa = %+v", periksa)
	}

	resp = kirim(t, c, "POST", l.srv.URL+"/api/impor", map[string]any{
		"data": json.RawMessage(file), "nama": periksa.NamaTersedia, "pemetaan": map[string]any{},
	})
	harapKode(t, resp, 201)
	var hasil struct {
		ProfilID int64 `json:"profilId"`
	}
	bacaBody(t, resp, &hasil)
	js, _ := l.st.ListJadwal(ctx, hasil.ProfilID)
	if len(js) != 2 {
		t.Fatalf("hasil impor %d bel", len(js))
	}

	resp = kirimMentah(t, c, "POST", l.srv.URL+"/api/impor/periksa", []byte(`{"format":"lain"}`))
	harapKode(t, resp, 400)
	if m := pesanError(t, resp); !strings.Contains(m, "bukan file jadwal") {
		t.Fatalf("pesan = %q", m)
	}
}

func TestImpor_PemetaanKurangDanBaru(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	file := []byte(`{"format":"belsekolah-jadwal","versi":1,"nama":"Luar","jenisBel":[{"nama":"Bel Literasi","warna":"#123456"}],"jadwal":[{"hari":2,"jam":"07:15","jenisBel":"Bel Literasi","label":"Literasi"}]}`)
	resp := kirimMentah(t, c, "POST", l.srv.URL+"/api/impor/periksa", file)
	harapKode(t, resp, 200)
	var periksa struct {
		JenisBelTakDikenal []string `json:"jenisBelTakDikenal"`
	}
	bacaBody(t, resp, &periksa)
	if len(periksa.JenisBelTakDikenal) != 1 || periksa.JenisBelTakDikenal[0] != "Bel Literasi" {
		t.Fatalf("tak dikenal = %v", periksa.JenisBelTakDikenal)
	}
	resp = kirim(t, c, "POST", l.srv.URL+"/api/impor", map[string]any{"data": json.RawMessage(file), "nama": "Luar", "pemetaan": map[string]any{}})
	harapKode(t, resp, 400)
	if m := pesanError(t, resp); !strings.Contains(m, "Bel Literasi") {
		t.Fatalf("pesan = %q", m)
	}
	resp = kirim(t, c, "POST", l.srv.URL+"/api/impor", map[string]any{
		"data": json.RawMessage(file), "nama": "Luar",
		"pemetaan": map[string]any{"Bel Literasi": map[string]any{"baru": true}},
	})
	harapKode(t, resp, 201)
	resp.Body.Close()
}
