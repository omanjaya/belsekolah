# Bel Sekolah — Rencana 2: API + Login + Backup + Docker Dev Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Binary `belsekolah` menyajikan REST API lengkap (login admin, status beranda, bel manual/darurat, CRUD profil/jadwal/periode/libur/jenis bel/audio, riwayat, pengaturan, backup/restore) dan bisa diuji end-to-end di container Docker Debian.

**Architecture:** Paket `api` (net/http ServeMux Go 1.22+) memakai `store` (CRUD + validasi), `auth` (bcrypt + sesi di memori), `scheduler` (status + bel manual), `player`, `backup` (snapshot `VACUUM INTO`, zip, restore tertunda yang diterapkan saat start), dan `sistem` (NTP, ffprobe). Frontend sementara berupa halaman placeholder di `internal/webui/dist` (diganti Rencana 4). Docker hanya untuk development: image Debian trixie berisi mpv/ffmpeg dengan ALSA `null`.

**Tech Stack:** Go 1.27 standard library, `golang.org/x/crypto/bcrypt`, `modernc.org/sqlite`, Docker + Compose, `ffprobe`, `mpv`.

**Spec:** `docs/superpowers/specs/2026-10-15-bel-sekolah-design.md` (terutama §9–§14)

## Global Constraints

- Module path `belsekolah`; tanpa cgo (`CGO_ENABLED=0` harus bisa build).
- Semua teks yang terlihat pengguna (pesan error API, log) berbahasa Indonesia.
- API di bawah `/api`, JSON camelCase; error berbentuk `{"error": "<pesan>"}` dengan kode 400 validasi, 401 belum login, 403 wajib ganti password / origin asing, 404 tidak ada, 409 dipakai/duplikat/tumpang tindih, 429 login terkunci, 500 lainnya.
- Setiap perubahan data yang sukses memanggil `Penjadwal.Reload()`.
- Login: 5 gagal per IP → kunci 5 menit; sesi di memori, kedaluwarsa 7 hari sejak terakhir dipakai; cookie `sesi` `HttpOnly`, `SameSite=Strict`, `Path=/`.
- Password minimal 8 karakter, maksimal 72 byte; password awal acak ditulis ke `<data>/password-awal.txt` (0600) dan dihapus setelah diganti.
- Upload audio: hanya `.mp3`/`.wav`, maks 20 MB, divalidasi ffprobe, nama file dibuat server (`upload-<hex>.<ext>`).
- Path file audio selalu `filepath.Join(audioDir, filepath.Base(a.File))`.
- Restore: validasi zip → folder `restore-tertunda` → aplikasi keluar kode 3 → diterapkan saat start sebelum database dibuka.
- Backup harian: `<data>/backup/bel-YYYY-MM-DD.db`, simpan 7 terakhir.
- Hari 1=Senin…7=Minggu; tanggal `YYYY-MM-DD`; jam `HH:MM`.

## Paralelisasi

- Gelombang 1 (paralel): Task 1, 2, 3, 4, 5
- Gelombang 2: Task 6 (butuh 1, 2, 3, 5)
- Gelombang 3: Task 7 (butuh 4, 6)
- Gelombang 4: Task 8 (butuh semua)

## Struktur File

```
internal/store/errors.go          (baru)  error domain + validasi umum
internal/store/jadwal.go          (ubah)  CRUD profil/jadwal/periode/libur, profil default
internal/store/audio.go           (ubah)  CRUD audio & jenis bel, validasi urutan
internal/store/riwayat.go         (ubah)  CariRiwayat
internal/store/pengaturan.go      (baru)  volume
internal/store/backup.go          (baru)  BackupKe (Task 4)
internal/store/crud_test.go       (baru)
internal/auth/auth.go, auth_test.go
internal/scheduler/scheduler.go   (ubah)  Status, BunyikanManual, FileUrutan, log tidak spam
internal/scheduler/status_test.go (baru)
internal/backup/backup.go, backup_test.go
internal/sistem/ntp.go, ffprobe.go, ffprobe_test.go
internal/webui/webui.go, dist/index.html
internal/api/server.go, respon.go, dto.go, sesi.go, beranda.go, pengaturan.go, rute_data.go
internal/api/profil.go, jadwal.go, jenisbel.go, audio.go, riwayat.go, backup.go
internal/api/bantu_test.go, inti_test.go, data_test.go
cmd/belsekolah/main.go            (ubah)
Dockerfile, compose.yaml, .dockerignore, deploy/docker/asound.conf
e2e/e2e_test.go, scripts/e2e.sh
```

---

### Task 1: `store` — CRUD lengkap, validasi, error domain

**Files:**
- Create: `internal/store/errors.go`, `internal/store/pengaturan.go`, `internal/store/crud_test.go`
- Modify: `internal/store/jadwal.go`, `internal/store/audio.go`, `internal/store/riwayat.go`

**Interfaces:**
- Consumes: kode store yang ada (`Store`, `insert`, `GetSetting`, `SetSetting`, `scanAudio`, `kolomAudio`, `validasiRentang`, `semuaJadwal/semuaPeriode/semuaLibur`, `ErrTidakAda`, `ErrPeriodeTumpangTindih`, `KunciProfilDefault`), `schedule.*`.
- Produces:
  - `var ErrDipakai, ErrDuplikat error`; `type ErrorValidasi struct{ Pesan string }` (method `Error() string`, dipakai sebagai `*ErrorValidasi`)
  - `const KunciVolume = "volume"`; `func (*Store) Volume(ctx) (int, error)` (default 100); `SetVolume(ctx, v int) error`
  - `type Profil struct{ ID int64; Nama string }`; `ListProfil(ctx) ([]Profil, error)`; `GetProfil(ctx, id) (Profil, error)`; `UpdateProfil(ctx, id int64, nama string) error`; `DeleteProfil(ctx, id) error`; `ProfilDefault(ctx) (int64, error)`; `SetProfilDefault(ctx, id int64) error`
  - `ListJadwal(ctx, profilID int64) ([]schedule.Jadwal, error)`; `GetJadwal(ctx, id) (schedule.Jadwal, error)`; `UpdateJadwal(ctx, j schedule.Jadwal) error`; `DeleteJadwal(ctx, id) error`; `SalinHari(ctx, profilID int64, dari, ke int) error`
  - `ListPeriode(ctx) ([]schedule.Periode, error)`; `GetPeriode(ctx, id)`; `UpdatePeriode(ctx, p) error`; `DeletePeriode(ctx, id) error`
  - `ListLibur(ctx) ([]schedule.Libur, error)`; `GetLibur(ctx, id)`; `UpdateLibur(ctx, l) error`; `DeleteLibur(ctx, id) error`
  - `ListAudio(ctx) ([]Audio, error)`; `UpdateNamaAudio(ctx, id int64, nama string) error`; `DeleteAudio(ctx, id) (Audio, error)`
  - `GetJenisBel(ctx, id) (JenisBel, error)`; `UpdateJenisBel(ctx, id int64, nama, warna string) error`; `DeleteJenisBel(ctx, id) error`
  - `type FilterRiwayat struct{ Tanggal, Status string; Batas int }`; `CariRiwayat(ctx, f FilterRiwayat) ([]Riwayat, error)`
  - Perilaku baru pada fungsi lama: `CreateProfil`, `CreateJenisBel`, `CreateAudio`, `CreateJadwal`, `CreatePeriode`, `CreateLibur`, `SetUrutanBel` memvalidasi input dan mengembalikan `*ErrorValidasi`/`ErrDuplikat` sesuai aturan di bawah.

Aturan:
- Nama (profil, jenis bel, audio): di-trim, tidak kosong, maks 100 karakter.
- Warna jenis bel: `#RRGGBB`.
- Label jadwal: di-trim, maks 100 karakter. Keterangan periode/libur: di-trim, maks 200 karakter.
- Urutan bel: 1–10 audio; audio yang tidak ada → `*ErrorValidasi` "suara tidak ditemukan".
- Pelanggaran UNIQUE → `ErrDuplikat`. Foreign key saat insert/update → `*ErrorValidasi` (referensi tidak ditemukan); saat delete → `ErrDipakai` dengan alasan.
- Profil default tidak bisa dihapus; audio `bawaan` tidak bisa dihapus; profil yang dipakai periode tidak bisa dihapus; jenis bel yang dipakai jadwal tidak bisa dihapus; audio yang dipakai jenis bel tidak bisa dihapus.
- Update/Delete pada ID yang tidak ada → `ErrTidakAda`.

- [ ] **Step 1: Tulis test yang gagal**

`internal/store/crud_test.go`:

```go
package store

import (
	"errors"
	"testing"
	"time"

	"belsekolah/internal/schedule"
)

func harapValidasi(t *testing.T, err error) {
	t.Helper()
	var v *ErrorValidasi
	if !errors.As(err, &v) {
		t.Fatalf("err = %v (%T), mau *ErrorValidasi", err, err)
	}
	if v.Pesan == "" {
		t.Fatal("pesan validasi kosong")
	}
}

func dataDasar(t *testing.T, s *Store) (profil, jenis, audio int64) {
	t.Helper()
	profil = harus(s.CreateProfil(ctx, "Normal"))(t)
	if err := s.SetProfilDefault(ctx, profil); err != nil {
		t.Fatal(err)
	}
	audio = harus(s.CreateAudio(ctx, Audio{Nama: "Ding", Sumber: "bawaan", File: "ding.wav", Dibuat: time.Now()}))(t)
	jenis = harus(s.CreateJenisBel(ctx, "Masuk", "#2563eb"))(t)
	if err := s.SetUrutanBel(ctx, jenis, []int64{audio}); err != nil {
		t.Fatal(err)
	}
	return
}

func TestValidasiNamaDanWarna(t *testing.T) {
	s := bukaStore(t)
	_, err := s.CreateProfil(ctx, "   ")
	harapValidasi(t, err)
	_, err = s.CreateJenisBel(ctx, "Masuk", "biru")
	harapValidasi(t, err)
	_, err = s.CreateJenisBel(ctx, string(make([]rune, 101)), "#000000")
	harapValidasi(t, err)
	id := harus(s.CreateProfil(ctx, "  Ujian  "))(t)
	if p := harus(s.GetProfil(ctx, id))(t); p.Nama != "Ujian" {
		t.Errorf("nama = %q, mau di-trim", p.Nama)
	}
	if _, err := s.CreateProfil(ctx, "Ujian"); !errors.Is(err, ErrDuplikat) {
		t.Errorf("err = %v, mau ErrDuplikat", err)
	}
}

func TestProfilCRUD(t *testing.T) {
	s := bukaStore(t)
	normal, _, _ := dataDasar(t, s)
	ujian := harus(s.CreateProfil(ctx, "Ujian"))(t)

	if err := s.UpdateProfil(ctx, ujian, "Ujian Akhir"); err != nil {
		t.Fatal(err)
	}
	daftar := harus(s.ListProfil(ctx))(t)
	if len(daftar) != 2 || daftar[1].Nama != "Ujian Akhir" {
		t.Fatalf("ListProfil = %+v", daftar)
	}
	if err := s.UpdateProfil(ctx, 999, "X"); !errors.Is(err, ErrTidakAda) {
		t.Errorf("update ID tak ada: %v", err)
	}
	if err := s.DeleteProfil(ctx, normal); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus profil default: %v, mau ErrDipakai", err)
	}
	harus(s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-10-19", Selesai: "2026-10-23", ProfilID: ujian}))(t)
	if err := s.DeleteProfil(ctx, ujian); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus profil yang dipakai periode: %v, mau ErrDipakai", err)
	}
	if def := harus(s.ProfilDefault(ctx))(t); def != normal {
		t.Errorf("ProfilDefault = %d, mau %d", def, normal)
	}
	if err := s.SetProfilDefault(ctx, 999); !errors.Is(err, ErrTidakAda) {
		t.Errorf("SetProfilDefault tak ada: %v", err)
	}
	if err := s.DeleteProfil(ctx, 999); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus ID tak ada: %v", err)
	}
}

func TestProfilDefaultBelumDiatur(t *testing.T) {
	s := bukaStore(t)
	if _, err := s.ProfilDefault(ctx); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v, mau ErrTidakAda", err)
	}
}

func TestJadwalCRUDDanSalin(t *testing.T) {
	s := bukaStore(t)
	profil, jenis, _ := dataDasar(t, s)
	senin1 := harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "07:00", JenisBelID: jenis, Label: " Jam ke-1 "}))(t)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "09:30", JenisBelID: jenis, Label: "Istirahat"}))(t)
	harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 2, Jam: "10:00", JenisBelID: jenis, Label: "Lama"}))(t)

	j := harus(s.GetJadwal(ctx, senin1))(t)
	if j.Label != "Jam ke-1" {
		t.Errorf("label = %q, mau di-trim", j.Label)
	}
	j.Jam = "07:15"
	if err := s.UpdateJadwal(ctx, j); err != nil {
		t.Fatal(err)
	}
	j.Jam = "9:00"
	harapValidasi(t, s.UpdateJadwal(ctx, j))
	j.Jam, j.JenisBelID = "07:15", 999
	harapValidasi(t, s.UpdateJadwal(ctx, j))

	_, err := s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: 1, Jam: "09:30", JenisBelID: jenis})
	if !errors.Is(err, ErrDuplikat) {
		t.Errorf("jadwal ganda: %v, mau ErrDuplikat", err)
	}
	_, err = s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: 999, Hari: 1, Jam: "11:00", JenisBelID: jenis})
	harapValidasi(t, err)

	if err := s.SalinHari(ctx, profil, 1, 2); err != nil {
		t.Fatal(err)
	}
	var selasa []string
	for _, x := range harus(s.ListJadwal(ctx, profil))(t) {
		if x.Hari == 2 {
			selasa = append(selasa, x.Jam+" "+x.Label)
		}
	}
	if len(selasa) != 2 || selasa[0] != "07:15 Jam ke-1" || selasa[1] != "09:30 Istirahat" {
		t.Fatalf("Selasa setelah salin = %v (jadwal lama harus diganti)", selasa)
	}
	harapValidasi(t, s.SalinHari(ctx, profil, 1, 1))
	harapValidasi(t, s.SalinHari(ctx, profil, 0, 2))
	if err := s.SalinHari(ctx, 999, 1, 2); !errors.Is(err, ErrTidakAda) {
		t.Errorf("salin profil tak ada: %v", err)
	}

	if err := s.DeleteJadwal(ctx, senin1); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteJadwal(ctx, senin1); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus dua kali: %v", err)
	}
	if err := s.DeleteJenisBel(ctx, jenis); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus jenis bel yang dipakai jadwal: %v, mau ErrDipakai", err)
	}
}

func TestPeriodeDanLiburCRUD(t *testing.T) {
	s := bukaStore(t)
	profil, _, _ := dataDasar(t, s)
	a := harus(s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-10-19", Selesai: "2026-10-23", ProfilID: profil, Keterangan: "PTS"}))(t)
	b := harus(s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-11-02", Selesai: "2026-11-06", ProfilID: profil}))(t)

	p := harus(s.GetPeriode(ctx, a))(t)
	p.Selesai = "2026-10-24" // memperpanjang diri sendiri tidak dianggap tumpang tindih
	if err := s.UpdatePeriode(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.Selesai = "2026-11-03"
	if err := s.UpdatePeriode(ctx, p); !errors.Is(err, ErrPeriodeTumpangTindih) {
		t.Errorf("update menabrak periode lain: %v", err)
	}
	_, err := s.CreatePeriode(ctx, schedule.Periode{Mulai: "2026-12-01", Selesai: "2026-12-02", ProfilID: 999})
	harapValidasi(t, err)
	if err := s.DeletePeriode(ctx, b); err != nil {
		t.Fatal(err)
	}
	if n := len(harus(s.ListPeriode(ctx))(t)); n != 1 {
		t.Errorf("jumlah periode = %d, mau 1", n)
	}

	l := harus(s.CreateLibur(ctx, schedule.Libur{Mulai: "2026-12-25", Selesai: "2026-12-25", Keterangan: " Natal "}))(t)
	libur := harus(s.GetLibur(ctx, l))(t)
	if libur.Keterangan != "Natal" {
		t.Errorf("keterangan = %q", libur.Keterangan)
	}
	libur.Selesai = "2026-12-26"
	if err := s.UpdateLibur(ctx, libur); err != nil {
		t.Fatal(err)
	}
	libur.Mulai = "2026-12-30"
	harapValidasi(t, s.UpdateLibur(ctx, libur))
	if err := s.DeleteLibur(ctx, l); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteLibur(ctx, l); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus libur dua kali: %v", err)
	}
	_, err = s.CreateLibur(ctx, schedule.Libur{Mulai: "2026-12-25", Selesai: "2026-12-25", Keterangan: string(make([]rune, 201))})
	harapValidasi(t, err)
}

func TestAudioDanJenisBelCRUD(t *testing.T) {
	s := bukaStore(t)
	_, jenis, bawaan := dataDasar(t, s)
	up := harus(s.CreateAudio(ctx, Audio{Nama: "Lagu", Sumber: "upload", File: "upload-1.mp3", Dibuat: time.Now()}))(t)
	_, err := s.CreateAudio(ctx, Audio{Nama: " ", Sumber: "upload", File: "x.mp3", Dibuat: time.Now()})
	harapValidasi(t, err)

	if err := s.UpdateNamaAudio(ctx, up, "Lagu Wajib"); err != nil {
		t.Fatal(err)
	}
	if n := len(harus(s.ListAudio(ctx))(t)); n != 2 {
		t.Errorf("ListAudio = %d, mau 2", n)
	}
	if _, err := s.DeleteAudio(ctx, bawaan); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus audio bawaan: %v, mau ErrDipakai", err)
	}
	if err := s.SetUrutanBel(ctx, jenis, []int64{up}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DeleteAudio(ctx, up); !errors.Is(err, ErrDipakai) {
		t.Errorf("hapus audio yang dipakai: %v, mau ErrDipakai", err)
	}
	harapValidasi(t, s.SetUrutanBel(ctx, jenis, nil))
	harapValidasi(t, s.SetUrutanBel(ctx, jenis, []int64{999}))
	harapValidasi(t, s.SetUrutanBel(ctx, jenis, make([]int64, 11)))
	if err := s.SetUrutanBel(ctx, jenis, []int64{bawaan}); err != nil {
		t.Fatal(err)
	}
	a := harus(s.DeleteAudio(ctx, up))(t)
	if a.File != "upload-1.mp3" {
		t.Errorf("DeleteAudio mengembalikan %+v", a)
	}

	if err := s.UpdateJenisBel(ctx, jenis, "Masuk Kelas", "#ff0000"); err != nil {
		t.Fatal(err)
	}
	j := harus(s.GetJenisBel(ctx, jenis))(t)
	if j.Nama != "Masuk Kelas" || j.Warna != "#ff0000" {
		t.Errorf("jenis bel = %+v", j)
	}
	harapValidasi(t, s.UpdateJenisBel(ctx, jenis, "Masuk", "merah"))
	if _, err := s.GetJenisBel(ctx, 999); !errors.Is(err, ErrTidakAda) {
		t.Errorf("GetJenisBel tak ada: %v", err)
	}
	baru := harus(s.CreateJenisBel(ctx, "Pulang", "#9333ea"))(t)
	if err := s.DeleteJenisBel(ctx, baru); err != nil {
		t.Fatal(err)
	}
}

func TestVolume(t *testing.T) {
	s := bukaStore(t)
	if v := harus(s.Volume(ctx))(t); v != 100 {
		t.Errorf("volume default = %d, mau 100", v)
	}
	if err := s.SetVolume(ctx, 65); err != nil {
		t.Fatal(err)
	}
	if v := harus(s.Volume(ctx))(t); v != 65 {
		t.Errorf("volume = %d, mau 65", v)
	}
	harapValidasi(t, s.SetVolume(ctx, 101))
	harapValidasi(t, s.SetVolume(ctx, -1))
}

func TestCariRiwayat(t *testing.T) {
	s := bukaStore(t)
	wib := time.FixedZone("WIB", 7*3600)
	for i, st := range []string{"berhasil", "gagal", "berhasil"} {
		w := time.Date(2026, 10, 12+i/2, 7+i, 0, 0, 0, wib)
		if err := s.InsertRiwayat(ctx, Riwayat{Waktu: w, Sumber: SumberManual, Status: st, Label: st}); err != nil {
			t.Fatal(err)
		}
	}
	semua := harus(s.CariRiwayat(ctx, FilterRiwayat{}))(t)
	if len(semua) != 3 {
		t.Fatalf("semua = %d, mau 3", len(semua))
	}
	tgl := harus(s.CariRiwayat(ctx, FilterRiwayat{Tanggal: "2026-10-12"}))(t)
	if len(tgl) != 2 {
		t.Errorf("tanggal 12 = %d, mau 2", len(tgl))
	}
	gagal := harus(s.CariRiwayat(ctx, FilterRiwayat{Status: "gagal"}))(t)
	if len(gagal) != 1 || gagal[0].Status != "gagal" {
		t.Errorf("status gagal = %+v", gagal)
	}
	if n := len(harus(s.CariRiwayat(ctx, FilterRiwayat{Batas: 1}))(t)); n != 1 {
		t.Errorf("batas 1 = %d", n)
	}
	_, err := s.CariRiwayat(ctx, FilterRiwayat{Tanggal: "12-10-2026"})
	harapValidasi(t, err)
	_, err = s.CariRiwayat(ctx, FilterRiwayat{Status: "aneh"})
	harapValidasi(t, err)
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `go test ./internal/store/`
Expected: FAIL — `undefined: ErrorValidasi`, `SetProfilDefault`, dll.

- [ ] **Step 3: Tulis `errors.go`**

```go
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	ErrDipakai  = errors.New("data masih dipakai")
	ErrDuplikat = errors.New("data dengan nama atau waktu yang sama sudah ada")
)

