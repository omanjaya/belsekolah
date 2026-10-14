package api

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"belsekolah/internal/nada"
)

func TestProfilAPI(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	u := l.srv.URL + "/api/profil"

	var p profilDTO
	resp := kirim(t, c, "POST", u, map[string]string{"nama": "Ujian"})
	harapKode(t, resp, 201)
	bacaBody(t, resp, &p)
	if p.ID == 0 || p.Nama != "Ujian" {
		t.Fatalf("profil = %+v", p)
	}
	resp = kirim(t, c, "POST", u, map[string]string{"nama": "Ujian"})
	harapKode(t, resp, 409)
	resp.Body.Close()
	resp = kirim(t, c, "POST", u, map[string]string{"nama": ""})
	harapKode(t, resp, 400)
	resp.Body.Close()

	resp = kirim(t, c, "PUT", fmt.Sprintf("%s/%d", u, p.ID), map[string]string{"nama": "Ujian Akhir"})
	harapKode(t, resp, 200)
	bacaBody(t, resp, &p)
	if p.Nama != "Ujian Akhir" {
		t.Errorf("nama = %q", p.Nama)
	}
	var daftar []profilDTO
	resp = kirim(t, c, "GET", u, nil)
	bacaBody(t, resp, &daftar)
	if len(daftar) != 2 {
		t.Fatalf("daftar = %+v", daftar)
	}
	resp = kirim(t, c, "DELETE", fmt.Sprintf("%s/%d", u, daftar[0].ID), nil)
	harapKode(t, resp, 409) // profil default
	resp.Body.Close()
	harapKode(t, kirim(t, c, "DELETE", fmt.Sprintf("%s/%d", u, p.ID), nil), 204)
	harapKode(t, kirim(t, c, "DELETE", fmt.Sprintf("%s/%d", u, p.ID), nil), 404)
	harapKode(t, kirim(t, c, "DELETE", u+"/abc", nil), 404)
	if l.penjadwal.jumlahReload() < 2 {
		t.Errorf("reload = %d, setiap perubahan harus memicu reload", l.penjadwal.jumlahReload())
	}
}

func TestJadwalAPI(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil, _ := l.st.ProfilDefault(ctx)
	jenis, _ := l.st.ListJenisBel(ctx)
	u := l.srv.URL + "/api/jadwal"

	harapKode(t, kirim(t, c, "GET", u, nil), 400)

	var j jadwalDTO
	resp := kirim(t, c, "POST", u, jadwalDTO{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis[0].ID, Label: "Jam ke-1"})
	harapKode(t, resp, 201)
	bacaBody(t, resp, &j)
	resp = kirim(t, c, "POST", u, jadwalDTO{ProfilID: profil, Hari: 1, Jam: "7:00", JenisBelID: jenis[0].ID})
	harapKode(t, resp, 400)
	resp.Body.Close()

	j.Jam = "07:15"
	resp = kirim(t, c, "PUT", fmt.Sprintf("%s/%d", u, j.ID), j)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &j)
	if j.Jam != "07:15" {
		t.Errorf("jam = %q", j.Jam)
	}

	resp = kirim(t, c, "POST", u+"/salin", map[string]any{"profilId": profil, "dariHari": 1, "keHari": 2})
	harapKode(t, resp, 200)
	resp.Body.Close()
	var daftar []jadwalDTO
	resp = kirim(t, c, "GET", fmt.Sprintf("%s?profil=%d", u, profil), nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &daftar)
	if len(daftar) != 2 || daftar[1].Hari != 2 || daftar[1].Jam != "07:15" {
		t.Fatalf("jadwal = %+v", daftar)
	}
	harapKode(t, kirim(t, c, "DELETE", fmt.Sprintf("%s/%d", u, j.ID), nil), 204)

	resp = kirim(t, c, "GET", u+"?profil=999", nil)
	harapKode(t, resp, 200)
	var kosong []jadwalDTO
	bacaBody(t, resp, &kosong)
	if kosong == nil {
		t.Error("list kosong harus [] bukan null")
	}
}

