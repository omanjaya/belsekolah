# Rencana 6: Backend Zona Waktu, Cadangan & Template Jadwal

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Menambah API zona waktu WIB/WITA/WIT, ekspor/impor jadwal, duplikat profil, template buatan sendiri, dan pembuat jadwal bawaan.

**Architecture:** Dua paket Go murni baru — `internal/formatjadwal` (format file jadwal + validasi) dan `internal/pembuat` (pembuat jadwal dari parameter) — plus `internal/zona` (WIB/WITA/WIT). `store` mendapat fungsi transaksional (duplikat, ekspor, impor, template). `api` mendapat handler baru. Zona aktif dipilih saat start dari database lalu flag; mengubahnya memulai ulang aplikasi (exit 3) seperti restore.

**Tech Stack:** Go 1.25, SQLite (modernc), `net/http` ServeMux.

**Spec:** `docs/superpowers/specs/2026-10-15-tampilan-zona-template-design.md` (§4–§8), dasar `docs/superpowers/specs/2026-10-15-bel-sekolah-design.md`.

## Global Constraints

- Semua pesan error untuk pengguna dalam bahasa Indonesia; nama identifier mengikuti gaya kode yang ada (bahasa Indonesia).
- Database memakai **satu koneksi** (`SetMaxOpenConns(1)`): **jangan pernah memanggil `s.db` di dalam transaksi yang sedang terbuka** — semua pembacaan yang dibutuhkan dilakukan sebelum `BeginTx`, atau lewat `tx`.
- Endpoint baru wajib login (`rute(...)` di `daftarRuteData`), otomatis melewati `cekOrigin` dan `blokirSaatRestore`.
- Setiap perubahan data yang berhasil memanggil `s.ubahData()`.
- Impor **tidak pernah menimpa** profil yang ada.
- Nama jenis bel dicocokkan **tanpa membedakan huruf besar/kecil**.
- Format file: `{"format":"belsekolah-jadwal","versi":1,...}`, maks 256 KB, maks 1000 bel.
- Zona: `WIB`→`Asia/Jakarta`, `WITA`→`Asia/Makassar`, `WIT`→`Asia/Jayapura`; kunci pengaturan `zona_waktu` berisi nama IANA.
- `go vet ./...` dan `go test -race ./...` hijau di akhir setiap tugas.
- Commit diakhiri baris kosong + `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Struktur File

| File | Tanggung jawab |
|---|---|
| `internal/zona/zona.go` (+test) | Daftar WIB/WITA/WIT, konversi kode↔IANA, pemilihan zona saat start |
| `internal/formatjadwal/formatjadwal.go` (+test) | Struct format file, `Baca`, `Validasi`, `ErrorValidasi`, helper |
| `internal/pembuat/pembuat.go` (+test) | `Parameter`, `Buat` → `formatjadwal.Data` |
| `internal/store/schema.sql` | Tabel `template` |
| `internal/store/zona.go` | `ZonaWaktu`/`SetZonaWaktu` |
| `internal/store/cadangan.go` (+`cadangan_test.go`) | Jumlah bel per profil, duplikat, ekspor, impor, nama tersedia |
| `internal/store/template.go` (+`template_test.go`) | CRUD template |
| `internal/api/zona_test.go` | Tes API zona |
| `internal/api/cadangan.go` (+`cadangan_test.go`) | Handler duplikat/ekspor/periksa/impor |
| `internal/api/template.go` (+`template_test.go`) | Handler template + pembuat |
| `cmd/belsekolah/main.go` | Pemilihan zona setelah database dibuka |
| `e2e/e2e_test.go` | Ganti zona → restart → status melaporkan zona baru |

---

### Task 1: Paket `zona` dan zona waktu di store, start, serta API pengaturan/status/sistem

**Files:**
- Create: `internal/zona/zona.go`, `internal/zona/zona_test.go`, `internal/store/zona.go`, `internal/api/zona_test.go`
- Modify: `internal/store/store_test.go` (tes kecil), `cmd/belsekolah/main.go`, `internal/api/server.go`, `internal/api/dto.go`, `internal/api/pengaturan.go`, `internal/api/beranda.go`, `internal/api/bantu_test.go`, `internal/api/perbaikan_test.go` (bila pesan 503 diuji)

**Interfaces:**
- Produces: `zona.IANA(kode string) (string, bool)`, `zona.Kode(iana string) string`, `zona.Pilih(tersimpan, flag string) (*time.Location, []string)`, `store.KunciZonaWaktu`, `(*Store).ZonaWaktu(ctx) (string, error)` ("" bila belum diatur), `(*Store).SetZonaWaktu(ctx, iana string) error`, `api.Deps.Zona string` (nama IANA zona aktif), field JSON `zonaWaktu` di pengaturan/status/sistem, `zonaIana` di sistem, `mulaiUlang` di respons PUT pengaturan.

- [ ] **Step 1: Tulis tes paket zona (gagal)**

`internal/zona/zona_test.go`:
```go
package zona

import "testing"

func TestIANAdanKode(t *testing.T) {
	for _, c := range []struct{ kode, iana string }{
		{"WIB", "Asia/Jakarta"}, {"WITA", "Asia/Makassar"}, {"WIT", "Asia/Jayapura"},
	} {
		got, ok := IANA(c.kode)
		if !ok || got != c.iana {
			t.Errorf("IANA(%q) = %q,%v; mau %q", c.kode, got, ok, c.iana)
		}
		if k := Kode(c.iana); k != c.kode {
			t.Errorf("Kode(%q) = %q; mau %q", c.iana, k, c.kode)
		}
	}
	if _, ok := IANA("wita"); ok {
		t.Error("kode harus huruf besar persis")
	}
	if k := Kode("Europe/Berlin"); k != "Europe/Berlin" {
		t.Errorf("Kode di luar daftar = %q; mau nama IANA apa adanya", k)
	}
}

func TestPilih(t *testing.T) {
	cases := []struct {
		nama, tersimpan, flag, mau string
		peringatan              bool
	}{
		{"tersimpan menang", "Asia/Makassar", "Asia/Jakarta", "Asia/Makassar", false},
		{"kosong pakai flag", "", "Asia/Jayapura", "Asia/Jayapura", false},
		{"tersimpan rusak pakai flag", "Bukan/Zona", "Asia/Jakarta", "Asia/Jakarta", true},
		{"flag rusak pakai Jakarta", "", "Bukan/Zona", "Asia/Jakarta", true},
		{"semua kosong", "", "", "Asia/Jakarta", false},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			loc, per := Pilih(c.tersimpan, c.flag)
			if loc.String() != c.mau {
				t.Errorf("zona = %q; mau %q", loc.String(), c.mau)
			}
			if (len(per) > 0) != c.peringatan {
				t.Errorf("peringatan = %v; mau ada=%v", per, c.peringatan)
			}
		})
	}
}
```

- [ ] **Step 2: Jalankan, pastikan gagal**

Run: `go test ./internal/zona/`
Expected: FAIL (paket/fungsi belum ada)

- [ ] **Step 3: Implementasi `internal/zona/zona.go`**

```go
// Package zona mengenal zona waktu Indonesia (WIB/WITA/WIT) dan memilih zona aktif saat start.
package zona

import (
	"fmt"
	"time"
	_ "time/tzdata" // zona tertanam, agar tidak bergantung pada tzdata OS
)

// Bawaan dipakai bila tidak ada zona lain yang valid.
const Bawaan = "Asia/Jakarta"

var daftar = []struct{ kode, iana string }{
	{"WIB", "Asia/Jakarta"},
	{"WITA", "Asia/Makassar"},
	{"WIT", "Asia/Jayapura"},
}

// IANA mengubah WIB/WITA/WIT menjadi nama zona IANA.
func IANA(kode string) (string, bool) {
	for _, z := range daftar {
		if z.kode == kode {
			return z.iana, true
		}
	}
	return "", false
}

// Kode mengubah nama IANA menjadi WIB/WITA/WIT; zona lain dikembalikan apa adanya.
func Kode(iana string) string {
	for _, z := range daftar {
		if z.iana == iana {
			return z.kode
		}
	}
	return iana
}

// Pilih menentukan zona aktif: zona tersimpan bila valid, lalu flag, lalu Asia/Jakarta.
// Nilai yang tidak valid dilewati dan dilaporkan sebagai peringatan.
func Pilih(tersimpan, flag string) (*time.Location, []string) {
	var peringatan []string
	for _, c := range []struct{ asal, nama string }{{"pengaturan", tersimpan}, {"flag -tz", flag}} {
		if c.nama == "" {
			continue
		}
		loc, err := time.LoadLocation(c.nama)
		if err == nil {
			return loc, peringatan
		}
		peringatan = append(peringatan, fmt.Sprintf("zona waktu dari %s tidak valid (%q), diabaikan", c.asal, c.nama))
	}
	loc, _ := time.LoadLocation(Bawaan)
	return loc, peringatan
}
```

- [ ] **Step 4: Jalankan tes zona, pastikan lulus**

Run: `go test ./internal/zona/`
Expected: PASS

- [ ] **Step 5: Store — tes lalu implementasi**

Tambah ke `internal/store/store_test.go`:
```go
func TestZonaWaktu(t *testing.T) {
	s := bukaStore(t)
	if z := harus(s.ZonaWaktu(ctx))(t); z != "" {
		t.Fatalf("zona awal = %q, mau kosong", z)
	}
	if err := s.SetZonaWaktu(ctx, "Asia/Makassar"); err != nil {
		t.Fatal(err)
	}
	if z := harus(s.ZonaWaktu(ctx))(t); z != "Asia/Makassar" {
		t.Fatalf("zona = %q", z)
	}
}
```

`internal/store/zona.go`:
```go
package store

import (
	"context"
	"errors"
)

// KunciZonaWaktu menyimpan nama IANA zona waktu sekolah (diutamakan di atas flag -tz).
const KunciZonaWaktu = "zona_waktu"

// ZonaWaktu mengembalikan "" bila belum pernah diatur.
func (s *Store) ZonaWaktu(ctx context.Context) (string, error) {
	z, err := s.GetSetting(ctx, KunciZonaWaktu)
	if errors.Is(err, ErrTidakAda) {
		return "", nil
	}
	return z, err
}