// ErrorValidasi menandai input yang tidak valid; Pesan aman ditampilkan ke pengguna.
type ErrorValidasi struct {
	Pesan string
}

func (e *ErrorValidasi) Error() string { return e.Pesan }

func validasi(format string, a ...any) error {
	return &ErrorValidasi{Pesan: fmt.Sprintf(format, a...)}
}

// dipakai membuat error ErrDipakai beserta alasannya.
func dipakai(alasan string) error {
	return fmt.Errorf("%w: %s", ErrDipakai, alasan)
}

// terjemahkan mengubah error constraint SQLite menjadi error domain.
// errFK dikembalikan untuk pelanggaran foreign key; nil berarti kembalikan err apa adanya.
func terjemahkan(err, errFK error) error {
	if err == nil {
		return nil
	}
	pesan := err.Error()
	switch {
	case strings.Contains(pesan, "UNIQUE constraint failed"):
		return ErrDuplikat
	case strings.Contains(pesan, "FOREIGN KEY constraint failed"):
		if errFK == nil {
			return err
		}
		return errFK
	}
	return err
}

// cekDiubah mengembalikan ErrTidakAda jika perintah tidak mengenai baris mana pun.
func cekDiubah(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTidakAda
	}
	return nil
}

func validasiTeks(nilai, nama string, maks int, wajib bool) (string, error) {
	nilai = strings.TrimSpace(nilai)
	if wajib && nilai == "" {
		return "", validasi("%s tidak boleh kosong", nama)
	}
	if utf8.RuneCountInString(nilai) > maks {
		return "", validasi("%s maksimal %d karakter", nama, maks)
	}
	return nilai, nil
}

func validasiNama(nama string) (string, error) {
	return validasiTeks(nama, "nama", 100, true)
}

var polaWarna = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func validasiWarna(w string) error {
	if !polaWarna.MatchString(w) {
		return validasi("warna harus berformat #RRGGBB")
	}
	return nil
}
```

- [ ] **Step 4: Ubah `validasiRentang`, Create*, dan tambah CRUD di `jadwal.go`**

Ganti `CreateProfil`, `CreateJadwal`, `validasiRentang`, `CreatePeriode`, `CreateLibur` dan tambahkan fungsi baru. Isi lengkap bagian yang berubah/baru (fungsi `LoadDataJadwal`, `semuaJadwal`, `semuaPeriode`, `semuaLibur`, `CountProfil` tetap):

```go
type Profil struct {
	ID   int64
	Nama string
}

func (s *Store) CreateProfil(ctx context.Context, nama string) (int64, error) {
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	id, err := s.insert(ctx, `INSERT INTO profil (nama) VALUES (?)`, nama)
	return id, terjemahkan(err, nil)
}

func (s *Store) ListProfil(ctx context.Context) ([]Profil, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, nama FROM profil ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Profil
	for rows.Next() {
		var p Profil
		if err := rows.Scan(&p.ID, &p.Nama); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetProfil(ctx context.Context, id int64) (Profil, error) {
	var p Profil
	err := s.db.QueryRowContext(ctx, `SELECT id, nama FROM profil WHERE id = ?`, id).Scan(&p.ID, &p.Nama)
	if errors.Is(err, sql.ErrNoRows) {
		return Profil{}, ErrTidakAda
	}
	return p, err
}

func (s *Store) UpdateProfil(ctx context.Context, id int64, nama string) error {
	nama, err := validasiNama(nama)
	if err != nil {
		return err
	}
	return terjemahkan(cekDiubah(s.db.ExecContext(ctx, `UPDATE profil SET nama = ? WHERE id = ?`, nama, id)), nil)
}

// DeleteProfil menolak profil default dan profil yang dipakai periode khusus.
// Jadwal milik profil ikut terhapus.
func (s *Store) DeleteProfil(ctx context.Context, id int64) error {
	def, err := s.ProfilDefault(ctx)
	if err != nil && !errors.Is(err, ErrTidakAda) {
		return err
	}
	if def == id {
		return dipakai("profil default tidak bisa dihapus")
	}
	return terjemahkan(
		cekDiubah(s.db.ExecContext(ctx, `DELETE FROM profil WHERE id = ?`, id)),
		dipakai("profil masih dipakai oleh periode khusus"))
}

// ProfilDefault mengembalikan ErrTidakAda jika belum diatur.
func (s *Store) ProfilDefault(ctx context.Context) (int64, error) {
	nilai, err := s.GetSetting(ctx, KunciProfilDefault)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(nilai, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("profil default tidak valid %q: %w", nilai, err)
	}
	return id, nil
}

func (s *Store) SetProfilDefault(ctx context.Context, id int64) error {
	if _, err := s.GetProfil(ctx, id); err != nil {
		return err
	}
	return s.SetSetting(ctx, KunciProfilDefault, strconv.FormatInt(id, 10))
}

func validasiJadwal(j schedule.Jadwal) (schedule.Jadwal, error) {
	if _, _, err := schedule.ParseJam(j.Jam); err != nil {
		return j, validasi("%s", err.Error())
	}
	if j.Hari < 1 || j.Hari > 7 {
		return j, validasi("hari harus 1 (Senin) sampai 7 (Minggu), bukan %d", j.Hari)
	}
	label, err := validasiTeks(j.Label, "label", 100, false)
	if err != nil {
		return j, err
	}
	j.Label = label
	return j, nil
}

var errRefJadwal = validasi("profil atau jenis bel tidak ditemukan")

func (s *Store) CreateJadwal(ctx context.Context, j schedule.Jadwal) (int64, error) {
	j, err := validasiJadwal(j)
	if err != nil {
		return 0, err
	}
	id, err := s.insert(ctx,
		`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label) VALUES (?, ?, ?, ?, ?)`,
		j.ProfilID, j.Hari, j.Jam, j.JenisBelID, j.Label)
	return id, terjemahkan(err, errRefJadwal)
}

const kolomJadwal = `id, profil_id, hari, jam, jenis_bel_id, label`

func scanJadwal(r pemindai) (schedule.Jadwal, error) {
	var j schedule.Jadwal
	err := r.Scan(&j.ID, &j.ProfilID, &j.Hari, &j.Jam, &j.JenisBelID, &j.Label)
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrTidakAda
	}
	return j, err
}

func (s *Store) ListJadwal(ctx context.Context, profilID int64) ([]schedule.Jadwal, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+kolomJadwal+` FROM jadwal WHERE profil_id = ? ORDER BY hari, jam`, profilID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []schedule.Jadwal
	for rows.Next() {
		j, err := scanJadwal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) GetJadwal(ctx context.Context, id int64) (schedule.Jadwal, error) {
	return scanJadwal(s.db.QueryRowContext(ctx, `SELECT `+kolomJadwal+` FROM jadwal WHERE id = ?`, id))
}

func (s *Store) UpdateJadwal(ctx context.Context, j schedule.Jadwal) error {
	j, err := validasiJadwal(j)
	if err != nil {
		return err
	}
	return terjemahkan(cekDiubah(s.db.ExecContext(ctx,
		`UPDATE jadwal SET profil_id = ?, hari = ?, jam = ?, jenis_bel_id = ?, label = ? WHERE id = ?`,
		j.ProfilID, j.Hari, j.Jam, j.JenisBelID, j.Label, j.ID)), errRefJadwal)
}

func (s *Store) DeleteJadwal(ctx context.Context, id int64) error {
	return cekDiubah(s.db.ExecContext(ctx, `DELETE FROM jadwal WHERE id = ?`, id))
}

// SalinHari mengganti seluruh jadwal hari `ke` dengan salinan jadwal hari `dari` pada profil yang sama.
func (s *Store) SalinHari(ctx context.Context, profilID int64, dari, ke int) error {
	if dari < 1 || dari > 7 || ke < 1 || ke > 7 {
		return validasi("hari harus 1 (Senin) sampai 7 (Minggu)")
	}
	if dari == ke {
		return validasi("hari asal dan hari tujuan tidak boleh sama")
	}
	if _, err := s.GetProfil(ctx, profilID); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM jadwal WHERE profil_id = ? AND hari = ?`, profilID, ke); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label)
		 SELECT profil_id, ?, jam, jenis_bel_id, label FROM jadwal WHERE profil_id = ? AND hari = ?`,
		ke, profilID, dari); err != nil {
		return err
	}
	return tx.Commit()
}

func validasiRentang(mulai, selesai string) error {
	a, err := time.Parse(schedule.FormatTanggal, mulai)
	if err != nil {
		return validasi("tanggal mulai tidak valid %q, gunakan format YYYY-MM-DD", mulai)
	}
	b, err := time.Parse(schedule.FormatTanggal, selesai)
	if err != nil {
		return validasi("tanggal selesai tidak valid %q, gunakan format YYYY-MM-DD", selesai)
	}
	if b.Before(a) {
		return validasi("tanggal selesai tidak boleh sebelum tanggal mulai")
	}
	return nil
}

func validasiKeterangan(k string) (string, error) {
	return validasiTeks(k, "keterangan", 200, false)
}

var errRefProfil = validasi("profil tidak ditemukan")

// simpanPeriode menjalankan insert (id == 0) atau update dalam satu transaksi bersama cek tumpang tindih.
func (s *Store) simpanPeriode(ctx context.Context, p schedule.Periode) (int64, error) {
	if err := validasiRentang(p.Mulai, p.Selesai); err != nil {
		return 0, err
	}
	ket, err := validasiKeterangan(p.Keterangan)
	if err != nil {
		return 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM periode_khusus WHERE tgl_mulai <= ? AND tgl_selesai >= ? AND id != ?`,
		p.Selesai, p.Mulai, p.ID).Scan(&n); err != nil {
		return 0, err
	}
	if n > 0 {
		return 0, ErrPeriodeTumpangTindih
	}
	id := p.ID
	if id == 0 {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO periode_khusus (tgl_mulai, tgl_selesai, profil_id, keterangan) VALUES (?, ?, ?, ?)`,
			p.Mulai, p.Selesai, p.ProfilID, ket)
		if err != nil {
			return 0, terjemahkan(err, errRefProfil)
		}
		if id, err = res.LastInsertId(); err != nil {
			return 0, err
		}
	} else {
		err := cekDiubah(tx.ExecContext(ctx,
			`UPDATE periode_khusus SET tgl_mulai = ?, tgl_selesai = ?, profil_id = ?, keterangan = ? WHERE id = ?`,
			p.Mulai, p.Selesai, p.ProfilID, ket, p.ID))
		if err != nil {
			return 0, terjemahkan(err, errRefProfil)
		}
	}
	return id, tx.Commit()
}

// CreatePeriode menolak periode yang bertabrakan dengan periode lain.
func (s *Store) CreatePeriode(ctx context.Context, p schedule.Periode) (int64, error) {
	p.ID = 0
	return s.simpanPeriode(ctx, p)
}

// UpdatePeriode menolak perubahan yang membuat periode bertabrakan dengan periode lain.
func (s *Store) UpdatePeriode(ctx context.Context, p schedule.Periode) error {
	if p.ID == 0 {
		return ErrTidakAda
	}
	_, err := s.simpanPeriode(ctx, p)
	return err
}

func (s *Store) GetPeriode(ctx context.Context, id int64) (schedule.Periode, error) {
	var p schedule.Periode
	err := s.db.QueryRowContext(ctx,
		`SELECT id, tgl_mulai, tgl_selesai, profil_id, keterangan FROM periode_khusus WHERE id = ?`, id).
		Scan(&p.ID, &p.Mulai, &p.Selesai, &p.ProfilID, &p.Keterangan)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrTidakAda
	}
	return p, err
}

func (s *Store) ListPeriode(ctx context.Context) ([]schedule.Periode, error) {
	return s.semuaPeriode(ctx)
}

func (s *Store) DeletePeriode(ctx context.Context, id int64) error {
	return cekDiubah(s.db.ExecContext(ctx, `DELETE FROM periode_khusus WHERE id = ?`, id))
}

func (s *Store) CreateLibur(ctx context.Context, l schedule.Libur) (int64, error) {
	if err := validasiRentang(l.Mulai, l.Selesai); err != nil {
		return 0, err
	}
	ket, err := validasiKeterangan(l.Keterangan)
	if err != nil {
		return 0, err
	}
	return s.insert(ctx,
		`INSERT INTO libur (tgl_mulai, tgl_selesai, keterangan) VALUES (?, ?, ?)`,
		l.Mulai, l.Selesai, ket)
}

func (s *Store) GetLibur(ctx context.Context, id int64) (schedule.Libur, error) {
	var l schedule.Libur
	err := s.db.QueryRowContext(ctx,
		`SELECT id, tgl_mulai, tgl_selesai, keterangan FROM libur WHERE id = ?`, id).
		Scan(&l.ID, &l.Mulai, &l.Selesai, &l.Keterangan)
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrTidakAda
	}
	return l, err
}

func (s *Store) ListLibur(ctx context.Context) ([]schedule.Libur, error) {
	return s.semuaLibur(ctx)
}

func (s *Store) UpdateLibur(ctx context.Context, l schedule.Libur) error {
	if err := validasiRentang(l.Mulai, l.Selesai); err != nil {
		return err
	}
	ket, err := validasiKeterangan(l.Keterangan)
	if err != nil {
		return err
	}
	return cekDiubah(s.db.ExecContext(ctx,
		`UPDATE libur SET tgl_mulai = ?, tgl_selesai = ?, keterangan = ? WHERE id = ?`,
		l.Mulai, l.Selesai, ket, l.ID))
}