func TestPeriodeDanLiburAPI(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	profil, _ := l.st.ProfilDefault(ctx)

	var p periodeDTO
	resp := kirim(t, c, "POST", l.srv.URL+"/api/periode", periodeDTO{Mulai: "2026-10-19", Selesai: "2026-10-23", ProfilID: profil, Keterangan: "PTS"})
	harapKode(t, resp, 201)
	bacaBody(t, resp, &p)
	resp = kirim(t, c, "POST", l.srv.URL+"/api/periode", periodeDTO{Mulai: "2026-10-20", Selesai: "2026-10-21", ProfilID: profil})
	harapKode(t, resp, 409)
	resp.Body.Close()
	p.Keterangan = "PTS Ganjil"
	resp = kirim(t, c, "PUT", fmt.Sprintf("%s/api/periode/%d", l.srv.URL, p.ID), p)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &p)
	if p.Keterangan != "PTS Ganjil" {
		t.Errorf("periode = %+v", p)
	}
	var daftarP []periodeDTO
	resp = kirim(t, c, "GET", l.srv.URL+"/api/periode", nil)
	bacaBody(t, resp, &daftarP)
	if len(daftarP) != 1 {
		t.Errorf("periode = %+v", daftarP)
	}
	harapKode(t, kirim(t, c, "DELETE", fmt.Sprintf("%s/api/periode/%d", l.srv.URL, p.ID), nil), 204)

	var lb liburDTO
	resp = kirim(t, c, "POST", l.srv.URL+"/api/libur", liburDTO{Mulai: "2026-12-25", Selesai: "2026-12-25", Keterangan: "Natal"})
	harapKode(t, resp, 201)
	bacaBody(t, resp, &lb)
	lb.Selesai = "2026-12-24"
	resp = kirim(t, c, "PUT", fmt.Sprintf("%s/api/libur/%d", l.srv.URL, lb.ID), lb)
	harapKode(t, resp, 400)
	resp.Body.Close()
	var daftarL []liburDTO
	resp = kirim(t, c, "GET", l.srv.URL+"/api/libur", nil)
	bacaBody(t, resp, &daftarL)
	if len(daftarL) != 1 || daftarL[0].Keterangan != "Natal" {
		t.Errorf("libur = %+v", daftarL)
	}
	harapKode(t, kirim(t, c, "DELETE", fmt.Sprintf("%s/api/libur/%d", l.srv.URL, lb.ID), nil), 204)
}