func (s *Store) SetZonaWaktu(ctx context.Context, iana string) error {
	return s.SetSetting(ctx, KunciZonaWaktu, iana)
}
```

Run: `go test ./internal/store/ -run TestZonaWaktu` → PASS

- [ ] **Step 6: Start aplikasi memilih zona setelah database dibuka**

Di `cmd/belsekolah/main.go`:
- Hapus pemanggilan `aturZonaWaktu(*tz)` dari `main()` dan hapus fungsi `aturZonaWaktu`. `-reset-password` tidak lagi bergantung pada zona.
- Ubah tanda tangan menjadi `func jalankan(dataDir, addr, tz string, log *slog.Logger) error` dan panggil `jalankan(*dataDir, *addr, *tz, log)`.
- Tepat setelah `defer st.Close()` (sebelum `bawaan.Pasang`), sisipkan:
```go
	tersimpan, err := st.ZonaWaktu(ctx)
	if err != nil {
		return fmt.Errorf("baca zona waktu: %w", err)
	}
	loc, peringatan := zona.Pilih(tersimpan, tz)
	for _, p := range peringatan {
		log.Warn(p)
	}
	// Belum ada goroutine yang membaca jam, jadi aman mengganti time.Local di sini.
	time.Local = loc
```
- Tambah `Zona: loc.String(),` ke `api.Deps{...}`.
- Ubah log start: `"zona_waktu", loc.String()`.
- Import `belsekolah/internal/zona`. Biarkan `_ "time/tzdata"` yang ada.
- Ubah deskripsi flag: `flag.String("tz", "Asia/Jakarta", "zona waktu awal; pilihan di web lebih diutamakan")`.

Run: `go build ./... && go vet ./cmd/...` → sukses.

- [ ] **Step 7: Tes API zona (gagal)**

Di `internal/api/bantu_test.go`, tambah `Zona: "Asia/Jakarta",` ke `Deps` dalam `siapkan`.

`internal/api/zona_test.go`:
```go
package api

import (
	"net/http"
	"testing"
	"time"

	"belsekolah/internal/store"
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
	go func() {
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
	if z, _ := l.st.ZonaWaktu(ctx); z != "" {
		t.Fatalf("zona tidak boleh tersimpan saat restore berlangsung, tersimpan %q", z)
	}
}

var _ = store.KunciZonaWaktu
```
Tambah import `context` dan `errors` ke `zona_test.go`, lalu hapus baris `var _ = store.KunciZonaWaktu` beserta import `store` bila tidak terpakai.

Tambah helper ke `internal/api/bantu_test.go` (import `mime/multipart`, `strconv`):
```go
func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// unggahRestore mengirim isi sebagai file backup.zip ke POST /api/restore.
func unggahRestore(t *testing.T, l *lingkungan, c *http.Client, isi []byte) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", "backup.zip")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(isi)
	mw.Close()
	req, _ := http.NewRequest("POST", l.srv.URL+"/api/restore", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}
```
Bila tes restore yang ada sudah punya helper multipart serupa, boleh dipakai ulang asal namanya `unggahRestore` tidak bentrok.

Run: `go test ./internal/api/ -run TestZona` → FAIL

- [ ] **Step 8: Implementasi API**

`internal/api/server.go`:
- Tambah ke `Deps`: `Zona string // nama IANA zona aktif; "" = lokasi dari Now()`.
- Tambah ke `server`: `akanMulaiUlang atomic.Bool // zona diganti; aplikasi akan dimulai ulang`.
- Di `blokirSaatRestore`, kondisi blokir menjadi `s.restore.Load() == restoreSiap || s.akanMulaiUlang.Load()`; pesan untuk kasus zona: `"aplikasi sedang dimulai ulang untuk menerapkan zona waktu"` (pesan restore tetap seperti sekarang).
- Tambah helper:
```go
// zonaAktif mengembalikan nama IANA zona yang sedang dipakai proses.
func (s *server) zonaAktif() string {
	if s.Zona != "" {
		return s.Zona
	}
	return s.Now().Location().String()
}
```

`internal/api/dto.go`:
- `pengaturanDTO` tambah `ZonaWaktu string `json:"zonaWaktu"`` dan `MulaiUlang bool `json:"mulaiUlang,omitempty"``.
- `statusDTO` tambah `ZonaWaktu string `json:"zonaWaktu"``.
- `sistemDTO` tambah `ZonaIana string `json:"zonaIana"`` (field `ZonaWaktu` yang ada kini berisi kode).

`internal/api/pengaturan.go`:
- `bacaPengaturan`: isi `p.ZonaWaktu = zona.Kode(s.zonaAktif())`.
- `ubahPengaturan`, setelah validasi volume dan **sebelum** menyimpan apa pun:
```go
	var ianaBaru string
	if p.ZonaWaktu != "" {
		iana, ok := zona.IANA(p.ZonaWaktu)
		if !ok {
			tulisError(w, http.StatusBadRequest, "zona waktu harus WIB, WITA, atau WIT")
			return
		}
		if iana != s.zonaAktif() {
			if s.restore.Load() != restoreIdle {
				tulisError(w, http.StatusServiceUnavailable, "restore sedang berjalan, coba lagi nanti")
				return
			}
			ianaBaru = iana
		}
	}
```
  Setelah volume & profil default tersimpan:
```go
	if ianaBaru != "" {
		if err := s.Store.SetZonaWaktu(r.Context(), ianaBaru); err != nil {
			s.tulisErrorData(w, err)
			return
		}
		s.akanMulaiUlang.Store(true)
		s.Log.Warn("zona waktu diganti, aplikasi akan dimulai ulang", "zona", ianaBaru)
		out, err := s.bacaPengaturan(r)
		if err != nil {
			s.tulisErrorData(w, err)
			return
		}
		out.ZonaWaktu = zona.Kode(ianaBaru)
		out.MulaiUlang = true
		tulisJSON(w, http.StatusOK, out)
		if s.MulaiUlang != nil {
			go func() {
				time.Sleep(500 * time.Millisecond) // beri waktu respons terkirim
				s.MulaiUlang()
			}()
		}
		return
	}
```
- `sistem`: `ZonaWaktu: zona.Kode(s.zonaAktif()), ZonaIana: s.zonaAktif(),`.

`internal/api/beranda.go` (`status`): isi `ZonaWaktu: zona.Kode(s.zonaAktif())` di `statusDTO`.

`cmd/belsekolah/main.go`: `errMulaiUlang` kini juga dipakai untuk zona — ubah pesannya menjadi `"mulai ulang atas permintaan (restore atau zona waktu)"` dan log di `main` menjadi `"keluar untuk dimulai ulang; layanan akan berjalan lagi"`.

- [ ] **Step 9: Jalankan semua tes**

Run: `go vet ./... && go test -race ./...`
Expected: PASS (perbaiki tes lama yang memeriksa format `zonaWaktu` lama di `/api/sistem` bila ada).

- [ ] **Step 10: Commit**

```bash
git add internal/zona internal/store/zona.go internal/store/store_test.go internal/api cmd/belsekolah/main.go
git commit -m "feat: zona waktu WIB/WITA/WIT dari pengaturan, ganti zona memulai ulang aplikasi

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Paket `formatjadwal`

**Files:**
- Create: `internal/formatjadwal/formatjadwal.go`, `internal/formatjadwal/formatjadwal_test.go`

**Interfaces:**
- Consumes: `schedule.ParseJam(s string) (jam, menit int, err error)`
- Produces:
  - `const Format = "belsekolah-jadwal"`, `Versi = 1`, `MaksBel = 1000`, `MaksUkuran = 256 << 10`
  - `type JenisBel struct{ Nama, Warna string }` (json `nama`,`warna`)
  - `type Bel struct{ Hari int; Jam, JenisBel, Label string }` (json `hari`,`jam`,`jenisBel`,`label`)
  - `type Data struct{ Format string; Versi int; Nama string; JenisBel []JenisBel; Jadwal []Bel }` (json `format`,`versi`,`nama`,`jenisBel`,`jadwal`)
  - `type ErrorValidasi struct{ Pesan string }` (+ `Error()`)
  - `func Baru(nama string) Data`
  - `func Baca(b []byte) (Data, error)`
  - `func (d Data) Validasi() error`
  - `func (d Data) JenisDipakai() []JenisBel` — jenis bel yang dirujuk jadwal, urutan kemunculan di `d.JenisBel`
  - `func (d Data) HariBerisi() []int` — hari (naik) yang punya bel

- [ ] **Step 1: Tulis tes (gagal)**

`internal/formatjadwal/formatjadwal_test.go`:
```go
package formatjadwal

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func contoh() Data {
	d := Baru("Normal")
	d.JenisBel = []JenisBel{{"Masuk", "#2563eb"}, {"Pulang", "#9333ea"}, {"Tak Dipakai", "#000000"}}
	d.Jadwal = []Bel{
		{Hari: 1, Jam: "07:00", JenisBel: "Masuk", Label: "Masuk"},
		{Hari: 1, Jam: "13:00", JenisBel: "Pulang", Label: "Pulang"},
		{Hari: 5, Jam: "07:00", JenisBel: "Masuk", Label: ""},
	}
	return d
}

func harapTidakValid(t *testing.T, err error, sub string) {
	t.Helper()
	var v *ErrorValidasi
	if !errors.As(err, &v) {
		t.Fatalf("err = %v (%T), mau *ErrorValidasi", err, err)
	}
	if !strings.Contains(v.Pesan, sub) {
		t.Fatalf("pesan %q tidak memuat %q", v.Pesan, sub)
	}
}

func TestBaca_RoundTrip(t *testing.T) {
	b, err := json.Marshal(contoh())
	if err != nil {
		t.Fatal(err)
	}
	d, err := Baca(b)
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "Normal" || len(d.Jadwal) != 3 || d.Jadwal[1].Jam != "13:00" {
		t.Fatalf("hasil baca salah: %+v", d)
	}
}

func TestBaca_Ditolak(t *testing.T) {
	besar := `{"format":"belsekolah-jadwal","versi":1,"nama":"` + strings.Repeat("a", MaksUkuran) + `"}`
	cases := []struct{ nama, isi, sub string }{
		{"bukan json", `ini bukan json`, "bukan file jadwal"},
		{"format lain", `{"format":"lain","versi":1,"nama":"x","jenisBel":[],"jadwal":[]}`, "bukan file jadwal"},
		{"versi baru", `{"format":"belsekolah-jadwal","versi":2,"nama":"x","jenisBel":[],"jadwal":[]}`, "versi"},
		{"terlalu besar", besar, "terlalu besar"},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			_, err := Baca([]byte(c.isi))
			harapTidakValid(t, err, c.sub)
		})
	}
}

