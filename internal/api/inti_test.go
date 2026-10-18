package api

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"belsekolah/internal/schedule"
	"belsekolah/internal/scheduler"
	"belsekolah/internal/store"
)

func TestSesiDanLogin(t *testing.T) {
	l := siapkan(t)
	c := l.klien()

	var sesi struct {
		Masuk              bool `json:"masuk"`
		WajibGantiPassword bool `json:"wajibGantiPassword"`
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/sesi", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &sesi)
	if sesi.Masuk {
		t.Fatal("belum login tapi masuk=true")
	}

	harapKode(t, kirim(t, c, "GET", l.srv.URL+"/api/status", nil), 401)

	resp = kirim(t, c, "POST", l.srv.URL+"/api/login", map[string]string{"password": "salah"})
	harapKode(t, resp, 401)
	if p := pesanError(t, resp); p != "password salah" {
		t.Errorf("pesan = %q", p)
	}

	resp = kirim(t, c, "POST", l.srv.URL+"/api/login", map[string]string{"password": l.password})
	harapKode(t, resp, 200)
	resp.Body.Close()
	cookie := resp.Header.Get("Set-Cookie")
	if !berisi(cookie, "sesi=") || !berisi(cookie, "HttpOnly") || !berisi(cookie, "SameSite=Strict") {
		t.Errorf("cookie = %q", cookie)
	}

	resp = kirim(t, c, "GET", l.srv.URL+"/api/status", nil)
	harapKode(t, resp, 403)
	if p := pesanError(t, resp); !berisi(p, "ganti password") {
		t.Errorf("pesan 403 = %q", p)
	}

	resp = kirim(t, c, "POST", l.srv.URL+"/api/password", map[string]string{"lama": l.password, "baru": "pendek"})
	harapKode(t, resp, 400)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/password", map[string]string{"lama": "keliru", "baru": "rahasia-baru-1"})
	harapKode(t, resp, 400)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/password", map[string]string{"lama": l.password, "baru": "rahasia-baru-1"})
	harapKode(t, resp, 200)
	resp.Body.Close()
	if _, err := os.Stat(filepath.Join(l.dataDir, "password-awal.txt")); !os.IsNotExist(err) {
		t.Error("password-awal.txt harus dihapus setelah password diganti")
	}

	harapKode(t, kirim(t, c, "GET", l.srv.URL+"/api/status", nil), 200)

	resp = kirim(t, c, "GET", l.srv.URL+"/api/sesi", nil)
	bacaBody(t, resp, &sesi)
	if !sesi.Masuk || sesi.WajibGantiPassword {
		t.Errorf("sesi = %+v", sesi)
	}

	harapKode(t, kirim(t, c, "POST", l.srv.URL+"/api/logout", nil), 200)
	harapKode(t, kirim(t, c, "GET", l.srv.URL+"/api/status", nil), 401)
}

func TestLoginTerkunci(t *testing.T) {
	l := siapkan(t)
	c := l.klien()
	for i := 0; i < 4; i++ {
		resp := kirim(t, c, "POST", l.srv.URL+"/api/login", map[string]string{"password": "salah"})
		harapKode(t, resp, 401)
		resp.Body.Close()
	}
	resp := kirim(t, c, "POST", l.srv.URL+"/api/login", map[string]string{"password": "salah"})
	harapKode(t, resp, 429)
	resp.Body.Close()
}

func TestBodyTidakValid(t *testing.T) {
	l := siapkan(t)
	c := l.klien()
	resp := kirim(t, c, "POST", l.srv.URL+"/api/login", map[string]any{"password": "x", "lain": 1})
	harapKode(t, resp, 400)
	resp.Body.Close()
}

func TestOriginAsingDitolak(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	req, _ := http.NewRequest("POST", l.srv.URL+"/api/bel/stop", nil)
	req.Header.Set("Origin", "http://situs-jahat.example")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	harapKode(t, resp, 403)
	resp.Body.Close()

	req, _ = http.NewRequest("POST", l.srv.URL+"/api/bel/stop", nil)
	req.Header.Set("Origin", l.srv.URL)
	resp, _ = c.Do(req)
	harapKode(t, resp, 200)
	resp.Body.Close()
}

func TestStatusBeranda(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	jenis, _ := l.st.ListJenisBel(ctx)
	masuk := jenis[0]
	l.penjadwal.status = scheduler.Status{
		Siap: true, Tanggal: "2026-10-12", ProfilID: 1,
		Bel: []scheduler.StatusBel{
			{JadwalID: 1, Waktu: l.now.Add(-time.Hour), JenisBelID: masuk.ID, Label: "Jam ke-1", Sudah: true},
			{JadwalID: 2, Waktu: l.now.Add(30 * time.Minute), JenisBelID: masuk.ID, Label: "Jam ke-2"},
			{JadwalID: 3, Waktu: l.now.Add(2 * time.Hour), JenisBelID: 999, Label: "Tanpa jenis"},
		},
	}
	l.st.InsertRiwayat(ctx, store.Riwayat{Waktu: l.now.Add(-time.Hour), JadwalID: 1, Label: "Jam ke-1", Sumber: store.SumberJadwal, Status: "gagal", Pesan: "mpv rusak"})
	salah := false
	l.ntp = &salah

	var st struct {
		Tanggal string `json:"tanggal"`
		Hari    int    `json:"hari"`
		Profil  *struct {
			Nama string `json:"nama"`
		} `json:"profil"`
		Libur *struct{} `json:"libur"`
		Bel   []struct {
			JadwalID int64  `json:"jadwalId"`
			Jam      string `json:"jam"`
			Sudah    bool   `json:"sudah"`
			JenisBel *struct {
				Nama  string `json:"nama"`
				Warna string `json:"warna"`
			} `json:"jenisBel"`
		} `json:"bel"`
		Berikutnya *struct {
			JadwalID int64 `json:"jadwalId"`
		} `json:"berikutnya"`
		Peringatan []string `json:"peringatan"`
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/status", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &st)
	if st.Tanggal != "2026-10-12" || st.Hari != 1 || st.Profil == nil || st.Profil.Nama != "Normal" || st.Libur != nil {
		t.Fatalf("status = %+v", st)
	}
	if len(st.Bel) != 3 || st.Bel[0].Jam != l.now.Add(-time.Hour).Format("15:04") || !st.Bel[0].Sudah ||
		st.Bel[0].JenisBel == nil || st.Bel[0].JenisBel.Nama != masuk.Nama || st.Bel[2].JenisBel != nil {
		t.Fatalf("bel = %+v", st.Bel)
	}
	if st.Berikutnya == nil || st.Berikutnya.JadwalID != 2 {
		t.Fatalf("berikutnya = %+v", st.Berikutnya)
	}
	if len(st.Peringatan) != 2 || !berisi(st.Peringatan[0], "gagal") || !berisi(st.Peringatan[1], "tersinkron") {
		t.Fatalf("peringatan = %v", st.Peringatan)
	}
}

func TestStatusBelumSiapDanLibur(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	var st struct {
		Tanggal    string   `json:"tanggal"`
		Peringatan []string `json:"peringatan"`
		Bel        []any    `json:"bel"`
		Libur      *struct {
			Keterangan string `json:"keterangan"`
		} `json:"libur"`
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/status", nil)
	bacaBody(t, resp, &st)
	if st.Tanggal != "2026-10-12" || len(st.Peringatan) != 1 || st.Bel == nil {
		t.Fatalf("status belum siap = %+v (bel harus [] bukan null)", st)
	}

	l.penjadwal.status = scheduler.Status{Siap: true, Tanggal: "2026-10-12",
		Libur: &schedule.Libur{ID: 1, Mulai: "2026-10-12", Selesai: "2026-10-12", Keterangan: "Libur nasional"}}
	resp = kirim(t, c, "GET", l.srv.URL+"/api/status", nil)
	bacaBody(t, resp, &st)
	if st.Libur == nil || st.Libur.Keterangan != "Libur nasional" {
		t.Fatalf("libur = %+v", st.Libur)
	}
}

func TestBelManualDaruratStop(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	jenis, _ := l.st.ListJenisBel(ctx)

	resp := kirim(t, c, "POST", l.srv.URL+"/api/bel/manual", map[string]int64{"jenisBelId": jenis[0].ID})
	harapKode(t, resp, 202)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/bel/darurat", map[string]int64{"jenisBelId": jenis[4].ID})
	harapKode(t, resp, 202)
	resp.Body.Close()
	resp = kirim(t, c, "POST", l.srv.URL+"/api/bel/manual", map[string]int64{"jenisBelId": 999})
	harapKode(t, resp, 404)
	resp.Body.Close()
	harapKode(t, kirim(t, c, "POST", l.srv.URL+"/api/bel/stop", nil), 200)

	m := l.penjadwal.manual
	if len(m) != 2 || m[0].Darurat || m[0].Label != "Manual: "+jenis[0].Nama || !m[1].Darurat || m[1].Label != "Darurat: "+jenis[4].Nama {
		t.Fatalf("panggilan manual = %+v", m)
	}
	if l.pemutar.stop != 1 {
		t.Errorf("stop = %d", l.pemutar.stop)
	}
}

func TestPengaturan(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	var p struct {
		Volume          int   `json:"volume"`
		ProfilDefaultID int64 `json:"profilDefaultId"`
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/pengaturan", nil)
	bacaBody(t, resp, &p)
	if p.Volume != 100 || p.ProfilDefaultID == 0 {
		t.Fatalf("pengaturan = %+v", p)
	}
	ujian, _ := l.st.CreateProfil(ctx, "Ujian")

	resp = kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", map[string]any{"volume": 150, "profilDefaultId": ujian})
	harapKode(t, resp, 400)
	resp.Body.Close()
	resp = kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", map[string]any{"volume": 70, "profilDefaultId": 999})
	harapKode(t, resp, 404)
	resp.Body.Close()

	sebelum := l.penjadwal.jumlahReload()
	resp = kirim(t, c, "PUT", l.srv.URL+"/api/pengaturan", map[string]any{"volume": 70, "profilDefaultId": ujian})
	harapKode(t, resp, 200)
	bacaBody(t, resp, &p)
	if p.Volume != 70 || p.ProfilDefaultID != ujian {
		t.Fatalf("pengaturan baru = %+v", p)
	}
	if l.pemutar.volume != 70 || l.penjadwal.jumlahReload() != sebelum+1 {
		t.Errorf("volume pemutar = %d, reload = %d", l.pemutar.volume, l.penjadwal.jumlahReload())
	}
	if v, _ := l.st.Volume(ctx); v != 70 {
		t.Errorf("volume tersimpan = %d", v)
	}
}

func TestSistem(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	benar := true
	l.ntp = &benar
	var s struct {
		Versi     string   `json:"versi"`
		ZonaWaktu string   `json:"zonaWaktu"`
		IP        []string `json:"ip"`
		NTP       *bool    `json:"ntp"`
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/api/sistem", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &s)
	if s.Versi != "uji" || s.ZonaWaktu == "" || s.IP == nil || s.NTP == nil || !*s.NTP {
		t.Fatalf("sistem = %+v", s)
	}
}

func TestUIDanAPITakDikenal(t *testing.T) {
	l := siapkan(t)
	c := l.klien()
	for _, p := range []string{"/", "/jadwal", "/pengaturan/password"} {
		resp := kirim(t, c, "GET", l.srv.URL+p, nil)
		harapKode(t, resp, 200)
		resp.Body.Close()
		if ct := resp.Header.Get("Content-Type"); !berisi(ct, "text/html") {
			t.Errorf("%s content-type = %q", p, ct)
		}
	}
	resp := kirim(t, c, "GET", l.srv.URL+"/app.js", nil)
	harapKode(t, resp, 200)
	resp.Body.Close()

	resp = kirim(t, c, "GET", l.srv.URL+"/api/tidak-ada", nil)
	harapKode(t, resp, 404)
	pesanError(t, resp)
}

func TestTesStop(t *testing.T) {
	l := siapkan(t)
	harapKode(t, kirim(t, http.DefaultClient, "POST", l.srv.URL+"/api/tes/stop", nil), 401)
	c := l.masuk(t)
	harapKode(t, kirim(t, c, "POST", l.srv.URL+"/api/tes/stop", nil), 200)
	l.pemutar.mu.Lock()
	defer l.pemutar.mu.Unlock()
	if l.pemutar.stopTes != 1 || l.pemutar.stop != 0 {
		t.Fatalf("stopTes = %d, stop = %d; mau 1 dan 0", l.pemutar.stopTes, l.pemutar.stop)
	}
}