func (s *Store) DeleteLibur(ctx context.Context, id int64) error {
	return cekDiubah(s.db.ExecContext(ctx, `DELETE FROM libur WHERE id = ?`, id))
}
```

Tambahkan import `database/sql` di `jadwal.go`. `LoadDataJadwal` tetap seperti sekarang (masih memakai `GetSetting` langsung). Hapus versi lama `CreateProfil`, `CreateJadwal`, `validasiRentang`, `CreatePeriode`, `CreateLibur`.

- [ ] **Step 5: Ubah `audio.go`**

Ganti `CreateAudio`, `CreateJenisBel`, `SetUrutanBel` dan tambah fungsi baru:

```go
func (s *Store) CreateAudio(ctx context.Context, a Audio) (int64, error) {
	nama, err := validasiNama(a.Nama)
	if err != nil {
		return 0, err
	}
	return s.insert(ctx,
		`INSERT INTO audio (nama, sumber, file, durasi_ms, teks_tts, suara_tts, dibuat)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		nama, a.Sumber, a.File, a.DurasiMS, a.TeksTTS, a.SuaraTTS, a.Dibuat.Format(time.RFC3339))
}

func (s *Store) ListAudio(ctx context.Context) ([]Audio, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+kolomAudio+` FROM audio a ORDER BY a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Audio
	for rows.Next() {
		a, err := scanAudio(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) UpdateNamaAudio(ctx context.Context, id int64, nama string) error {
	nama, err := validasiNama(nama)
	if err != nil {
		return err
	}
	return cekDiubah(s.db.ExecContext(ctx, `UPDATE audio SET nama = ? WHERE id = ?`, nama, id))
}

// DeleteAudio menghapus baris audio dan mengembalikannya agar pemanggil bisa menghapus filenya.
func (s *Store) DeleteAudio(ctx context.Context, id int64) (Audio, error) {
	a, err := s.GetAudio(ctx, id)
	if err != nil {
		return Audio{}, err
	}
	if a.Sumber == "bawaan" {
		return Audio{}, dipakai("suara bawaan tidak bisa dihapus")
	}
	err = terjemahkan(
		cekDiubah(s.db.ExecContext(ctx, `DELETE FROM audio WHERE id = ?`, id)),
		dipakai("suara masih dipakai oleh jenis bel"))
	if err != nil {
		return Audio{}, err
	}
	return a, nil
}

func (s *Store) CreateJenisBel(ctx context.Context, nama, warna string) (int64, error) {
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	if err := validasiWarna(warna); err != nil {
		return 0, err
	}
	id, err := s.insert(ctx, `INSERT INTO jenis_bel (nama, warna) VALUES (?, ?)`, nama, warna)
	return id, terjemahkan(err, nil)
}

func (s *Store) GetJenisBel(ctx context.Context, id int64) (JenisBel, error) {
	var j JenisBel
	err := s.db.QueryRowContext(ctx, `SELECT id, nama, warna FROM jenis_bel WHERE id = ?`, id).
		Scan(&j.ID, &j.Nama, &j.Warna)
	if errors.Is(err, sql.ErrNoRows) {
		return j, ErrTidakAda
	}
	return j, err
}

func (s *Store) UpdateJenisBel(ctx context.Context, id int64, nama, warna string) error {
	nama, err := validasiNama(nama)
	if err != nil {
		return err
	}
	if err := validasiWarna(warna); err != nil {
		return err
	}
	return terjemahkan(cekDiubah(s.db.ExecContext(ctx,
		`UPDATE jenis_bel SET nama = ?, warna = ? WHERE id = ?`, nama, warna, id)), nil)
}

func (s *Store) DeleteJenisBel(ctx context.Context, id int64) error {
	return terjemahkan(
		cekDiubah(s.db.ExecContext(ctx, `DELETE FROM jenis_bel WHERE id = ?`, id)),
		dipakai("jenis bel masih dipakai di jadwal"))
}

// SetUrutanBel mengganti seluruh urutan audio milik sebuah jenis bel (1–10 audio).
func (s *Store) SetUrutanBel(ctx context.Context, jenisBelID int64, audioIDs []int64) error {
	if len(audioIDs) == 0 {
		return validasi("pilih minimal satu suara")
	}
	if len(audioIDs) > 10 {
		return validasi("maksimal 10 suara dalam satu jenis bel")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM urutan_bel WHERE jenis_bel_id = ?`, jenisBelID); err != nil {
		return err
	}
	for i, audioID := range audioIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO urutan_bel (jenis_bel_id, urutan, audio_id) VALUES (?, ?, ?)`,
			jenisBelID, i+1, audioID); err != nil {
			return terjemahkan(err, validasi("suara atau jenis bel tidak ditemukan"))
		}
	}
	return tx.Commit()
}
```

`terjemahkan(err, nil)` berarti pelanggaran foreign key dikembalikan apa adanya (tidak mungkin terjadi di sana).

- [ ] **Step 6: `pengaturan.go` dan `CariRiwayat`**

`internal/store/pengaturan.go`:

```go
package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// KunciVolume menyimpan volume pemutaran 0–100.
const KunciVolume = "volume"

// Volume mengembalikan volume tersimpan, atau 100 jika belum pernah diatur.
func (s *Store) Volume(ctx context.Context) (int, error) {
	nilai, err := s.GetSetting(ctx, KunciVolume)
	if errors.Is(err, ErrTidakAda) {
		return 100, nil
	}
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(nilai)
	if err != nil {
		return 0, fmt.Errorf("volume tersimpan tidak valid %q: %w", nilai, err)
	}
	return v, nil
}

func (s *Store) SetVolume(ctx context.Context, v int) error {
	if v < 0 || v > 100 {
		return validasi("volume harus 0 sampai 100")
	}
	return s.SetSetting(ctx, KunciVolume, strconv.Itoa(v))
}
```

Di `riwayat.go`, pindahkan pemindaian baris ke fungsi `scanRiwayat(rows *sql.Rows) (Riwayat, error)`, tambahkan `CariRiwayat`, dan jadikan `RiwayatTerbaru` pembungkus:

```go
// FilterRiwayat menyaring riwayat; nilai kosong berarti tanpa filter. Batas default 200, maks 1000.
type FilterRiwayat struct {
	Tanggal string
	Status  string
	Batas   int
}

var statusRiwayat = map[string]bool{
	"berhasil": true, "gagal": true, "terlewat": true, "cadangan": true, "dihentikan": true,
}

func (s *Store) CariRiwayat(ctx context.Context, f FilterRiwayat) ([]Riwayat, error) {
	if f.Batas <= 0 || f.Batas > 1000 {
		f.Batas = 200
	}
	q := `SELECT id, waktu, jadwal_id, jenis_bel_id, label, sumber, status, pesan FROM riwayat_bel WHERE 1 = 1`
	var args []any
	if f.Tanggal != "" {
		if _, err := time.Parse(schedule.FormatTanggal, f.Tanggal); err != nil {
			return nil, validasi("tanggal tidak valid %q, gunakan format YYYY-MM-DD", f.Tanggal)
		}
		q += ` AND tanggal = ?`
		args = append(args, f.Tanggal)
	}
	if f.Status != "" {
		if !statusRiwayat[f.Status] {
			return nil, validasi("status riwayat tidak dikenal %q", f.Status)
		}
		q += ` AND status = ?`
		args = append(args, f.Status)
	}
	q += ` ORDER BY waktu DESC, id DESC LIMIT ?`
	args = append(args, f.Batas)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Riwayat
	for rows.Next() {
		r, err := scanRiwayat(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RiwayatTerbaru mengembalikan riwayat terbaru lebih dulu.
func (s *Store) RiwayatTerbaru(ctx context.Context, batas int) ([]Riwayat, error) {
	return s.CariRiwayat(ctx, FilterRiwayat{Batas: batas})
}

func scanRiwayat(rows *sql.Rows) (Riwayat, error) {
	var r Riwayat
	var waktu string
	var jadwalID, jenisBelID sql.NullInt64
	if err := rows.Scan(&r.ID, &waktu, &jadwalID, &jenisBelID, &r.Label, &r.Sumber, &r.Status, &r.Pesan); err != nil {
		return r, err
	}
	t, err := time.Parse(time.RFC3339, waktu)
	if err != nil {
		return r, fmt.Errorf("waktu riwayat %d tidak valid: %w", r.ID, err)
	}
	r.Waktu = t.Local()
	r.JadwalID, r.JenisBelID = jadwalID.Int64, jenisBelID.Int64
	return r, nil
}
```

- [ ] **Step 7: Jalankan seluruh test store dan paket yang memakainya**

Run: `go test ./internal/store/ ./internal/bawaan/ ./internal/scheduler/ -v -count=1 2>&1 | tail -40 && go vet ./...`
Expected: PASS semua, termasuk test lama (`TestCreateJadwal_Validasi`, `TestCreatePeriode_Validasi`, bawaan).

- [ ] **Step 8: Commit**

```bash
git add internal/store
git commit -m "feat(store): CRUD lengkap dengan validasi dan error domain"
```

---

### Task 2: Paket `auth`

**Files:**
- Create: `internal/auth/auth.go`, `internal/auth/auth_test.go`
- Modify: `go.mod`, `go.sum` (tambah `golang.org/x/crypto`)

**Interfaces:**
- Consumes: `store.ErrTidakAda`.
- Produces:
  - `const KunciHash = "password_hash"`, `KunciWajibGanti = "wajib_ganti_password"`, `NamaFilePasswordAwal = "password-awal.txt"`, `BatasGagal = 5`, `LamaKunci = 5 * time.Minute`, `UmurSesi = 7 * 24 * time.Hour`, `PanjangMinimal = 8`
  - `var BiayaHash = bcrypt.DefaultCost` (diturunkan ke `bcrypt.MinCost` di test paket lain)
  - `var ErrPasswordSalah, ErrTerkunci, ErrPasswordPendek, ErrPasswordPanjang, ErrPasswordSama, ErrBelumAdaPassword error`
  - `type Penyimpan interface { GetSetting(ctx, kunci string) (string, error); SetSetting(ctx, kunci, nilai string) error }`
  - `func New(st Penyimpan, now func() time.Time) *Auth`
  - `func (*Auth) SiapkanPasswordAwal(ctx) (password string, dibuat bool, err error)`
  - `func (*Auth) ResetPassword(ctx) (string, error)` — juga menghapus semua sesi
  - `func (*Auth) Login(ctx, ip, password string) (token string, err error)`
  - `func (*Auth) Logout(token string)`, `Valid(token string) bool`
  - `func (*Auth) WajibGanti(ctx) (bool, error)`
  - `func (*Auth) GantiPassword(ctx, lama, baru string) error`
  - `func PasswordAcak() string` — 10 karakter dari alfabet tanpa huruf/angka mirip (`abcdefghjkmnpqrstuvwxyz23456789`)

- [ ] **Step 1: Tambah dependensi**

```bash
go get golang.org/x/crypto@latest
```

- [ ] **Step 2: Tulis test yang gagal**

`internal/auth/auth_test.go`:

```go
package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"belsekolah/internal/store"
)

var ctx = context.Background()

type petaPenyimpan struct {
	mu   sync.Mutex
	data map[string]string
}

func (p *petaPenyimpan) GetSetting(_ context.Context, k string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.data[k]
	if !ok {
		return "", store.ErrTidakAda
	}
	return v, nil
}

func (p *petaPenyimpan) SetSetting(_ context.Context, k, v string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.data[k] = v
	return nil
}

type jam struct{ t time.Time }

func (j *jam) now() time.Time   { return j.t }
func (j *jam) maju(d time.Duration) { j.t = j.t.Add(d) }

func siapkan(t *testing.T) (*Auth, *jam, string) {
	t.Helper()
	BiayaHash = bcrypt.MinCost
	j := &jam{t: time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)}
	a := New(&petaPenyimpan{data: map[string]string{}}, j.now)
	pw, dibuat, err := a.SiapkanPasswordAwal(ctx)
	if err != nil || !dibuat || len(pw) != 10 {
		t.Fatalf("SiapkanPasswordAwal = %q, %v, %v", pw, dibuat, err)
	}
	return a, j, pw
}

func TestPasswordAwalHanyaSekali(t *testing.T) {
	a, _, _ := siapkan(t)
	pw, dibuat, err := a.SiapkanPasswordAwal(ctx)
	if err != nil || dibuat || pw != "" {
		t.Fatalf("panggilan kedua = %q, %v, %v; mau tidak membuat ulang", pw, dibuat, err)
	}
	if w, _ := a.WajibGanti(ctx); !w {
		t.Error("password awal harus wajib diganti")
	}
}

func TestPasswordAcak(t *testing.T) {
	a, b := PasswordAcak(), PasswordAcak()
	if len(a) != 10 || a == b {
		t.Fatalf("PasswordAcak = %q, %q", a, b)
	}
	if strings.ContainsAny(a, "il1o0") {
		t.Errorf("password %q memuat karakter yang mudah tertukar", a)
	}
}

func TestLoginDanSesi(t *testing.T) {
	a, j, pw := siapkan(t)
	if _, err := a.Login(ctx, "10.0.0.2", "salah"); !errors.Is(err, ErrPasswordSalah) {
		t.Fatalf("login salah: %v", err)
	}
	token, err := a.Login(ctx, "10.0.0.2", pw)
	if err != nil || len(token) < 32 {
		t.Fatalf("login benar: %q, %v", token, err)
	}
	if !a.Valid(token) || a.Valid("palsu") || a.Valid("") {
		t.Fatal("validasi token salah")
	}
	j.maju(6 * 24 * time.Hour)
	if !a.Valid(token) {
		t.Fatal("sesi harus masih berlaku setelah 6 hari")
	}
	j.maju(6 * 24 * time.Hour) // 6 hari sejak terakhir dipakai: masih berlaku (sliding)
	if !a.Valid(token) {
		t.Fatal("sesi harus diperpanjang setiap dipakai")
	}
	j.maju(7*24*time.Hour + time.Second)
	if a.Valid(token) {
		t.Fatal("sesi harus kedaluwarsa setelah 7 hari tidak dipakai")
	}
	token2, _ := a.Login(ctx, "10.0.0.2", pw)
	a.Logout(token2)
	if a.Valid(token2) {
		t.Fatal("token harus tidak berlaku setelah logout")
	}
}

func TestKunciSetelahLimaGagal(t *testing.T) {
	a, j, pw := siapkan(t)
	for i := 1; i <= 4; i++ {
		if _, err := a.Login(ctx, "10.0.0.9", "salah"); !errors.Is(err, ErrPasswordSalah) {
			t.Fatalf("percobaan %d: %v", i, err)
		}
	}
	if _, err := a.Login(ctx, "10.0.0.9", "salah"); !errors.Is(err, ErrTerkunci) {
		t.Fatalf("percobaan ke-5: %v, mau ErrTerkunci", err)
	}
	if _, err := a.Login(ctx, "10.0.0.9", pw); !errors.Is(err, ErrTerkunci) {
		t.Fatalf("password benar saat terkunci: %v, mau ErrTerkunci", err)
	}
	if _, err := a.Login(ctx, "10.0.0.10", pw); err != nil {
		t.Fatalf("IP lain tidak boleh ikut terkunci: %v", err)
	}
	j.maju(5*time.Minute + time.Second)
	if _, err := a.Login(ctx, "10.0.0.9", pw); err != nil {
		t.Fatalf("setelah 5 menit harus bisa login: %v", err)
	}
}

func TestGantiPassword(t *testing.T) {
	a, _, pw := siapkan(t)
	if err := a.GantiPassword(ctx, "salah", "rahasia-baru"); !errors.Is(err, ErrPasswordSalah) {
		t.Errorf("lama salah: %v", err)
	}
	if err := a.GantiPassword(ctx, pw, "pendek"); !errors.Is(err, ErrPasswordPendek) {
		t.Errorf("baru pendek: %v", err)
	}
	if err := a.GantiPassword(ctx, pw, strings.Repeat("x", 73)); !errors.Is(err, ErrPasswordPanjang) {
		t.Errorf("baru terlalu panjang: %v", err)
	}
	if err := a.GantiPassword(ctx, pw, pw); !errors.Is(err, ErrPasswordSama) {
		t.Errorf("baru sama dengan lama: %v", err)
	}
	if err := a.GantiPassword(ctx, pw, "rahasia-baru"); err != nil {
		t.Fatal(err)
	}
	if w, _ := a.WajibGanti(ctx); w {
		t.Error("setelah diganti tidak lagi wajib ganti")
	}
	if _, err := a.Login(ctx, "1.1.1.1", pw); !errors.Is(err, ErrPasswordSalah) {
		t.Error("password lama tidak boleh berlaku lagi")
	}
	if _, err := a.Login(ctx, "1.1.1.1", "rahasia-baru"); err != nil {
		t.Errorf("password baru: %v", err)
	}
}

func TestResetPassword(t *testing.T) {
	a, _, pw := siapkan(t)
	if err := a.GantiPassword(ctx, pw, "rahasia-baru"); err != nil {
		t.Fatal(err)
	}
	token, _ := a.Login(ctx, "1.1.1.1", "rahasia-baru")
	baru, err := a.ResetPassword(ctx)
	if err != nil || len(baru) != 10 {
		t.Fatalf("ResetPassword = %q, %v", baru, err)
	}
	if a.Valid(token) {
		t.Error("reset harus menghapus semua sesi")
	}
	if w, _ := a.WajibGanti(ctx); !w {
		t.Error("password hasil reset wajib diganti")
	}
	if _, err := a.Login(ctx, "1.1.1.1", baru); err != nil {
		t.Errorf("login dengan password reset: %v", err)
	}
}

func TestBelumAdaPassword(t *testing.T) {
	BiayaHash = bcrypt.MinCost
	a := New(&petaPenyimpan{data: map[string]string{}}, time.Now)
	if _, err := a.Login(ctx, "1.1.1.1", "apa"); !errors.Is(err, ErrBelumAdaPassword) {
		t.Fatalf("err = %v", err)
	}
}
```

- [ ] **Step 3: Jalankan test, pastikan gagal**

Run: `go test ./internal/auth/`
Expected: FAIL — `undefined: New`.

- [ ] **Step 4: Implementasi**

`internal/auth/auth.go`:

```go
// Package auth mengelola password admin tunggal dan sesi login.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"belsekolah/internal/store"
)

const (
	KunciHash            = "password_hash"
	KunciWajibGanti      = "wajib_ganti_password"
	NamaFilePasswordAwal = "password-awal.txt"
	BatasGagal           = 5
	LamaKunci            = 5 * time.Minute
	UmurSesi             = 7 * 24 * time.Hour
	PanjangMinimal       = 8
	panjangMaksimalByte  = 72 // batas bcrypt
)

// BiayaHash adalah cost bcrypt; paket test menurunkannya ke bcrypt.MinCost agar cepat.
var BiayaHash = bcrypt.DefaultCost

var (
	ErrPasswordSalah    = errors.New("password salah")
	ErrTerkunci         = errors.New("terlalu banyak percobaan login, coba lagi dalam 5 menit")
	ErrPasswordPendek   = fmt.Errorf("password baru minimal %d karakter", PanjangMinimal)
	ErrPasswordPanjang  = fmt.Errorf("password baru maksimal %d byte", panjangMaksimalByte)
	ErrPasswordSama     = errors.New("password baru harus berbeda dari password lama")
	ErrBelumAdaPassword = errors.New("password admin belum dibuat")
)

type Penyimpan interface {
	GetSetting(ctx context.Context, kunci string) (string, error)
	SetSetting(ctx context.Context, kunci, nilai string) error
}

type catatanGagal struct {
	gagal       int
	kunciSampai time.Time
}

type Auth struct {
	st  Penyimpan
	now func() time.Time

	mu    sync.Mutex
	sesi  map[string]time.Time      // token → terakhir dipakai
	gagal map[string]*catatanGagal // alamat IP → percobaan gagal
}

func New(st Penyimpan, now func() time.Time) *Auth {
	return &Auth{st: st, now: now, sesi: map[string]time.Time{}, gagal: map[string]*catatanGagal{}}
}

// SiapkanPasswordAwal membuat password acak jika belum ada password sama sekali.
func (a *Auth) SiapkanPasswordAwal(ctx context.Context) (string, bool, error) {
	_, err := a.st.GetSetting(ctx, KunciHash)
	if err == nil {
		return "", false, nil
	}
	if !errors.Is(err, store.ErrTidakAda) {
		return "", false, err
	}
	pw := PasswordAcak()
	if err := a.simpan(ctx, pw, true); err != nil {
		return "", false, err
	}
	return pw, true, nil
}

// ResetPassword membuat password acak baru yang wajib diganti, dan mengakhiri semua sesi.
func (a *Auth) ResetPassword(ctx context.Context) (string, error) {
	pw := PasswordAcak()
	if err := a.simpan(ctx, pw, true); err != nil {
		return "", err
	}
	a.mu.Lock()
	a.sesi = map[string]time.Time{}
	a.mu.Unlock()
	return pw, nil
}

func (a *Auth) simpan(ctx context.Context, pw string, wajibGanti bool) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), BiayaHash)
	if err != nil {
		return err
	}
	if err := a.st.SetSetting(ctx, KunciHash, string(hash)); err != nil {
		return err
	}
	nilai := "0"
	if wajibGanti {
		nilai = "1"
	}
	return a.st.SetSetting(ctx, KunciWajibGanti, nilai)
}