func TestValidasi(t *testing.T) {
	cases := []struct {
		nama string
		ubah func(*Data)
		sub  string
	}{
		{"nama kosong", func(d *Data) { d.Nama = " " }, "nama"},
		{"hari salah", func(d *Data) { d.Jadwal[0].Hari = 8 }, "bel ke-1"},
		{"jam salah", func(d *Data) { d.Jadwal[1].Jam = "7:00" }, "bel ke-2"},
		{"jenis tak ada", func(d *Data) { d.Jadwal[2].JenisBel = "Hilang" }, "Hilang"},
		{"bel ganda", func(d *Data) { d.Jadwal[1].Jam = "07:00" }, "ganda"},
		{"label panjang", func(d *Data) { d.Jadwal[0].Label = strings.Repeat("x", 101) }, "label"},
		{"jenis ganda beda huruf", func(d *Data) { d.JenisBel[1].Nama = "masuk" }, "ganda"},
		{"warna salah", func(d *Data) { d.JenisBel[0].Warna = "biru" }, "warna"},
		{"terlalu banyak", func(d *Data) {
			d.Jadwal = nil
			for i := 0; i <= MaksBel; i++ {
				d.Jadwal = append(d.Jadwal, Bel{Hari: 1 + i%7, Jam: "07:00", JenisBel: "Masuk"})
			}
		}, "maksimal"},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			d := contoh()
			c.ubah(&d)
			harapTidakValid(t, d.Validasi(), c.sub)
		})
	}
	if err := contoh().Validasi(); err != nil {
		t.Fatalf("contoh valid ditolak: %v", err)
	}
}

func TestJenisDipakaiDanHariBerisi(t *testing.T) {
	d := contoh()
	j := d.JenisDipakai()
	if len(j) != 2 || j[0].Nama != "Masuk" || j[1].Nama != "Pulang" {
		t.Fatalf("JenisDipakai = %+v", j)
	}
	h := d.HariBerisi()
	if len(h) != 2 || h[0] != 1 || h[1] != 5 {
		t.Fatalf("HariBerisi = %v", h)
	}
}
```

- [ ] **Step 2: Jalankan, pastikan gagal**

Run: `go test ./internal/formatjadwal/` → FAIL

- [ ] **Step 3: Implementasi**

`internal/formatjadwal/formatjadwal.go`:
```go
// Package formatjadwal mendefinisikan file jadwal (ekspor/impor profil dan template).
package formatjadwal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"belsekolah/internal/schedule"
)

const (
	Format     = "belsekolah-jadwal"
	Versi      = 1
	MaksBel    = 1000
	MaksUkuran = 256 << 10
)

type JenisBel struct {
	Nama  string `json:"nama"`
	Warna string `json:"warna"`
}

type Bel struct {
	Hari     int    `json:"hari"`
	Jam      string `json:"jam"`
	JenisBel string `json:"jenisBel"`
	Label    string `json:"label"`
}

type Data struct {
	Format   string     `json:"format"`
	Versi    int        `json:"versi"`
	Nama     string     `json:"nama"`
	JenisBel []JenisBel `json:"jenisBel"`
	Jadwal   []Bel      `json:"jadwal"`
}

// ErrorValidasi menandai file atau isi yang tidak valid; Pesan aman ditampilkan ke pengguna.
type ErrorValidasi struct{ Pesan string }

func (e *ErrorValidasi) Error() string { return e.Pesan }

func salah(format string, a ...any) error {
	return &ErrorValidasi{Pesan: fmt.Sprintf(format, a...)}
}

// Baru membuat Data kosong berformat versi terbaru.
func Baru(nama string) Data {
	return Data{Format: Format, Versi: Versi, Nama: nama, JenisBel: []JenisBel{}, Jadwal: []Bel{}}
}

// Baca mengurai dan memvalidasi isi file jadwal.
func Baca(b []byte) (Data, error) {
	var d Data
	if len(b) > MaksUkuran {
		return d, salah("file jadwal terlalu besar (maksimal %d KB)", MaksUkuran>>10)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&d); err != nil || d.Format != Format {
		return Data{}, salah("bukan file jadwal bel sekolah")
	}
	if d.Versi != Versi {
		return Data{}, salah("versi file jadwal %d tidak didukung (aplikasi ini memakai versi %d)", d.Versi, Versi)
	}
	if err := d.Validasi(); err != nil {
		return Data{}, err
	}
	return d, nil
}

var polaWarna = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Validasi memeriksa isi Data dengan aturan yang sama seperti input manual.
func (d Data) Validasi() error {
	nama := strings.TrimSpace(d.Nama)
	if nama == "" || utf8.RuneCountInString(nama) > 100 {
		return salah("nama jadwal wajib diisi, maksimal 100 karakter")
	}
	if len(d.Jadwal) > MaksBel {
		return salah("maksimal %d bel dalam satu file jadwal", MaksBel)
	}
	jenis := make(map[string]bool, len(d.JenisBel))
	for _, j := range d.JenisBel {
		n := strings.TrimSpace(j.Nama)
		if n == "" || utf8.RuneCountInString(n) > 100 {
			return salah("nama jenis bel wajib diisi, maksimal 100 karakter")
		}
		if !polaWarna.MatchString(j.Warna) {
			return salah("warna jenis bel %q harus berformat #RRGGBB", j.Nama)
		}
		k := strings.ToLower(n)
		if jenis[k] {
			return salah("jenis bel %q ganda", j.Nama)
		}
		jenis[k] = true
	}
	waktu := make(map[string]bool, len(d.Jadwal))
	for i, b := range d.Jadwal {
		if b.Hari < 1 || b.Hari > 7 {
			return salah("bel ke-%d: hari harus 1 (Senin) sampai 7 (Minggu)", i+1)
		}
		if _, _, err := schedule.ParseJam(b.Jam); err != nil {
			return salah("bel ke-%d: %s", i+1, err.Error())
		}
		if utf8.RuneCountInString(strings.TrimSpace(b.Label)) > 100 {
			return salah("bel ke-%d: label maksimal 100 karakter", i+1)
		}
		if !jenis[strings.ToLower(strings.TrimSpace(b.JenisBel))] {
			return salah("bel ke-%d: jenis bel %q tidak ada di daftar jenis bel", i+1, b.JenisBel)
		}
		k := fmt.Sprintf("%d %s", b.Hari, b.Jam)
		if waktu[k] {
			return salah("bel ke-%d: bel ganda di hari dan jam yang sama (%s)", i+1, b.Jam)
		}
		waktu[k] = true
	}
	return nil
}

// JenisDipakai mengembalikan jenis bel yang dirujuk jadwal, sesuai urutan di d.JenisBel.
func (d Data) JenisDipakai() []JenisBel {
	dipakai := make(map[string]bool)
	for _, b := range d.Jadwal {
		dipakai[strings.ToLower(strings.TrimSpace(b.JenisBel))] = true
	}
	var out []JenisBel
	for _, j := range d.JenisBel {
		if dipakai[strings.ToLower(strings.TrimSpace(j.Nama))] {
			out = append(out, j)
		}
	}
	return out
}

// HariBerisi mengembalikan hari (1–7, naik) yang memiliki bel.
func (d Data) HariBerisi() []int {
	ada := make(map[int]bool)
	for _, b := range d.Jadwal {
		ada[b.Hari] = true
	}
	out := make([]int, 0, len(ada))
	for h := range ada {
		out = append(out, h)
	}
	sort.Ints(out)
	return out
}
```

- [ ] **Step 4: Jalankan tes**

Run: `go test ./internal/formatjadwal/` → PASS

- [ ] **Step 5: Commit**

```bash
git add internal/formatjadwal
git commit -m "feat: format file jadwal (ekspor/impor) beserta validasinya

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Paket `pembuat` (pembuat jadwal bawaan)

**Files:**
- Create: `internal/pembuat/pembuat.go`, `internal/pembuat/pembuat_test.go`

**Interfaces:**
- Consumes: `formatjadwal.Baru`, `formatjadwal.Data`, `formatjadwal.Bel`, `formatjadwal.JenisBel`, `formatjadwal.ErrorValidasi`, `schedule.ParseJam`
- Produces:
  - `type Istirahat struct{ SetelahJP, Menit int }` (json `setelahJP`,`menit`)
  - `type Upacara struct{ Aktif bool; Menit int }` (json `aktif`,`menit`)
  - `type Kegiatan struct{ Aktif bool; Menit int; Label string }` (json `aktif`,`menit`,`label`)
  - `type Ujian struct{ JamMulai string; JumlahSesi, LamaSesi, Jeda int }` (json `jamMulai`,`jumlahSesi`,`lamaSesi`,`jeda`)
  - `type Parameter struct{ Jenjang, Varian string; HariSekolah int; JamMasuk string; LamaJP, JumlahJP, JumlahJPJumat, JumlahJPSabtu int; Istirahat []Istirahat; UpacaraSenin Upacara; KegiatanJumat Kegiatan; Ujian Ujian }` (json camelCase: `jenjang`,`varian`,`hariSekolah`,`jamMasuk`,`lamaJP`,`jumlahJP`,`jumlahJPJumat`,`jumlahJPSabtu`,`istirahat`,`upacaraSenin`,`kegiatanJumat`,`ujian`)
  - `func Buat(p Parameter) (formatjadwal.Data, error)` — error selalu `*formatjadwal.ErrorValidasi`
  - Nama jenis bel yang dipakai: `"Masuk"`, `"Ganti Pelajaran"`, `"Istirahat"`, `"Pulang"`

- [ ] **Step 1: Tulis tes (gagal)**