func TestJenisBelAPI(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	audio, _ := l.st.ListAudio(ctx)
	u := l.srv.URL + "/api/jenis-bel"

	var daftar []jenisBelDTO
	resp := kirim(t, c, "GET", u, nil)
	bacaBody(t, resp, &daftar)
	if len(daftar) != 5 || len(daftar[0].AudioIDs) == 0 {
		t.Fatalf("jenis bel = %+v", daftar)
	}

	var j jenisBelDTO
	resp = kirim(t, c, "POST", u, jenisBelDTO{Nama: "Upacara", Warna: "#f59e0b", AudioIDs: []int64{audio[0].ID, audio[1].ID}})
	harapKode(t, resp, 201)
	bacaBody(t, resp, &j)
	if j.ID == 0 || len(j.AudioIDs) != 2 {
		t.Fatalf("jenis baru = %+v", j)
	}
	resp = kirim(t, c, "POST", u, jenisBelDTO{Nama: "Kosong", Warna: "#000000", AudioIDs: []int64{999}})
	harapKode(t, resp, 400)
	resp.Body.Close()
	resp = kirim(t, c, "GET", u, nil)
	bacaBody(t, resp, &daftar)
	if len(daftar) != 6 {
		t.Errorf("jenis bel gagal dibuat tidak boleh tersisa: %d", len(daftar))
	}

	j.AudioIDs = []int64{audio[2].ID}
	j.Nama = "Upacara Bendera"
	resp = kirim(t, c, "PUT", fmt.Sprintf("%s/%d", u, j.ID), j)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &j)
	if j.Nama != "Upacara Bendera" || len(j.AudioIDs) != 1 || j.AudioIDs[0] != audio[2].ID {
		t.Fatalf("jenis diubah = %+v", j)
	}

	l.penjadwal.files = map[int64][]string{j.ID: {"/x/a.wav"}}
	harapKode(t, kirim(t, c, "POST", fmt.Sprintf("%s/%d/tes", u, j.ID), nil), 202)
	if n := len(l.pemutar.diputar); n != 1 || l.pemutar.diputar[0].Files[0] != "/x/a.wav" {
		t.Fatalf("tes jenis bel diputar = %+v", l.pemutar.diputar)
	}

	profil, _ := l.st.ProfilDefault(ctx)
	resp = kirim(t, c, "POST", l.srv.URL+"/api/jadwal", jadwalDTO{ProfilID: profil, Hari: 3, Jam: "08:00", JenisBelID: j.ID})
	harapKode(t, resp, 201)
	resp.Body.Close()
	resp = kirim(t, c, "DELETE", fmt.Sprintf("%s/%d", u, j.ID), nil)
	harapKode(t, resp, 409)
	resp.Body.Close()
}