func (a *Auth) cocok(ctx context.Context, pw string) (bool, error) {
	hash, err := a.st.GetSetting(ctx, KunciHash)
	if errors.Is(err, store.ErrTidakAda) {
		return false, ErrBelumAdaPassword
	}
	if err != nil {
		return false, err
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil, nil
}

// Login memeriksa password dan membuat sesi baru. Lima kegagalan dari IP yang sama mengunci IP itu 5 menit.
func (a *Auth) Login(ctx context.Context, ip, password string) (string, error) {
	a.mu.Lock()
	c := a.gagal[ip]
	terkunci := c != nil && a.now().Before(c.kunciSampai)
	a.mu.Unlock()
	if terkunci {
		return "", ErrTerkunci
	}

	ok, err := a.cocok(ctx, password)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if !ok {
		c := a.gagal[ip]
		if c == nil {
			c = &catatanGagal{}
			a.gagal[ip] = c
		}
		c.gagal++
		if c.gagal >= BatasGagal {
			c.gagal = 0
			c.kunciSampai = a.now().Add(LamaKunci)
			return "", ErrTerkunci
		}
		return "", ErrPasswordSalah
	}
	delete(a.gagal, ip)
	token := tokenAcak()
	a.sesi[token] = a.now()
	return token, nil
}

func (a *Auth) Logout(token string) {
	a.mu.Lock()
	delete(a.sesi, token)
	a.mu.Unlock()
}

// Valid memeriksa token dan memperpanjang masa berlakunya.
func (a *Auth) Valid(token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	terakhir, ok := a.sesi[token]
	if !ok {
		return false
	}
	if a.now().Sub(terakhir) > UmurSesi {
		delete(a.sesi, token)
		return false
	}
	a.sesi[token] = a.now()
	return true
}

func (a *Auth) WajibGanti(ctx context.Context) (bool, error) {
	v, err := a.st.GetSetting(ctx, KunciWajibGanti)
	if errors.Is(err, store.ErrTidakAda) {
		return false, nil
	}
	return v == "1", err
}

func (a *Auth) GantiPassword(ctx context.Context, lama, baru string) error {
	ok, err := a.cocok(ctx, lama)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPasswordSalah
	}
	switch {
	case utf8.RuneCountInString(baru) < PanjangMinimal:
		return ErrPasswordPendek
	case len(baru) > panjangMaksimalByte:
		return ErrPasswordPanjang
	case baru == lama:
		return ErrPasswordSama
	}
	return a.simpan(ctx, baru, false)
}

const alfabet = "abcdefghjkmnpqrstuvwxyz23456789"

// PasswordAcak membuat password 10 karakter tanpa karakter yang mudah tertukar.
func PasswordAcak() string {
	b := make([]byte, 10)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alfabet))))
		if err != nil {
			panic(err) // crypto/rand tidak pernah gagal pada OS yang didukung
		}
		b[i] = alfabet[n.Int64()]
	}
	return string(b)
}

func tokenAcak() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
```

- [ ] **Step 5: Jalankan test, pastikan lulus**

Run: `go test -race ./internal/auth/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/auth
git commit -m "feat(auth): password admin, sesi, dan kunci login"
```

---

### Task 3: `scheduler` — status, bel manual, path aman, log tidak spam

**Files:**
- Modify: `internal/scheduler/scheduler.go`, `internal/scheduler/scheduler_test.go` (fake `UrutanBel` bisa diatur)
- Create: `internal/scheduler/status_test.go`

**Interfaces:**
- Consumes: `store.Riwayat`, `store.SumberManual`, `store.SumberDarurat`, `player.Darurat`, `player.Biasa`.
- Produces:
  - `type StatusBel struct { JadwalID int64; Waktu time.Time; JenisBelID int64; Label string; Sudah bool }`
  - `type Status struct { Siap bool; Tanggal string; ProfilID int64; Libur *schedule.Libur; Bel []StatusBel }`
  - `func (*Scheduler) Status() Status` — salinan aman; `Siap` false sebelum muat pertama
  - `func (*Scheduler) BunyikanManual(ctx context.Context, jenisBelID int64, label string, darurat bool)` — riwayat `sumber` manual/darurat, `Waktu` = jam saat dibunyikan, `JadwalID` 0; prioritas darurat memakai `player.Darurat`
  - `func (*Scheduler) FileUrutan(ctx context.Context, jenisBelID int64) []string` — `filepath.Join(audioDir, filepath.Base(a.File))`
  - `Run` mencatat error yang sama paling sering sekali per menit dan mencatat "pulih" saat kembali normal.

- [ ] **Step 1: Buat fake `UrutanBel` bisa diatur**

Di `scheduler_test.go`, tambahkan field `urutan []store.Audio` pada `fakeSumber` dan ubah methodnya:

```go
func (f *fakeSumber) UrutanBel(_ context.Context, jenisBelID int64) ([]store.Audio, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.urutan != nil {
		return f.urutan, nil
	}
	return []store.Audio{{ID: 1, File: "ding.wav"}}, nil
}
```

- [ ] **Step 2: Tulis test yang gagal**

`internal/scheduler/status_test.go`:

```go
package scheduler

import (
	"testing"

	"belsekolah/internal/player"
	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

func TestStatusSebelumDanSesudahMuat(t *testing.T) {
	s, clock, _, _ := siapkan(dataSenin())
	if st := s.Status(); st.Siap {
		t.Fatal("Status harus belum siap sebelum Tick pertama")
	}
	clock.set("2026-10-12", 7, 0, 5)
	tick(t, s)
	st := s.Status()
	if !st.Siap || st.Tanggal != "2026-10-12" || st.ProfilID != 1 || st.Libur != nil {
		t.Fatalf("status = %+v", st)
	}
	if len(st.Bel) != 1 || st.Bel[0].Label != "Masuk" || !st.Bel[0].Sudah || st.Bel[0].JenisBelID != 10 {
		t.Fatalf("bel = %+v", st.Bel)
	}
}

func TestStatusLiburDisalin(t *testing.T) {
	d := dataSenin()
	d.Libur = []schedule.Libur{{ID: 7, Mulai: "2026-10-12", Selesai: "2026-10-12", Keterangan: "Libur"}}
	s, clock, _, _ := siapkan(d)
	clock.set("2026-10-12", 6, 0, 0)
	tick(t, s)
	st := s.Status()
	if st.Libur == nil || st.Libur.ID != 7 {
		t.Fatalf("libur = %+v", st.Libur)
	}
	st.Libur.Keterangan = "diubah"
	if s.Status().Libur.Keterangan != "Libur" {
		t.Fatal("Status harus mengembalikan salinan, bukan pointer ke data internal")
	}
}

func TestBunyikanManual(t *testing.T) {
	s, clock, sumber, pemutar := siapkan(dataSenin())
	clock.set("2026-10-12", 10, 15, 0)
	s.BunyikanManual(ctx, 10, "Manual: Masuk", false)
	s.BunyikanManual(ctx, 10, "Darurat: Masuk", true)
	if len(pemutar.diputar) != 2 {
		t.Fatalf("diputar = %d, mau 2", len(pemutar.diputar))
	}
	if pemutar.diputar[0].Prioritas != player.Biasa || pemutar.diputar[1].Prioritas != player.Darurat {
		t.Errorf("prioritas = %v, %v", pemutar.diputar[0].Prioritas, pemutar.diputar[1].Prioritas)
	}
	pemutar.diputar[0].Selesai(player.Berhasil, "")
	pemutar.diputar[1].Selesai(player.Dihentikan, "dihentikan oleh admin")
	r := sumber.semuaRiwayat()
	if len(r) != 2 {
		t.Fatalf("riwayat = %+v", r)
	}
	if r[0].Sumber != store.SumberManual || r[0].JadwalID != 0 || r[0].Label != "Manual: Masuk" ||
		r[0].Status != "berhasil" || !r[0].Waktu.Equal(clock.t) {
		t.Errorf("riwayat manual = %+v", r[0])
	}
	if r[1].Sumber != store.SumberDarurat || r[1].Status != "dihentikan" || r[1].Pesan == "" {
		t.Errorf("riwayat darurat = %+v", r[1])
	}
}

func TestFileUrutanMembuangPath(t *testing.T) {
	s, _, sumber, _ := siapkan(dataSenin())
	sumber.urutan = []store.Audio{{File: "../../etc/passwd"}, {File: "a.wav"}}
	files := s.FileUrutan(ctx, 10)
	if len(files) != 2 || files[0] != "/data/audio/passwd" || files[1] != "/data/audio/a.wav" {
		t.Fatalf("files = %v", files)
	}
}
```

- [ ] **Step 3: Jalankan test, pastikan gagal**

Run: `go test ./internal/scheduler/`
Expected: FAIL — `s.Status undefined`.

- [ ] **Step 4: Implementasi**

Di `scheduler.go`: tambah tipe `StatusBel`, `Status`, method `Status`, `BunyikanManual`, `FileUrutan`, ganti `bunyikan`, `catat`, cabang terlewat di `Tick`, dan `Run`:

```go
// StatusBel adalah satu bel hari ini beserta apakah sudah dibunyikan atau dicatat.
type StatusBel struct {
	JadwalID   int64
	Waktu      time.Time
	JenisBelID int64
	Label      string
	Sudah      bool
}

// Status adalah gambaran jadwal hari ini menurut pemeriksaan terakhir.
type Status struct {
	Siap     bool // false sebelum data dimuat pertama kali
	Tanggal  string
	ProfilID int64
	Libur    *schedule.Libur
	Bel      []StatusBel
}

const statusTerlewat = "terlewat"

func (s *Scheduler) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{Siap: s.tanggal != "", Tanggal: s.tanggal, ProfilID: s.hasil.ProfilID}
	if l := s.hasil.Libur; l != nil {
		salinan := *l
		st.Libur = &salinan
	}
	for _, b := range s.hasil.Bel {
		st.Bel = append(st.Bel, StatusBel{
			JadwalID: b.JadwalID, Waktu: b.Waktu, JenisBelID: b.JenisBelID,
			Label: b.Label, Sudah: s.sudah[b.JadwalID],
		})
	}
	return st
}