`internal/pembuat/pembuat_test.go`:
```go
package pembuat

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"belsekolah/internal/formatjadwal"
)

func smpNormal() Parameter {
	return Parameter{
		Jenjang: "SMP", Varian: "normal", HariSekolah: 5,
		JamMasuk: "07:00", LamaJP: 40, JumlahJP: 7, JumlahJPJumat: 5, JumlahJPSabtu: 5,
		Istirahat:    []Istirahat{{SetelahJP: 3, Menit: 15}, {SetelahJP: 6, Menit: 15}},
		UpacaraSenin: Upacara{Aktif: true, Menit: 40},
	}
}

// ringkas menulis bel satu hari sebagai "HH:MM Jenis|Label" dipisah koma.
func ringkas(d formatjadwal.Data, hari int) string {
	var s []string
	for _, b := range d.Jadwal {
		if b.Hari == hari {
			s = append(s, fmt.Sprintf("%s %s|%s", b.Jam, b.JenisBel, b.Label))
		}
	}
	return strings.Join(s, ", ")
}

func TestBuat_SMPNormal(t *testing.T) {
	d, err := Buat(smpNormal())
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "SMP Normal 5 hari" {
		t.Errorf("nama = %q", d.Nama)
	}
	senin := "07:00 Masuk|Upacara, 07:40 Ganti Pelajaran|Jam ke-1, 08:20 Ganti Pelajaran|Jam ke-2, " +
		"09:00 Ganti Pelajaran|Jam ke-3, 09:40 Istirahat|Istirahat, 09:55 Masuk|Masuk setelah istirahat, " +
		"10:35 Ganti Pelajaran|Jam ke-5, 11:15 Ganti Pelajaran|Jam ke-6, 11:55 Istirahat|Istirahat, " +
		"12:10 Masuk|Masuk setelah istirahat, 12:50 Pulang|Pulang"
	selasa := "07:00 Masuk|Masuk, 07:40 Ganti Pelajaran|Jam ke-2, 08:20 Ganti Pelajaran|Jam ke-3, " +
		"09:00 Istirahat|Istirahat, 09:15 Masuk|Masuk setelah istirahat, 09:55 Ganti Pelajaran|Jam ke-5, " +
		"10:35 Ganti Pelajaran|Jam ke-6, 11:15 Istirahat|Istirahat, 11:30 Masuk|Masuk setelah istirahat, 12:10 Pulang|Pulang"
	jumat := "07:00 Masuk|Masuk, 07:40 Ganti Pelajaran|Jam ke-2, 08:20 Ganti Pelajaran|Jam ke-3, " +
		"09:00 Istirahat|Istirahat, 09:15 Masuk|Masuk setelah istirahat, 09:55 Ganti Pelajaran|Jam ke-5, 10:35 Pulang|Pulang"
	for _, c := range []struct {
		hari int
		mau  string
	}{{1, senin}, {2, selasa}, {3, selasa}, {4, selasa}, {5, jumat}, {6, ""}, {7, ""}} {
		if got := ringkas(d, c.hari); got != c.mau {
			t.Errorf("hari %d:\n got %s\nmau %s", c.hari, got, c.mau)
		}
	}
	if err := d.Validasi(); err != nil {
		t.Fatalf("hasil tidak lolos validasi format: %v", err)
	}
	if len(d.JenisBel) != 4 {
		t.Fatalf("jenis bel = %+v", d.JenisBel)
	}
}

func TestBuat_EnamHariDanKegiatanJumat(t *testing.T) {
	p := smpNormal()
	p.HariSekolah = 6
	p.JumlahJPSabtu = 2
	p.UpacaraSenin.Aktif = false
	p.KegiatanJumat = Kegiatan{Aktif: true, Menit: 30, Label: ""}
	d, err := Buat(p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "SMP Normal 6 hari" {
		t.Errorf("nama = %q", d.Nama)
	}
	if got := ringkas(d, 1); !strings.HasPrefix(got, "07:00 Masuk|Masuk, 07:40 Ganti Pelajaran|Jam ke-2") {
		t.Errorf("Senin tanpa upacara = %s", got)
	}
	if got := ringkas(d, 5); !strings.HasPrefix(got, "07:00 Masuk|Kegiatan Jumat, 07:30 Ganti Pelajaran|Jam ke-1, 08:10 Ganti Pelajaran|Jam ke-2") {
		t.Errorf("Jumat dengan kegiatan = %s", got)
	}
	if got := ringkas(d, 6); got != "07:00 Masuk|Masuk, 07:40 Ganti Pelajaran|Jam ke-2, 08:20 Pulang|Pulang" {
		t.Errorf("Sabtu = %s", got)
	}
}

func TestBuat_Ujian(t *testing.T) {
	p := Parameter{Jenjang: "SMA", Varian: "ujian", HariSekolah: 5,
		Ujian: Ujian{JamMulai: "07:30", JumlahSesi: 2, LamaSesi: 120, Jeda: 30}}
	d, err := Buat(p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "SMA/SMK Ujian 5 hari" {
		t.Errorf("nama = %q", d.Nama)
	}
	mau := "07:30 Masuk|Sesi 1 dimulai, 09:30 Istirahat|Sesi 1 selesai, 10:00 Masuk|Sesi 2 dimulai, 12:00 Pulang|Ujian selesai"
	for h := 1; h <= 5; h++ {
		if got := ringkas(d, h); got != mau {
			t.Errorf("hari %d = %s", h, got)
		}
	}
}

func TestBuat_Ramadan(t *testing.T) {
	p := smpNormal()
	p.Varian = "ramadan"
	p.LamaJP = 30
	d, err := Buat(p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Nama != "SMP Ramadan 5 hari" {
		t.Errorf("nama = %q", d.Nama)
	}
	if got := ringkas(d, 2); !strings.HasPrefix(got, "07:00 Masuk|Masuk, 07:30 Ganti Pelajaran|Jam ke-2") {
		t.Errorf("Selasa Ramadan = %s", got)
	}
}

func TestBuat_Ditolak(t *testing.T) {
	cases := []struct {
		nama string
		ubah func(*Parameter)
		sub  string
	}{
		{"jenjang", func(p *Parameter) { p.Jenjang = "TK" }, "jenjang"},
		{"varian", func(p *Parameter) { p.Varian = "libur" }, "varian"},
		{"hari", func(p *Parameter) { p.HariSekolah = 4 }, "hari sekolah"},
		{"jam masuk", func(p *Parameter) { p.JamMasuk = "7.00" }, "jam masuk"},
		{"lama jp", func(p *Parameter) { p.LamaJP = 5 }, "lama jam pelajaran"},
		{"jumlah jp", func(p *Parameter) { p.JumlahJP = 13 }, "jumlah jam pelajaran"},
		{"jumlah jumat", func(p *Parameter) { p.JumlahJPJumat = 0 }, "Jumat"},
		{"istirahat banyak", func(p *Parameter) {
			p.Istirahat = append(p.Istirahat, Istirahat{SetelahJP: 1, Menit: 10})
		}, "istirahat"},
		{"istirahat tidak naik", func(p *Parameter) { p.Istirahat[1].SetelahJP = 3 }, "istirahat"},
		{"istirahat di akhir", func(p *Parameter) { p.Istirahat[1].SetelahJP = 7 }, "istirahat"},
		{"istirahat menit", func(p *Parameter) { p.Istirahat[0].Menit = 61 }, "istirahat"},
		{"upacara menit", func(p *Parameter) { p.UpacaraSenin.Menit = 5 }, "upacara"},
		{"tengah malam", func(p *Parameter) { p.JamMasuk = "22:00"; p.LamaJP = 90; p.JumlahJP = 12 }, "tengah malam"},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			p := smpNormal()
			c.ubah(&p)
			_, err := Buat(p)
			var v *formatjadwal.ErrorValidasi
			if !errors.As(err, &v) || !strings.Contains(v.Pesan, c.sub) {
				t.Fatalf("err = %v, mau ErrorValidasi memuat %q", err, c.sub)
			}
		})
	}
	u := Parameter{Jenjang: "SD", Varian: "ujian", HariSekolah: 5, Ujian: Ujian{JamMulai: "07:30", JumlahSesi: 5, LamaSesi: 60, Jeda: 10}}
	if _, err := Buat(u); err == nil || !strings.Contains(err.Error(), "sesi") {
		t.Fatalf("jumlah sesi 5 harus ditolak, err = %v", err)
	}
}
```

- [ ] **Step 2: Jalankan, pastikan gagal**

Run: `go test ./internal/pembuat/` → FAIL

- [ ] **Step 3: Implementasi**

