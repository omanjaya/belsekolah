package api

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

type ringkasTemplate struct {
	ID        int64  `json:"id"`
	Nama      string `json:"nama"`
	JumlahBel int    `json:"jumlahBel"`
	Hari      []int  `json:"hari"`
	Dibuat    string `json:"dibuat"`
}

func TestTemplate_DariProfilDanCRUD(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil := profilBerisi(t, l)

	resp := kirim(t, c, "POST", l.srv.URL+"/api/template", map[string]any{"nama": "Jadwal Sekolah", "profilId": profil})
	harapKode(t, resp, 201)
	var tp ringkasTemplate
	bacaBody(t, resp, &tp)
	if tp.Nama != "Jadwal Sekolah" || tp.JumlahBel != 2 || len(tp.Hari) != 1 || tp.Hari[0] != 1 || tp.Dibuat == "" {
		t.Fatalf("template = %+v", tp)
	}
	resp = kirim(t, c, "POST", l.srv.URL+"/api/template", map[string]any{"nama": "Jadwal Sekolah", "profilId": profil})
	harapKode(t, resp, 409)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/template", map[string]any{"nama": "Kosong"})
	harapKode(t, resp, 400)
	resp.Body.Close()

	resp = kirim(t, c, "PUT", l.srv.URL+"/api/template/"+itoa(tp.ID), map[string]string{"nama": "Ganti"})
	harapKode(t, resp, 200)
	resp.Body.Close()

	resp = kirim(t, c, "GET", l.srv.URL+"/api/template/"+itoa(tp.ID), nil)
	harapKode(t, resp, 200)
	var detail struct {
		ringkasTemplate
		Data struct {
			Nama   string            `json:"nama"`
			Jadwal []json.RawMessage `json:"jadwal"`
		} `json:"data"`
	}
	bacaBody(t, resp, &detail)
	if detail.Nama != "Ganti" || detail.Data.Nama != "Ganti" || len(detail.Data.Jadwal) != 2 {
		t.Fatalf("detail = %+v", detail)
	}

	resp = kirim(t, c, "GET", l.srv.URL+"/api/template/"+itoa(tp.ID)+"/ekspor", nil)
	harapKode(t, resp, 200)
	if cd := resp.Header.Get("Content-Disposition"); cd != `attachment; filename="jadwal-ganti.json"` {
		t.Fatalf("Content-Disposition = %q", cd)
	}
	file, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	resp = kirim(t, c, "POST", l.srv.URL+"/api/template", map[string]any{"nama": "Dari File", "data": json.RawMessage(file)})
	harapKode(t, resp, 201)
	resp.Body.Close()

	var daftar []ringkasTemplate
	resp = kirim(t, c, "GET", l.srv.URL+"/api/template", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &daftar)
	if len(daftar) != 2 {
		t.Fatalf("daftar = %+v", daftar)
	}
	resp = kirim(t, c, "DELETE", l.srv.URL+"/api/template/"+itoa(tp.ID), nil)
	harapKode(t, resp, 204)
	resp.Body.Close()
	resp = kirim(t, c, "GET", l.srv.URL+"/api/template/"+itoa(tp.ID), nil)
	harapKode(t, resp, 404)
	resp.Body.Close()
}

func TestTemplate_Buat(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	param := map[string]any{
		"jenjang": "SD", "varian": "normal", "hariSekolah": 5, "jamMasuk": "07:00",
		"lamaJP": 35, "jumlahJP": 6, "jumlahJPJumat": 4, "jumlahJPSabtu": 4,
		"istirahat":     []map[string]int{{"setelahJP": 3, "menit": 15}},
		"upacaraSenin":  map[string]any{"aktif": true, "menit": 40},
		"kegiatanJumat": map[string]any{"aktif": false, "menit": 30, "label": ""},
		"ujian":         map[string]any{"jamMulai": "07:30", "jumlahSesi": 2, "lamaSesi": 120, "jeda": 30},
	}
	sebelum, _ := l.st.CountProfil(ctx)
	resp := kirim(t, c, "POST", l.srv.URL+"/api/template/buat", param)
	harapKode(t, resp, 200)
	var d struct {
		Format string            `json:"format"`
		Nama   string            `json:"nama"`
		Jadwal []json.RawMessage `json:"jadwal"`
	}
	bacaBody(t, resp, &d)
	if d.Format != "belsekolah-jadwal" || d.Nama != "SD Normal 5 hari" || len(d.Jadwal) == 0 {
		t.Fatalf("hasil = %+v", d)
	}
	if n, _ := l.st.CountProfil(ctx); n != sebelum {
		t.Fatal("template/buat tidak boleh menyimpan apa pun")
	}
	param["lamaJP"] = 5
	resp = kirim(t, c, "POST", l.srv.URL+"/api/template/buat", param)
	harapKode(t, resp, 400)
	if m := pesanError(t, resp); !strings.Contains(m, "lama jam pelajaran") {
		t.Fatalf("pesan = %q", m)
	}
}