// Run memanggil Tick setiap detik sampai ctx selesai. Error yang sama dicatat paling sering sekali per menit.
func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	var errTerakhir string
	var dicatat time.Time
	for {
		if err := s.Tick(ctx); err != nil {
			if err.Error() != errTerakhir || time.Since(dicatat) >= time.Minute {
				s.log.Error("pemeriksaan jadwal gagal", "err", err)
				errTerakhir, dicatat = err.Error(), time.Now()
			}
		} else if errTerakhir != "" {
			s.log.Info("pemeriksaan jadwal pulih")
			errTerakhir = ""
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
```

Di `Tick`, ganti baris catat terlewat menjadi:

```go
		if telat := now.Sub(b.Waktu); telat >= Toleransi {
			s.catat(ctx, store.Riwayat{
				Waktu: b.Waktu, JadwalID: b.JadwalID, JenisBelID: b.JenisBelID, Label: b.Label,
				Sumber: store.SumberJadwal, Status: statusTerlewat,
				Pesan: fmt.Sprintf("terlewat %s", telat.Round(time.Second)),
			})
			continue
		}
```

Ganti `bunyikan` dan `catat`, tambah `BunyikanManual`, `putar`, `FileUrutan`:

```go
func (s *Scheduler) bunyikan(ctx context.Context, b schedule.Bel) {
	s.log.Info("bel berbunyi", "label", b.Label, "waktu", b.Waktu.Format("15:04"))
	s.putar(ctx, store.Riwayat{
		Waktu: b.Waktu, JadwalID: b.JadwalID, JenisBelID: b.JenisBelID,
		Label: b.Label, Sumber: store.SumberJadwal,
	}, player.Biasa)
}

// BunyikanManual membunyikan jenis bel di luar jadwal dan mencatatnya di riwayat.
// Aman dipanggil dari goroutine mana pun; tidak memakai kunci scheduler.
func (s *Scheduler) BunyikanManual(ctx context.Context, jenisBelID int64, label string, darurat bool) {
	sumber, prioritas := store.SumberManual, player.Biasa
	if darurat {
		sumber, prioritas = store.SumberDarurat, player.Darurat
	}
	s.log.Info("bel manual", "label", label, "darurat", darurat)
	s.putar(ctx, store.Riwayat{
		Waktu: s.clock.Now(), JenisBelID: jenisBelID, Label: label, Sumber: sumber,
	}, prioritas)
}

func (s *Scheduler) putar(ctx context.Context, r store.Riwayat, prioritas player.Prioritas) {
	s.pemutar.Play(player.Permintaan{
		Files:     s.FileUrutan(ctx, r.JenisBelID),
		Prioritas: prioritas,
		Selesai: func(status player.Status, pesan string) {
			r.Status, r.Pesan = string(status), pesan
			s.catat(context.Background(), r)
		},
	})
}

// FileUrutan mengembalikan path file audio sebuah jenis bel. Jika gagal dimuat,
// hasilnya kosong dan pemutar akan memakai nada cadangan.
func (s *Scheduler) FileUrutan(ctx context.Context, jenisBelID int64) []string {
	audio, err := s.sumber.UrutanBel(ctx, jenisBelID)
	if err != nil {
		s.log.Error("muat urutan bel gagal", "jenis_bel", jenisBelID, "err", err)
	}
	var files []string
	for _, a := range audio {
		files = append(files, filepath.Join(s.audioDir, filepath.Base(a.File)))
	}
	return files
}

func (s *Scheduler) catat(ctx context.Context, r store.Riwayat) {
	if err := s.sumber.InsertRiwayat(ctx, r); err != nil {
		s.log.Error("simpan riwayat gagal", "label", r.Label, "err", err)
	}
}
```

- [ ] **Step 5: Jalankan test, pastikan lulus**

Run: `go test -race -count=3 ./internal/scheduler/ -v`
Expected: PASS (test lama dan baru).

- [ ] **Step 6: Commit**

```bash
git add internal/scheduler
git commit -m "feat(scheduler): status hari ini, bel manual, path audio aman"
```

---

### Task 4: Paket `backup` + `Store.BackupKe`

**Files:**
- Create: `internal/store/backup.go`, `internal/backup/backup.go`, `internal/backup/backup_test.go`

**Interfaces:**
- Consumes: `store.Open`, `(*Store).CountProfil`, `(*Store).LoadDataJadwal`, `(*Store).CreateProfil`, `(*Store).Close`, `schedule.FormatTanggal`.
- Produces:
  - `func (*store.Store) BackupKe(ctx, path string) error` — `VACUUM INTO`; file tujuan tidak boleh sudah ada
  - `var backup.ErrBackupTidakValid` — semua kesalahan isi file backup membungkus error ini
  - `const backup.SimpanHarian = 7`
  - `func backup.Harian(ctx, st *store.Store, dir string, now time.Time, simpan int) error`
  - `func backup.JalankanHarian(ctx, st *store.Store, dir string, log *slog.Logger)` — cek saat start lalu tiap jam
  - `func backup.TulisZip(ctx, w io.Writer, st *store.Store, audioDir string) error` — isi: `bel.db`, `audio/<nama>` (file reguler, nama tidak diawali titik)
  - `func backup.Siapkan(ctx, zipPath, dataDir string) error` — validasi + ekstrak ke `<data>/restore-tertunda`
  - `func backup.TerapkanTertunda(dataDir string, now time.Time) (bool, error)` — dipanggil sebelum database dibuka

- [ ] **Step 1: Tulis test yang gagal**

`internal/backup/backup_test.go`:

```go
package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"belsekolah/internal/store"
)

var ctx = context.Background()

func siapkanData(t *testing.T) (dataDir string, st *store.Store) {
	t.Helper()
	dataDir = t.TempDir()
	if err := os.MkdirAll(filepath.Join(dataDir, "audio"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dataDir, "bel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if _, err := st.CreateProfil(ctx, "Normal"); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.wav", "b.mp3", ".unggah-123"} {
		if err := os.WriteFile(filepath.Join(dataDir, "audio", n), []byte("isi "+n), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dataDir, st
}

func namaDiZip(t *testing.T, b []byte) []string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range zr.File {
		out = append(out, f.Name)
	}
	sort.Strings(out)
	return out
}

func buatZip(t *testing.T, path string, isi map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for nama, data := range isi {
		w, err := zw.Create(nama)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}
	zw.Close()
	f.Close()
}

func TestHarianSekaliPerHariDanDipangkas(t *testing.T) {
	dataDir, st := siapkanData(t)
	dir := filepath.Join(dataDir, "backup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tgl := range []string{"2026-10-01", "2026-10-02", "2026-10-03", "2026-10-04", "2026-10-05", "2026-10-06", "2026-10-07"} {
		os.WriteFile(filepath.Join(dir, "bel-"+tgl+".db"), []byte("lama"), 0o644)
	}
	now := time.Date(2026, 10, 15, 2, 0, 0, 0, time.Local)
	if err := Harian(ctx, st, dir, now, 7); err != nil {
		t.Fatal(err)
	}
	if err := Harian(ctx, st, dir, now.Add(time.Hour), 7); err != nil {
		t.Fatalf("panggilan kedua di hari yang sama harus aman: %v", err)
	}
	ents, _ := os.ReadDir(dir)
	var nama []string
	for _, e := range ents {
		nama = append(nama, e.Name())
	}
	if len(nama) != 7 || nama[0] != "bel-2026-10-02.db" || nama[6] != "bel-2026-10-15.db" {
		t.Fatalf("isi folder backup = %v", nama)
	}
	baru, err := store.Open(filepath.Join(dir, "bel-2026-10-15.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer baru.Close()
	if n, _ := baru.CountProfil(ctx); n != 1 {
		t.Errorf("backup harian berisi %d profil, mau 1", n)
	}
}

func TestTulisZip(t *testing.T) {
	dataDir, st := siapkanData(t)
	var buf bytes.Buffer
	if err := TulisZip(ctx, &buf, st, filepath.Join(dataDir, "audio")); err != nil {
		t.Fatal(err)
	}
	got := namaDiZip(t, buf.Bytes())
	want := []string{"audio/a.wav", "audio/b.mp3", "bel.db"}
	if len(got) != len(want) {
		t.Fatalf("isi zip = %v, mau %v (file berawalan titik dilewati)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("isi zip = %v, mau %v", got, want)
		}
	}
}

func TestSiapkanDanTerapkan(t *testing.T) {
	asal, st := siapkanData(t)
	zipPath := filepath.Join(t.TempDir(), "backup.zip")
	f, _ := os.Create(zipPath)
	if err := TulisZip(ctx, f, st, filepath.Join(asal, "audio")); err != nil {
		t.Fatal(err)
	}
	f.Close()

	tujuan := t.TempDir()
	os.MkdirAll(filepath.Join(tujuan, "audio"), 0o755)
	os.WriteFile(filepath.Join(tujuan, "audio", "lama.wav"), []byte("lama"), 0o644)
	stLama, _ := store.Open(filepath.Join(tujuan, "bel.db"))
	stLama.CreateProfil(ctx, "Lama")
	stLama.CreateProfil(ctx, "Lama 2")
	stLama.Close()

	if err := Siapkan(ctx, zipPath, tujuan); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 15, 9, 30, 0, 0, time.Local)
	diterapkan, err := TerapkanTertunda(tujuan, now)
	if err != nil || !diterapkan {
		t.Fatalf("TerapkanTertunda = %v, %v", diterapkan, err)
	}
	if _, err := os.Stat(filepath.Join(tujuan, "restore-tertunda")); !os.IsNotExist(err) {
		t.Error("folder tertunda harus dihapus setelah diterapkan")
	}
	hasil, err := store.Open(filepath.Join(tujuan, "bel.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer hasil.Close()
	if n, _ := hasil.CountProfil(ctx); n != 1 {
		t.Errorf("profil setelah restore = %d, mau 1 (dari backup)", n)
	}
	if _, err := os.Stat(filepath.Join(tujuan, "audio", "a.wav")); err != nil {
		t.Error("audio dari backup harus ada")
	}
	if _, err := os.Stat(filepath.Join(tujuan, "audio", "lama.wav")); !os.IsNotExist(err) {
		t.Error("audio lama harus dipindahkan")
	}
	if _, err := os.Stat(filepath.Join(tujuan, "backup", "bel-sebelum-restore-20261015-093000.db")); err != nil {
		t.Errorf("database lama harus disimpan: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tujuan, "backup", "audio-sebelum-restore-20261015-093000", "lama.wav")); err != nil {
		t.Errorf("audio lama harus disimpan: %v", err)
	}

	again, err := TerapkanTertunda(tujuan, now)
	if err != nil || again {
		t.Errorf("tanpa restore tertunda: %v, %v", again, err)
	}
}

func TestSiapkanMenolakIsiTidakValid(t *testing.T) {
	_, st := siapkanData(t)
	snap := filepath.Join(t.TempDir(), "snap.db")
	if err := st.BackupKe(ctx, snap); err != nil {
		t.Fatal(err)
	}
	db, _ := os.ReadFile(snap)
	kosong := filepath.Join(t.TempDir(), "kosong.db")
	stKosong, _ := store.Open(kosong)
	stKosong.Close()
	dbKosong, _ := os.ReadFile(kosong)

	kasus := map[string]map[string][]byte{
		"tanpa bel.db":    {"audio/a.wav": []byte("x")},
		"path traversal":  {"bel.db": db, "audio/../evil": []byte("x")},
		"folder bertingkat": {"bel.db": db, "audio/sub/a.wav": []byte("x")},
		"file asing":      {"bel.db": db, "etc/passwd": []byte("x")},
		"db rusak":        {"bel.db": []byte("bukan sqlite")},
		"db tanpa profil": {"bel.db": dbKosong},
		"nama berawalan titik": {"bel.db": db, "audio/.x": []byte("x")},
	}
	for nama, isi := range kasus {
		t.Run(nama, func(t *testing.T) {
			dataDir := t.TempDir()
			zipPath := filepath.Join(t.TempDir(), "x.zip")
			buatZip(t, zipPath, isi)
			err := Siapkan(ctx, zipPath, dataDir)
			if !errors.Is(err, ErrBackupTidakValid) {
				t.Fatalf("err = %v, mau ErrBackupTidakValid", err)
			}
			if _, err := os.Stat(filepath.Join(dataDir, "restore-tertunda")); !os.IsNotExist(err) {
				t.Error("restore tidak valid tidak boleh meninggalkan folder tertunda")
			}
		})
	}
	bukanZip := filepath.Join(t.TempDir(), "x.zip")
	os.WriteFile(bukanZip, []byte("bukan zip"), 0o644)
	if err := Siapkan(ctx, bukanZip, t.TempDir()); !errors.Is(err, ErrBackupTidakValid) {
		t.Errorf("bukan zip: %v", err)
	}
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `go test ./internal/backup/`
Expected: FAIL — `undefined: Harian`.

- [ ] **Step 3: `store/backup.go`**

```go
package store

import "context"

// BackupKe menyalin database secara konsisten ke path. File tujuan tidak boleh sudah ada.
func (s *Store) BackupKe(ctx context.Context, path string) error {
	_, err := s.db.ExecContext(ctx, `VACUUM INTO ?`, path)
	return err
}
```

- [ ] **Step 4: `backup/backup.go`**

```go
// Package backup membuat backup harian, file zip backup, dan menerapkan restore saat start.
package backup

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

// ErrBackupTidakValid membungkus semua kesalahan isi file backup.
var ErrBackupTidakValid = errors.New("file backup tidak valid")

const (
	SimpanHarian   = 7
	folderTertunda = "restore-tertunda"
	namaDB         = "bel.db"
	awalanAudio    = "audio/"
	batasEkstrak   = 1 << 30 // total 1 GB, pelindung zip bomb
)

func tidakValid(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrBackupTidakValid, fmt.Sprintf(format, a...))
}

// Harian membuat backup database untuk tanggal `now` jika belum ada, lalu menyisakan `simpan` file terbaru.
func Harian(ctx context.Context, st *store.Store, dir string, now time.Time, simpan int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tujuan := filepath.Join(dir, "bel-"+now.Format(schedule.FormatTanggal)+".db")
	if _, err := os.Stat(tujuan); errors.Is(err, os.ErrNotExist) {
		if err := st.BackupKe(ctx, tujuan); err != nil {
			return fmt.Errorf("backup harian: %w", err)
		}
	}
	lama, err := filepath.Glob(filepath.Join(dir, "bel-????-??-??.db"))
	if err != nil {
		return err
	}
	sort.Strings(lama)
	for len(lama) > simpan {
		if err := os.Remove(lama[0]); err != nil {
			return err
		}
		lama = lama[1:]
	}
	return nil
}

// JalankanHarian memeriksa backup harian saat start lalu setiap jam sampai ctx selesai.
func JalankanHarian(ctx context.Context, st *store.Store, dir string, log *slog.Logger) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if err := Harian(ctx, st, dir, time.Now(), SimpanHarian); err != nil {
			log.Error("backup harian gagal", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// TulisZip menulis snapshot database dan seluruh file audio ke w.
func TulisZip(ctx context.Context, w io.Writer, st *store.Store, audioDir string) error {
	tmp, err := os.MkdirTemp("", "belsekolah-backup-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	snap := filepath.Join(tmp, namaDB)
	if err := st.BackupKe(ctx, snap); err != nil {
		return fmt.Errorf("snapshot database: %w", err)
	}

	zw := zip.NewWriter(w)
	if err := tambahFile(zw, namaDB, snap); err != nil {
		return err
	}
	ents, err := os.ReadDir(audioDir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if !e.Type().IsRegular() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if err := tambahFile(zw, awalanAudio+e.Name(), filepath.Join(audioDir, e.Name())); err != nil {
			return err
		}
	}
	return zw.Close()
}

func tambahFile(zw *zip.Writer, nama, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w, err := zw.Create(nama)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

// Siapkan memvalidasi zip backup lalu mengekstraknya ke folder tertunda.
// Restore baru diterapkan oleh TerapkanTertunda saat aplikasi start berikutnya.
func Siapkan(ctx context.Context, zipPath, dataDir string) (err error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return tidakValid("bukan file zip")
	}
	defer zr.Close()

	staging := filepath.Join(dataDir, folderTertunda+".baru")
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(staging)
		}
	}()
	if err := os.MkdirAll(filepath.Join(staging, "audio"), 0o755); err != nil {
		return err
	}

	adaDB := false
	var total int64
	for _, f := range zr.File {
		switch {
		case f.Name == namaDB:
			adaDB = true
			if err := ekstrak(f, filepath.Join(staging, namaDB), &total); err != nil {
				return err
			}
		case f.Name == awalanAudio:
			continue
		case strings.HasPrefix(f.Name, awalanAudio):
			nama := strings.TrimPrefix(f.Name, awalanAudio)
			if nama == "" || nama != filepath.Base(nama) || strings.HasPrefix(nama, ".") ||
				strings.ContainsAny(nama, `/\`) || !f.Mode().IsRegular() {
				return tidakValid("isi tidak dikenal %q", f.Name)
			}
			if err := ekstrak(f, filepath.Join(staging, "audio", nama), &total); err != nil {
				return err
			}
		default:
			return tidakValid("isi tidak dikenal %q", f.Name)
		}
	}
	if !adaDB {
		return tidakValid("tidak berisi %s", namaDB)
	}
	if err := periksaDB(ctx, filepath.Join(staging, namaDB)); err != nil {
		return err
	}

	final := filepath.Join(dataDir, folderTertunda)
	if err := os.RemoveAll(final); err != nil {
		return err
	}
	return os.Rename(staging, final)
}

func ekstrak(f *zip.File, tujuan string, total *int64) error {
	r, err := f.Open()
	if err != nil {
		return tidakValid("tidak bisa membaca %q", f.Name)
	}
	defer r.Close()
	w, err := os.OpenFile(tujuan, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	n, err := io.Copy(w, io.LimitReader(r, batasEkstrak-*total+1))
	cerr := w.Close()
	*total += n
	if *total > batasEkstrak {
		return tidakValid("isi terlalu besar")
	}
	if err != nil {
		return tidakValid("tidak bisa membaca %q", f.Name)
	}
	return cerr
}

func periksaDB(ctx context.Context, path string) error {
	st, err := store.Open(path)
	if err != nil {
		return tidakValid("database rusak")
	}
	defer st.Close()
	if _, err := st.LoadDataJadwal(ctx); err != nil {
		return tidakValid("database rusak")
	}
	n, err := st.CountProfil(ctx)
	if err != nil || n == 0 {
		return tidakValid("database tidak berisi profil jadwal")
	}
	return nil
}

// TerapkanTertunda menerapkan restore yang disiapkan, sebelum database dibuka.
// Data lama dipindahkan ke <data>/backup dengan akhiran waktu.
func TerapkanTertunda(dataDir string, now time.Time) (bool, error) {
	tertunda := filepath.Join(dataDir, folderTertunda)
	if _, err := os.Stat(tertunda); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	arsip := filepath.Join(dataDir, "backup")
	if err := os.MkdirAll(arsip, 0o755); err != nil {
		return false, err
	}
	stempel := now.Format("20060102-150405")
	db := filepath.Join(dataDir, namaDB)
	for _, akhiran := range []string{"", "-wal", "-shm"} {
		lama := db + akhiran
		if _, err := os.Stat(lama); err == nil {
			if err := os.Rename(lama, filepath.Join(arsip, "bel-sebelum-restore-"+stempel+".db"+akhiran)); err != nil {
				return false, err
			}
		}
	}
	if err := os.Rename(filepath.Join(tertunda, namaDB), db); err != nil {
		return false, err
	}
	audio := filepath.Join(dataDir, "audio")
	if _, err := os.Stat(audio); err == nil {
		if err := os.Rename(audio, filepath.Join(arsip, "audio-sebelum-restore-"+stempel)); err != nil {
			return false, err
		}
	}
	if err := os.Rename(filepath.Join(tertunda, "audio"), audio); err != nil {
		return false, err
	}
	return true, os.RemoveAll(tertunda)
}
```

Catatan untuk `periksaDB`: `store.Open` pada file non-SQLite gagal saat migrasi skema (pesan "file is not a database"), sehingga dikembalikan sebagai tidak valid.

- [ ] **Step 5: Jalankan test, pastikan lulus**

Run: `go test -race ./internal/backup/ ./internal/store/ -v -count=1 2>&1 | tail -30`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/store/backup.go internal/backup
git commit -m "feat(backup): backup harian, zip, dan restore tertunda"
```

---

### Task 5: Paket `sistem` (NTP, ffprobe) + `webui` placeholder

**Files:**
- Create: `internal/sistem/ntp.go`, `internal/sistem/ffprobe.go`, `internal/sistem/ffprobe_test.go`, `internal/webui/webui.go`, `internal/webui/dist/index.html`

**Interfaces:**
- Consumes: `nada.DingDong` (test).
- Produces:
  - `func sistem.CekNTP() func() *bool` — mengembalikan fungsi bercache 60 detik; `nil` jika status tidak diketahui (mis. `timedatectl` tidak ada)
  - `func sistem.DurasiAudio(ctx context.Context, path string) (int64, error)` — durasi ms via ffprobe; error jika bukan audio
  - `var sistem.ErrBukanAudio`
  - `func webui.FS() fs.FS` — isi `internal/webui/dist`

- [ ] **Step 1: Tulis test yang gagal**

`internal/sistem/ffprobe_test.go`:

```go
package sistem

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"belsekolah/internal/nada"
)

func TestDurasiAudio(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe tidak terpasang")
	}
	dir := t.TempDir()
	wav := filepath.Join(dir, "a.wav")
	os.WriteFile(wav, nada.DingDong(), 0o644)
	d, err := DurasiAudio(context.Background(), wav)
	if err != nil {
		t.Fatal(err)
	}
	if d < 1700 || d > 1900 {
		t.Errorf("durasi = %d ms, mau sekitar 1800", d)
	}

	palsu := filepath.Join(dir, "palsu.mp3")
	os.WriteFile(palsu, []byte("ini bukan audio"), 0o644)
	if _, err := DurasiAudio(context.Background(), palsu); !errors.Is(err, ErrBukanAudio) {
		t.Errorf("file palsu: %v, mau ErrBukanAudio", err)
	}
}

func TestCekNTPTidakPanik(t *testing.T) {
	cek := CekNTP()
	_ = cek() // nil di macOS (tanpa timedatectl), bool di Debian
	_ = cek() // panggilan kedua memakai cache
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `go test ./internal/sistem/`
Expected: FAIL — `undefined: DurasiAudio`.

- [ ] **Step 3: Implementasi**

`internal/sistem/ffprobe.go`:

```go
// Package sistem membungkus program sistem: ffprobe dan timedatectl.
package sistem

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
)

var ErrBukanAudio = errors.New("file bukan audio yang valid")

// DurasiAudio mengembalikan durasi file audio dalam milidetik menggunakan ffprobe.
func DurasiAudio(ctx context.Context, path string) (int64, error) {
	out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error",
		"-show_entries", "format=duration:stream=codec_type",
		"-of", "json", path).Output()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return 0, ErrBukanAudio
		}
		return 0, err // ffprobe tidak terpasang, dsb.
	}
	var hasil struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &hasil); err != nil {
		return 0, ErrBukanAudio
	}
	adaAudio := false
	for _, s := range hasil.Streams {
		if s.CodecType == "audio" {
			adaAudio = true
		}
	}
	detik, err := strconv.ParseFloat(hasil.Format.Duration, 64)
	if !adaAudio || err != nil || detik <= 0 {
		return 0, ErrBukanAudio
	}
	return int64(detik * 1000), nil
}
```

`internal/sistem/ntp.go`:

```go
package sistem

import (
	"os/exec"
	"strings"
	"sync"
	"time"
)

// CekNTP mengembalikan fungsi yang melaporkan apakah jam sistem tersinkron NTP.
// Hasil di-cache 60 detik; nil berarti tidak diketahui (timedatectl tidak tersedia).
func CekNTP() func() *bool {
	var mu sync.Mutex
	var nilai *bool
	var kapan time.Time
	return func() *bool {
		mu.Lock()
		defer mu.Unlock()
		if !kapan.IsZero() && time.Since(kapan) < time.Minute {
			return nilai
		}
		kapan = time.Now()
		out, err := exec.Command("timedatectl", "show", "-p", "NTPSynchronized", "--value").Output()
		if err != nil {
			nilai = nil
			return nil
		}
		v := strings.TrimSpace(string(out)) == "yes"
		nilai = &v
		return nilai
	}
}
```

`internal/webui/webui.go`:

```go
// Package webui menyimpan hasil build frontend. Isi dist/ diganti oleh build Rencana 4.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var files embed.FS

// FS mengembalikan isi folder dist.
func FS() fs.FS {
	sub, err := fs.Sub(files, "dist")
	if err != nil {
		panic(err) // dist selalu ada saat kompilasi
	}
	return sub
}
```

`internal/webui/dist/index.html`:

```html
<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Bel Sekolah</title>
</head>
<body style="font-family: system-ui, sans-serif; padding: 2rem;">
<h1>Bel Sekolah</h1>
<p>Antarmuka sedang disiapkan. API tersedia di <code>/api</code>.</p>
</body>
</html>
```

- [ ] **Step 4: Jalankan test, pastikan lulus**

Run: `go test ./internal/sistem/ ./internal/webui/ -v && go vet ./...`
Expected: PASS (`TestDurasiAudio` berjalan karena ffprobe ada di `/opt/homebrew/bin`).

- [ ] **Step 5: Commit**

```bash
git add internal/sistem internal/webui
git commit -m "feat(sistem): cek NTP, durasi audio via ffprobe, dan UI placeholder"
```

---

### Task 6: Paket `api` — inti (server, sesi, beranda, bel, pengaturan, sistem, UI)

**Files:**
- Create: `internal/api/server.go`, `respon.go`, `dto.go`, `sesi.go`, `beranda.go`, `pengaturan.go`, `rute_data.go`, `bantu_test.go`, `inti_test.go`

**Interfaces:**
- Consumes: Task 1 (`store.*` baru), Task 2 (`auth.*`), Task 3 (`scheduler.Status`, `StatusBel`), `player.Permintaan`, `schedule.Hari`, `schedule.FormatTanggal`.
- Produces:
  - `type api.Penjadwal interface { Status() scheduler.Status; Reload(); BunyikanManual(ctx, jenisBelID int64, label string, darurat bool); FileUrutan(ctx, jenisBelID int64) []string }`
  - `type api.Pemutar interface { Play(player.Permintaan); Stop(); SetVolume(int) }`
  - `type api.Deps struct { Store *store.Store; Auth *auth.Auth; Penjadwal Penjadwal; Pemutar Pemutar; DataDir, AudioDir, Versi string; Web fs.FS; Log *slog.Logger; Now func() time.Time; CekNTP func() *bool; DurasiAudio func(ctx context.Context, path string) (int64, error); MulaiUlang func() }`
  - `func api.New(d Deps) http.Handler`
  - Untuk Task 7 (paket yang sama): tipe `server` (embed `Deps`), helper `tulisJSON`, `tulisError`, `bacaJSON`, `(*server).tulisErrorData`, `idDariPath`, `(*server).ubahData(w)` (memanggil Reload), DTO + konverter di `dto.go`, fungsi `(*server).daftarRuteData(rute func(pola string, h http.HandlerFunc))` di `rute_data.go` (Task 6 membuatnya kosong), helper test di `bantu_test.go`.

- [ ] **Step 1: Tulis helper test**

`internal/api/bantu_test.go`:

```go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"golang.org/x/crypto/bcrypt"

	"belsekolah/internal/auth"
	"belsekolah/internal/bawaan"
	"belsekolah/internal/player"
	"belsekolah/internal/scheduler"
	"belsekolah/internal/store"
)

type panggilManual struct {
	JenisBelID int64
	Label      string
	Darurat    bool
}

type fakePenjadwal struct {
	mu     sync.Mutex
	status scheduler.Status
	reload int
	manual []panggilManual
	files  map[int64][]string
}

func (f *fakePenjadwal) Status() scheduler.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakePenjadwal) Reload() {
	f.mu.Lock()
	f.reload++
	f.mu.Unlock()
}

func (f *fakePenjadwal) BunyikanManual(_ context.Context, id int64, label string, darurat bool) {
	f.mu.Lock()
	f.manual = append(f.manual, panggilManual{id, label, darurat})
	f.mu.Unlock()
}

func (f *fakePenjadwal) FileUrutan(_ context.Context, id int64) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.files[id]
}

func (f *fakePenjadwal) jumlahReload() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reload
}

type fakePemutar struct {
	mu      sync.Mutex
	diputar []player.Permintaan
	stop    int
	volume  int
}

func (f *fakePemutar) Play(r player.Permintaan) {
	f.mu.Lock()
	f.diputar = append(f.diputar, r)
	f.mu.Unlock()
}

func (f *fakePemutar) Stop() {
	f.mu.Lock()
	f.stop++
	f.mu.Unlock()
}

func (f *fakePemutar) SetVolume(v int) {
	f.mu.Lock()
	f.volume = v
	f.mu.Unlock()
}

type lingkungan struct {
	srv        *httptest.Server
	st         *store.Store
	penjadwal  *fakePenjadwal
	pemutar    *fakePemutar
	dataDir    string
	audioDir   string
	password   string
	now        time.Time
	ntp        *bool
	mulaiUlang atomic.Int32
}

var ctx = context.Background()

// durasiPalsu menganggap file berawalan "RIFF" sebagai audio 1500 ms.
func durasiPalsu(_ context.Context, path string) (int64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if bytes.HasPrefix(b, []byte("RIFF")) {
		return 1500, nil
	}
	return 0, errors.New("file bukan audio yang valid")
}

func siapkan(t *testing.T) *lingkungan {
	t.Helper()
	auth.BiayaHash = bcrypt.MinCost
	l := &lingkungan{penjadwal: &fakePenjadwal{}, pemutar: &fakePemutar{}}
	l.now = time.Date(2026, 10, 12, 8, 0, 0, 0, time.Local) // Senin
	l.dataDir = t.TempDir()
	l.audioDir = filepath.Join(l.dataDir, "audio")
	if err := os.MkdirAll(l.audioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(l.dataDir, "bel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	l.st = st
	if err := bawaan.Pasang(ctx, st, l.audioDir, l.now); err != nil {
		t.Fatal(err)
	}
	au := auth.New(st, func() time.Time { return l.now })
	pw, _, err := au.SiapkanPasswordAwal(ctx)
	if err != nil {
		t.Fatal(err)
	}
	l.password = pw
	os.WriteFile(filepath.Join(l.dataDir, auth.NamaFilePasswordAwal), []byte(pw), 0o600)

	h := New(Deps{
		Store: st, Auth: au, Penjadwal: l.penjadwal, Pemutar: l.pemutar,
		DataDir: l.dataDir, AudioDir: l.audioDir, Versi: "uji",
		Web:         fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>Bel</title>")}, "app.js": {Data: []byte("js")}},
		Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:         func() time.Time { return l.now },
		CekNTP:      func() *bool { return l.ntp },
		DurasiAudio: durasiPalsu,
		MulaiUlang:  func() { l.mulaiUlang.Add(1) },
	})
	l.srv = httptest.NewServer(h)
	t.Cleanup(l.srv.Close)
	return l
}

func (l *lingkungan) klien() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// masuk login dan (jika perlu) mengganti password awal, lalu mengembalikan klien yang sudah login.
func (l *lingkungan) masuk(t *testing.T) *http.Client {
	t.Helper()
	c := l.klien()
	resp := kirim(t, c, "POST", l.srv.URL+"/api/login", map[string]string{"password": l.password})
	harapKode(t, resp, 200)
	var hasil struct {
		WajibGantiPassword bool `json:"wajibGantiPassword"`
	}
	bacaBody(t, resp, &hasil)
	if hasil.WajibGantiPassword {
		baru := "rahasia-baru-123"
		resp := kirim(t, c, "POST", l.srv.URL+"/api/password", map[string]string{"lama": l.password, "baru": baru})
		harapKode(t, resp, 200)
		resp.Body.Close()
		l.password = baru
	}
	return c
}

func kirim(t *testing.T, c *http.Client, metode, url string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(metode, url, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func harapKode(t *testing.T, resp *http.Response, kode int) {
	t.Helper()
	if resp.StatusCode != kode {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("%s %s: kode %d, mau %d; body: %s", resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, kode, b)
	}
}

func bacaBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
}

func pesanError(t *testing.T, resp *http.Response) string {
	t.Helper()
	var e struct {
		Error string `json:"error"`
	}
	bacaBody(t, resp, &e)
	if e.Error == "" {
		t.Fatal("respons error tanpa field error")
	}
	return e.Error
}

func berisi(s, sub string) bool { return strings.Contains(s, sub) }
```

- [ ] **Step 2: Tulis test inti yang gagal**

`internal/api/inti_test.go`:

```go
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
```

- [ ] **Step 3: Jalankan test, pastikan gagal**

Run: `go test ./internal/api/`
Expected: FAIL — `undefined: New`, `Deps`.

- [ ] **Step 4: `respon.go`**

```go
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"belsekolah/internal/store"
)

func tulisJSON(w http.ResponseWriter, kode int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(kode)
	json.NewEncoder(w).Encode(v)
}

func tulisError(w http.ResponseWriter, kode int, pesan string) {
	tulisJSON(w, kode, map[string]string{"error": pesan})
}

func tulisPesan(w http.ResponseWriter, kode int, pesan string) {
	tulisJSON(w, kode, map[string]string{"pesan": pesan})
}

// bacaJSON membaca body JSON (maks 1 MB, field tak dikenal ditolak). Menulis 400 dan mengembalikan false jika gagal.
func bacaJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		tulisError(w, http.StatusBadRequest, "format data tidak valid")
		return false
	}
	return true
}

// tulisErrorData memetakan error store ke kode HTTP.
func (s *server) tulisErrorData(w http.ResponseWriter, err error) {
	var v *store.ErrorValidasi
	switch {
	case errors.As(err, &v):
		tulisError(w, http.StatusBadRequest, v.Pesan)
	case errors.Is(err, store.ErrTidakAda):
		tulisError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, store.ErrDipakai), errors.Is(err, store.ErrDuplikat), errors.Is(err, store.ErrPeriodeTumpangTindih):
		tulisError(w, http.StatusConflict, err.Error())
	default:
		s.Log.Error("kesalahan server", "err", err)
		tulisError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
	}
}

// idDariPath membaca {id}; menulis 404 dan mengembalikan false jika bukan angka.
func idDariPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		tulisError(w, http.StatusNotFound, "data tidak ditemukan")
		return 0, false
	}
	return id, true
}
```

- [ ] **Step 5: `dto.go`**

```go
package api

import (
	"time"

	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

type profilDTO struct {
	ID   int64  `json:"id"`
	Nama string `json:"nama"`
}

type jenisBelRingkasDTO struct {
	ID    int64  `json:"id"`
	Nama  string `json:"nama"`
	Warna string `json:"warna"`
}

type jenisBelDTO struct {
	ID       int64   `json:"id"`
	Nama     string  `json:"nama"`
	Warna    string  `json:"warna"`
	AudioIDs []int64 `json:"audioIds"`
}

type audioDTO struct {
	ID       int64     `json:"id"`
	Nama     string    `json:"nama"`
	Sumber   string    `json:"sumber"`
	DurasiMS int64     `json:"durasiMs"`
	TeksTTS  string    `json:"teksTts,omitempty"`
	SuaraTTS string    `json:"suaraTts,omitempty"`
	Dibuat   time.Time `json:"dibuat"`
}

type jadwalDTO struct {
	ID         int64  `json:"id"`
	ProfilID   int64  `json:"profilId"`
	Hari       int    `json:"hari"`
	Jam        string `json:"jam"`
	JenisBelID int64  `json:"jenisBelId"`
	Label      string `json:"label"`
}

type periodeDTO struct {
	ID         int64  `json:"id"`
	Mulai      string `json:"mulai"`
	Selesai    string `json:"selesai"`
	ProfilID   int64  `json:"profilId"`
	Keterangan string `json:"keterangan"`
}

type liburDTO struct {
	ID         int64  `json:"id"`
	Mulai      string `json:"mulai"`
	Selesai    string `json:"selesai"`
	Keterangan string `json:"keterangan"`
}

type riwayatDTO struct {
	ID         int64     `json:"id"`
	Waktu      time.Time `json:"waktu"`
	JadwalID   int64     `json:"jadwalId,omitempty"`
	JenisBelID int64     `json:"jenisBelId,omitempty"`
	Label      string    `json:"label"`
	Sumber     string    `json:"sumber"`
	Status     string    `json:"status"`
	Pesan      string    `json:"pesan,omitempty"`
}

type belHariIniDTO struct {
	JadwalID int64               `json:"jadwalId"`
	Jam      string              `json:"jam"`
	Waktu    time.Time           `json:"waktu"`
	Label    string              `json:"label"`
	JenisBel *jenisBelRingkasDTO `json:"jenisBel"`
	Sudah    bool                `json:"sudah"`
}

type statusDTO struct {
	Sekarang   time.Time       `json:"sekarang"`
	Tanggal    string          `json:"tanggal"`
	Hari       int             `json:"hari"`
	Profil     *profilDTO      `json:"profil"`
	Libur      *liburDTO       `json:"libur"`
	Bel        []belHariIniDTO `json:"bel"`
	Berikutnya *belHariIniDTO  `json:"berikutnya"`
	Peringatan []string        `json:"peringatan"`
}

type pengaturanDTO struct {
	Volume          int   `json:"volume"`
	ProfilDefaultID int64 `json:"profilDefaultId"`
}

type sistemDTO struct {
	Versi     string    `json:"versi"`
	ZonaWaktu string    `json:"zonaWaktu"`
	Sekarang  time.Time `json:"sekarang"`
	IP        []string  `json:"ip"`
	NTP       *bool     `json:"ntp"`
}

func keProfilDTO(p store.Profil) profilDTO { return profilDTO{ID: p.ID, Nama: p.Nama} }

func keAudioDTO(a store.Audio) audioDTO {
	return audioDTO{ID: a.ID, Nama: a.Nama, Sumber: a.Sumber, DurasiMS: a.DurasiMS,
		TeksTTS: a.TeksTTS, SuaraTTS: a.SuaraTTS, Dibuat: a.Dibuat}
}

func keJadwalDTO(j schedule.Jadwal) jadwalDTO {
	return jadwalDTO{ID: j.ID, ProfilID: j.ProfilID, Hari: j.Hari, Jam: j.Jam, JenisBelID: j.JenisBelID, Label: j.Label}
}

func (d jadwalDTO) keJadwal() schedule.Jadwal {
	return schedule.Jadwal{ID: d.ID, ProfilID: d.ProfilID, Hari: d.Hari, Jam: d.Jam, JenisBelID: d.JenisBelID, Label: d.Label}
}

func kePeriodeDTO(p schedule.Periode) periodeDTO {
	return periodeDTO{ID: p.ID, Mulai: p.Mulai, Selesai: p.Selesai, ProfilID: p.ProfilID, Keterangan: p.Keterangan}
}

func (d periodeDTO) kePeriode() schedule.Periode {
	return schedule.Periode{ID: d.ID, Mulai: d.Mulai, Selesai: d.Selesai, ProfilID: d.ProfilID, Keterangan: d.Keterangan}
}

func keLiburDTO(l schedule.Libur) liburDTO {
	return liburDTO{ID: l.ID, Mulai: l.Mulai, Selesai: l.Selesai, Keterangan: l.Keterangan}
}

func (d liburDTO) keLibur() schedule.Libur {
	return schedule.Libur{ID: d.ID, Mulai: d.Mulai, Selesai: d.Selesai, Keterangan: d.Keterangan}
}

func keRiwayatDTO(r store.Riwayat) riwayatDTO {
	return riwayatDTO{ID: r.ID, Waktu: r.Waktu, JadwalID: r.JadwalID, JenisBelID: r.JenisBelID,
		Label: r.Label, Sumber: r.Sumber, Status: r.Status, Pesan: r.Pesan}
}
```

- [ ] **Step 5b: `rute_data.go` (kosong, diisi Task 7)**

```go
package api

import "net/http"

// daftarRuteData mendaftarkan rute CRUD data. Diisi oleh Task 7.
func (s *server) daftarRuteData(rute func(pola string, h http.HandlerFunc)) {}
```

- [ ] **Step 6: `server.go`**

```go
// Package api menyajikan REST API dan antarmuka web bel sekolah.
package api

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"belsekolah/internal/auth"
	"belsekolah/internal/player"
	"belsekolah/internal/scheduler"
	"belsekolah/internal/store"
)

type Penjadwal interface {
	Status() scheduler.Status
	Reload()
	BunyikanManual(ctx context.Context, jenisBelID int64, label string, darurat bool)
	FileUrutan(ctx context.Context, jenisBelID int64) []string
}

type Pemutar interface {
	Play(req player.Permintaan)
	Stop()
	SetVolume(v int)
}

type Deps struct {
	Store       *store.Store
	Auth        *auth.Auth
	Penjadwal   Penjadwal
	Pemutar     Pemutar
	DataDir     string
	AudioDir    string
	Versi       string
	Web         fs.FS
	Log         *slog.Logger
	Now         func() time.Time
	CekNTP      func() *bool                                        // nil = tidak diketahui
	DurasiAudio func(ctx context.Context, path string) (int64, error) // durasi ms; error jika bukan audio
	MulaiUlang  func()                                              // dipanggil setelah restore disiapkan
}

type server struct {
	Deps
}

const namaCookie = "sesi"

func New(d Deps) http.Handler {
	s := &server{Deps: d}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("GET /api/sesi", s.sesi)
	mux.Handle("POST /api/logout", s.wajibLogin(s.logout, false))
	mux.Handle("POST /api/password", s.wajibLogin(s.gantiPassword, false))

	rute := func(pola string, h http.HandlerFunc) { mux.Handle(pola, s.wajibLogin(h, true)) }
	rute("GET /api/status", s.status)
	rute("POST /api/bel/manual", s.belManual)
	rute("POST /api/bel/darurat", s.belDarurat)
	rute("POST /api/bel/stop", s.belStop)
	rute("GET /api/pengaturan", s.ambilPengaturan)
	rute("PUT /api/pengaturan", s.ubahPengaturan)
	rute("GET /api/sistem", s.sistem)
	s.daftarRuteData(rute)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		tulisError(w, http.StatusNotFound, "alamat API tidak dikenal")
	})
	mux.Handle("/", s.ui())
	return s.pulihkanPanic(s.cekOrigin(mux))
}

// ubahData dipanggil setelah perubahan data berhasil agar penjadwal memuat ulang.
func (s *server) ubahData() {
	s.Penjadwal.Reload()
}

func (s *server) wajibLogin(h http.HandlerFunc, wajibPasswordBaru bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(namaCookie)
		if err != nil || !s.Auth.Valid(c.Value) {
			tulisError(w, http.StatusUnauthorized, "silakan login terlebih dahulu")
			return
		}
		if wajibPasswordBaru {
			wajib, err := s.Auth.WajibGanti(r.Context())
			if err != nil {
				s.tulisErrorData(w, err)
				return
			}
			if wajib {
				tulisError(w, http.StatusForbidden, "ganti password awal terlebih dahulu")
				return
			}
		}
		h(w, r)
	})
}

// cekOrigin menolak permintaan pengubah data dari situs lain (pelengkap SameSite=Strict).
func (s *server) cekOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if o := r.Header.Get("Origin"); o != "" {
				u, err := url.Parse(o)
				if err != nil || u.Host != r.Host {
					tulisError(w, http.StatusForbidden, "permintaan dari situs lain ditolak")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) pulihkanPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.Log.Error("panic di handler", "path", r.URL.Path, "nilai", v)
				tulisError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// ui menyajikan frontend; path yang bukan file dikembalikan ke index.html (SPA).
func (s *server) ui() http.Handler {
	if s.Web == nil {
		return http.NotFoundHandler()
	}
	berkas := http.FileServerFS(s.Web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p != "" {
			if info, err := fs.Stat(s.Web, p); err != nil || info.IsDir() {
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/"
				berkas.ServeHTTP(w, r2)
				return
			}
		}
		berkas.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 7: `sesi.go`**

```go
package api

import (
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"belsekolah/internal/auth"
)

func alamatIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Password string `json:"password"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	token, err := s.Auth.Login(r.Context(), alamatIP(r), p.Password)
	switch {
	case errors.Is(err, auth.ErrTerkunci):
		tulisError(w, http.StatusTooManyRequests, err.Error())
		return
	case errors.Is(err, auth.ErrPasswordSalah):
		tulisError(w, http.StatusUnauthorized, "password salah")
		return
	case err != nil:
		s.tulisErrorData(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: namaCookie, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteStrictMode,
		MaxAge: int(auth.UmurSesi.Seconds()),
	})
	wajib, err := s.Auth.WajibGanti(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, map[string]bool{"wajibGantiPassword": wajib})
}

func (s *server) sesi(w http.ResponseWriter, r *http.Request) {
	masuk := false
	if c, err := r.Cookie(namaCookie); err == nil {
		masuk = s.Auth.Valid(c.Value)
	}
	wajib := false
	if masuk {
		var err error
		if wajib, err = s.Auth.WajibGanti(r.Context()); err != nil {
			s.tulisErrorData(w, err)
			return
		}
	}
	tulisJSON(w, http.StatusOK, map[string]bool{"masuk": masuk, "wajibGantiPassword": wajib})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(namaCookie); err == nil {
		s.Auth.Logout(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: namaCookie, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteStrictMode, MaxAge: -1})
	tulisPesan(w, http.StatusOK, "berhasil keluar")
}

func (s *server) gantiPassword(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Lama string `json:"lama"`
		Baru string `json:"baru"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	err := s.Auth.GantiPassword(r.Context(), p.Lama, p.Baru)
	switch {
	case errors.Is(err, auth.ErrPasswordSalah):
		tulisError(w, http.StatusBadRequest, "password lama salah")
		return
	case errors.Is(err, auth.ErrPasswordPendek), errors.Is(err, auth.ErrPasswordPanjang), errors.Is(err, auth.ErrPasswordSama):
		tulisError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		s.tulisErrorData(w, err)
		return
	}
	if err := os.Remove(filepath.Join(s.DataDir, auth.NamaFilePasswordAwal)); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.Log.Warn("gagal menghapus file password awal", "err", err)
	}
	tulisPesan(w, http.StatusOK, "password berhasil diganti")
}
```

- [ ] **Step 8: `beranda.go`**

```go
package api

import (
	"fmt"
	"net/http"

	"belsekolah/internal/schedule"
	"belsekolah/internal/store"
)

func (s *server) status(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := s.Now()
	st := s.Penjadwal.Status()
	out := statusDTO{
		Sekarang: now, Tanggal: st.Tanggal, Hari: schedule.Hari(now),
		Bel: []belHariIniDTO{}, Peringatan: []string{},
	}
	if !st.Siap {
		out.Tanggal = now.Format(schedule.FormatTanggal)
		out.Peringatan = append(out.Peringatan, "Penjadwal belum siap, tunggu beberapa detik")
	}
	if st.ProfilID != 0 {
		if p, err := s.Store.GetProfil(ctx, st.ProfilID); err == nil {
			d := keProfilDTO(p)
			out.Profil = &d
		}
	}
	if st.Libur != nil {
		d := keLiburDTO(*st.Libur)
		out.Libur = &d
	}

	jenis, err := s.Store.ListJenisBel(ctx)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	peta := make(map[int64]jenisBelRingkasDTO, len(jenis))
	for _, j := range jenis {
		peta[j.ID] = jenisBelRingkasDTO{ID: j.ID, Nama: j.Nama, Warna: j.Warna}
	}
	for _, b := range st.Bel {
		item := belHariIniDTO{JadwalID: b.JadwalID, Jam: b.Waktu.Format("15:04"), Waktu: b.Waktu, Label: b.Label, Sudah: b.Sudah}
		if j, ok := peta[b.JenisBelID]; ok {
			item.JenisBel = &j
		}
		out.Bel = append(out.Bel, item)
		if out.Berikutnya == nil && !b.Sudah && b.Waktu.After(now) {
			salinan := item
			out.Berikutnya = &salinan
		}
	}

	riwayat, err := s.Store.CariRiwayat(ctx, store.FilterRiwayat{Tanggal: out.Tanggal, Batas: 200})
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	for i := len(riwayat) - 1; i >= 0; i-- { // urut dari yang paling awal
		rw := riwayat[i]
		switch rw.Status {
		case "gagal":
			out.Peringatan = append(out.Peringatan, fmt.Sprintf("Bel %q pukul %s gagal berbunyi: %s", rw.Label, rw.Waktu.Format("15:04"), rw.Pesan))
		case "cadangan":
			out.Peringatan = append(out.Peringatan, fmt.Sprintf("Bel %q pukul %s memakai nada cadangan: %s", rw.Label, rw.Waktu.Format("15:04"), rw.Pesan))
		}
	}
	if s.CekNTP != nil {
		if ok := s.CekNTP(); ok != nil && !*ok {
			out.Peringatan = append(out.Peringatan, "Jam belum tersinkron dengan internet (NTP); periksa koneksi atau atur jam manual")
		}
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) belManual(w http.ResponseWriter, r *http.Request)  { s.bel(w, r, false) }
func (s *server) belDarurat(w http.ResponseWriter, r *http.Request) { s.bel(w, r, true) }

func (s *server) bel(w http.ResponseWriter, r *http.Request, darurat bool) {
	var p struct {
		JenisBelID int64 `json:"jenisBelId"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	j, err := s.Store.GetJenisBel(r.Context(), p.JenisBelID)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	label := "Manual: " + j.Nama
	if darurat {
		label = "Darurat: " + j.Nama
	}
	s.Penjadwal.BunyikanManual(r.Context(), j.ID, label, darurat)
	tulisPesan(w, http.StatusAccepted, "bel dibunyikan")
}

func (s *server) belStop(w http.ResponseWriter, r *http.Request) {
	s.Pemutar.Stop()
	tulisPesan(w, http.StatusOK, "suara dihentikan")
}
```

- [ ] **Step 9: `pengaturan.go` (pengaturan + sistem)**

```go
package api

import (
	"errors"
	"net"
	"net/http"

	"belsekolah/internal/store"
)

func (s *server) bacaPengaturan(r *http.Request) (pengaturanDTO, error) {
	var p pengaturanDTO
	v, err := s.Store.Volume(r.Context())
	if err != nil {
		return p, err
	}
	p.Volume = v
	def, err := s.Store.ProfilDefault(r.Context())
	if err != nil && !errors.Is(err, store.ErrTidakAda) {
		return p, err
	}
	p.ProfilDefaultID = def
	return p, nil
}

func (s *server) ambilPengaturan(w http.ResponseWriter, r *http.Request) {
	p, err := s.bacaPengaturan(r)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, p)
}

func (s *server) ubahPengaturan(w http.ResponseWriter, r *http.Request) {
	var p pengaturanDTO
	if !bacaJSON(w, r, &p) {
		return
	}
	if p.Volume < 0 || p.Volume > 100 {
		tulisError(w, http.StatusBadRequest, "volume harus 0 sampai 100")
		return
	}
	if err := s.Store.SetProfilDefault(r.Context(), p.ProfilDefaultID); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if err := s.Store.SetVolume(r.Context(), p.Volume); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.Pemutar.SetVolume(p.Volume)
	s.ubahData()
	s.ambilPengaturan(w, r)
}

func (s *server) sistem(w http.ResponseWriter, r *http.Request) {
	now := s.Now()
	out := sistemDTO{
		Versi:     s.Versi,
		ZonaWaktu: now.Location().String() + " (" + now.Format("MST -07:00") + ")",
		Sekarang:  now,
		IP:        []string{},
	}
	if alamat, err := net.InterfaceAddrs(); err == nil {
		for _, a := range alamat {
			if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() && ipn.IP.To4() != nil {
				out.IP = append(out.IP, ipn.IP.String())
			}
		}
	}
	if s.CekNTP != nil {
		out.NTP = s.CekNTP()
	}
	tulisJSON(w, http.StatusOK, out)
}
```

- [ ] **Step 10: Jalankan test, pastikan lulus**

Run: `go test -race ./internal/api/ -v -count=1 2>&1 | tail -40 && go vet ./...`
Expected: PASS semua test di `inti_test.go`.

- [ ] **Step 11: Commit**

```bash
git add internal/api
git commit -m "feat(api): server, login, beranda, bel manual, pengaturan, sistem"
```

---

### Task 7: Paket `api` — rute data (profil, jadwal, periode, libur, jenis bel, audio, riwayat, backup)

**Files:**
- Modify: `internal/api/rute_data.go`
- Create: `internal/api/profil.go`, `jadwal.go`, `jenisbel.go`, `audio.go`, `riwayat.go`, `backup.go`, `data_test.go`

**Interfaces:**
- Consumes: semua dari Task 6 (helper, DTO, `server`), Task 1 (`store.*`), Task 4 (`backup.TulisZip`, `backup.Siapkan`, `backup.ErrBackupTidakValid`), `player.Permintaan`.
- Produces: rute sesuai tabel API di spec §14 (bagian data).

Aturan respons: `POST` sukses → 201 + objek; `PUT` → 200 + objek; `DELETE` → 204 tanpa body; list → 200 + array (tidak pernah `null`). Setiap perubahan sukses memanggil `s.ubahData()`.

- [ ] **Step 1: Tulis test yang gagal**

`internal/api/data_test.go`:

```go
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
```

Tambahkan ke `bantu_test.go`:

```go
func storeRiwayat(l *lingkungan, status string) store.Riwayat {
	return store.Riwayat{Waktu: l.now, Label: "Uji", Sumber: store.SumberManual, Status: status}
}

func tungguSampai(t *testing.T, kondisi func() bool) {
	t.Helper()
	batas := time.Now().Add(3 * time.Second)
	for time.Now().Before(batas) {
		if kondisi() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("kondisi tidak terpenuhi dalam 3 detik")
}
```

- [ ] **Step 2: Jalankan test, pastikan gagal**

Run: `go test ./internal/api/ -run 'API' -count=1`
Expected: FAIL — rute data belum ada (404).

- [ ] **Step 3: `rute_data.go`**

```go
package api

import "net/http"

// daftarRuteData mendaftarkan rute CRUD data, pustaka suara, riwayat, dan backup.
func (s *server) daftarRuteData(rute func(pola string, h http.HandlerFunc)) {
	rute("GET /api/profil", s.listProfil)
	rute("POST /api/profil", s.buatProfil)
	rute("PUT /api/profil/{id}", s.ubahProfil)
	rute("DELETE /api/profil/{id}", s.hapusProfil)

	rute("GET /api/jadwal", s.listJadwal)
	rute("POST /api/jadwal", s.buatJadwal)
	rute("POST /api/jadwal/salin", s.salinJadwal)
	rute("PUT /api/jadwal/{id}", s.ubahJadwal)
	rute("DELETE /api/jadwal/{id}", s.hapusJadwal)

	rute("GET /api/periode", s.listPeriode)
	rute("POST /api/periode", s.buatPeriode)
	rute("PUT /api/periode/{id}", s.ubahPeriode)
	rute("DELETE /api/periode/{id}", s.hapusPeriode)

	rute("GET /api/libur", s.listLibur)
	rute("POST /api/libur", s.buatLibur)
	rute("PUT /api/libur/{id}", s.ubahLibur)
	rute("DELETE /api/libur/{id}", s.hapusLibur)

	rute("GET /api/jenis-bel", s.listJenisBel)
	rute("POST /api/jenis-bel", s.buatJenisBel)
	rute("PUT /api/jenis-bel/{id}", s.ubahJenisBel)
	rute("DELETE /api/jenis-bel/{id}", s.hapusJenisBel)
	rute("POST /api/jenis-bel/{id}/tes", s.tesJenisBel)

	rute("GET /api/audio", s.listAudio)
	rute("POST /api/audio", s.unggahAudio)
	rute("PUT /api/audio/{id}", s.ubahAudio)
	rute("DELETE /api/audio/{id}", s.hapusAudio)
	rute("GET /api/audio/{id}/file", s.fileAudio)
	rute("POST /api/audio/{id}/tes", s.tesAudio)

	rute("GET /api/riwayat", s.listRiwayat)
	rute("GET /api/backup", s.unduhBackup)
	rute("POST /api/restore", s.pulihkan)
}
```

- [ ] **Step 4: `profil.go` dan `jadwal.go` (profil, jadwal, periode, libur)**

`internal/api/profil.go`:

```go
package api

import "net/http"

func (s *server) listProfil(w http.ResponseWriter, r *http.Request) {
	ps, err := s.Store.ListProfil(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]profilDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, keProfilDTO(p))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimProfil(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	p, err := s.Store.GetProfil(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, keProfilDTO(p))
}

func (s *server) buatProfil(w http.ResponseWriter, r *http.Request) {
	var p profilDTO
	if !bacaJSON(w, r, &p) {
		return
	}
	id, err := s.Store.CreateProfil(r.Context(), p.Nama)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimProfil(w, r, id, http.StatusCreated)
}

func (s *server) ubahProfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var p profilDTO
	if !bacaJSON(w, r, &p) {
		return
	}
	if err := s.Store.UpdateProfil(r.Context(), id, p.Nama); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimProfil(w, r, id, http.StatusOK)
}

func (s *server) hapusProfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteProfil(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}
```

`internal/api/jadwal.go`:

```go
package api

import (
	"net/http"
	"strconv"
)

func (s *server) listJadwal(w http.ResponseWriter, r *http.Request) {
	profil, err := strconv.ParseInt(r.URL.Query().Get("profil"), 10, 64)
	if err != nil {
		tulisError(w, http.StatusBadRequest, "parameter profil wajib diisi")
		return
	}
	js, err := s.Store.ListJadwal(r.Context(), profil)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]jadwalDTO, 0, len(js))
	for _, j := range js {
		out = append(out, keJadwalDTO(j))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimJadwal(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	j, err := s.Store.GetJadwal(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, keJadwalDTO(j))
}

func (s *server) buatJadwal(w http.ResponseWriter, r *http.Request) {
	var d jadwalDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	id, err := s.Store.CreateJadwal(r.Context(), d.keJadwal())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimJadwal(w, r, id, http.StatusCreated)
}

func (s *server) ubahJadwal(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var d jadwalDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	d.ID = id
	if err := s.Store.UpdateJadwal(r.Context(), d.keJadwal()); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimJadwal(w, r, id, http.StatusOK)
}

func (s *server) hapusJadwal(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteJadwal(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) salinJadwal(w http.ResponseWriter, r *http.Request) {
	var p struct {
		ProfilID int64 `json:"profilId"`
		DariHari int   `json:"dariHari"`
		KeHari   int   `json:"keHari"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	if err := s.Store.SalinHari(r.Context(), p.ProfilID, p.DariHari, p.KeHari); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	tulisPesan(w, http.StatusOK, "jadwal berhasil disalin")
}

func (s *server) listPeriode(w http.ResponseWriter, r *http.Request) {
	ps, err := s.Store.ListPeriode(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]periodeDTO, 0, len(ps))
	for _, p := range ps {
		out = append(out, kePeriodeDTO(p))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimPeriode(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	p, err := s.Store.GetPeriode(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, kePeriodeDTO(p))
}

func (s *server) buatPeriode(w http.ResponseWriter, r *http.Request) {
	var d periodeDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	id, err := s.Store.CreatePeriode(r.Context(), d.kePeriode())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimPeriode(w, r, id, http.StatusCreated)
}

func (s *server) ubahPeriode(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var d periodeDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	d.ID = id
	if err := s.Store.UpdatePeriode(r.Context(), d.kePeriode()); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimPeriode(w, r, id, http.StatusOK)
}

func (s *server) hapusPeriode(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeletePeriode(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) listLibur(w http.ResponseWriter, r *http.Request) {
	ls, err := s.Store.ListLibur(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]liburDTO, 0, len(ls))
	for _, l := range ls {
		out = append(out, keLiburDTO(l))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimLibur(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	l, err := s.Store.GetLibur(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, keLiburDTO(l))
}

func (s *server) buatLibur(w http.ResponseWriter, r *http.Request) {
	var d liburDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	id, err := s.Store.CreateLibur(r.Context(), d.keLibur())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimLibur(w, r, id, http.StatusCreated)
}

func (s *server) ubahLibur(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var d liburDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	d.ID = id
	if err := s.Store.UpdateLibur(r.Context(), d.keLibur()); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimLibur(w, r, id, http.StatusOK)
}

func (s *server) hapusLibur(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteLibur(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: `jenisbel.go`**

```go
package api

import (
	"net/http"

	"belsekolah/internal/player"
)

func (s *server) keJenisBelDTO(r *http.Request, id int64) (jenisBelDTO, error) {
	j, err := s.Store.GetJenisBel(r.Context(), id)
	if err != nil {
		return jenisBelDTO{}, err
	}
	urutan, err := s.Store.UrutanBel(r.Context(), id)
	if err != nil {
		return jenisBelDTO{}, err
	}
	out := jenisBelDTO{ID: j.ID, Nama: j.Nama, Warna: j.Warna, AudioIDs: make([]int64, 0, len(urutan))}
	for _, a := range urutan {
		out.AudioIDs = append(out.AudioIDs, a.ID)
	}
	return out, nil
}

func (s *server) listJenisBel(w http.ResponseWriter, r *http.Request) {
	js, err := s.Store.ListJenisBel(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]jenisBelDTO, 0, len(js))
	for _, j := range js {
		d, err := s.keJenisBelDTO(r, j.ID)
		if err != nil {
			s.tulisErrorData(w, err)
			return
		}
		out = append(out, d)
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimJenisBel(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	d, err := s.keJenisBelDTO(r, id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, d)
}

func (s *server) buatJenisBel(w http.ResponseWriter, r *http.Request) {
	var d jenisBelDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	id, err := s.Store.CreateJenisBel(r.Context(), d.Nama, d.Warna)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if err := s.Store.SetUrutanBel(r.Context(), id, d.AudioIDs); err != nil {
		if errHapus := s.Store.DeleteJenisBel(r.Context(), id); errHapus != nil {
			s.Log.Error("gagal membersihkan jenis bel", "id", id, "err", errHapus)
		}
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimJenisBel(w, r, id, http.StatusCreated)
}

func (s *server) ubahJenisBel(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var d jenisBelDTO
	if !bacaJSON(w, r, &d) {
		return
	}
	if err := s.Store.UpdateJenisBel(r.Context(), id, d.Nama, d.Warna); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if err := s.Store.SetUrutanBel(r.Context(), id, d.AudioIDs); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimJenisBel(w, r, id, http.StatusOK)
}

func (s *server) hapusJenisBel(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteJenisBel(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}

// tesJenisBel memutar jenis bel di speaker tanpa mencatat riwayat.
func (s *server) tesJenisBel(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	if _, err := s.Store.GetJenisBel(r.Context(), id); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.Pemutar.Play(player.Permintaan{Files: s.Penjadwal.FileUrutan(r.Context(), id)})
	tulisPesan(w, http.StatusAccepted, "suara diputar")
}
```

- [ ] **Step 6: `audio.go`**

```go
package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"belsekolah/internal/player"
	"belsekolah/internal/store"
)

const batasUnggahAudio = 20 << 20

func (s *server) pathAudio(a store.Audio) string {
	return filepath.Join(s.AudioDir, filepath.Base(a.File))
}

func (s *server) listAudio(w http.ResponseWriter, r *http.Request) {
	as, err := s.Store.ListAudio(r.Context())
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]audioDTO, 0, len(as))
	for _, a := range as {
		out = append(out, keAudioDTO(a))
	}
	tulisJSON(w, http.StatusOK, out)
}

func (s *server) kirimAudio(w http.ResponseWriter, r *http.Request, id int64, kode int) {
	a, err := s.Store.GetAudio(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, kode, keAudioDTO(a))
}

func namaAcak(ext string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return "upload-" + hex.EncodeToString(b) + ext
}

func (s *server) unggahAudio(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, batasUnggahAudio+(1<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		tulisError(w, http.StatusBadRequest, "file terlalu besar (maksimal 20 MB) atau form tidak valid")
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, hdr, err := r.FormFile("file")
	if err != nil {
		tulisError(w, http.StatusBadRequest, "pilih file audio")
		return
	}
	defer f.Close()
	if hdr.Size > batasUnggahAudio {
		tulisError(w, http.StatusBadRequest, "file terlalu besar (maksimal 20 MB)")
		return
	}
	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	if ext != ".mp3" && ext != ".wav" {
		tulisError(w, http.StatusBadRequest, "hanya file MP3 atau WAV yang diterima")
		return
	}
	nama := strings.TrimSpace(r.FormValue("nama"))
	if nama == "" {
		nama = strings.TrimSuffix(filepath.Base(hdr.Filename), filepath.Ext(hdr.Filename))
	}

	tmp, err := os.CreateTemp(s.AudioDir, ".unggah-*")
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // tidak berpengaruh setelah berhasil di-rename
	_, err = io.Copy(tmp, f)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}

	durasi, err := s.DurasiAudio(r.Context(), tmpPath)
	if err != nil {
		tulisError(w, http.StatusBadRequest, "file bukan audio yang valid")
		return
	}
	namaFile := namaAcak(ext)
	tujuan := filepath.Join(s.AudioDir, namaFile)
	if err := os.Rename(tmpPath, tujuan); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	id, err := s.Store.CreateAudio(r.Context(), store.Audio{
		Nama: nama, Sumber: "upload", File: namaFile, DurasiMS: durasi, Dibuat: s.Now(),
	})
	if err != nil {
		os.Remove(tujuan)
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimAudio(w, r, id, http.StatusCreated)
}

func (s *server) ubahAudio(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	var p struct {
		Nama string `json:"nama"`
	}
	if !bacaJSON(w, r, &p) {
		return
	}
	if err := s.Store.UpdateNamaAudio(r.Context(), id, p.Nama); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	s.kirimAudio(w, r, id, http.StatusOK)
}

func (s *server) hapusAudio(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	a, err := s.Store.DeleteAudio(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if err := os.Remove(s.pathAudio(a)); err != nil && !errors.Is(err, os.ErrNotExist) {
		s.Log.Warn("gagal menghapus file audio", "file", a.File, "err", err)
	}
	s.ubahData()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) fileAudio(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	a, err := s.Store.GetAudio(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	http.ServeFile(w, r, s.pathAudio(a))
}

// tesAudio memutar satu suara di speaker tanpa mencatat riwayat.
func (s *server) tesAudio(w http.ResponseWriter, r *http.Request) {
	id, ok := idDariPath(w, r)
	if !ok {
		return
	}
	a, err := s.Store.GetAudio(r.Context(), id)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.Pemutar.Play(player.Permintaan{Files: []string{s.pathAudio(a)}})
	tulisPesan(w, http.StatusAccepted, "suara diputar")
}
```

- [ ] **Step 7: `riwayat.go` dan `backup.go`**

`internal/api/riwayat.go`:

```go
package api

import (
	"net/http"
	"strconv"

	"belsekolah/internal/store"
)

func (s *server) listRiwayat(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := store.FilterRiwayat{Tanggal: q.Get("tanggal"), Status: q.Get("status")}
	if b := q.Get("batas"); b != "" {
		n, err := strconv.Atoi(b)
		if err != nil {
			tulisError(w, http.StatusBadRequest, "parameter batas harus angka")
			return
		}
		f.Batas = n
	}
	rs, err := s.Store.CariRiwayat(r.Context(), f)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	out := make([]riwayatDTO, 0, len(rs))
	for _, rw := range rs {
		out = append(out, keRiwayatDTO(rw))
	}
	tulisJSON(w, http.StatusOK, out)
}
```

`internal/api/backup.go`:

```go
package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"belsekolah/internal/backup"
)

// unduhBackup membuat zip di file sementara lebih dulu, agar kegagalan bisa dilaporkan sebagai 500.
func (s *server) unduhBackup(w http.ResponseWriter, r *http.Request) {
	tmp, err := os.CreateTemp("", "belsekolah-*.zip")
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if err := backup.TulisZip(r.Context(), tmp, s.Store, s.AudioDir); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		s.tulisErrorData(w, err)
		return
	}
	nama := fmt.Sprintf("bel-sekolah-%s.zip", s.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nama+`"`)
	http.ServeContent(w, r, nama, s.Now(), tmp)
}

func (s *server) pulihkan(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		tulisError(w, http.StatusBadRequest, "file backup terlalu besar atau form tidak valid")
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile("file")
	if err != nil {
		tulisError(w, http.StatusBadRequest, "pilih file backup (.zip)")
		return
	}
	defer f.Close()

	tmp, err := os.CreateTemp("", "belsekolah-restore-*.zip")
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	defer os.Remove(tmp.Name())
	_, err = io.Copy(tmp, f)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}

	if err := backup.Siapkan(r.Context(), tmp.Name(), s.DataDir); err != nil {
		if errors.Is(err, backup.ErrBackupTidakValid) {
			tulisError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.tulisErrorData(w, err)
		return
	}
	s.Log.Warn("restore disiapkan, aplikasi akan dimulai ulang")
	tulisPesan(w, http.StatusAccepted, "backup diterima, aplikasi dimulai ulang dalam beberapa detik")
	if s.MulaiUlang != nil {
		go func() {
			time.Sleep(500 * time.Millisecond) // beri waktu respons terkirim
			s.MulaiUlang()
		}()
	}
}
```

- [ ] **Step 8: Jalankan semua test api**

Run: `go test -race ./internal/api/ -count=1 -v 2>&1 | tail -40 && go vet ./...`
Expected: PASS semua test (`inti_test.go` dan `data_test.go`).

- [ ] **Step 9: Commit**

```bash
git add internal/api
git commit -m "feat(api): CRUD data, pustaka suara, riwayat, backup dan restore"
```

---

### Task 8: `main` + Docker dev + tes end-to-end

**Files:**
- Modify: `cmd/belsekolah/main.go`
- Create: `Dockerfile`, `compose.yaml`, `.dockerignore`, `deploy/docker/asound.conf`, `e2e/e2e_test.go`, `scripts/e2e.sh`
- Modify: `docs/superpowers/plans/catatan-untuk-rencana-2.md` (tandai yang selesai)

**Interfaces:**
- Consumes: semua paket sebelumnya.
- Produces: binary dengan flag `-data`, `-tz`, `-addr` (default `:80`), `-reset-password`; exit code 3 setelah restore; image Docker dev; `scripts/e2e.sh`.

- [ ] **Step 1: Tulis `main.go` baru**

```go
// Command belsekolah menjalankan bel sekolah otomatis beserta antarmuka webnya.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata" // zona waktu tertanam, agar Debian minimal tanpa tzdata tetap jalan

	"belsekolah/internal/api"
	"belsekolah/internal/auth"
	"belsekolah/internal/backup"
	"belsekolah/internal/bawaan"
	"belsekolah/internal/nada"
	"belsekolah/internal/player"
	"belsekolah/internal/scheduler"
	"belsekolah/internal/sistem"
	"belsekolah/internal/store"
	"belsekolah/internal/webui"
)

// versi diisi saat build: -ldflags "-X main.versi=..."
var versi = "dev"

var _ scheduler.Sumber = (*store.Store)(nil)

// errMulaiUlang membuat proses keluar dengan kode 3 agar systemd/Docker memulainya lagi.
var errMulaiUlang = errors.New("mulai ulang setelah restore")

func main() {
	dataDir := flag.String("data", "/var/lib/belsekolah", "folder data (database dan audio)")
	tz := flag.String("tz", "Asia/Jakarta", "zona waktu sekolah")
	addr := flag.String("addr", ":80", "alamat web server")
	reset := flag.Bool("reset-password", false, "buat password admin acak baru lalu keluar")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := aturZonaWaktu(*tz); err != nil {
		log.Error("aplikasi berhenti", "err", err)
		os.Exit(1)
	}
	if *reset {
		if err := resetPassword(*dataDir); err != nil {
			log.Error("reset password gagal", "err", err)
			os.Exit(1)
		}
		return
	}
	err := jalankan(*dataDir, *addr, log)
	switch {
	case errors.Is(err, errMulaiUlang):
		log.Info("keluar untuk menerapkan restore; layanan akan dimulai ulang")
		os.Exit(3)
	case err != nil:
		log.Error("aplikasi berhenti", "err", err)
		os.Exit(1)
	}
}

// aturZonaWaktu menjadikan zona sekolah sebagai time.Local, sehingga bel berbunyi
// menurut jam lokal sekolah walau OS memakai UTC.
func aturZonaWaktu(nama string) error {
	if nama == "" {
		return errors.New("zona waktu wajib diisi, misalnya Asia/Jakarta")
	}
	loc, err := time.LoadLocation(nama)
	if err != nil {
		return fmt.Errorf("zona waktu tidak valid %q: %w", nama, err)
	}
	time.Local = loc
	return nil
}

func tulisPasswordAwal(dataDir, pw string) error {
	return os.WriteFile(filepath.Join(dataDir, auth.NamaFilePasswordAwal), []byte(pw+"\n"), 0o600)
}

func resetPassword(dataDir string) error {
	st, err := store.Open(filepath.Join(dataDir, "bel.db"))
	if err != nil {
		return err
	}
	defer st.Close()
	pw, err := auth.New(st, time.Now).ResetPassword(context.Background())
	if err != nil {
		return err
	}
	if err := tulisPasswordAwal(dataDir, pw); err != nil {
		return err
	}
	fmt.Printf("Password admin baru: %s\nMulai ulang layanan agar sesi lama berakhir (systemctl restart belsekolah).\n", pw)
	return nil
}

func jalankan(dataDir, addr string, log *slog.Logger) error {
	sinyal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, batal := context.WithCancelCause(sinyal)
	defer batal(nil)

	if diterapkan, err := backup.TerapkanTertunda(dataDir, time.Now()); err != nil {
		return fmt.Errorf("terapkan restore: %w", err)
	} else if diterapkan {
		log.Warn("restore dari backup diterapkan; data lama disimpan di folder backup")
	}

	audioDir := filepath.Join(dataDir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return err
	}
	// Nada cadangan ditulis ulang setiap start, terpisah dari folder audio.
	cadangan := filepath.Join(dataDir, "cadangan.wav")
	if err := os.WriteFile(cadangan, nada.Cadangan(), 0o644); err != nil {
		return fmt.Errorf("tulis nada cadangan: %w", err)
	}

	st, err := store.Open(filepath.Join(dataDir, "bel.db"))
	if err != nil {
		return err
	}
	defer st.Close()

	if err := bawaan.Pasang(ctx, st, audioDir, time.Now()); err != nil {
		return fmt.Errorf("pasang data awal: %w", err)
	}

	au := auth.New(st, time.Now)
	if pw, dibuat, err := au.SiapkanPasswordAwal(ctx); err != nil {
		return fmt.Errorf("siapkan password awal: %w", err)
	} else if dibuat {
		if err := tulisPasswordAwal(dataDir, pw); err != nil {
			return err
		}
		log.Warn("password admin awal dibuat; ganti setelah login pertama",
			"password", pw, "file", filepath.Join(dataDir, auth.NamaFilePasswordAwal))
	}

	pemutar := player.New(player.Mpv{}, cadangan)
	if v, err := st.Volume(ctx); err == nil {
		pemutar.SetVolume(v)
	} else {
		log.Warn("volume tersimpan tidak terbaca, memakai 100", "err", err)
	}
	selesai := make(chan struct{})
	go func() {
		defer close(selesai)
		pemutar.Jalankan(ctx)
	}()
	// Didaftarkan setelah defer st.Close(), jadi berjalan lebih dulu: pemutar harus
	// selesai (Selesai -> InsertRiwayat) sebelum database ditutup.
	defer func() { <-selesai }()

	sch := scheduler.New(scheduler.JamSistem{}, st, pemutar, audioDir, log)
	go backup.JalankanHarian(ctx, st, filepath.Join(dataDir, "backup"), log)

	srv := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 10 * time.Second,
		Handler: api.New(api.Deps{
			Store: st, Auth: au, Penjadwal: sch, Pemutar: pemutar,
			DataDir: dataDir, AudioDir: audioDir, Versi: versi,
			Web: webui.FS(), Log: log, Now: time.Now,
			CekNTP: sistem.CekNTP(), DurasiAudio: sistem.DurasiAudio,
			MulaiUlang: func() { batal(errMulaiUlang) },
		}),
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			batal(fmt.Errorf("web server: %w", err))
		}
	}()

	log.Info("bel sekolah berjalan", "versi", versi, "alamat", addr, "data", dataDir,
		"zona_waktu", time.Now().Format("MST -07:00"))
	sch.Run(ctx)

	tutup, batalTutup := context.WithTimeout(context.Background(), 5*time.Second)
	defer batalTutup()
	if err := srv.Shutdown(tutup); err != nil {
		log.Warn("web server tidak berhenti rapi", "err", err)
	}
	log.Info("bel sekolah berhenti")

	if sebab := context.Cause(ctx); sebab != nil && !errors.Is(sebab, context.Canceled) {
		return sebab
	}
	return nil
}
```

- [ ] **Step 2: Build dan uji jalan lokal**

```bash
go vet ./... && go test -race ./... && go build -o belsekolah ./cmd/belsekolah
./belsekolah -data ./tmp-data -addr 127.0.0.1:8089 & APP=$!
sleep 2
cat tmp-data/password-awal.txt
curl -s 127.0.0.1:8089/api/sesi; echo
curl -s -o /dev/null -w "%{http_code}\n" 127.0.0.1:8089/api/status
curl -s 127.0.0.1:8089/ | head -3
kill $APP; wait $APP; echo "exit $?"
./belsekolah -data ./tmp-data -tz "" ; echo "exit $?"
./belsekolah -data ./tmp-data -reset-password
rm -rf tmp-data belsekolah
```

Expected: password 10 karakter; `{"masuk":false,"wajibGantiPassword":false}`; `401`; HTML placeholder; `exit 0` setelah kill; `-tz ""` keluar `1` dengan pesan "zona waktu wajib diisi"; reset mencetak "Password admin baru: ...".

- [ ] **Step 3: Berkas Docker**

`deploy/docker/asound.conf`:

```
# Container development tidak punya kartu suara: arahkan ALSA ke perangkat null
# agar mpv tetap berjalan seperti di mini PC.
pcm.!default {
    type null
}
```

`Dockerfile`:

```dockerfile
# syntax=docker/dockerfile:1
# Image development/pengujian. Produksi di mini PC memakai binary + systemd (Rencana 5).
FROM golang:1.27-trixie AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSI=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.versi=${VERSI}" -o /out/belsekolah ./cmd/belsekolah

FROM debian:trixie-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends mpv ffmpeg ca-certificates tzdata \
 && rm -rf /var/lib/apt/lists/*
COPY deploy/docker/asound.conf /etc/asound.conf
COPY --from=build /out/belsekolah /usr/local/bin/belsekolah
VOLUME /var/lib/belsekolah
EXPOSE 8080
ENTRYPOINT ["belsekolah"]
CMD ["-data", "/var/lib/belsekolah", "-addr", ":8080", "-tz", "Asia/Jakarta"]
```

Jika tag `golang:1.27-trixie` tidak tersedia, pakai tag Go terbaru yang ≥ versi di `go.mod` (cek `docker pull golang:1.27`), dan catat di laporan.

`compose.yaml`:

```yaml
# Lingkungan development: docker compose up --build, lalu buka http://localhost:8080
services:
  belsekolah:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - data:/var/lib/belsekolah
    environment:
      TZ: Asia/Jakarta
    restart: unless-stopped

volumes:
  data:
```

`.dockerignore`:

```
.git
.claude
.superpowers
tmp-data
dist
belsekolah
web/node_modules
```

- [ ] **Step 4: Tes end-to-end**

`e2e/e2e_test.go`:

```go
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
	JadwalID int64  `json:"jadwalId"`
	Label    string `json:"label"`
	Sumber   string `json:"sumber"`
	Status   string `json:"status"`
	Pesan    string `json:"pesan"`
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
		r := tungguRiwayat(t, k, func(r riwayat) bool { return r.JadwalID == j.ID }, 150*time.Second)
		if r.Status != "berhasil" {
			t.Fatalf("bel terjadwal status %q: %s", r.Status, r.Pesan)
		}
	})

	t.Run("darurat lalu hentikan", func(t *testing.T) {
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

		time.Sleep(3 * time.Second)
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
}
```

`scripts/e2e.sh`:

```bash
#!/usr/bin/env bash
# Menjalankan tes end-to-end terhadap container Docker yang baru (data kosong).
set -euo pipefail
cd "$(dirname "$0")/.."

docker compose down -v --remove-orphans
docker compose up -d --build
trap 'docker compose logs --no-color belsekolah | tail -50; docker compose down -v' EXIT

BEL_URL=http://localhost:8080 go test -tags e2e -count=1 -v -timeout 10m ./e2e/
```

```bash
chmod +x scripts/e2e.sh
```

- [ ] **Step 5: Jalankan e2e**

Run: `./scripts/e2e.sh`
Expected: build image sukses; semua subtest `TestAlurLengkap` PASS (± 2–3 menit karena menunggu bel terjadwal). Bila ada yang gagal, baca log container yang dicetak skrip, perbaiki akar masalahnya (bukan melonggarkan tes), lalu ulangi.

- [ ] **Step 6: Perbarui catatan dan commit**

Di `docs/superpowers/plans/catatan-untuk-rencana-2.md`, tambahkan di atas daftar: `Diselesaikan di Rencana 2: sanitasi path audio, rate-limit log, validasi input API, -tz kosong ditolak.` dan hapus butir-butir tersebut dari daftar.

```bash
go vet ./... && go test -race ./...
git add cmd Dockerfile compose.yaml .dockerignore deploy e2e scripts docs/superpowers/plans/catatan-untuk-rencana-2.md
git commit -m "feat: web server, Docker dev, dan tes end-to-end"
```