`internal/pembuat/pembuat.go`:
```go
// Package pembuat menyusun jadwal bel dari parameter sekolah (template bawaan).
package pembuat

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/schedule"
)

// Nama jenis bel; sama dengan jenis bel awal di paket bawaan.
const (
	jenisMasuk     = "Masuk"
	jenisGanti     = "Ganti Pelajaran"
	jenisIstirahat = "Istirahat"
	jenisPulang    = "Pulang"
)

// Warna sama dengan jenisBelAwal di paket bawaan.
var jenisBel = []formatjadwal.JenisBel{
	{Nama: jenisMasuk, Warna: "#2563eb"},
	{Nama: jenisGanti, Warna: "#0d9488"},
	{Nama: jenisIstirahat, Warna: "#16a34a"},
	{Nama: jenisPulang, Warna: "#9333ea"},
}

type Istirahat struct {
	SetelahJP int `json:"setelahJP"`
	Menit     int `json:"menit"`
}

type Upacara struct {
	Aktif bool `json:"aktif"`
	Menit int  `json:"menit"`
}

type Kegiatan struct {
	Aktif bool   `json:"aktif"`
	Menit int    `json:"menit"`
	Label string `json:"label"`
}

type Ujian struct {
	JamMulai   string `json:"jamMulai"`
	JumlahSesi int    `json:"jumlahSesi"`
	LamaSesi   int    `json:"lamaSesi"`
	Jeda       int    `json:"jeda"`
}

type Parameter struct {
	Jenjang       string      `json:"jenjang"` // SD, SMP, SMA
	Varian        string      `json:"varian"`  // normal, ramadan, ujian
	HariSekolah   int         `json:"hariSekolah"`
	JamMasuk      string      `json:"jamMasuk"`
	LamaJP        int         `json:"lamaJP"`
	JumlahJP      int         `json:"jumlahJP"`
	JumlahJPJumat int         `json:"jumlahJPJumat"`
	JumlahJPSabtu int         `json:"jumlahJPSabtu"`
	Istirahat     []Istirahat `json:"istirahat"`
	UpacaraSenin  Upacara     `json:"upacaraSenin"`
	KegiatanJumat Kegiatan    `json:"kegiatanJumat"`
	Ujian         Ujian       `json:"ujian"`
}

var namaJenjang = map[string]string{"SD": "SD", "SMP": "SMP", "SMA": "SMA/SMK"}
var namaVarian = map[string]string{"normal": "Normal", "ramadan": "Ramadan", "ujian": "Ujian"}

func salah(format string, a ...any) error {
	return &formatjadwal.ErrorValidasi{Pesan: fmt.Sprintf(format, a...)}
}

func diLuar(v, min, maks int) bool { return v < min || v > maks }

func menitDari(jam, nama string) (int, error) {
	h, m, err := schedule.ParseJam(jam)
	if err != nil {
		return 0, salah("%s tidak valid, gunakan format HH:MM", nama)
	}
	return h*60 + m, nil
}

func (p *Parameter) validasi() error {
	if _, ok := namaJenjang[p.Jenjang]; !ok {
		return salah("jenjang harus SD, SMP, atau SMA")
	}
	if _, ok := namaVarian[p.Varian]; !ok {
		return salah("varian harus normal, ramadan, atau ujian")
	}
	if p.HariSekolah != 5 && p.HariSekolah != 6 {
		return salah("hari sekolah harus 5 atau 6")
	}
	if p.Varian == "ujian" {
		u := p.Ujian
		if _, err := menitDari(u.JamMulai, "jam mulai ujian"); err != nil {
			return err
		}
		switch {
		case diLuar(u.JumlahSesi, 1, 4):
			return salah("jumlah sesi ujian harus 1 sampai 4")
		case diLuar(u.LamaSesi, 30, 180):
			return salah("lama sesi ujian harus 30 sampai 180 menit")
		case diLuar(u.Jeda, 5, 60):
			return salah("jeda antarsesi harus 5 sampai 60 menit")
		}
		return nil
	}
	if _, err := menitDari(p.JamMasuk, "jam masuk"); err != nil {
		return err
	}
	switch {
	case diLuar(p.LamaJP, 10, 90):
		return salah("lama jam pelajaran harus 10 sampai 90 menit")
	case diLuar(p.JumlahJP, 1, 12):
		return salah("jumlah jam pelajaran harus 1 sampai 12")
	case diLuar(p.JumlahJPJumat, 1, 12):
		return salah("jumlah jam pelajaran hari Jumat harus 1 sampai 12")
	case p.HariSekolah == 6 && diLuar(p.JumlahJPSabtu, 1, 12):
		return salah("jumlah jam pelajaran hari Sabtu harus 1 sampai 12")
	case len(p.Istirahat) > 2:
		return salah("maksimal 2 kali istirahat")
	}
	sebelum := 0
	for i, s := range p.Istirahat {
		if s.SetelahJP <= sebelum || s.SetelahJP >= p.JumlahJP {
			return salah("istirahat ke-%d harus setelah jam pelajaran yang urut dan sebelum jam terakhir", i+1)
		}
		if diLuar(s.Menit, 5, 60) {
			return salah("lama istirahat ke-%d harus 5 sampai 60 menit", i+1)
		}
		sebelum = s.SetelahJP
	}
	if p.UpacaraSenin.Aktif && diLuar(p.UpacaraSenin.Menit, 10, 90) {
		return salah("lama upacara harus 10 sampai 90 menit")
	}
	if p.KegiatanJumat.Aktif {
		if diLuar(p.KegiatanJumat.Menit, 10, 90) {
			return salah("lama kegiatan Jumat harus 10 sampai 90 menit")
		}
		if utf8.RuneCountInString(strings.TrimSpace(p.KegiatanJumat.Label)) > 100 {
			return salah("nama kegiatan Jumat maksimal 100 karakter")
		}
	}
	return nil
}

// penyusun menampung bel satu hari dan mencatat bila jam melewati tengah malam.
type penyusun struct {
	hari    int
	bel     []formatjadwal.Bel
	lewatan bool
}

func (s *penyusun) tambah(menit int, jenis, label string) {
	if menit >= 24*60 {
		s.lewatan = true
		return
	}
	s.bel = append(s.bel, formatjadwal.Bel{
		Hari: s.hari, Jam: fmt.Sprintf("%02d:%02d", menit/60, menit%60), JenisBel: jenis, Label: label,
	})
}

func (p *Parameter) jumlahJPHari(hari int) int {
	switch hari {
	case 5:
		return p.JumlahJPJumat
	case 6:
		return p.JumlahJPSabtu
	}
	return p.JumlahJP
}

func (p *Parameter) hariPelajaran(s *penyusun) {
	t, _ := menitDari(p.JamMasuk, "jam masuk")
	label, tambahan := "Masuk", 0
	switch {
	case s.hari == 1 && p.UpacaraSenin.Aktif:
		label, tambahan = "Upacara", p.UpacaraSenin.Menit
	case s.hari == 5 && p.KegiatanJumat.Aktif:
		label, tambahan = strings.TrimSpace(p.KegiatanJumat.Label), p.KegiatanJumat.Menit
		if label == "" {
			label = "Kegiatan Jumat"
		}
	}
	s.tambah(t, jenisMasuk, label)
	if tambahan > 0 {
		t += tambahan
		s.tambah(t, jenisGanti, "Jam ke-1")
	}
	istirahat := make(map[int]int, len(p.Istirahat))
	for _, i := range p.Istirahat {
		istirahat[i.SetelahJP] = i.Menit
	}
	jumlah := p.jumlahJPHari(s.hari)
	for k := 1; k <= jumlah; k++ {
		t += p.LamaJP // akhir JP k
		if k == jumlah {
			s.tambah(t, jenisPulang, "Pulang")
			break
		}
		if m, ok := istirahat[k]; ok {
			s.tambah(t, jenisIstirahat, "Istirahat")
			t += m
			s.tambah(t, jenisMasuk, "Masuk setelah istirahat")
		} else {
			s.tambah(t, jenisGanti, fmt.Sprintf("Jam ke-%d", k+1))
		}
	}
}

func (p *Parameter) hariUjian(s *penyusun) {
	t, _ := menitDari(p.Ujian.JamMulai, "jam mulai ujian")
	for i := 1; i <= p.Ujian.JumlahSesi; i++ {
		s.tambah(t, jenisMasuk, fmt.Sprintf("Sesi %d dimulai", i))
		t += p.Ujian.LamaSesi
		if i == p.Ujian.JumlahSesi {
			s.tambah(t, jenisPulang, "Ujian selesai")
			break
		}
		s.tambah(t, jenisIstirahat, fmt.Sprintf("Sesi %d selesai", i))
		t += p.Ujian.Jeda
	}
}

// Buat menyusun jadwal Senin sampai hari sekolah terakhir dari parameter.
func Buat(p Parameter) (formatjadwal.Data, error) {
	if err := p.validasi(); err != nil {
		return formatjadwal.Data{}, err
	}
	d := formatjadwal.Baru(fmt.Sprintf("%s %s %d hari", namaJenjang[p.Jenjang], namaVarian[p.Varian], p.HariSekolah))
	d.JenisBel = append([]formatjadwal.JenisBel(nil), jenisBel...)
	for hari := 1; hari <= p.HariSekolah; hari++ {
		s := &penyusun{hari: hari}
		if p.Varian == "ujian" {
			p.hariUjian(s)
		} else {
			p.hariPelajaran(s)
		}
		if s.lewatan {
			return formatjadwal.Data{}, salah("jadwal melewati tengah malam; kurangi jumlah atau lama jam pelajaran")
		}
		d.Jadwal = append(d.Jadwal, s.bel...)
	}
	if err := d.Validasi(); err != nil {
		return formatjadwal.Data{}, err
	}
	return d, nil
}
```

- [ ] **Step 4: Jalankan tes**

Run: `go test ./internal/pembuat/` → PASS. Bila nilai jam di tes berbeda dari keluaran, **periksa ulang perhitungan tangan di tes terhadap aturan spec §6.2 sebelum mengubah kode** — tes di atas adalah perhitungan aturan spec.

- [ ] **Step 5: Commit**

```bash
git add internal/pembuat
git commit -m "feat: pembuat jadwal bawaan (SD/SMP/SMA, normal/ramadan/ujian, 5/6 hari)

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: Store — jumlah bel, duplikat, ekspor, impor, template

**Files:**
- Create: `internal/store/cadangan.go`, `internal/store/cadangan_test.go`, `internal/store/template.go`, `internal/store/template_test.go`
- Modify: `internal/store/schema.sql`

**Interfaces:**
- Consumes: `formatjadwal.Data/Baru/JenisBel/Bel/ErrorValidasi`, `(*Store).ListJenisBel`, `(*Store).FindAudio`, `(*Store).ListAudio`, `(*Store).GetProfil`, `(*Store).ListJadwal`, `validasiNama`, `terjemahkan`, `cekDiubah`, `validasi`
- Produces:
  - `func (s *Store) JumlahBelPerProfil(ctx) (map[int64]int, error)`
  - `func (s *Store) DuplikatProfil(ctx, id int64, nama string) (int64, error)`
  - `func (s *Store) EksporProfil(ctx, id int64) (formatjadwal.Data, error)`
  - `func (s *Store) JenisBelTakDikenal(ctx, d formatjadwal.Data) ([]string, error)`
  - `func (s *Store) NamaProfilTersedia(ctx, nama string) (string, error)`
  - `type PemetaanJenis struct{ JenisBelID int64; Baru bool }` (json `jenisBelId`,`baru`)
  - `func (s *Store) ImporJadwal(ctx, d formatjadwal.Data, nama string, pemetaan map[string]PemetaanJenis) (int64, error)`
  - `type Template struct{ ID int64; Nama string; Data formatjadwal.Data; Dibuat time.Time }`
  - `func (s *Store) CreateTemplate(ctx, nama string, d formatjadwal.Data, now time.Time) (int64, error)`
  - `func (s *Store) ListTemplate(ctx) ([]Template, error)` (urut nama)
  - `func (s *Store) GetTemplate(ctx, id int64) (Template, error)`
  - `func (s *Store) UpdateNamaTemplate(ctx, id int64, nama string) error`
  - `func (s *Store) DeleteTemplate(ctx, id int64) error`
  - Error validasi dari `formatjadwal` diubah menjadi `*store.ErrorValidasi` dengan pesan yang sama.

- [ ] **Step 1: Skema**

Tambahkan ke akhir `internal/store/schema.sql`:
```sql
CREATE TABLE IF NOT EXISTS template (
  id      INTEGER PRIMARY KEY,
  nama    TEXT NOT NULL UNIQUE,
  data    TEXT NOT NULL,
  dibuat  TEXT NOT NULL
);
```

- [ ] **Step 2: Tulis tes cadangan (gagal)**

`internal/store/cadangan_test.go`:
```go
package store

import (
	"errors"
	"reflect"
	"testing"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/schedule"
)

// isiProfil membuat profil "Normal" (default) dengan 3 bel memakai jenis "Masuk".
func isiProfil(t *testing.T, s *Store) (profil, jenis int64) {
	t.Helper()
	profil, jenis, _ = dataDasar(t, s)
	for _, b := range []schedule.Jadwal{
		{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, Label: "Masuk"},
		{ProfilID: profil, Hari: 1, Jam: "12:00", JenisBelID: jenis, Label: "Pulang"},
		{ProfilID: profil, Hari: 5, Jam: "07:00", JenisBelID: jenis, Label: ""},
	} {
		harus(s.CreateJadwal(ctx, b))(t)
	}
	return
}

func TestJumlahBelPerProfil(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	kosong := harus(s.CreateProfil(ctx, "Kosong"))(t)
	n := harus(s.JumlahBelPerProfil(ctx))(t)
	if n[profil] != 3 || n[kosong] != 0 {
		t.Fatalf("jumlah = %v", n)
	}
}

