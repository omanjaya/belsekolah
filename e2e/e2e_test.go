//go:build e2e

// Package e2e menguji binary di container Docker lewat HTTP. Jalankan dengan scripts/e2e.sh.
package e2e

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var dasar = func() string {
	if u := os.Getenv("BEL_URL"); u != "" {
		return u
	}
	return "http://localhost:8080"
}()

type klien struct {
	t *testing.T
	c *http.Client
}

func baru(t *testing.T) *klien {
	jar, _ := cookiejar.New(nil)
	return &klien{t: t, c: &http.Client{Jar: jar, Timeout: 30 * time.Second}}
}

func (k *klien) do(metode, path string, body any, kode int, hasil any) {
	k.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(metode, dasar+path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := k.c.Do(req)
	if err != nil {
		k.t.Fatalf("%s %s: %v", metode, path, err)
	}
	defer resp.Body.Close()
	isi, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != kode {
		k.t.Fatalf("%s %s: kode %d, mau %d: %s", metode, path, resp.StatusCode, kode, isi)
	}
	if hasil != nil {
		if b, ok := hasil.(*[]byte); ok {
			*b = isi
			return
		}
		if err := json.Unmarshal(isi, hasil); err != nil {
			k.t.Fatalf("decode %s: %v", path, err)
		}
	}
}

func (k *klien) unggah(path, nama, namaFile string, isi []byte, kode int, hasil any) {
	k.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if nama != "" {
		mw.WriteField("nama", nama)
	}
	fw, _ := mw.CreateFormFile("file", namaFile)
	fw.Write(isi)
	mw.Close()
	req, _ := http.NewRequest("POST", dasar+path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := k.c.Do(req)
	if err != nil {
		k.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != kode {
		k.t.Fatalf("unggah %s: kode %d, mau %d: %s", path, resp.StatusCode, kode, b)
	}
	if hasil != nil {
		json.Unmarshal(b, hasil)
	}
}

func tungguSiap(t *testing.T) {
	t.Helper()
	batas := time.Now().Add(90 * time.Second)
	for time.Now().Before(batas) {
		resp, err := http.Get(dasar + "/api/sesi")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatal("aplikasi tidak siap dalam 90 detik")
}

func passwordAwal(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("docker", "compose", "exec", "-T", "belsekolah",
		"cat", "/var/lib/belsekolah/password-awal.txt").Output()
	if err != nil {
		t.Fatalf("baca password awal: %v", err)
	}
	return strings.TrimSpace(string(out))
}

type riwayat struct {
	Waktu    time.Time `json:"waktu"`
	JadwalID int64     `json:"jadwalId"`
	Label    string    `json:"label"`
	Sumber   string    `json:"sumber"`
	Status   string    `json:"status"`
	Pesan    string    `json:"pesan"`
}

func tungguRiwayat(t *testing.T, k *klien, cocok func(riwayat) bool, batas time.Duration) riwayat {
	t.Helper()
	akhir := time.Now().Add(batas)
	for time.Now().Before(akhir) {
		var rs []riwayat
		k.do("GET", "/api/riwayat?batas=50", nil, 200, &rs)
		for _, r := range rs {
			if cocok(r) {
				return r
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("riwayat yang dicari tidak muncul dalam %s", batas)
	return riwayat{}
}

func TestAlurLengkap(t *testing.T) {
	tungguSiap(t)
	k := baru(t)
	pwAwal := passwordAwal(t)
	pwBaru := "uji-e2e-rahasia"

	var login struct {
		WajibGantiPassword bool `json:"wajibGantiPassword"`
	}
	k.do("POST", "/api/login", map[string]string{"password": pwAwal}, 200, &login)
	if !login.WajibGantiPassword {
		t.Fatal("instalasi baru harus wajib ganti password")
	}
	k.do("GET", "/api/status", nil, 403, nil)
	k.do("POST", "/api/password", map[string]string{"lama": pwAwal, "baru": pwBaru}, 200, nil)

	var jenis []struct {
		ID   int64  `json:"id"`
		Nama string `json:"nama"`
	}
	k.do("GET", "/api/jenis-bel", nil, 200, &jenis)
	if len(jenis) != 5 {
		t.Fatalf("jenis bel = %d, mau 5", len(jenis))
	}
	idJenis := map[string]int64{}
	for _, j := range jenis {
		idJenis[j.Nama] = j.ID
	}

	t.Run("bel manual berbunyi lewat mpv", func(t *testing.T) {
		k.do("POST", "/api/bel/manual", map[string]int64{"jenisBelId": idJenis["Ganti Pelajaran"]}, 202, nil)
		r := tungguRiwayat(t, k, func(r riwayat) bool { return r.Sumber == "manual" }, 30*time.Second)
		if r.Status != "berhasil" {
			t.Fatalf("bel manual status %q: %s", r.Status, r.Pesan)
		}
	})

	t.Run("unggah audio", func(t *testing.T) {
		var wav []byte
		var daftar []struct {
			ID int64 `json:"id"`
		}
		k.do("GET", "/api/audio", nil, 200, &daftar)
		k.do("GET", fmt.Sprintf("/api/audio/%d/file", daftar[0].ID), nil, 200, &wav)
		var a struct {
			ID       int64 `json:"id"`
			DurasiMS int64 `json:"durasiMs"`
		}
		k.unggah("/api/audio", "Uji Unggah", "uji.wav", wav, 201, &a)
		if a.DurasiMS <= 0 {
			t.Fatalf("durasi = %d (ffprobe di container)", a.DurasiMS)
		}
		k.unggah("/api/audio", "Palsu", "palsu.mp3", []byte("bukan audio"), 400, nil)
	})

	t.Run("TTS asli lewat edge-tts", func(t *testing.T) {
		if os.Getenv("BEL_E2E_TTS") != "1" {
			t.Skip("set BEL_E2E_TTS=1 untuk menguji layanan TTS asli (butuh internet)")
		}
		var suara []struct {
			ID string `json:"id"`
		}
		k.do("GET", "/api/tts/suara", nil, 200, &suara)
		if len(suara) != 2 {
			t.Fatalf("suara = %+v", suara)
		}
		var pratinjau []byte
		k.do("POST", "/api/tts/pratinjau", map[string]string{"teks": "Tes suara", "suara": "id-ID-GadisNeural"}, 200, &pratinjau)
		if len(pratinjau) < 1000 {
			t.Fatalf("pratinjau terlalu kecil: %d byte", len(pratinjau))
		}
		var a struct {
			ID       int64  `json:"id"`
			Sumber   string `json:"sumber"`
			DurasiMS int64  `json:"durasiMs"`
		}
		k.do("POST", "/api/audio/tts", map[string]string{
			"nama": "Pengumuman Istirahat", "teks": "Perhatian, waktunya istirahat.", "suara": "id-ID-ArdiNeural",
		}, 201, &a)
		if a.Sumber != "tts" || a.DurasiMS < 500 {
			t.Fatalf("audio TTS = %+v", a)
		}
		k.do("PUT", fmt.Sprintf("/api/audio/%d/tts", a.ID), map[string]string{
			"teks": "Perhatian, waktunya istirahat. Jangan lupa makan.", "suara": "id-ID-GadisNeural",
		}, 200, nil)
		k.do("POST", fmt.Sprintf("/api/audio/%d/tes", a.ID), nil, 202, nil)
	})

	t.Run("bel terjadwal berbunyi tepat waktu", func(t *testing.T) {
		var p struct {
			ProfilDefaultID int64 `json:"profilDefaultId"`
		}
		k.do("GET", "/api/pengaturan", nil, 200, &p)
		var st struct {
			Sekarang time.Time `json:"sekarang"`
			Hari     int       `json:"hari"`
		}
		k.do("GET", "/api/status", nil, 200, &st)
		target := st.Sekarang.Truncate(time.Minute).Add(time.Minute)
		if target.Sub(st.Sekarang) < 15*time.Second {
			target = target.Add(time.Minute)
		}
		if target.Day() != st.Sekarang.Day() {
			t.Skip("terlalu dekat tengah malam untuk uji jadwal")
		}
		var j struct {
			ID int64 `json:"id"`
		}
		k.do("POST", "/api/jadwal", map[string]any{
			"profilId": p.ProfilDefaultID, "hari": st.Hari, "jam": target.Format("15:04"),
			"jenisBelId": idJenis["Masuk"], "label": "Uji E2E",
		}, 201, &j)
		if j.ID == 0 {
			t.Fatal("id jadwal baru = 0")
		}
		r := tungguRiwayat(t, k, func(r riwayat) bool { return r.JadwalID == j.ID }, 150*time.Second)
		if r.Status != "berhasil" {
			t.Fatalf("bel terjadwal status %q: %s", r.Status, r.Pesan)
		}
		if d := r.Waktu.Sub(target); d < -5*time.Second || d > 5*time.Second {
			t.Fatalf("waktu riwayat %s, mau sekitar %s", r.Waktu, target)
		}
	})

	t.Run("darurat lalu hentikan", func(t *testing.T) {
		// Aman: urutan Darurat 2x Sirene 4,5 dtk (~9 dtk, real-time lewat --ao=null),
		// sedangkan stop dikirim 1 dtk setelah mulai.
		if idJenis["Darurat"] == 0 {
			t.Fatal("jenis bel Darurat tidak ada")
		}
		k.do("POST", "/api/bel/darurat", map[string]int64{"jenisBelId": idJenis["Darurat"]}, 202, nil)
		time.Sleep(time.Second)
		k.do("POST", "/api/bel/stop", nil, 200, nil)
		r := tungguRiwayat(t, k, func(r riwayat) bool { return r.Sumber == "darurat" }, 30*time.Second)
		if r.Status != "dihentikan" {
			t.Fatalf("darurat status %q, mau dihentikan", r.Status)
		}
	})

	t.Run("backup lalu restore", func(t *testing.T) {
		var zipData []byte
		k.do("GET", "/api/backup", nil, 200, &zipData)
		zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
		if err != nil {
			t.Fatal(err)
		}
		adaUnggahan := false
		for _, f := range zr.File {
			if strings.HasPrefix(f.Name, "audio/upload-") {
				adaUnggahan = true
			}
		}
		if !adaUnggahan {
			t.Fatal("backup tidak berisi audio unggahan")
		}
		k.do("POST", "/api/profil", map[string]string{"nama": "Setelah Backup"}, 201, nil)
		k.unggah("/api/restore", "", "backup.zip", zipData, 202, nil)

		// Tunggu proses lama benar-benar berhenti (permintaan mulai gagal), baru tunggu siap.
		mati := false
		for batas := time.Now().Add(30 * time.Second); time.Now().Before(batas); time.Sleep(100 * time.Millisecond) {
			c := &http.Client{Timeout: 2 * time.Second}
			resp, err := c.Get(dasar + "/api/sesi")
			if err != nil {
				mati = true
				break
			}
			resp.Body.Close()
		}
		if !mati {
			t.Fatal("proses lama tidak berhenti dalam 30 detik setelah restore")
		}
		tungguSiap(t)
		k2 := baru(t)
		k2.do("POST", "/api/login", map[string]string{"password": pwBaru}, 200, nil)
		var profil []struct {
			Nama string `json:"nama"`
		}
		k2.do("GET", "/api/profil", nil, 200, &profil)
		for _, p := range profil {
			if p.Nama == "Setelah Backup" {
				t.Fatal("restore harus mengembalikan data ke kondisi backup")
			}
		}
	})

	t.Run("ganti zona waktu memulai ulang aplikasi", func(t *testing.T) {
		k3 := baru(t)
		k3.do("POST", "/api/login", map[string]string{"password": pwBaru}, 200, nil)
		var sesi struct {
			Boot string `json:"boot"`
		}
		k3.do("GET", "/api/sesi", nil, 200, &sesi)
		var p map[string]any
		k3.do("GET", "/api/pengaturan", nil, 200, &p)
		p["zonaWaktu"] = "WITA"
		var hasil struct {
			MulaiUlang bool `json:"mulaiUlang"`
		}
		k3.do("PUT", "/api/pengaturan", p, 200, &hasil)
		if !hasil.MulaiUlang {
			t.Fatal("ganti zona harus membalas mulaiUlang")
		}
		baruBoot := ""
		for batas := time.Now().Add(30 * time.Second); time.Now().Before(batas); time.Sleep(200 * time.Millisecond) {
			c := &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{DisableKeepAlives: true}}
			resp, err := c.Get(dasar + "/api/sesi")
			if err != nil {
				continue
			}
			var s struct {
				Boot string `json:"boot"`
			}
			json.NewDecoder(resp.Body).Decode(&s)
			resp.Body.Close()
			if s.Boot != "" && s.Boot != sesi.Boot {
				baruBoot = s.Boot
				break
			}
		}
		if baruBoot == "" {
			t.Fatal("aplikasi tidak dimulai ulang dalam 30 detik setelah ganti zona")
		}
		k4 := baru(t)
		k4.do("POST", "/api/login", map[string]string{"password": pwBaru}, 200, nil)
		var st struct {
			ZonaWaktu string `json:"zonaWaktu"`
		}
		k4.do("GET", "/api/status", nil, 200, &st)
		if st.ZonaWaktu != "WITA" {
			t.Fatalf("zona setelah restart = %q, mau WITA", st.ZonaWaktu)
		}
	})
}