func unggah(t *testing.T, c *http.Client, url, nama, namaFile string, isi []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if nama != "" {
		mw.WriteField("nama", nama)
	}
	fw, _ := mw.CreateFormFile("file", namaFile)
	fw.Write(isi)
	mw.Close()
	req, _ := http.NewRequest("POST", url, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestAudioAPI(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	u := l.srv.URL + "/api/audio"

	var a audioDTO
	resp := unggah(t, c, u, "Lagu Sekolah", "lagu.WAV", nada.DingDong())
	harapKode(t, resp, 201)
	bacaBody(t, resp, &a)
	if a.Nama != "Lagu Sekolah" || a.Sumber != "upload" || a.DurasiMS != 1500 {
		t.Fatalf("audio = %+v", a)
	}
	tersimpan, _ := l.st.GetAudio(ctx, a.ID)
	if filepath.Base(tersimpan.File) != tersimpan.File || filepath.Ext(tersimpan.File) != ".wav" || tersimpan.File == "lagu.WAV" {
		t.Errorf("nama file tersimpan = %q, harus dibuat server", tersimpan.File)
	}

	resp = unggah(t, c, u, "", "tanpa-nama.wav", nada.DingDong())
	harapKode(t, resp, 201)
	var b audioDTO
	bacaBody(t, resp, &b)
	if b.Nama != "tanpa-nama" {
		t.Errorf("nama default = %q", b.Nama)
	}

	resp = unggah(t, c, u, "Palsu", "palsu.mp3", []byte("bukan audio"))
	harapKode(t, resp, 400)
	resp.Body.Close()
	resp = unggah(t, c, u, "Exe", "virus.exe", nada.DingDong())
	harapKode(t, resp, 400)
	resp.Body.Close()
	ents, _ := os.ReadDir(l.audioDir)
	for _, e := range ents {
		if e.Name()[0] == '.' {
			t.Errorf("file sementara tertinggal: %s", e.Name())
		}
	}

	resp = kirim(t, c, "GET", fmt.Sprintf("%s/%d/file", u, a.ID), nil)
	harapKode(t, resp, 200)
	isi, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !bytes.Equal(isi, nada.DingDong()) {
		t.Error("isi file yang diunduh berbeda")
	}

	harapKode(t, kirim(t, c, "POST", fmt.Sprintf("%s/%d/tes", u, a.ID), nil), 202)
	if len(l.pemutar.diputar) != 1 || l.pemutar.diputar[0].Files[0] != filepath.Join(l.audioDir, tersimpan.File) {
		t.Fatalf("tes audio = %+v", l.pemutar.diputar)
	}

	resp = kirim(t, c, "PUT", fmt.Sprintf("%s/%d", u, a.ID), map[string]string{"nama": "Mars Sekolah"})
	harapKode(t, resp, 200)
	bacaBody(t, resp, &a)
	if a.Nama != "Mars Sekolah" {
		t.Errorf("nama = %q", a.Nama)
	}

	var daftar []audioDTO
	resp = kirim(t, c, "GET", u, nil)
	bacaBody(t, resp, &daftar)
	if len(daftar) != 5 {
		t.Fatalf("audio = %d, mau 5 (3 bawaan + 2 upload)", len(daftar))
	}
	resp = kirim(t, c, "DELETE", fmt.Sprintf("%s/%d", u, daftar[0].ID), nil)
	harapKode(t, resp, 409) // bawaan
	resp.Body.Close()
	harapKode(t, kirim(t, c, "DELETE", fmt.Sprintf("%s/%d", u, a.ID), nil), 204)
	if _, err := os.Stat(filepath.Join(l.audioDir, tersimpan.File)); !os.IsNotExist(err) {
		t.Error("file audio harus ikut terhapus")
	}
}

func TestRiwayatAPI(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)
	l.st.InsertRiwayat(ctx, storeRiwayat(l, "berhasil"))
	l.st.InsertRiwayat(ctx, storeRiwayat(l, "gagal"))
	var daftar []riwayatDTO
	resp := kirim(t, c, "GET", l.srv.URL+"/api/riwayat?tanggal=2026-10-12&status=gagal&batas=10", nil)
	harapKode(t, resp, 200)
	bacaBody(t, resp, &daftar)
	if len(daftar) != 1 || daftar[0].Status != "gagal" {
		t.Fatalf("riwayat = %+v", daftar)
	}
	harapKode(t, kirim(t, c, "GET", l.srv.URL+"/api/riwayat?tanggal=kemarin", nil), 400)
	harapKode(t, kirim(t, c, "GET", l.srv.URL+"/api/riwayat?batas=abc", nil), 400)
}

func TestBackupDanRestoreAPI(t *testing.T) {
	l := siapkan(t)
	c := l.masuk(t)

	resp := kirim(t, c, "GET", l.srv.URL+"/api/backup", nil)
	harapKode(t, resp, 200)
	if ct := resp.Header.Get("Content-Type"); ct != "application/zip" {
		t.Errorf("content-type = %q", ct)
	}
	if cd := resp.Header.Get("Content-Disposition"); !berisi(cd, "bel-sekolah-2026-10-12.zip") {
		t.Errorf("content-disposition = %q", cd)
	}
	isi, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	zr, err := zip.NewReader(bytes.NewReader(isi), int64(len(isi)))
	if err != nil {
		t.Fatal(err)
	}
	adaDB := false
	for _, f := range zr.File {
		if f.Name == "bel.db" {
			adaDB = true
		}
	}
	if !adaDB {
		t.Fatal("zip backup tidak berisi bel.db")
	}

	resp = unggah(t, c, l.srv.URL+"/api/restore", "", "backup.zip", []byte("bukan zip"))
	harapKode(t, resp, 400)
	resp.Body.Close()
	if l.mulaiUlang.Load() != 0 {
		t.Fatal("restore gagal tidak boleh memulai ulang")
	}

	resp = unggah(t, c, l.srv.URL+"/api/restore", "", "backup.zip", isi)
	harapKode(t, resp, 202)
	resp.Body.Close()
	if _, err := os.Stat(filepath.Join(l.dataDir, "restore-tertunda", "bel.db")); err != nil {
		t.Fatalf("restore tertunda tidak disiapkan: %v", err)
	}
	tungguSampai(t, func() bool { return l.mulaiUlang.Load() == 1 })
}