func TestDuplikatProfil(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	baru := harus(s.DuplikatProfil(ctx, profil, "Normal (salinan)"))(t)
	asal := harus(s.ListJadwal(ctx, profil))(t)
	salin := harus(s.ListJadwal(ctx, baru))(t)
	if len(salin) != len(asal) {
		t.Fatalf("salinan %d bel, asal %d", len(salin), len(asal))
	}
	for i := range asal {
		if salin[i].Hari != asal[i].Hari || salin[i].Jam != asal[i].Jam || salin[i].JenisBelID != asal[i].JenisBelID || salin[i].Label != asal[i].Label {
			t.Fatalf("bel %d berbeda: %+v vs %+v", i, salin[i], asal[i])
		}
	}
	if _, err := s.DuplikatProfil(ctx, profil, "Normal"); !errors.Is(err, ErrDuplikat) {
		t.Fatalf("nama bentrok: err = %v", err)
	}
	if _, err := s.DuplikatProfil(ctx, 999, "X"); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("profil tak ada: err = %v", err)
	}
	if n := harus(s.CountProfil(ctx))(t); n != 2 {
		t.Fatalf("profil = %d setelah duplikat gagal, mau 2", n)
	}
}

func TestEksporImpor_RoundTrip(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s)
	d := harus(s.EksporProfil(ctx, profil))(t)
	if d.Format != formatjadwal.Format || d.Nama != "Normal" || len(d.Jadwal) != 3 || len(d.JenisBel) != 1 {
		t.Fatalf("ekspor = %+v", d)
	}
	if tak := harus(s.JenisBelTakDikenal(ctx, d))(t); len(tak) != 0 {
		t.Fatalf("tak dikenal = %v", tak)
	}
	nama := harus(s.NamaProfilTersedia(ctx, d.Nama))(t)
	if nama != "Normal (impor)" {
		t.Fatalf("nama tersedia = %q", nama)
	}
	baru := harus(s.ImporJadwal(ctx, d, nama, nil))(t)
	asal := harus(s.ListJadwal(ctx, profil))(t)
	hasil := harus(s.ListJadwal(ctx, baru))(t)
	for i := range asal {
		asal[i].ID, asal[i].ProfilID, hasil[i].ID, hasil[i].ProfilID = 0, 0, 0, 0
	}
	if !reflect.DeepEqual(asal, hasil) {
		t.Fatalf("round-trip berbeda:\n%+v\n%+v", asal, hasil)
	}
	if hasil[0].JenisBelID != jenis {
		t.Fatal("jenis bel yang namanya sama harus dipakai ulang")
	}
	if nama := harus(s.NamaProfilTersedia(ctx, "Normal"))(t); nama != "Normal (impor 2)" {
		t.Fatalf("nama tersedia kedua = %q", nama)
	}
}

func dataAsing() formatjadwal.Data {
	d := formatjadwal.Baru("Asing")
	d.JenisBel = []formatjadwal.JenisBel{{"MASUK", "#111111"}, {"Bel Literasi", "#222222"}, {"Bel Doa", "#333333"}}
	d.Jadwal = []formatjadwal.Bel{
		{Hari: 1, Jam: "07:00", JenisBel: "MASUK"},
		{Hari: 1, Jam: "07:15", JenisBel: "Bel Literasi", Label: "Literasi"},
		{Hari: 1, Jam: "07:30", JenisBel: "Bel Doa", Label: "Doa"},
	}
	return d
}

func TestImpor_Pemetaan(t *testing.T) {
	s := bukaStore(t)
	_, jenis := isiProfil(t, s)
	d := dataAsing()
	tak := harus(s.JenisBelTakDikenal(ctx, d))(t)
	if !reflect.DeepEqual(tak, []string{"Bel Literasi", "Bel Doa"}) {
		t.Fatalf("tak dikenal = %v (MASUK harus cocok dengan Masuk)", tak)
	}
	// Pemetaan kurang → ditolak, tidak ada sisa.
	_, err := s.ImporJadwal(ctx, d, "Asing", map[string]PemetaanJenis{"Bel Literasi": {Baru: true}})
	harapValidasi(t, err)
	if n := harus(s.CountProfil(ctx))(t); n != 1 {
		t.Fatalf("profil = %d setelah impor gagal", n)
	}
	if js := harus(s.ListJenisBel(ctx))(t); len(js) != 1 {
		t.Fatalf("jenis bel = %d setelah impor gagal (transaksi harus dibatalkan)", len(js))
	}
	// Jenis tujuan tidak ada → ditolak.
	_, err = s.ImporJadwal(ctx, d, "Asing", map[string]PemetaanJenis{"Bel Literasi": {Baru: true}, "Bel Doa": {JenisBelID: 999}})
	harapValidasi(t, err)

	id := harus(s.ImporJadwal(ctx, d, "Asing", map[string]PemetaanJenis{
		"Bel Literasi": {Baru: true}, "Bel Doa": {JenisBelID: jenis},
	}))(t)
	js := harus(s.ListJadwal(ctx, id))(t)
	if len(js) != 3 || js[0].JenisBelID != jenis || js[2].JenisBelID != jenis {
		t.Fatalf("hasil impor = %+v", js)
	}
	semua := harus(s.ListJenisBel(ctx))(t)
	if len(semua) != 2 || semua[1].Nama != "Bel Literasi" || semua[1].Warna != "#222222" {
		t.Fatalf("jenis bel baru = %+v", semua)
	}
	urutan := harus(s.UrutanBel(ctx, semua[1].ID))(t)
	if len(urutan) != 1 {
		t.Fatalf("jenis bel baru harus punya 1 suara, ada %d", len(urutan))
	}
	if _, err := s.ImporJadwal(ctx, d, "Asing", map[string]PemetaanJenis{"Bel Literasi": {Baru: true}, "Bel Doa": {Baru: true}}); !errors.Is(err, ErrDuplikat) {
		t.Fatalf("nama profil bentrok: err = %v", err)
	}
}

func TestImpor_DataTidakValid(t *testing.T) {
	s := bukaStore(t)
	isiProfil(t, s)
	d := dataAsing()
	d.Jadwal[0].Hari = 9
	_, err := s.ImporJadwal(ctx, d, "X", nil)
	harapValidasi(t, err)
}
```

- [ ] **Step 3: Tulis tes template (gagal)**

`internal/store/template_test.go`:
```go
package store

import (
	"errors"
	"testing"
	"time"

	"belsekolah/internal/formatjadwal"
)

func TestTemplateCRUD(t *testing.T) {
	s := bukaStore(t)
	d := dataAsing()
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	id := harus(s.CreateTemplate(ctx, "Template A", d, now))(t)
	if _, err := s.CreateTemplate(ctx, "Template A", d, now); !errors.Is(err, ErrDuplikat) {
		t.Fatalf("nama ganda: err = %v", err)
	}
	bad := d
	bad.Jadwal = []formatjadwal.Bel{{Hari: 0, Jam: "07:00", JenisBel: "MASUK"}}
	_, err := s.CreateTemplate(ctx, "Rusak", bad, now)
	harapValidasi(t, err)

	tp := harus(s.GetTemplate(ctx, id))(t)
	if tp.Nama != "Template A" || tp.Data.Nama != "Template A" || len(tp.Data.Jadwal) != 3 || !tp.Dibuat.Equal(now) {
		t.Fatalf("template = %+v", tp)
	}
	if err := s.UpdateNamaTemplate(ctx, id, "Template B"); err != nil {
		t.Fatal(err)
	}
	tp = harus(s.GetTemplate(ctx, id))(t)
	if tp.Nama != "Template B" || tp.Data.Nama != "Template B" {
		t.Fatalf("ganti nama: %+v", tp)
	}
	harus(s.CreateTemplate(ctx, "Awal", d, now))(t)
	list := harus(s.ListTemplate(ctx))(t)
	if len(list) != 2 || list[0].Nama != "Awal" {
		t.Fatalf("list harus urut nama: %+v", list)
	}
	if err := s.DeleteTemplate(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTemplate(ctx, id); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("setelah hapus: %v", err)
	}
	if err := s.DeleteTemplate(ctx, id); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("hapus dua kali: %v", err)
	}
}
```

Run: `go test ./internal/store/` → FAIL (fungsi belum ada)

- [ ] **Step 4: Implementasi `internal/store/cadangan.go`**

```go
package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"belsekolah/internal/formatjadwal"
)

// dariFormat mengubah error validasi format file menjadi error validasi store.
func dariFormat(err error) error {
	var v *formatjadwal.ErrorValidasi
	if errors.As(err, &v) {
		return validasi("%s", v.Pesan)
	}
	return err
}

// JumlahBelPerProfil mengembalikan jumlah bel per ID profil (profil tanpa bel tidak ada di peta).
func (s *Store) JumlahBelPerProfil(ctx context.Context) (map[int64]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT profil_id, COUNT(*) FROM jadwal GROUP BY profil_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]int)
	for rows.Next() {
		var id int64
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
}

// DuplikatProfil membuat profil baru berisi salinan seluruh bel profil id.
func (s *Store) DuplikatProfil(ctx context.Context, id int64, nama string) (int64, error) {
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	if _, err := s.GetProfil(ctx, id); err != nil {
		return 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO profil (nama) VALUES (?)`, nama)
	if err != nil {
		return 0, terjemahkan(err, nil)
	}
	baru, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label)
		 SELECT ?, hari, jam, jenis_bel_id, label FROM jadwal WHERE profil_id = ?`, baru, id); err != nil {
		return 0, err
	}
	return baru, tx.Commit()
}

// EksporProfil menyusun file jadwal dari satu profil.
func (s *Store) EksporProfil(ctx context.Context, id int64) (formatjadwal.Data, error) {
	p, err := s.GetProfil(ctx, id)
	if err != nil {
		return formatjadwal.Data{}, err
	}
	jadwal, err := s.ListJadwal(ctx, id)
	if err != nil {
		return formatjadwal.Data{}, err
	}
	jenis, err := s.ListJenisBel(ctx)
	if err != nil {
		return formatjadwal.Data{}, err
	}
	peta := make(map[int64]JenisBel, len(jenis))
	for _, j := range jenis {
		peta[j.ID] = j
	}
	d := formatjadwal.Baru(p.Nama)
	masuk := make(map[int64]bool)
	for _, b := range jadwal {
		j, ok := peta[b.JenisBelID]
		if !ok {
			return formatjadwal.Data{}, fmt.Errorf("jenis bel %d untuk jadwal %d tidak ada", b.JenisBelID, b.ID)
		}
		if !masuk[j.ID] {
			masuk[j.ID] = true
			d.JenisBel = append(d.JenisBel, formatjadwal.JenisBel{Nama: j.Nama, Warna: j.Warna})
		}
		d.Jadwal = append(d.Jadwal, formatjadwal.Bel{Hari: b.Hari, Jam: b.Jam, JenisBel: j.Nama, Label: b.Label})
	}
	return d, nil
}

func kunciNama(n string) string { return strings.ToLower(strings.TrimSpace(n)) }

