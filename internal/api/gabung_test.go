package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGabung_PeriksaDanGabung(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil := profilBerisi(t, l)
	js, _ := l.st.ListJenisBel(ctx)
	nama := js[0].Nama
	data := json.RawMessage(`{"format":"belsekolah-jadwal","versi":1,"nama":"Tambah banyak",
		"jenisBel":[{"nama":"` + nama + `","warna":"#2563eb"}],
		"jadwal":[{"hari":1,"jam":"07:00","jenisBel":"` + nama + `","label":"Upacara"},
		          {"hari":1,"jam":"12:00","jenisBel":"` + nama + `","label":"Uji"},
		          {"hari":2,"jam":"07:00","jenisBel":"` + nama + `","label":"Baru"}]}`)
	url := l.srv.URL + "/api/profil/" + itoa(profil) + "/gabung"

	resp := kirim(t, c, "POST", url+"/periksa", map[string]any{"data": data})
	harapKode(t, resp, 200)
	var p struct {
		JenisBelTakDikenal []string                            `json:"jenisBelTakDikenal"`
		Ringkasan          struct{ Baru, Kembar, Bentrok int } `json:"ringkasan"`
		Baris              []struct {
			Hari   int                               `json:"hari"`
			Jam    string                            `json:"jam"`
			Status string                            `json:"status"`
			Lama   *struct{ JenisBel, Label string } `json:"lama"`
		} `json:"baris"`
	}
	bacaBody(t, resp, &p)
	if p.Ringkasan.Baru != 1 || p.Ringkasan.Kembar != 1 || p.Ringkasan.Bentrok != 1 || p.JenisBelTakDikenal == nil {
		t.Fatalf("periksa = %+v", p)
	}
	if p.Baris[0].Status != "bentrok" || p.Baris[0].Lama == nil || p.Baris[0].Lama.Label != "Uji" {
		t.Fatalf("baris[0] = %+v", p.Baris[0])
	}

	reload := l.penjadwal.jumlahReload()
	resp = kirim(t, c, "POST", url, map[string]any{"data": data, "timpa": []map[string]any{{"hari": 1, "jam": "07:00"}}})
	harapKode(t, resp, 200)
	var h struct{ Ditambah, Ditimpa, Dilewati int }
	bacaBody(t, resp, &h)
	if h.Ditambah != 1 || h.Ditimpa != 1 || h.Dilewati != 1 {
		t.Fatalf("gabung = %+v", h)
	}
	if l.penjadwal.jumlahReload() == reload {
		t.Fatal("gabung harus memuat ulang penjadwal")
	}
	jadwal, _ := l.st.ListJadwal(ctx, profil)
	if len(jadwal) != 3 {
		t.Fatalf("jadwal = %d bel", len(jadwal))
	}
}

func TestGabung_Galat(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil := profilBerisi(t, l)
	url := l.srv.URL + "/api/profil/" + itoa(profil) + "/gabung"
	asing := json.RawMessage(`{"format":"belsekolah-jadwal","versi":1,"nama":"X",
		"jenisBel":[{"nama":"Bel Asing","warna":"#123456"}],
		"jadwal":[{"hari":1,"jam":"08:00","jenisBel":"Bel Asing","label":""}]}`)

	resp := kirim(t, c, "POST", url, map[string]any{"data": asing})
	harapKode(t, resp, 400)
	if m := pesanError(t, resp); !strings.Contains(m, "Bel Asing") {
		t.Fatalf("pesan = %q", m)
	}
	resp = kirim(t, c, "POST", url+"/periksa", map[string]any{"data": json.RawMessage(`{"format":"lain"}`)})
	harapKode(t, resp, 400)
	resp.Body.Close()
	resp = kirim(t, c, "POST", url+"/periksa", map[string]any{"data": asing, "lain": 1})
	harapKode(t, resp, 400)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/profil/9999/gabung/periksa", map[string]any{"data": asing})
	harapKode(t, resp, 404)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/profil/9999/gabung", map[string]any{"data": asing, "pemetaan": map[string]any{"Bel Asing": map[string]any{"baru": true}}})
	harapKode(t, resp, 404)
	resp.Body.Close()

	tamu := l.klien() // klien tanpa login
	resp = kirim(t, tamu, "POST", url+"/periksa", map[string]any{"data": asing})
	harapKode(t, resp, 401)
	resp.Body.Close()
}