// JenisBelTakDikenal mengembalikan nama jenis bel yang dipakai d tetapi belum ada di database.
func (s *Store) JenisBelTakDikenal(ctx context.Context, d formatjadwal.Data) ([]string, error) {
	ada, err := s.ListJenisBel(ctx)
	if err != nil {
		return nil, err
	}
	kenal := make(map[string]bool, len(ada))
	for _, j := range ada {
		kenal[kunciNama(j.Nama)] = true
	}
	out := []string{}
	for _, j := range d.JenisDipakai() {
		if !kenal[kunciNama(j.Nama)] {
			out = append(out, j.Nama)
		}
	}
	return out, nil
}

// NamaProfilTersedia mengembalikan nama, atau nama + " (impor)", " (impor 2)", ... yang belum dipakai.
func (s *Store) NamaProfilTersedia(ctx context.Context, nama string) (string, error) {
	nama = strings.TrimSpace(nama)
	if r := []rune(nama); len(r) > 85 {
		nama = string(r[:85])
	}
	ps, err := s.ListProfil(ctx)
	if err != nil {
		return "", err
	}
	dipakai := make(map[string]bool, len(ps))
	for _, p := range ps {
		dipakai[p.Nama] = true
	}
	if !dipakai[nama] {
		return nama, nil
	}
	for i := 1; ; i++ {
		calon := nama + " (impor)"
		if i > 1 {
			calon = fmt.Sprintf("%s (impor %d)", nama, i)
		}
		if !dipakai[calon] && utf8.RuneCountInString(calon) <= 100 {
			return calon, nil
		}
	}
}

// PemetaanJenis menentukan jenis bel tujuan untuk satu nama jenis bel yang tidak dikenal.
type PemetaanJenis struct {
	JenisBelID int64 `json:"jenisBelId"`
	Baru       bool  `json:"baru"`
}

// suaraJenisBaru memilih suara untuk jenis bel baru hasil impor: "Ding Dong" bawaan, atau suara pertama.
func (s *Store) suaraJenisBaru(ctx context.Context) (int64, error) {
	a, err := s.FindAudio(ctx, "bawaan", "Ding Dong")
	if err == nil {
		return a.ID, nil
	}
	if !errors.Is(err, ErrTidakAda) {
		return 0, err
	}
	semua, err := s.ListAudio(ctx)
	if err != nil {
		return 0, err
	}
	if len(semua) == 0 {
		return 0, validasi("belum ada suara untuk jenis bel baru; tambahkan suara dahulu")
	}
	return semua[0].ID, nil
}

// ImporJadwal membuat profil baru dari d dalam satu transaksi. Jenis bel yang namanya sudah ada
// (tanpa membedakan huruf besar/kecil) dipakai langsung; sisanya harus ada di pemetaan.
func (s *Store) ImporJadwal(ctx context.Context, d formatjadwal.Data, nama string, pemetaan map[string]PemetaanJenis) (int64, error) {
	if err := d.Validasi(); err != nil {
		return 0, dariFormat(err)
	}
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	// Semua pembacaan sebelum transaksi: database hanya punya satu koneksi.
	ada, err := s.ListJenisBel(ctx)
	if err != nil {
		return 0, err
	}
	idAda := make(map[string]int64, len(ada))
	idSah := make(map[int64]bool, len(ada))
	for _, j := range ada {
		idAda[kunciNama(j.Nama)] = j.ID
		idSah[j.ID] = true
	}
	dipakai := d.JenisDipakai()
	perluSuara := false
	for _, j := range dipakai {
		if _, ok := idAda[kunciNama(j.Nama)]; ok {
			continue
		}
		p, ok := pemetaan[j.Nama]
		switch {
		case !ok || (!p.Baru && p.JenisBelID == 0):
			return 0, validasi("jenis bel %q belum dipetakan", j.Nama)
		case p.Baru:
			perluSuara = true
		case !idSah[p.JenisBelID]:
			return 0, validasi("jenis bel tujuan untuk %q tidak ditemukan", j.Nama)
		}
	}
	var suara int64
	if perluSuara {
		if suara, err = s.suaraJenisBaru(ctx); err != nil {
			return 0, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `INSERT INTO profil (nama) VALUES (?)`, nama)
	if err != nil {
		return 0, terjemahkan(err, nil)
	}
	profil, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	idJenis := make(map[string]int64, len(dipakai))
	for _, j := range dipakai {
		k := kunciNama(j.Nama)
		if id, ok := idAda[k]; ok {
			idJenis[k] = id
			continue
		}
		p := pemetaan[j.Nama]
		if !p.Baru {
			idJenis[k] = p.JenisBelID
			continue
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO jenis_bel (nama, warna) VALUES (?, ?)`, strings.TrimSpace(j.Nama), j.Warna)
		if err != nil {
			return 0, terjemahkan(err, nil)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO urutan_bel (jenis_bel_id, urutan, audio_id) VALUES (?, 1, ?)`, id, suara); err != nil {
			return 0, err
		}
		idJenis[k] = id
	}
	for _, b := range d.Jadwal {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label) VALUES (?, ?, ?, ?, ?)`,
			profil, b.Hari, b.Jam, idJenis[kunciNama(b.JenisBel)], strings.TrimSpace(b.Label)); err != nil {
			return 0, terjemahkan(err, errRefJadwal)
		}
	}
	return profil, tx.Commit()
}
```

- [ ] **Step 5: Implementasi `internal/store/template.go`**

```go
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"belsekolah/internal/formatjadwal"
)

// Template adalah jadwal tersimpan yang bisa dipakai untuk membuat profil baru.
type Template struct {
	ID     int64
	Nama   string
	Data   formatjadwal.Data
	Dibuat time.Time
}

func (s *Store) CreateTemplate(ctx context.Context, nama string, d formatjadwal.Data, now time.Time) (int64, error) {
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	d.Nama = nama
	if err := d.Validasi(); err != nil {
		return 0, dariFormat(err)
	}
	b, err := json.Marshal(d)
	if err != nil {
		return 0, err
	}
	id, err := s.insert(ctx, `INSERT INTO template (nama, data, dibuat) VALUES (?, ?, ?)`,
		nama, string(b), now.Format(time.RFC3339))
	return id, terjemahkan(err, nil)
}

func scanTemplate(r pemindai) (Template, error) {
	var t Template
	var data, dibuat string
	err := r.Scan(&t.ID, &t.Nama, &data, &dibuat)
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrTidakAda
	}
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal([]byte(data), &t.Data); err != nil {
		return t, fmt.Errorf("data template %d rusak: %w", t.ID, err)
	}
	if t.Dibuat, err = time.Parse(time.RFC3339, dibuat); err != nil {
		return t, fmt.Errorf("tanggal template %d rusak: %w", t.ID, err)
	}
	return t, nil
}

func (s *Store) ListTemplate(ctx context.Context) ([]Template, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, nama, data, dibuat FROM template ORDER BY nama`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTemplate(ctx context.Context, id int64) (Template, error) {
	return scanTemplate(s.db.QueryRowContext(ctx, `SELECT id, nama, data, dibuat FROM template WHERE id = ?`, id))
}

// UpdateNamaTemplate mengganti nama template sekaligus nama di dalam datanya.
func (s *Store) UpdateNamaTemplate(ctx context.Context, id int64, nama string) error {
	nama, err := validasiNama(nama)
	if err != nil {
		return err
	}
	t, err := s.GetTemplate(ctx, id)
	if err != nil {
		return err
	}
	t.Data.Nama = nama
	b, err := json.Marshal(t.Data)
	if err != nil {
		return err
	}
	return terjemahkan(cekDiubah(s.db.ExecContext(ctx,
		`UPDATE template SET nama = ?, data = ? WHERE id = ?`, nama, string(b), id)), nil)
}

func (s *Store) DeleteTemplate(ctx context.Context, id int64) error {
	return cekDiubah(s.db.ExecContext(ctx, `DELETE FROM template WHERE id = ?`, id))
}
```

- [ ] **Step 6: Jalankan tes store**

Run: `go test -race ./internal/store/` → PASS

- [ ] **Step 7: Commit**

```bash
git add internal/store
git commit -m "feat(store): duplikat profil, ekspor/impor jadwal transaksional, dan tabel template

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: API — profil (jumlah bel, duplikat, ekspor), impor, template, pembuat

**Files:**
- Create: `internal/api/cadangan.go`, `internal/api/cadangan_test.go`, `internal/api/template.go`, `internal/api/template_test.go`
- Modify: `internal/api/rute_data.go`, `internal/api/dto.go`, `internal/api/profil.go`, `internal/api/respon.go`

**Interfaces:**
- Consumes: semua fungsi store Task 4, `formatjadwal.Baca`, `formatjadwal.ErrorValidasi`, `pembuat.Buat`, `pembuat.Parameter`
- Produces (HTTP):
  - `GET /api/profil` → `[{id, nama, jumlahBel}]`
  - `POST /api/profil/{id}/duplikat` `{nama}` → `201 {id, nama, jumlahBel}`
  - `GET /api/profil/{id}/ekspor` → `200` file JSON, `Content-Disposition: attachment; filename="jadwal-<slug>.json"`
  - `POST /api/impor/periksa` (body = isi file) → `200 {nama, jumlahBel, jenisBelTakDikenal: [string], namaTersedia}`
  - `POST /api/impor` `{data, nama, pemetaan: {"<nama>": {jenisBelId}|{baru:true}}}` → `201 {profilId}`
  - `GET /api/template` → `[{id, nama, jumlahBel, hari: [int], dibuat}]`
  - `POST /api/template` `{nama, profilId}` atau `{nama, data}` → `201` ringkasan
  - `GET /api/template/{id}` → ringkasan + `data`
  - `PUT /api/template/{id}` `{nama}` → `200` ringkasan
  - `DELETE /api/template/{id}` → `204`
  - `GET /api/template/{id}/ekspor` → file
  - `POST /api/template/buat` (body = `pembuat.Parameter`) → `200` data format jadwal (tanpa efek samping)

- [ ] **Step 1: Tulis tes API (gagal)**

`internal/api/cadangan_test.go`:
```go
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
```

`internal/api/template_test.go`:
```go
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
		"istirahat":    []map[string]int{{"setelahJP": 3, "menit": 15}},
		"upacaraSenin": map[string]any{"aktif": true, "menit": 40},
		"kegiatanJumat": map[string]any{"aktif": false, "menit": 30, "label": ""},
		"ujian":        map[string]any{"jamMulai": "07:30", "jumlahSesi": 2, "lamaSesi": 120, "jeda": 30},
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
```
Helper `itoa` ditambahkan di Task 1.

Run: `go test ./internal/api/ -run 'TestProfil_|TestEkspor|TestImpor_|TestTemplate_'` → FAIL

- [ ] **Step 2: DTO & profil**

`internal/api/dto.go`: tambah `JumlahBel int `json:"jumlahBel"`` ke `profilDTO`. Tambah:
```go
type templateDTO struct {
	ID        int64              `json:"id"`
	Nama      string             `json:"nama"`
	JumlahBel int                `json:"jumlahBel"`
	Hari      []int              `json:"hari"`
	Dibuat    time.Time          `json:"dibuat"`
	Data      *formatjadwal.Data `json:"data,omitempty"`
}

func keTemplateDTO(t store.Template, denganData bool) templateDTO {
	out := templateDTO{ID: t.ID, Nama: t.Nama, JumlahBel: len(t.Data.Jadwal), Hari: t.Data.HariBerisi(), Dibuat: t.Dibuat}
	if denganData {
		d := t.Data
		out.Data = &d
	}
	return out
}
```

`internal/api/profil.go`:
- `listProfil` mengambil `s.Store.JumlahBelPerProfil` dan mengisi `JumlahBel` untuk setiap profil.
- `kirimProfil` juga mengisi `JumlahBel` (pakai `JumlahBelPerProfil`, ambil entri id).

`internal/api/respon.go` — `tulisErrorData` menangani error format:
```go
	var f *formatjadwal.ErrorValidasi
	switch {
	case errors.As(err, &v):
		tulisError(w, http.StatusBadRequest, v.Pesan)
	case errors.As(err, &f):
		tulisError(w, http.StatusBadRequest, f.Pesan)
	...
```

- [ ] **Step 3: Implementasi `internal/api/cadangan.go`**

```go
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/store"
)

var polaBukanSlug = regexp.MustCompile(`[^a-z0-9]+`)

// namaFileJadwal membentuk "jadwal-<slug>.json" yang aman untuk header.
func namaFileJadwal(nama string) string {
	slug := strings.Trim(polaBukanSlug.ReplaceAllString(strings.ToLower(nama), "-"), "-")
	if slug == "" {
		slug = "profil"
	}
	return "jadwal-" + slug + ".json"
}

func kirimFileJadwal(w http.ResponseWriter, d formatjadwal.Data) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, namaFileJadwal(d.Nama)))
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(d)
}

func (s *server) duplikatProfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var p profilDTO
	if !bacaJSON(w, r, &p) {
		return
	}
	baru, err := s.Store.DuplikatProfil(r.Context(), id, p.Nama)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimProfil(w, r, baru, http.StatusCreated)
}

func (s *server) eksporProfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	d, err := s.Store.EksporProfil(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	kirimFileJadwal(w, d)
}

// bacaFileJadwal membaca body mentah sebagai file jadwal. Menulis 400 dan mengembalikan false jika gagal.
func bacaFileJadwal(w http.ResponseWriter, r *http.Request) (formatjadwal.Data, bool) {
	b, err := io.ReadAll(io.LimitReader(r.Body, formatjadwal.MaksUkuran+1))
	if err != nil {
		tulisError(w, http.StatusBadRequest, "file jadwal tidak terbaca")
		return formatjadwal.Data{}, false
	}
	d, err := formatjadwal.Baca(b)
	if err != nil {
		tulisError(w, http.StatusBadRequest, err.Error())
		return formatjadwal.Data{}, false
	}
	return d, true
}

type periksaDTO struct {
	Nama               string   `json:"nama"`
	JumlahBel          int      `json:"jumlahBel"`
	JenisBelTakDikenal []string `json:"jenisBelTakDikenal"`
	NamaTersedia       string   `json:"namaTersedia"`
}

func (s *server) periksaImpor(w http.ResponseWriter, r *http.Request) {
	d, ok := bacaFileJadwal(w, r)
	if !ok {
		return
	}
	tak, err := s.Store.JenisBelTakDikenal(r.Context(), d)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	nama, err := s.Store.NamaProfilTersedia(r.Context(), d.Nama)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, periksaDTO{Nama: d.Nama, JumlahBel: len(d.Jadwal), JenisBelTakDikenal: tak, NamaTersedia: nama})
}

type imporDTO struct {
	Data     json.RawMessage                `json:"data"`
	Nama     string                         `json:"nama"`
	Pemetaan map[string]store.PemetaanJenis `json:"pemetaan"`
}

func (s *server) impor(w http.ResponseWriter, r *http.Request) {
	var in imporDTO
	if !bacaJSON(w, r, &in) {
		return
	}
	d, err := formatjadwal.Baca(in.Data)
	if err != nil {
		tulisError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := s.Store.ImporJadwal(r.Context(), d, in.Nama, in.Pemetaan)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	tulisJSON(w, http.StatusCreated, map[string]int64{"profilId": id})
}
```
Catatan: pesan 409 untuk `ErrDuplikat` mengikuti `tulisErrorData` yang ada.

- [ ] **Step 4: Implementasi `internal/api/template.go`**

```go
package api

import (
	"encoding/json"
	"net/http"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/pembuat"
)

func (s *server) listTemplate(w http.ResponseWriter, r *http.Request) {
	ts, err := s.Store.ListTemplate(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]templateDTO, 0, len(ts))
	for _, t := range ts {
		out = append(out, keTemplateDTO(t, false))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimTemplate(w http.ResponseWriter, r *http.Request, id int64, kode int, denganData bool) {
	t, err := s.Store.GetTemplate(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, keTemplateDTO(t, denganData))
}

type buatTemplateDTO struct {
	Nama     string          `json:"nama"`
	ProfilID int64           `json:"profilId"`
	Data     json.RawMessage `json:"data"`
}

func (s *server) buatTemplate(w http.ResponseWriter, r *http.Request) {
	var in buatTemplateDTO
	if !bacaJSON(w, r, &in) {
		return
	}
	var d formatjadwal.Data
	var err error
	switch {
	case in.ProfilID > 0:
		d, err = s.Store.EksporProfil(r.Context(), in.ProfilID)
	case len(in.Data) > 0:
		d, err = formatjadwal.Baca(in.Data)
	default:
		tulisError(w, http.StatusBadRequest, "pilih profil atau file jadwal untuk template")
		return
	}
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	id, err := s.Store.CreateTemplate(r.Context(), in.Nama, d, s.Now())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.kirimTemplate(w, r, id, http.StatusCreated, false)
}

func (s *server) ambilTemplate(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	s.kirimTemplate(w, r, id, http.StatusOK, true)
}

func (s *server) ubahTemplate(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var in struct {
		Nama string `json:"nama"`
	}
	if !bacaJSON(w, r, &in) {
		return
	}
	if err := s.Store.UpdateNamaTemplate(r.Context(), id, in.Nama); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.kirimTemplate(w, r, id, http.StatusOK, false)
}

func (s *server) hapusTemplate(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteTemplate(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) eksporTemplate(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	t, err := s.Store.GetTemplate(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	kirimFileJadwal(w, t.Data)
}

// buatDariParameter menyusun jadwal dari parameter pembuat tanpa menyimpan apa pun.
func (s *server) buatDariParameter(w http.ResponseWriter, r *http.Request) {
	var p pembuat.Parameter
	if !bacaJSON(w, r, &p) {
		return
	}
	d, err := pembuat.Buat(p)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, d)
}
```
Template tidak memengaruhi penjadwal, jadi handler template tidak memanggil `s.ubahData()`.

- [ ] **Step 5: Daftarkan rute**

Di `internal/api/rute_data.go`, setelah rute profil:
```go
	rute("POST /api/profil/{id}/duplikat", s.duplikatProfil)
	rute("GET /api/profil/{id}/ekspor", s.eksporProfil)
	rute("POST /api/impor/periksa", s.periksaImpor)
	rute("POST /api/impor", s.impor)

	rute("GET /api/template", s.listTemplate)
	rute("POST /api/template", s.buatTemplate)
	rute("POST /api/template/buat", s.buatDariParameter)
	rute("GET /api/template/{id}", s.ambilTemplate)
	rute("PUT /api/template/{id}", s.ubahTemplate)
	rute("DELETE /api/template/{id}", s.hapusTemplate)
	rute("GET /api/template/{id}/ekspor", s.eksporTemplate)
```
(`POST /api/template/buat` lebih spesifik daripada `/api/template/{id}` yang hanya GET/PUT/DELETE, jadi tidak bentrok.)

- [ ] **Step 6: Jalankan semua tes**

Run: `go vet ./... && go test -race ./...` → PASS

- [ ] **Step 7: Commit**

```bash
git add internal/api
git commit -m "feat(api): duplikat & ekspor profil, impor jadwal, template, dan pembuat jadwal

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: e2e zona waktu & dokumentasi

**Files:**
- Modify: `e2e/e2e_test.go`, `README.md`, `PANDUAN-INSTAL.md`, `docs/superpowers/specs/2026-10-15-bel-sekolah-design.md` (§14: tambah rujukan ke spec tahap 2)

**Interfaces:**
- Consumes: API Task 1 & 5.

- [ ] **Step 1: Tambah subtes e2e**

Di `e2e/e2e_test.go`, di akhir `TestAlurLengkap` (setelah subtes "backup lalu restore"), tambah:
```go
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
			c := &http.Client{Timeout: 2 * time.Second}
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
```
Periksa apakah `/api/sesi` tanpa login mengembalikan `boot` (lihat `sesi.go`: `sesiDTO` selalu berisi `Boot`). Bila tidak, ambil `boot` lewat klien yang sudah login dan abaikan error 401 saat menunggu.

- [ ] **Step 2: Jalankan e2e**

Run: `BEL_PORT=18080 ./scripts/e2e.sh`
Expected: Playwright lama tetap lulus (frontend belum berubah; bila ada tes yang membaca teks zona waktu lama di Pengaturan, sesuaikan harapannya ke format baru `WIB`), e2e Go termasuk subtes zona lulus.

- [ ] **Step 3: Dokumentasi**

- `PANDUAN-INSTAL.md`: di bagian zona waktu, jelaskan bahwa zona bisa diganti dari **Pengaturan → Zona waktu** (WIB/WITA/WIT), aplikasi mulai ulang sekitar 2 detik, dan pilihan di web lebih diutamakan daripada `--tz` saat instal. Jam sistem Debian tidak ikut berubah.
- `README.md`: tambah baris di bagian fitur/struktur untuk `internal/zona`, `internal/formatjadwal`, `internal/pembuat`.
- Spec dasar §14: tambah kalimat "Tambahan tahap 2: lihat `2026-10-15-tampilan-zona-template-design.md` §7."

- [ ] **Step 4: Commit**

```bash
git add e2e/e2e_test.go README.md PANDUAN-INSTAL.md docs/superpowers/specs/2026-10-15-bel-sekolah-design.md
git commit -m "test(e2e): ganti zona waktu memulai ulang aplikasi; dokumentasi zona & template

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
