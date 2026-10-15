# Tambah Banyak Bel (Gabung) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tombol "Tambah banyak" di halaman Jadwal. Tombol ini menggabungkan banyak bel (dari ketikan/tempelan, template, file, atau profil lain) ke profil yang sedang dibuka, lewat pratinjau baru/kembar/bentrok.

**Architecture:**
- Browser mengubah ketiga sumber menjadi format file jadwal (`FileJadwal` / `formatjadwal.Data`).
- Server punya dua endpoint: `periksa` (baca saja) dan `gabung` (satu transaksi).
- Klasifikasi slot berupa fungsi murni di paket baru `internal/gabung`.
- Resolusi jenis bel diambil dari `Store.ImporJadwal` menjadi fungsi bersama.
- Di frontend: logika murni di `lib/tambahBanyak.ts`, dua komponen yang dipisah dari kode lama (`PemetaanJenis`, `FormPembuat`), tiga komponen baru (`SumberTambahBanyak`, `PratinjauGabung`, `TambahBanyak`).

**Tech Stack:** Go 1.27 (net/http, modernc SQLite satu koneksi), Svelte 5 (runes) + TypeScript + Tailwind 4, Vitest, Playwright (proyek `hp`, Pixel 7).

**Spec:** `docs/superpowers/specs/2026-10-15-tambah-banyak-gabung-design.md`

## Global Constraints

- Bahasa kode, komentar, pesan error, dan teks UI: **Bahasa Indonesia**, mengikuti gaya kode di sekitarnya.
- Database **satu koneksi**: di dalam transaksi jangan memanggil method `Store` yang memakai `s.db`. Baca sebelum `BeginTx`, atau query lewat `tx`.
- Batas bel: `formatjadwal.MaksBel` (1000), baik untuk masukan maupun total profil setelah digabung.
- Kembar = jenis bel (ID setelah resolusi, bukan 0) sama **dan** `strings.TrimSpace(label)` sama (peka huruf besar/kecil).
- Nama jenis bel dicocokkan tanpa membedakan huruf besar/kecil (`kunciNama`). `pemetaan` diberi kunci nama persis seperti di `data.jenisBel[].nama`.
- Bel lama yang tidak ada di masukan **tidak pernah** dihapus atau diubah.
- Endpoint baru wajib login dan didaftarkan lewat `rute(...)` di `internal/api/rute_data.go`, sehingga cekOrigin dan blokir restore otomatis berlaku.
- UI: masukan `min-h-11 rounded-xl`, tombol memakai `Tombol`, panel memakai `PanelGeser`, galat ditampilkan `role="alert"`, status memuat ditampilkan `role="status"`.
- Tabel lebih dari 3 kolom memakai `Tabel` di laptop dan kartu bertumpuk di HP (pola halaman lain).
- Jangan commit build E2E: `internal/webui/dist` tidak boleh berisi `__bel`.
- Tes: `go test -race ./...`, `cd web && npm run check && npx vitest run`. Playwright dijalankan dengan `PORT=<unik> npx playwright test`.
- Commit diakhiri baris kosong lalu `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Struktur File

| File | Tugas | Isi |
|---|---|---|
| `internal/gabung/gabung.go` (+`_test`) | 1 | Klasifikasi slot & rencana gabung (murni) |
| `internal/store/jenisimpor.go` | 2 | Resolusi jenis bel bersama (dipisah dari `cadangan.go`) |
| `internal/store/gabung.go` (+`_test`) | 2 | `PeriksaGabung`, `GabungJadwal` |
| `internal/store/cadangan.go` | 2 | `ImporJadwal` memakai fungsi bersama |
| `internal/api/gabung.go` (+`_test`), `rute_data.go` | 3 | Dua endpoint |
| `web/src/lib/tambahBanyak.ts` (+`.test.ts`), `web/src/lib/api.ts` | 4 | Pengurai tempelan, perluasan hari, filter hari, pemetaan, tipe API |
| `web/src/components/PemetaanJenis.svelte`, `AlurImpor.svelte` | 5 | Pemetaan jenis bersama |
| `web/src/components/FormPembuat.svelte`, `pages/Template.svelte` | 5 | Form pembuat bersama |
| `web/src/components/SumberTambahBanyak.svelte` | 6 | Tiga tab sumber |
| `web/src/components/PratinjauGabung.svelte`, `TambahBanyak.svelte`, `pages/Jadwal.svelte` | 7 | Pratinjau, orkestrasi langkah, tombol |
| `web/e2e/tambah-banyak.spec.ts`, `internal/webui/dist`, `README.md` | 8 | E2E, build produksi, dokumentasi |

Gelombang paralel (berkas tidak beririsan): **G1** = Tugas 1, 4, 5 · **G2** = Tugas 2, 6 · **G3** = Tugas 3, 7 · **G4** = Tugas 8.

---

### Task 1: Paket `internal/gabung` (klasifikasi murni)

**Files:**
- Create: `internal/gabung/gabung.go`
- Test: `internal/gabung/gabung_test.go`

**Interfaces:**
- Produces (dipakai Tugas 2):
  - `type Status string` dengan konstanta `Baru`, `Kembar`, `Bentrok` (nilai `"baru"`, `"kembar"`, `"bentrok"`)
  - `type Isi struct{ JenisBelID int64; Label string }`
  - `type Lama struct{ JadwalID int64; Hari int; Jam string; Isi }`
  - `type Masuk struct{ Hari int; Jam string; Isi }`
  - `type Slot struct{ Hari int \`json:"hari"\`; Jam string \`json:"jam"\` }`
  - `type Baris struct{ Indeks int; Hari int; Jam string; Status Status; Lama *Lama; Baru Masuk }`
  - `func Klasifikasi(lama []Lama, masuk []Masuk) []Baris`
  - `type Rencana struct{ Tambah []Baris; Timpa []Baris; Dilewati int }`
  - `func Rencanakan(baris []Baris, timpa []Slot) Rencana`

- [ ] **Step 1: Tulis tes yang gagal**

```go
package gabung

import (
	"reflect"
	"testing"
)

func TestKlasifikasi(t *testing.T) {
	lama := []Lama{
		{JadwalID: 10, Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Masuk"}},
		{JadwalID: 11, Hari: 1, Jam: "09:00", Isi: Isi{JenisBelID: 2, Label: "Istirahat"}},
		{JadwalID: 12, Hari: 2, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Masuk"}},
		{JadwalID: 13, Hari: 3, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Masuk"}},
		{JadwalID: 14, Hari: 5, Jam: "13:00", Isi: Isi{JenisBelID: 1, Label: "Pulang"}}, // tidak disentuh
	}
	masuk := []Masuk{
		{Hari: 3, Jam: "07:00", Isi: Isi{JenisBelID: 0, Label: "Masuk"}},     // 0: jenis belum dipetakan → bentrok
		{Hari: 1, Jam: "09:00", Isi: Isi{JenisBelID: 2, Label: " Istirahat "}}, // label di-trim → kembar
		{Hari: 1, Jam: "07:15", Isi: Isi{JenisBelID: 1, Label: "Dhuha"}},     // baru
		{Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Upacara"}},   // beda label → bentrok
		{Hari: 2, Jam: "07:00", Isi: Isi{JenisBelID: 3, Label: "Masuk"}},     // beda jenis → bentrok
		{Hari: 2, Jam: "07:01", Isi: Isi{JenisBelID: 1, Label: "masuk"}},     // baru
	}
	got := Klasifikasi(lama, masuk)
	type ringkas struct {
		Indeks   int
		Hari     int
		Jam      string
		Status   Status
		JadwalID int64
	}
	var r []ringkas
	for _, b := range got {
		var id int64
		if b.Lama != nil {
			id = b.Lama.JadwalID
		}
		r = append(r, ringkas{b.Indeks, b.Hari, b.Jam, b.Status, id})
	}
	mau := []ringkas{
		{3, 1, "07:00", Bentrok, 10},
		{2, 1, "07:15", Baru, 0},
		{1, 1, "09:00", Kembar, 11},
		{4, 2, "07:00", Bentrok, 12},
		{5, 2, "07:01", Baru, 0},
		{0, 3, "07:00", Bentrok, 13},
	}
	if !reflect.DeepEqual(r, mau) {
		t.Fatalf("Klasifikasi =\n%+v\nmau\n%+v", r, mau)
	}
	if got[0].Baru.Label != "Upacara" || got[0].Lama.Label != "Masuk" {
		t.Fatalf("isi baris pertama salah: %+v", got[0])
	}
}

func TestKlasifikasi_LabelPekaHuruf(t *testing.T) {
	lama := []Lama{{JadwalID: 1, Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "Upacara"}}}
	got := Klasifikasi(lama, []Masuk{{Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "upacara"}}})
	if got[0].Status != Bentrok {
		t.Fatalf("status = %s, mau bentrok", got[0].Status)
	}
}

func TestRencanakan(t *testing.T) {
	baris := Klasifikasi(
		[]Lama{
			{JadwalID: 10, Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "A"}},
			{JadwalID: 11, Hari: 1, Jam: "08:00", Isi: Isi{JenisBelID: 1, Label: "B"}},
			{JadwalID: 12, Hari: 1, Jam: "09:00", Isi: Isi{JenisBelID: 1, Label: "C"}},
		},
		[]Masuk{
			{Hari: 1, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "A2"}}, // bentrok, ditimpa
			{Hari: 1, Jam: "08:00", Isi: Isi{JenisBelID: 1, Label: "B2"}}, // bentrok, dipertahankan
			{Hari: 1, Jam: "09:00", Isi: Isi{JenisBelID: 1, Label: "C"}},  // kembar
			{Hari: 2, Jam: "07:00", Isi: Isi{JenisBelID: 1, Label: "D"}},  // baru
		},
	)
	r := Rencanakan(baris, []Slot{
		{Hari: 1, Jam: "07:00"},
		{Hari: 1, Jam: "09:00"}, // kembar: diabaikan
		{Hari: 2, Jam: "07:00"}, // baru: tetap ditambah, bukan ditimpa
		{Hari: 4, Jam: "10:00"}, // tidak ada di masukan: diabaikan
	})
	if len(r.Tambah) != 1 || r.Tambah[0].Hari != 2 {
		t.Fatalf("Tambah = %+v", r.Tambah)
	}
	if len(r.Timpa) != 1 || r.Timpa[0].Lama.JadwalID != 10 || r.Timpa[0].Baru.Label != "A2" {
		t.Fatalf("Timpa = %+v", r.Timpa)
	}
	if r.Dilewati != 2 {
		t.Fatalf("Dilewati = %d, mau 2", r.Dilewati)
	}
}

func TestKlasifikasi_Kosong(t *testing.T) {
	if got := Klasifikasi(nil, nil); len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
	r := Rencanakan(nil, nil)
	if r.Tambah == nil || r.Timpa == nil {
		t.Fatal("slice Rencana harus non-nil")
	}
}
```

- [ ] **Step 2: Jalankan tes, pastikan gagal**

Run: `go test ./internal/gabung/`
Expected: FAIL (paket/simbol belum ada)

- [ ] **Step 3: Implementasi**

```go
// Package gabung mengelompokkan bel masukan terhadap jadwal sebuah profil (baru/kembar/bentrok)
// dan menyusun rencana penggabungan. Murni, tanpa database.
package gabung

import (
	"sort"
	"strings"
)

type Status string

const (
	Baru    Status = "baru"
	Kembar  Status = "kembar"
	Bentrok Status = "bentrok"
)

// Isi adalah isi satu slot. JenisBelID 0 berarti jenis belum dikenal (belum dipetakan atau akan dibuat baru).
type Isi struct {
	JenisBelID int64
	Label      string
}

// Lama adalah bel yang sudah ada di profil tujuan.
type Lama struct {
	JadwalID int64
	Hari     int
	Jam      string
	Isi
}

// Masuk adalah bel masukan; slot (Hari, Jam) dianggap unik di antara masukan.
type Masuk struct {
	Hari int
	Jam  string
	Isi
}

// Slot menunjuk hari dan jam di profil tujuan.
type Slot struct {
	Hari int    `json:"hari"`
	Jam  string `json:"jam"`
}

// Baris adalah hasil klasifikasi satu bel masukan; Indeks menunjuk posisinya di slice masukan.
type Baris struct {
	Indeks int
	Hari   int
	Jam    string
	Status Status
	Lama   *Lama
	Baru   Masuk
}

func sama(a, b Isi) bool {
	return a.JenisBelID != 0 && a.JenisBelID == b.JenisBelID &&
		strings.TrimSpace(a.Label) == strings.TrimSpace(b.Label)
}

// Klasifikasi mengelompokkan setiap bel masukan, diurutkan menurut hari lalu jam.
func Klasifikasi(lama []Lama, masuk []Masuk) []Baris {
	ada := make(map[Slot]*Lama, len(lama))
	for i := range lama {
		ada[Slot{lama[i].Hari, lama[i].Jam}] = &lama[i]
	}
	out := make([]Baris, 0, len(masuk))
	for i, m := range masuk {
		b := Baris{Indeks: i, Hari: m.Hari, Jam: m.Jam, Baru: m, Status: Baru}
		if l, ok := ada[Slot{m.Hari, m.Jam}]; ok {
			salinan := *l
			b.Lama = &salinan
			if sama(m.Isi, l.Isi) {
				b.Status = Kembar
			} else {
				b.Status = Bentrok
			}
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Hari != out[j].Hari {
			return out[i].Hari < out[j].Hari
		}
		return out[i].Jam < out[j].Jam
	})
	return out
}

// Rencana adalah tindakan penggabungan. Dilewati = kembar + bentrok yang dipertahankan.
type Rencana struct {
	Tambah   []Baris
	Timpa    []Baris
	Dilewati int
}

// Rencanakan: baru ditambah, kembar dilewati, bentrok ditimpa hanya bila slotnya ada di timpa.
// Slot di timpa yang bukan bentrok diabaikan.
func Rencanakan(baris []Baris, timpa []Slot) Rencana {
	pilih := make(map[Slot]bool, len(timpa))
	for _, s := range timpa {
		pilih[s] = true
	}
	r := Rencana{Tambah: []Baris{}, Timpa: []Baris{}}
	for _, b := range baris {
		switch {
		case b.Status == Baru:
			r.Tambah = append(r.Tambah, b)
		case b.Status == Bentrok && pilih[Slot{b.Hari, b.Jam}]:
			r.Timpa = append(r.Timpa, b)
		default:
			r.Dilewati++
		}
	}
	return r
}
```

- [ ] **Step 4: Jalankan tes, pastikan lulus**

Run: `go test -race ./internal/gabung/ && go vet ./internal/gabung/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gabung
git commit -m "feat(gabung): klasifikasi bel baru/kembar/bentrok dan rencana gabung"
```

---

### Task 2: Store — resolusi jenis bersama, `PeriksaGabung`, `GabungJadwal`

**Files:**
- Create: `internal/store/jenisimpor.go`
- Create: `internal/store/gabung.go`
- Modify: `internal/store/cadangan.go` (`ImporJadwal` memakai fungsi bersama; isi fungsi lain tidak diubah)
- Test: `internal/store/gabung_test.go`

**Interfaces:**
- Consumes: `internal/gabung` (Tugas 1): `Klasifikasi`, `Rencanakan`, `Lama`, `Masuk`, `Isi`, `Slot`, `Status`.
- Produces (dipakai Tugas 3):
  - `type IsiGabung struct{ JenisBel string \`json:"jenisBel"\`; Label string \`json:"label"\` }`
  - `type BarisGabung struct{ Hari int \`json:"hari"\`; Jam string \`json:"jam"\`; Status gabung.Status \`json:"status"\`; Lama *IsiGabung \`json:"lama"\`; Baru IsiGabung \`json:"baru"\` }`
  - `type RingkasanGabung struct{ Baru int \`json:"baru"\`; Kembar int \`json:"kembar"\`; Bentrok int \`json:"bentrok"\` }`
  - `type HasilPeriksaGabung struct{ JenisBelTakDikenal []string \`json:"jenisBelTakDikenal"\`; Ringkasan RingkasanGabung \`json:"ringkasan"\`; Baris []BarisGabung \`json:"baris"\` }`
  - `type HasilGabung struct{ Ditambah int \`json:"ditambah"\`; Ditimpa int \`json:"ditimpa"\`; Dilewati int \`json:"dilewati"\` }`
  - `func (s *Store) PeriksaGabung(ctx context.Context, profilID int64, d formatjadwal.Data, pemetaan map[string]PemetaanJenis) (HasilPeriksaGabung, error)`
  - `func (s *Store) GabungJadwal(ctx context.Context, profilID int64, d formatjadwal.Data, pemetaan map[string]PemetaanJenis, timpa []gabung.Slot) (HasilGabung, error)`
- Error: data tidak valid, pemetaan kurang/salah, atau lebih dari 1000 bel → `*ErrorValidasi`. Profil tidak ada → `ErrTidakAda` (dari `GetProfil`).

- [ ] **Step 1: Pisahkan resolusi jenis bel ke `jenisimpor.go`**

Buat `internal/store/jenisimpor.go`:

```go
package store

import (
	"context"
	"database/sql"
	"strings"

	"belsekolah/internal/formatjadwal"
)

// petaJenis memotret jenis bel di database untuk mencocokkan nama di file jadwal.
type petaJenis struct {
	idAda   map[string]int64 // kunciNama -> id
	namaID  map[int64]string // id -> nama di database
}

func (s *Store) bacaPetaJenis(ctx context.Context) (petaJenis, error) {
	ada, err := s.ListJenisBel(ctx)
	if err != nil {
		return petaJenis{}, err
	}
	p := petaJenis{idAda: make(map[string]int64, len(ada)), namaID: make(map[int64]string, len(ada))}
	for _, j := range ada {
		p.idAda[kunciNama(j.Nama)] = j.ID
		p.namaID[j.ID] = j.Nama
	}
	return p, nil
}

// takDikenal mengembalikan nama jenis yang dipakai d tetapi tidak ada di database (tanpa memandang pemetaan).
func (p petaJenis) takDikenal(d formatjadwal.Data) []string {
	out := []string{}
	for _, j := range d.JenisDipakai() {
		if _, ok := p.idAda[kunciNama(j.Nama)]; !ok {
			out = append(out, j.Nama)
		}
	}
	return out
}

// idSementara: kunciNama -> id jenis untuk setiap jenis yang dipakai d, tanpa membuat apa pun.
// 0 bila belum dipetakan atau dipetakan ke jenis baru.
func (p petaJenis) idSementara(d formatjadwal.Data, pemetaan map[string]PemetaanJenis) map[string]int64 {
	out := make(map[string]int64)
	for _, j := range d.JenisDipakai() {
		k := kunciNama(j.Nama)
		if id, ok := p.idAda[k]; ok {
			out[k] = id
			continue
		}
		if m, ok := pemetaan[j.Nama]; ok && !m.Baru {
			if _, sah := p.namaID[m.JenisBelID]; sah {
				out[k] = m.JenisBelID
				continue
			}
		}
		out[k] = 0
	}
	return out
}

// periksaPemetaan memastikan setiap jenis tak dikenal dipetakan ke jenis yang ada atau ke jenis baru.
// perluSuara true bila ada jenis baru yang akan dibuat.
func (p petaJenis) periksaPemetaan(d formatjadwal.Data, pemetaan map[string]PemetaanJenis) (perluSuara bool, err error) {
	for _, j := range d.JenisDipakai() {
		if _, ok := p.idAda[kunciNama(j.Nama)]; ok {
			continue
		}
		m, ok := pemetaan[j.Nama]
		switch {
		case !ok || (!m.Baru && m.JenisBelID == 0):
			return false, validasi("jenis bel %q belum dipetakan", j.Nama)
		case m.Baru:
			perluSuara = true
		case p.namaID[m.JenisBelID] == "":
			return false, validasi("jenis bel tujuan untuk %q tidak ditemukan", j.Nama)
		}
	}
	return perluSuara, nil
}

// buatJenis (di dalam tx) mengembalikan kunciNama -> id untuk setiap jenis yang dipakai d,
// membuat jenis baru (dengan satu suara) sesuai pemetaan. Panggil periksaPemetaan dahulu.
func (p petaJenis) buatJenis(ctx context.Context, tx *sql.Tx, d formatjadwal.Data, pemetaan map[string]PemetaanJenis, suara int64) (map[string]int64, error) {
	dipakai := d.JenisDipakai()
	idJenis := make(map[string]int64, len(dipakai))
	for _, j := range dipakai {
		k := kunciNama(j.Nama)
		if id, ok := p.idAda[k]; ok {
			idJenis[k] = id
			continue
		}
		m := pemetaan[j.Nama]
		if !m.Baru {
			idJenis[k] = m.JenisBelID
			continue
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO jenis_bel (nama, warna) VALUES (?, ?)`, strings.TrimSpace(j.Nama), j.Warna)
		if err != nil {
			return nil, terjemahkan(err, nil)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO urutan_bel (jenis_bel_id, urutan, audio_id) VALUES (?, 1, ?)`, id, suara); err != nil {
			return nil, err
		}
		idJenis[k] = id
	}
	return idJenis, nil
}
```

- [ ] **Step 2: Tulis ulang `ImporJadwal` di `cadangan.go` memakai fungsi bersama**

Ganti badan `ImporJadwal` (komentar dokumentasinya tetap) menjadi:

```go
func (s *Store) ImporJadwal(ctx context.Context, d formatjadwal.Data, nama string, pemetaan map[string]PemetaanJenis) (int64, error) {
	if err := d.Validasi(); err != nil {
		return 0, dariFormat(err)
	}
	nama, err := validasiNama(nama)
	if err != nil {
		return 0, err
	}
	// Semua pembacaan sebelum transaksi: database hanya punya satu koneksi.
	peta, err := s.bacaPetaJenis(ctx)
	if err != nil {
		return 0, err
	}
	perluSuara, err := peta.periksaPemetaan(d, pemetaan)
	if err != nil {
		return 0, err
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
	idJenis, err := peta.buatJenis(ctx, tx, d, pemetaan, suara)
	if err != nil {
		return 0, err
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

Ubah juga `JenisBelTakDikenal` agar memakai `bacaPetaJenis` + `takDikenal`. Hasilnya tetap sama:

```go
func (s *Store) JenisBelTakDikenal(ctx context.Context, d formatjadwal.Data) ([]string, error) {
	peta, err := s.bacaPetaJenis(ctx)
	if err != nil {
		return nil, err
	}
	return peta.takDikenal(d), nil
}
```

Run: `go test -race ./internal/store/ ./internal/api/`
Expected: PASS (tes impor lama tetap hijau). Hapus import yang tidak terpakai bila kompilator memintanya.

- [ ] **Step 3: Tulis tes gabung yang gagal**

`internal/store/gabung_test.go`. Helper `isiProfil`, `dataAsing`, `harus`, `harapValidasi`, `bukaStore`, dan `ctx` sudah ada di paket ini. `isiProfil` membuat profil "Normal" berisi Sen 07:00 "Masuk", Sen 12:00 "Pulang", dan Jum 07:00 "" dengan jenis "Masuk". `dataAsing()` berisi jenis MASUK, Bel Literasi, dan Bel Doa, serta bel Sen 07:00 MASUK "", Sen 07:15 Bel Literasi "Literasi", dan Sen 07:30 Bel Doa "Doa".

```go
package store

import (
	"errors"
	"fmt"
	"testing"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/gabung"
	"belsekolah/internal/schedule"
)

// dataGabung: Sen 07:00 Masuk "Upacara" (bentrok), Sen 12:00 Masuk "Pulang" (kembar),
// Sel 07:00 Masuk "Masuk" (baru), Jum 07:00 masuk "" (kembar: nama jenis beda huruf).
func dataGabung() formatjadwal.Data {
	d := formatjadwal.Baru("Tambah banyak")
	d.JenisBel = []formatjadwal.JenisBel{{Nama: "Masuk", Warna: "#2563eb"}}
	d.Jadwal = []formatjadwal.Bel{
		{Hari: 1, Jam: "07:00", JenisBel: "Masuk", Label: "Upacara"},
		{Hari: 1, Jam: "12:00", JenisBel: "Masuk", Label: "Pulang"},
		{Hari: 2, Jam: "07:00", JenisBel: "Masuk", Label: "Masuk"},
		{Hari: 5, Jam: "07:00", JenisBel: "MASUK", Label: ""},
	}
	return d
}

func TestPeriksaGabung(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	h := harus(s.PeriksaGabung(ctx, profil, dataGabung(), nil))(t)
	if h.Ringkasan != (RingkasanGabung{Baru: 1, Kembar: 2, Bentrok: 1}) {
		t.Fatalf("ringkasan = %+v", h.Ringkasan)
	}
	if len(h.JenisBelTakDikenal) != 0 {
		t.Fatalf("tak dikenal = %v", h.JenisBelTakDikenal)
	}
	b := h.Baris[0]
	if b.Hari != 1 || b.Jam != "07:00" || b.Status != gabung.Bentrok || b.Lama == nil ||
		b.Lama.JenisBel != "Masuk" || b.Lama.Label != "Masuk" || b.Baru.Label != "Upacara" {
		t.Fatalf("baris[0] = %+v lama=%+v", b, b.Lama)
	}
	if h.Baris[2].Status != gabung.Baru || h.Baris[2].Lama != nil {
		t.Fatalf("baris[2] = %+v", h.Baris[2])
	}
	// Periksa tidak mengubah apa pun.
	if n := len(harus(s.ListJadwal(ctx, profil))(t)); n != 3 {
		t.Fatalf("jadwal = %d setelah periksa", n)
	}
}

func TestPeriksaGabung_JenisTakDikenal(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s)
	d := dataAsing() // Sen 07:00 MASUK "" (lama: Masuk "Masuk") → bentrok karena label beda
	h := harus(s.PeriksaGabung(ctx, profil, d, nil))(t)
	if fmt.Sprint(h.JenisBelTakDikenal) != "[Bel Literasi Bel Doa]" {
		t.Fatalf("tak dikenal = %v", h.JenisBelTakDikenal)
	}
	if h.Ringkasan.Baru != 2 || h.Ringkasan.Bentrok != 1 {
		t.Fatalf("ringkasan = %+v", h.Ringkasan)
	}
	// Setelah dipetakan, daftar tak dikenal tetap dilaporkan.
	h = harus(s.PeriksaGabung(ctx, profil, d, map[string]PemetaanJenis{
		"Bel Literasi": {Baru: true}, "Bel Doa": {JenisBelID: jenis},
	}))(t)
	if len(h.JenisBelTakDikenal) != 2 {
		t.Fatalf("tak dikenal setelah pemetaan = %v", h.JenisBelTakDikenal)
	}
}

func TestPeriksaGabung_ProfilTidakAda(t *testing.T) {
	s := bukaStore(t)
	isiProfil(t, s)
	if _, err := s.PeriksaGabung(ctx, 999, dataGabung(), nil); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v", err)
	}
	if _, err := s.GabungJadwal(ctx, 999, dataGabung(), nil, nil); !errors.Is(err, ErrTidakAda) {
		t.Fatalf("err = %v", err)
	}
}

func TestGabungJadwal(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s)
	h := harus(s.GabungJadwal(ctx, profil, dataGabung(), nil, []gabung.Slot{{Hari: 1, Jam: "07:00"}, {Hari: 1, Jam: "12:00"}}))(t)
	if h != (HasilGabung{Ditambah: 1, Ditimpa: 1, Dilewati: 2}) {
		t.Fatalf("hasil = %+v", h)
	}
	js := harus(s.ListJadwal(ctx, profil))(t)
	got := map[string]string{}
	for _, j := range js {
		if j.JenisBelID != jenis {
			t.Fatalf("jenis salah: %+v", j)
		}
		got[fmt.Sprintf("%d %s", j.Hari, j.Jam)] = j.Label
	}
	mau := map[string]string{"1 07:00": "Upacara", "1 12:00": "Pulang", "2 07:00": "Masuk", "5 07:00": ""}
	if fmt.Sprint(got) != fmt.Sprint(mau) {
		t.Fatalf("jadwal = %v, mau %v", got, mau)
	}
}

func TestGabungJadwal_PertahankanBawaan(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	h := harus(s.GabungJadwal(ctx, profil, dataGabung(), nil, nil))(t)
	if h != (HasilGabung{Ditambah: 1, Ditimpa: 0, Dilewati: 3}) {
		t.Fatalf("hasil = %+v", h)
	}
	for _, j := range harus(s.ListJadwal(ctx, profil))(t) {
		if j.Hari == 1 && j.Jam == "07:00" && j.Label != "Masuk" {
			t.Fatalf("bel lama berubah: %+v", j)
		}
	}
}

func TestGabungJadwal_JenisBaruDanPemetaan(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s)
	d := dataAsing()
	if _, err := s.GabungJadwal(ctx, profil, d, map[string]PemetaanJenis{"Bel Literasi": {Baru: true}}, nil); err == nil {
		t.Fatal("pemetaan kurang harus ditolak")
	} else {
		harapValidasi(t, err)
	}
	if n := len(harus(s.ListJenisBel(ctx))(t)); n != 1 {
		t.Fatalf("jenis bel = %d setelah gagal (transaksi harus batal)", n)
	}
	h := harus(s.GabungJadwal(ctx, profil, d, map[string]PemetaanJenis{
		"Bel Literasi": {Baru: true}, "Bel Doa": {JenisBelID: jenis},
	}, nil))(t)
	if h.Ditambah != 2 {
		t.Fatalf("hasil = %+v", h)
	}
	semua := harus(s.ListJenisBel(ctx))(t)
	if len(semua) != 2 || semua[1].Nama != "Bel Literasi" {
		t.Fatalf("jenis = %+v", semua)
	}
	if n := len(harus(s.UrutanBel(ctx, semua[1].ID))(t)); n != 1 {
		t.Fatalf("jenis baru harus punya 1 suara, ada %d", n)
	}
}

func TestGabungJadwal_BatasTotal(t *testing.T) {
	s := bukaStore(t)
	profil, jenis := isiProfil(t, s) // 3 bel
	// Isi profil sampai 999 bel: hari 2–7 × jam 00:00.. (langsung lewat CreateJadwal).
	n := 3
	for h := 2; h <= 7 && n < 999; h++ {
		for m := 0; m < 24*60 && n < 999; m += 1 {
			jam := fmt.Sprintf("%02d:%02d", m/60, m%60)
			harus(s.CreateJadwal(ctx, schedule.Jadwal{ProfilID: profil, Hari: h, Jam: jam, JenisBelID: jenis}))(t)
			n++
		}
	}
	d := formatjadwal.Baru("Tambah")
	d.JenisBel = []formatjadwal.JenisBel{{Nama: "Masuk", Warna: "#2563eb"}}
	d.Jadwal = []formatjadwal.Bel{
		{Hari: 1, Jam: "08:00", JenisBel: "Masuk"},
		{Hari: 1, Jam: "08:01", JenisBel: "Masuk"},
	}
	_, err := s.GabungJadwal(ctx, profil, d, nil, nil)
	harapValidasi(t, err)
	if c := len(harus(s.ListJadwal(ctx, profil))(t)); c != 999 {
		t.Fatalf("jadwal = %d setelah gagal", c)
	}
	d.Jadwal = d.Jadwal[:1]
	harus(s.GabungJadwal(ctx, profil, d, nil, nil))(t) // tepat 1000: boleh
}

func TestGabungJadwal_DataTidakValid(t *testing.T) {
	s := bukaStore(t)
	profil, _ := isiProfil(t, s)
	d := dataGabung()
	d.Jadwal[0].Jam = "7:00"
	_, err := s.GabungJadwal(ctx, profil, d, nil, nil)
	harapValidasi(t, err)
	_, err = s.PeriksaGabung(ctx, profil, d, nil)
	harapValidasi(t, err)
}
```

Catatan: di `dataGabung` sengaja hanya satu entri `JenisBel` ("Masuk"). Bel Jumat merujuk "MASUK", yang menurut `Validasi` cocok dengan "Masuk" tanpa membedakan huruf. Hapus dua baris `d.JenisBel = ...` pertama dan ganti dengan satu baris `d.JenisBel = []formatjadwal.JenisBel{{Nama: "Masuk", Warna: "#2563eb"}}`.

Run: `go test ./internal/store/ -run Gabung`
Expected: FAIL (`PeriksaGabung` / `GabungJadwal` belum ada)

- [ ] **Step 4: Implementasi `internal/store/gabung.go`**

```go
package store

import (
	"context"
	"database/sql"
	"strings"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/gabung"
)

type IsiGabung struct {
	JenisBel string `json:"jenisBel"`
	Label    string `json:"label"`
}

type BarisGabung struct {
	Hari   int           `json:"hari"`
	Jam    string        `json:"jam"`
	Status gabung.Status `json:"status"`
	Lama   *IsiGabung    `json:"lama"`
	Baru   IsiGabung     `json:"baru"`
}

type RingkasanGabung struct {
	Baru    int `json:"baru"`
	Kembar  int `json:"kembar"`
	Bentrok int `json:"bentrok"`
}

type HasilPeriksaGabung struct {
	JenisBelTakDikenal []string        `json:"jenisBelTakDikenal"`
	Ringkasan          RingkasanGabung `json:"ringkasan"`
	Baris              []BarisGabung   `json:"baris"`
}

type HasilGabung struct {
	Ditambah int `json:"ditambah"`
	Ditimpa  int `json:"ditimpa"`
	Dilewati int `json:"dilewati"`
}

// masukDari mengubah bel d menjadi masukan gabung dengan id jenis dari idJenis (kunciNama -> id).
func masukDari(d formatjadwal.Data, idJenis map[string]int64) []gabung.Masuk {
	out := make([]gabung.Masuk, len(d.Jadwal))
	for i, b := range d.Jadwal {
		out[i] = gabung.Masuk{Hari: b.Hari, Jam: b.Jam,
			Isi: gabung.Isi{JenisBelID: idJenis[kunciNama(b.JenisBel)], Label: strings.TrimSpace(b.Label)}}
	}
	return out
}

func jadwalLama(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, profilID int64) ([]gabung.Lama, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, hari, jam, jenis_bel_id, label FROM jadwal WHERE profil_id = ?`, profilID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []gabung.Lama
	for rows.Next() {
		var l gabung.Lama
		if err := rows.Scan(&l.JadwalID, &l.Hari, &l.Jam, &l.JenisBelID, &l.Label); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// PeriksaGabung mengklasifikasi bel d terhadap jadwal profil tanpa mengubah apa pun.
// Jenis yang belum dipetakan (atau dipetakan ke jenis baru) dianggap berbeda dari semua jenis yang ada.
func (s *Store) PeriksaGabung(ctx context.Context, profilID int64, d formatjadwal.Data, pemetaan map[string]PemetaanJenis) (HasilPeriksaGabung, error) {
	if err := d.Validasi(); err != nil {
		return HasilPeriksaGabung{}, dariFormat(err)
	}
	if _, err := s.GetProfil(ctx, profilID); err != nil {
		return HasilPeriksaGabung{}, err
	}
	peta, err := s.bacaPetaJenis(ctx)
	if err != nil {
		return HasilPeriksaGabung{}, err
	}
	lama, err := jadwalLama(ctx, s.db, profilID)
	if err != nil {
		return HasilPeriksaGabung{}, err
	}
	h := HasilPeriksaGabung{JenisBelTakDikenal: peta.takDikenal(d), Baris: []BarisGabung{}}
	for _, b := range gabung.Klasifikasi(lama, masukDari(d, peta.idSementara(d, pemetaan))) {
		asal := d.Jadwal[b.Indeks]
		bg := BarisGabung{Hari: b.Hari, Jam: b.Jam, Status: b.Status,
			Baru: IsiGabung{JenisBel: strings.TrimSpace(asal.JenisBel), Label: strings.TrimSpace(asal.Label)}}
		if b.Lama != nil {
			bg.Lama = &IsiGabung{JenisBel: peta.namaID[b.Lama.JenisBelID], Label: b.Lama.Label}
		}
		switch b.Status {
		case gabung.Baru:
			h.Ringkasan.Baru++
		case gabung.Kembar:
			h.Ringkasan.Kembar++
		case gabung.Bentrok:
			h.Ringkasan.Bentrok++
		}
		h.Baris = append(h.Baris, bg)
	}
	return h, nil
}

// GabungJadwal menambah bel baru dan menimpa bentrok yang dipilih dalam satu transaksi.
// Klasifikasi dihitung ulang dari database; bel lama di luar masukan tidak disentuh.
func (s *Store) GabungJadwal(ctx context.Context, profilID int64, d formatjadwal.Data, pemetaan map[string]PemetaanJenis, timpa []gabung.Slot) (HasilGabung, error) {
	if err := d.Validasi(); err != nil {
		return HasilGabung{}, dariFormat(err)
	}
	// Semua pembacaan lewat s.db sebelum transaksi: database hanya punya satu koneksi.
	if _, err := s.GetProfil(ctx, profilID); err != nil {
		return HasilGabung{}, err
	}
	peta, err := s.bacaPetaJenis(ctx)
	if err != nil {
		return HasilGabung{}, err
	}
	perluSuara, err := peta.periksaPemetaan(d, pemetaan)
	if err != nil {
		return HasilGabung{}, err
	}
	var suara int64
	if perluSuara {
		if suara, err = s.suaraJenisBaru(ctx); err != nil {
			return HasilGabung{}, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return HasilGabung{}, err
	}
	defer tx.Rollback()
	idJenis, err := peta.buatJenis(ctx, tx, d, pemetaan, suara)
	if err != nil {
		return HasilGabung{}, err
	}
	lama, err := jadwalLama(ctx, tx, profilID)
	if err != nil {
		return HasilGabung{}, err
	}
	r := gabung.Rencanakan(gabung.Klasifikasi(lama, masukDari(d, idJenis)), timpa)
	if len(lama)+len(r.Tambah) > formatjadwal.MaksBel {
		return HasilGabung{}, validasi("profil akan berisi %d bel, melebihi batas %d bel", len(lama)+len(r.Tambah), formatjadwal.MaksBel)
	}
	for _, b := range r.Tambah {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO jadwal (profil_id, hari, jam, jenis_bel_id, label) VALUES (?, ?, ?, ?, ?)`,
			profilID, b.Hari, b.Jam, b.Baru.JenisBelID, b.Baru.Label); err != nil {
			return HasilGabung{}, terjemahkan(err, errRefJadwal)
		}
	}
	for _, b := range r.Timpa {
		if _, err := tx.ExecContext(ctx, `UPDATE jadwal SET jenis_bel_id = ?, label = ? WHERE id = ?`,
			b.Baru.JenisBelID, b.Baru.Label, b.Lama.JadwalID); err != nil {
			return HasilGabung{}, terjemahkan(err, errRefJadwal)
		}
	}
	if err := tx.Commit(); err != nil {
		return HasilGabung{}, err
	}
	return HasilGabung{Ditambah: len(r.Tambah), Ditimpa: len(r.Timpa), Dilewati: r.Dilewati}, nil
}
```

Catatan untuk implementer: kalau `GetProfil` mengembalikan error lain untuk ID yang tidak ada, periksa `jadwal.go:51`. Tes `TestPeriksaGabung_ProfilTidakAda` menuntut `errors.Is(err, ErrTidakAda)`.

- [ ] **Step 5: Jalankan semua tes**

Run: `go vet ./... && go test -race ./internal/store/ ./internal/api/ ./internal/gabung/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/store
git commit -m "feat(store): periksa dan gabung jadwal ke profil; resolusi jenis bel dipakai bersama impor"
```

---

### Task 3: API — `POST /api/profil/{id}/gabung/periksa` dan `/gabung`

**Files:**
- Create: `internal/api/gabung.go`
- Modify: `internal/api/rute_data.go` (dua baris `rute(...)` setelah `GET /api/profil/{id}/ekspor`)
- Test: `internal/api/gabung_test.go`

**Interfaces:**
- Consumes: `Store.PeriksaGabung`, `Store.GabungJadwal`, `store.PemetaanJenis`, `gabung.Slot` (Tugas 1–2). Helper yang sudah ada: `idDariPath`, `bacaJSON`, `tulisJSON`, `tulisError`, `s.tulisErrorData`, `s.ubahData()`.
- Produces: kontrak JSON sesuai spec §4 (dipakai Tugas 7).

- [ ] **Step 1: Tulis tes yang gagal**

Helper `siapkan`, `masuk`, `kirim`, `harapKode`, `bacaBody`, `pesanError`, `itoa`, `profilBerisi` (2 bel Senin 07:00 & 12:00 label "Uji", jenis `js[0]`), dan `l.penjadwal.jumlahReload()` sudah ada di paket ini.

```go
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
		JenisBelTakDikenal []string `json:"jenisBelTakDikenal"`
		Ringkasan          struct{ Baru, Kembar, Bentrok int } `json:"ringkasan"`
		Baris              []struct {
			Hari   int    `json:"hari"`
			Jam    string `json:"jam"`
			Status string `json:"status"`
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

	tamu := l.klienBaru(t) // klien tanpa login
	resp = kirim(t, tamu, "POST", url+"/periksa", map[string]any{"data": asing})
	harapKode(t, resp, 401)
	resp.Body.Close()
}
```

Catatan: kalau helper klien tanpa login di `bantu_test.go` bernama lain, cari dengan `grep -n "func (l \*lingkungan)" internal/api/bantu_test.go` dan pakai yang ada. Kalau tidak ada, buat `&http.Client{Jar: ...}` seperti di `bantu_test.go`.

Run: `go test ./internal/api/ -run Gabung`
Expected: FAIL (404 karena rute belum ada)

- [ ] **Step 2: Implementasi**

`internal/api/gabung.go`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"belsekolah/internal/formatjadwal"
	"belsekolah/internal/gabung"
	"belsekolah/internal/store"
)

type gabungDTO struct {
	Data     json.RawMessage                `json:"data"`
	Pemetaan map[string]store.PemetaanJenis `json:"pemetaan"`
	Timpa    []gabung.Slot                  `json:"timpa"`
}

// bacaGabung membaca id profil dan badan permintaan gabung. Menulis error dan mengembalikan false jika gagal.
func bacaGabung(w http.ResponseWriter, r *http.Request) (int64, gabungDTO, formatjadwal.Data, bool) {
	var in gabungDTO
	id, ok := idDariPath(w, r)
	if !ok || !bacaJSON(w, r, &in) {
		return 0, in, formatjadwal.Data{}, false
	}
	d, err := formatjadwal.Baca(in.Data)
	if err != nil {
		tulisError(w, http.StatusBadRequest, err.Error())
		return 0, in, formatjadwal.Data{}, false
	}
	return id, in, d, true
}

func (s *server) periksaGabung(w http.ResponseWriter, r *http.Request) {
	id, in, d, ok := bacaGabung(w, r)
	if !ok {
		return
	}
	h, err := s.Store.PeriksaGabung(r.Context(), id, d, in.Pemetaan)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	tulisJSON(w, http.StatusOK, h)
}

func (s *server) gabungJadwal(w http.ResponseWriter, r *http.Request) {
	id, in, d, ok := bacaGabung(w, r)
	if !ok {
		return
	}
	h, err := s.Store.GabungJadwal(r.Context(), id, d, in.Pemetaan, in.Timpa)
	if err != nil {
		s.tulisErrorData(w, err)
		return
	}
	s.ubahData()
	tulisJSON(w, http.StatusOK, h)
}
```

Catatan: kalau `formatjadwal.Baca(nil)` panik atau menghasilkan pesan yang aneh untuk `data` kosong, tambahkan pengecekan `len(in.Data) == 0` → 400 `"data jadwal wajib diisi"`.

Di `rute_data.go`, setelah `rute("GET /api/profil/{id}/ekspor", s.eksporProfil)` tambahkan:

```go
	rute("POST /api/profil/{id}/gabung/periksa", s.periksaGabung)
	rute("POST /api/profil/{id}/gabung", s.gabungJadwal)
```

- [ ] **Step 3: Jalankan tes**

Run: `go vet ./... && go test -race ./...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/api
git commit -m "feat(api): endpoint periksa dan gabung jadwal ke profil"
```

---

### Task 4: Frontend lib — `tambahBanyak.ts` dan tipe API

**Files:**
- Create: `web/src/lib/tambahBanyak.ts`
- Test: `web/src/lib/tambahBanyak.test.ts`
- Modify: `web/src/lib/api.ts` (tambah tipe di bawah `ParameterPembuat`)

**Interfaces:**
- Produces (dipakai Tugas 5–7):
  - Di `api.ts`:
    ```ts
    export type StatusGabung = 'baru' | 'kembar' | 'bentrok'
    export interface IsiGabung { jenisBel: string; label: string }
    export interface BarisGabung { hari: number; jam: string; status: StatusGabung; lama: IsiGabung | null; baru: IsiGabung }
    export interface HasilPeriksaGabung {
      jenisBelTakDikenal: string[]
      ringkasan: { baru: number; kembar: number; bentrok: number }
      baris: BarisGabung[]
    }
    export interface HasilGabung { ditambah: number; ditimpa: number; dilewati: number }
    ```
  - Di `tambahBanyak.ts`: `BARU_JENIS`, `BarisKetik`, `GalatTempel`, `normalJam`, `uraiTempel`, `jamGanda`, `keFileJadwal`, `saringHari`, `hariBerisi`, `kePemetaan`, `kunciSlot` (lihat kode).

- [ ] **Step 1: Tulis tes yang gagal**

```ts
import { describe, expect, it } from 'vitest'
import { hariBerisi, jamGanda, keFileJadwal, kePemetaan, kunciSlot, normalJam, saringHari, uraiTempel } from './tambahBanyak'

describe('normalJam', () => {
  it('menerima titik, titik dua, dan satu digit jam', () => {
    expect(normalJam('7.15')).toBe('07:15')
    expect(normalJam('7:15')).toBe('07:15')
    expect(normalJam(' 07:15 ')).toBe('07:15')
    expect(normalJam('23.59')).toBe('23:59')
  })
  it('menolak yang tidak valid', () => {
    for (const s of ['', '24:00', '7:60', '7', 'jam', '07:5', '07:155']) expect(normalJam(s)).toBeNull()
  })
})

describe('uraiTempel', () => {
  it('memisah dengan TAB, titik koma, atau koma; kolom jenis kosong memakai bawaan', () => {
    const teks = '07:15\tSholat Dhuha\tMasuk\n7.30;Literasi;\n\n  \n08:00, Jam ke-1 ,Ganti Jam'
    const h = uraiTempel(teks, 'Bawaan')
    expect(h.galat).toEqual([])
    expect(h.baris).toEqual([
      { jam: '07:15', label: 'Sholat Dhuha', jenis: 'Masuk' },
      { jam: '07:30', label: 'Literasi', jenis: 'Bawaan' },
      { jam: '08:00', label: 'Jam ke-1', jenis: 'Ganti Jam' },
    ])
  })
  it('hanya jam juga boleh', () => {
    expect(uraiTempel('09:00', 'Masuk').baris).toEqual([{ jam: '09:00', label: '', jenis: 'Masuk' }])
  })
  it('baris rusak dilaporkan dengan nomor baris dan tidak ikut', () => {
    const h = uraiTempel('jam\tX\n07:00\tOk\n1;2;3;4\n08:00\t' + 'a'.repeat(101), 'Masuk')
    expect(h.baris).toEqual([{ jam: '07:00', label: 'Ok', jenis: 'Masuk' }])
    expect(h.galat.map((g) => g.nomor)).toEqual([1, 3, 4])
    expect(h.galat[0].alasan).toContain('jam')
  })
})

describe('jamGanda', () => {
  it('mengembalikan jam yang muncul lebih dari sekali, urut', () => {
    expect(jamGanda([{ jam: '08:00', label: '', jenis: 'A' }, { jam: '07:00', label: '', jenis: 'A' }, { jam: '08:00', label: 'x', jenis: 'B' }])).toEqual(['08:00'])
  })
})

describe('keFileJadwal', () => {
  it('memperluas baris × hari dan mengisi warna jenis tanpa beda huruf', () => {
    const d = keFileJadwal(
      [{ jam: '07:15', label: 'Dhuha', jenis: 'masuk' }, { jam: '09:00', label: 'Ist', jenis: 'Literasi' }],
      [3, 1],
      [{ nama: 'Masuk', warna: '#2563eb' }],
    )
    expect(d.format).toBe('belsekolah-jadwal')
    expect(d.versi).toBe(1)
    expect(d.nama).toBe('Tambah banyak')
    expect(d.jenisBel).toEqual([{ nama: 'masuk', warna: '#2563eb' }, { nama: 'Literasi', warna: '#64748b' }])
    expect(d.jadwal).toEqual([
      { hari: 1, jam: '07:15', jenisBel: 'masuk', label: 'Dhuha' },
      { hari: 1, jam: '09:00', jenisBel: 'Literasi', label: 'Ist' },
      { hari: 3, jam: '07:15', jenisBel: 'masuk', label: 'Dhuha' },
      { hari: 3, jam: '09:00', jenisBel: 'Literasi', label: 'Ist' },
    ])
  })
  it('nama jenis yang hanya beda huruf dicatat sekali', () => {
    const d = keFileJadwal([{ jam: '07:00', label: '', jenis: 'Masuk' }, { jam: '08:00', label: '', jenis: 'MASUK' }], [1], [])
    expect(d.jenisBel).toHaveLength(1)
  })
})

describe('saringHari & hariBerisi', () => {
  const d = keFileJadwal([{ jam: '07:00', label: '', jenis: 'A' }], [1, 2, 5], [])
  it('hariBerisi urut naik', () => expect(hariBerisi(d)).toEqual([1, 2, 5]))
  it('saringHari hanya menyisakan hari terpilih tanpa mengubah aslinya', () => {
    const s = saringHari(d, [5, 1])
    expect(s.jadwal.map((b) => b.hari)).toEqual([1, 5])
    expect(d.jadwal).toHaveLength(3)
  })
})

describe('kePemetaan & kunciSlot', () => {
  it('mengubah pilihan select ke pemetaan API', () => {
    expect(kePemetaan({ A: 'baru', B: '7' })).toEqual({ A: { baru: true }, B: { jenisBelId: 7 } })
  })
  it('kunciSlot', () => expect(kunciSlot(1, '07:00')).toBe('1 07:00'))
})
```

Run: `cd web && npx vitest run src/lib/tambahBanyak.test.ts`
Expected: FAIL (modul belum ada)

- [ ] **Step 2: Implementasi `web/src/lib/tambahBanyak.ts`**

```ts
import type { FileJadwal, Pemetaan } from './api'

/** Nilai <option> "Buat jenis bel baru" pada pemetaan jenis. */
export const BARU_JENIS = 'baru'
const WARNA_CADANGAN = '#64748b'
const MAKS_LABEL = 100

export interface BarisKetik { jam: string; label: string; jenis: string }
export interface GalatTempel { nomor: number; teks: string; alasan: string }

const dua = (n: number) => String(n).padStart(2, '0')
const kunci = (n: string) => n.trim().toLowerCase()

/** '7.15' | '7:15' | '07:15' → '07:15'; selain itu null. */
export function normalJam(s: string): string | null {
  const m = s.trim().match(/^(\d{1,2})[.:](\d{2})$/)
  if (!m) return null
  const j = Number(m[1])
  const mnt = Number(m[2])
  if (j > 23 || mnt > 59) return null
  return `${dua(j)}:${dua(mnt)}`
}

/** Satu bel per baris: `jam, keterangan, jenis bel` dipisah TAB, ';' atau ','. Jenis kosong → jenisBawaan. */
export function uraiTempel(teks: string, jenisBawaan: string): { baris: BarisKetik[]; galat: GalatTempel[] } {
  const baris: BarisKetik[] = []
  const galat: GalatTempel[] = []
  teks.split(/\r?\n/).forEach((mentah, i) => {
    if (!mentah.trim()) return
    const pemisah = mentah.includes('\t') ? '\t' : mentah.includes(';') ? ';' : ','
    const kolom = mentah.split(pemisah).map((k) => k.trim())
    const nomor = i + 1
    if (kolom.length > 3) {
      galat.push({ nomor, teks: mentah, alasan: 'terlalu banyak kolom (maksimal 3: jam, keterangan, jenis bel)' })
      return
    }
    const jam = normalJam(kolom[0] ?? '')
    if (!jam) {
      galat.push({ nomor, teks: mentah, alasan: `jam "${kolom[0]}" tidak valid, gunakan HH:MM` })
      return
    }
    const label = kolom[1] ?? ''
    if ([...label].length > MAKS_LABEL) {
      galat.push({ nomor, teks: mentah, alasan: `keterangan maksimal ${MAKS_LABEL} karakter` })
      return
    }
    baris.push({ jam, label, jenis: kolom[2] || jenisBawaan })
  })
  return { baris, galat }
}

/** Jam yang muncul lebih dari sekali (urut naik). */
export function jamGanda(baris: BarisKetik[]): string[] {
  const hitung = new Map<string, number>()
  for (const b of baris) hitung.set(b.jam, (hitung.get(b.jam) ?? 0) + 1)
  return [...hitung].filter(([, n]) => n > 1).map(([j]) => j).sort()
}

/** Setiap baris × setiap hari → file jadwal. Warna jenis dari daftar jenis (tanpa beda huruf), selain itu abu-abu. */
export function keFileJadwal(
  baris: BarisKetik[],
  hari: number[],
  jenis: { nama: string; warna: string }[],
  nama = 'Tambah banyak',
): FileJadwal {
  const warna = new Map(jenis.map((j) => [kunci(j.nama), j.warna]))
  const jenisBel: FileJadwal['jenisBel'] = []
  const sudah = new Set<string>()
  for (const b of baris) {
    const k = kunci(b.jenis)
    if (sudah.has(k)) continue
    sudah.add(k)
    jenisBel.push({ nama: b.jenis.trim(), warna: warna.get(k) ?? WARNA_CADANGAN })
  }
  const jadwal: FileJadwal['jadwal'] = []
  for (const h of [...hari].sort((a, b) => a - b)) {
    for (const b of baris) jadwal.push({ hari: h, jam: b.jam, jenisBel: b.jenis.trim(), label: b.label.trim() })
  }
  return { format: 'belsekolah-jadwal', versi: 1, nama, jenisBel, jadwal }
}

/** Salinan d yang hanya berisi bel pada hari terpilih. */
export function saringHari(d: FileJadwal, hari: number[]): FileJadwal {
  const pilih = new Set(hari)
  return { ...d, jenisBel: [...d.jenisBel], jadwal: d.jadwal.filter((b) => pilih.has(b.hari)) }
}

/** Hari (1–7, naik) yang punya bel. */
export function hariBerisi(d: FileJadwal): number[] {
  return [...new Set(d.jadwal.map((b) => b.hari))].sort((a, b) => a - b)
}

/** Pilihan <select> pemetaan ('baru' | id sebagai string) → bentuk API. */
export function kePemetaan(pilihan: Record<string, string>): Pemetaan {
  const out: Pemetaan = {}
  for (const [n, v] of Object.entries(pilihan)) out[n] = v === BARU_JENIS ? { baru: true } : { jenisBelId: Number(v) }
  return out
}

export const kunciSlot = (hari: number, jam: string) => `${hari} ${jam}`
```

Tambahkan tipe gabung di `api.ts` (lihat Interfaces).

- [ ] **Step 3: Jalankan tes**

Run: `cd web && npx vitest run && npm run check`
Expected: PASS, 0 error, 0 warning

- [ ] **Step 4: Commit**

```bash
git add web/src/lib/tambahBanyak.ts web/src/lib/tambahBanyak.test.ts web/src/lib/api.ts
git commit -m "feat(web): logika tambah banyak (urai tempelan, perluasan hari, filter, pemetaan) dan tipe API gabung"
```

---

### Task 5: Komponen bersama `PemetaanJenis` dan `FormPembuat` (refaktor tanpa perubahan perilaku)

**Files:**
- Create: `web/src/components/PemetaanJenis.svelte`
- Create: `web/src/components/FormPembuat.svelte`
- Modify: `web/src/components/AlurImpor.svelte`
- Modify: `web/src/pages/Template.svelte`

**Interfaces:**
- Produces (dipakai Tugas 6–7):
  - `PemetaanJenis.svelte` props: `{ nama: string[]; jenisBel: JenisBel[]; pilihan: Record<string, string> ($bindable); awalanId: string }`. Merender paragraf penjelas yang sama dengan AlurImpor sekarang ("Jenis bel berikut belum ada. Pilih jenis bel yang dipakai untuk masing-masing."), lalu satu `<Medan label={n} untuk="{awalanId}-{i}">` + `<select id="{awalanId}-{i}">` per nama, berisi opsi pertama `BARU_JENIS` "Buat jenis bel baru" dan opsi per jenis.
  - `FormPembuat.svelte` props: `{ hasil: FileJadwal | null ($bindable); menghitung: boolean ($bindable); galat: string ($bindable); hariSekolah: number[] ($bindable) }`. Isinya seluruh state pembuat yang sekarang ada di `Template.svelte` baris 37–98 (`param`, `istirahatAktif`, `ganti`, `angka`, `badan`, efek debounce 300 ms ke `POST /template/buat`, `hariSekolah`) beserta `<form aria-label="Pembuat jadwal">…</form>` (baris 233–358), **tanpa mengubah id input (`pb-*`) maupun teks**. `JENJANG`, `VARIAN`, `kelasInput`, dan `kelasCentang` ikut pindah. Hasil efek ditulis ke `hasil`, `galat`, dan `menghitung`. `hariSekolah` = `param.hariSekolah === 6 ? [1..6] : [1..5]`, ditulis lewat `$effect`.
- Consumes: `BARU_JENIS`, `kePemetaan` dari `lib/tambahBanyak.ts` (Tugas 4). Kalau Tugas 4 belum di-merge di worktree ini, buat dulu dua ekspor itu persis seperti di Tugas 4 (isi berkas yang sama), lalu biarkan cherry-pick menyatukannya.

- [ ] **Step 1: Buat `PemetaanJenis.svelte`** dari blok `{#if hasil.jenisBelTakDikenal.length > 0}…{/if}` di `AlurImpor.svelte` (isi di dalam `{#if}`). Ganti `hasil.jenisBelTakDikenal` dengan `nama` dan `impor-peta` dengan `awalanId`.

- [ ] **Step 2: Pakai di `AlurImpor.svelte`**: `{#if hasil.jenisBelTakDikenal.length > 0}<PemetaanJenis nama={hasil.jenisBelTakDikenal} {jenisBel} bind:pilihan awalanId="impor-peta" />{/if}`. Ganti `const BARU = 'baru'` dengan impor `BARU_JENIS`, dan perulangan pembentuk `pemetaan` dengan `kePemetaan(pilihan)`.

- [ ] **Step 3: Buat `FormPembuat.svelte`** dengan memindahkan kode seperti di Interfaces. Di `Template.svelte`, ganti state dan form tersebut dengan:

```svelte
<FormPembuat bind:hasil={pratinjau} bind:menghitung bind:galat={galatPratinjau} bind:hariSekolah />
```

Deklarasikan `let pratinjau = $state<FileJadwal | null>(null)`, `let menghitung = $state(false)`, `let galatPratinjau = $state('')`, `let hariSekolah = $state<number[]>([1, 2, 3, 4, 5])`. Sisa halaman (pratinjau, "Buat profil dari jadwal ini", Template saya) tidak berubah. Import yang tidak terpakai dihapus.

- [ ] **Step 4: Verifikasi tanpa perubahan perilaku**

Run: `cd web && npm run check && npx vitest run && PORT=18091 npx playwright test e2e/template.spec.ts e2e/cadangan.spec.ts`
Expected: 0 error/warning; semua tes lulus (spec yang hanya jalan di Docker boleh dilewati).

- [ ] **Step 5: Commit**

```bash
git add web/src/components/PemetaanJenis.svelte web/src/components/FormPembuat.svelte web/src/components/AlurImpor.svelte web/src/pages/Template.svelte
git commit -m "refactor(web): pisahkan PemetaanJenis dan FormPembuat agar dapat dipakai ulang"
```

---

### Task 6: `SumberTambahBanyak.svelte` (tiga tab sumber)

**Files:**
- Create: `web/src/components/SumberTambahBanyak.svelte`

**Interfaces:**
- Consumes:
  - `lib/tambahBanyak.ts`: `uraiTempel`, `jamGanda`, `keFileJadwal`, `saringHari`, `hariBerisi`, `BarisKetik`, `GalatTempel`
  - `FormPembuat.svelte` (Tugas 5)
  - `api`: `GET /template` (`Template[]`), `GET /template/{id}` (`Template` dengan `data`), `GET /profil` (`Profil[]`), `GET /profil/{id}/ekspor` lewat `api.get<FileJadwal>` (server menjawab `application/json` dengan Content-Disposition; `api.get` mengurai JSON)
  - `NAMA_HARI`
- Produces (dipakai Tugas 7): props `{ profilId: number; jenisBel: JenisBel[]; data: FileJadwal | null ($bindable); galat: string ($bindable) }`.
  - `data` adalah file jadwal yang **sudah difilter hari**. Nilainya `null` kalau sumber kosong atau tidak valid.
  - `galat` berisi alasan tombol Periksa nonaktif (misalnya "Jam 07:00 ganda"); string kosong kalau tidak ada.

**Perilaku (spec §5.2):**
- Tab (`role="tablist"`, tombol `role="tab"` dengan `aria-selected`): **"Ketik/tempel"** (bawaan), **"Template"**, **"File/profil lain"**. Isi tiap tab dibungkus `role="tabpanel"`.
- **Ketik/tempel:**
  - `<select id="tb-jenis-bawaan">` berlabel "Jenis bel bawaan", berisi jenis yang ada; bawaannya jenis pertama.
  - `<textarea id="tb-tempel">` berlabel "Tempel dari Excel/teks", dengan placeholder `07:15<TAB>Sholat Dhuha<TAB>Masuk`, dan tombol **"Tambahkan ke daftar"**. Tombol ini menjalankan `uraiTempel(teks, namaJenisBawaan)`, menambahkan `baris` ke daftar, mengosongkan textarea kalau tidak ada galat, dan menampilkan galat per baris (`role="alert"`, "Baris {nomor}: {alasan}").
  - Daftar isian: tiap baris berisi `<input type="time">` Jam (label "Jam baris {i+1}"), input Keterangan (maxlength 100, label "Keterangan baris {i+1}"), `<select>` Jenis bel berisi nama jenis yang ada (label "Jenis bel baris {i+1}"; kalau nama dari tempelan tidak ada di daftar, tambahkan sebagai `<option>` ekstra supaya nilainya tetap), dan tombol ikon hapus `aria-label="Hapus baris {i+1}"`. Ada tombol **"+ Baris"**. Daftar awal berisi satu baris kosong (jam '', jenis bawaan).
  - Fieldset "Hari tujuan": 7 checkbox `NAMA_HARI`, Sen–Jum tercentang.
  - `data` = `keFileJadwal(barisTerisi, hariTujuan, jenisBel)`, dengan `barisTerisi` = baris yang jamnya terisi. `data` bernilai null kalau tidak ada baris terisi atau tidak ada hari. `galat` = `Jam ${jamGanda(...).join(', ')} ganda` kalau ada jam ganda; saat itu `data` juga null.
- **Template:**
  - Tombol pilihan "Pembuat jadwal" / "Template saya".
  - "Pembuat jadwal" merender `<FormPembuat bind:hasil bind:menghitung bind:galat={galatPembuat} />`.
  - "Template saya" berupa `<select id="tb-template">` dari `GET /template`. Saat dipilih, ambil `GET /template/{id}` lalu pakai `.data`.
- **File/profil lain:**
  - `<input type="file" accept=".json,application/json">` berlabel "File jadwal (.json)". Baca dengan `await f.text()` lalu `JSON.parse` di dalam try; kalau gagal, `galat = 'File tidak bisa dibaca sebagai file jadwal'`.
  - **atau** `<select id="tb-profil">` "Profil lain" dari `GET /profil`, tanpa `profilId` sasaran. Saat dipilih, ambil `GET /profil/{id}/ekspor`.
- **Ambil hari** (untuk Template dan File/profil lain): fieldset berisi checkbox hanya untuk `hariBerisi(sumber)`, semuanya tercentang saat sumber berganti. `data = saringHari(sumber, hariTerpilih)`, atau null kalau hasilnya 0 bel.
- Ringkasan di bawah (`data-testid="ringkasan-sumber"`): `"{n} bel dari {h} hari"` atau "Belum ada bel".
- Semua permintaan memakai pola nomor-urut (`saya !== urutan`) supaya respons usang diabaikan. Galat muat ditampilkan `role="alert"`.

- [ ] **Step 1: Implementasi komponen** sesuai perilaku di atas, memakai kelas masukan `min-h-11 w-full rounded-xl border border-slate-300 ...` seperti komponen lain.
- [ ] **Step 2: Verifikasi**: `cd web && npm run check` (0 error/warning) dan `npx vitest run`. Komponen ini diuji end-to-end di Tugas 8.
- [ ] **Step 3: Commit**

```bash
git add web/src/components/SumberTambahBanyak.svelte
git commit -m "feat(web): sumber tambah banyak (ketik/tempel, template, file/profil lain)"
```

---

### Task 7: `PratinjauGabung`, `TambahBanyak`, dan tombol di halaman Jadwal

**Files:**
- Create: `web/src/components/PratinjauGabung.svelte`
- Create: `web/src/components/TambahBanyak.svelte`
- Modify: `web/src/pages/Jadwal.svelte`

**Interfaces:**
- Consumes: `SumberTambahBanyak` (Tugas 6), `PemetaanJenis` (Tugas 5), `kePemetaan`, `kunciSlot`, `BARU_JENIS` (Tugas 4), tipe `HasilPeriksaGabung`, `BarisGabung`, `HasilGabung` (Tugas 4), `konfirmasi`, `beriTahu`, `Tabel`, `KartuStatistik`, `NAMA_HARI`.
- Produces:
  - `PratinjauGabung.svelte` props: `{ hasil: HasilPeriksaGabung; timpa: Set<string> ($bindable) }`. `timpa` berisi `kunciSlot` bentrok yang akan ditimpa.
  - `TambahBanyak.svelte` props: `{ buka: boolean ($bindable); profil: Profil; jenisBel: JenisBel[]; onselesai: () => void }`.

**PratinjauGabung (spec §5.4):**
- Tiga `KartuStatistik`: "Baru", "Kembar", "Bentrok", dengan angka dari `hasil.ringkasan`.
- Kalau ada bentrok: radiogroup "Bel yang bentrok" dengan ◉ "Pertahankan yang lama" (bawaan) dan ○ "Timpa dengan yang baru". Mengubahnya mengisi atau mengosongkan `timpa` untuk semua baris bentrok.
- Checkbox "Tampilkan kembar" (bawaan tidak dicentang). Baris kembar disembunyikan kalau tidak dicentang.
- Dikelompokkan per hari (`<h3>` NAMA_HARI).
  - Di laptop (`hidden md:block`): `Tabel` dengan kolom Jam, Status, Lama, Baru, Tindakan.
  - Di HP (`md:hidden`): kartu bertumpuk dengan isi yang sama.
  - Status berupa lencana: baru `bg-emerald-50 text-emerald-700`, kembar `bg-slate-100 text-slate-600`, bentrok `bg-amber-50 text-amber-800`. Teksnya "Baru", "Kembar", "Bentrok".
  - Lama/Baru ditampilkan sebagai `"{label || '—'} · {jenisBel}"`.
  - Kolom Tindakan untuk bentrok berisi `<select aria-label="Tindakan {NAMA_HARI[hari]} {jam}">` dengan opsi "Pertahankan" / "Timpa", terikat ke keanggotaan `timpa`. Baris lain berisi "—".

**TambahBanyak (spec §5.1, 5.3, 5.5):**
- `PanelGeser lebar="lg" judul="Tambah banyak ke profil {profil.nama}"`.
- State `langkah: 'sumber' | 'pemetaan' | 'pratinjau'`. Saat `buka` berubah ke true, semuanya diatur ulang ke 'sumber' (data null, pilihan kosong, timpa kosong).
- **sumber**: `<SumberTambahBanyak profilId={profil.id} {jenisBel} bind:data bind:galat={galatSumber} />`. Aksi: Batal, **Periksa** (`disabled={!data || !!galatSumber}`).
- **Periksa**: `POST /profil/{id}/gabung/periksa` `{ data, pemetaan: kePemetaan(pilihan) }`.
  - Kalau `jenisBelTakDikenal` berisi nama yang belum ada di `pilihan` (pemeriksaan pertama), isi `pilihan[n] = BARU_JENIS` untuk tiap nama dan pindah ke 'pemetaan'.
  - Selain itu pindah ke 'pratinjau' dengan `timpa = new Set()`.
- **pemetaan**: `<PemetaanJenis nama={hasil.jenisBelTakDikenal} {jenisBel} bind:pilihan awalanId="tb-peta" />`. Aksi: **Kembali** (ke sumber), **Lanjut**. Lanjut memanggil periksa ulang dengan `pemetaan`, lalu pindah ke 'pratinjau'.
- **pratinjau**: `<PratinjauGabung {hasil} bind:timpa />`.
  - Aksi: **Kembali**, lalu **"Gabungkan ({n} perubahan)"** dengan `n = ringkasan.baru + timpa.size`. Tombol nonaktif kalau n = 0.
  - Kalau `timpa.size > 0`, tampilkan dulu `konfirmasi('Timpa bel lama?', '{timpa.size} bel lama akan diganti dengan yang baru.', 'Timpa')`. Kalau tidak dikonfirmasi, batalkan.
  - Kirim `POST /profil/{id}/gabung` `{ data, pemetaan, timpa: [...timpa].map(k => ({ hari: Number(k.split(' ')[0]), jam: k.split(' ')[1] })) }`.
  - Kalau sukses: `buka = false`, `beriTahu('{ditambah} ditambah, {ditimpa} ditimpa, {dilewati} dilewati')`, lalu `onselesai()`.
  - Kalau gagal: tetap di langkah ini dan tampilkan galat `role="alert"`.
- Galat periksa tampil di langkah aktif (`role="alert"`). Saat memeriksa, tampilkan "Memeriksa…" `role="status"`.

**Jadwal.svelte:**
- Import `TambahBanyak`.
- Tambah state `bukaBanyak = $state(false)`.
- Di `{#snippet kanan()}`, sebelum `+ Tambah bel`, tambahkan `<Tombol varian="kedua" onclick={() => (bukaBanyak = true)} disabled={!profilId}>Tambah banyak</Tombol>`.
- Di akhir halaman: `{#if profilAktif}<TambahBanyak bind:buka={bukaBanyak} profil={profilAktif} jenisBel={jenis} onselesai={() => muatJadwal()} />{/if}`, dengan `const profilAktif = $derived(profil.find((p) => p.id === profilId))`. Pakai `$derived` yang sama kalau sudah ada.
- Setelah gabung, muat ulang juga `jenis` (`api.get<JenisBel[]>('/jenis-bel')`) karena mungkin ada jenis baru.

- [ ] **Step 1: Implementasi `PratinjauGabung.svelte`**
- [ ] **Step 2: Implementasi `TambahBanyak.svelte`**
- [ ] **Step 3: Pasang di `Jadwal.svelte`**
- [ ] **Step 4: Verifikasi**: `cd web && npm run check && npx vitest run`, lalu `PORT=18092 npx playwright test e2e/jadwal.spec.ts` (tes jadwal lama tetap lulus).
- [ ] **Step 5: Commit**

```bash
git add web/src/components/PratinjauGabung.svelte web/src/components/TambahBanyak.svelte web/src/pages/Jadwal.svelte
git commit -m "feat(web): panel tambah banyak dengan pratinjau gabung di halaman Jadwal"
```

---

### Task 8: E2E, build produksi, dokumentasi

**Files:**
- Create: `web/e2e/tambah-banyak.spec.ts`
- Modify: `internal/webui/dist/**` (hasil `npm run build`)
- Modify: `README.md` (satu kalimat di bagian fitur/penggunaan: "Tambah banyak" di halaman Jadwal menggabungkan bel dari ketikan/tempelan, template, file, atau profil lain ke profil yang dibuka, dengan pratinjau bentrok)

**Interfaces:**
- Consumes: semua tugas sebelumnya. Helper e2e `apiMasuk`, `jenisBelId`, `unik` dari `e2e/bantu.ts`. Setiap tes membuat profil uji sendiri (`POST /api/profil`), membukanya lewat `/jadwal?profil={id}`, dan menghapusnya di `finally`.

- [ ] **Step 1: Tulis `web/e2e/tambah-banyak.spec.ts`** dengan 4 tes:
  1. **Ketik/tempel, bentrok dipertahankan lalu ditimpa:**
     - Profil uji berisi Senin 07:00 "Masuk" (jenis "Masuk") lewat API.
     - Buka `/jadwal?profil=ID`, klik "Tambah banyak", lalu isi `#tb-tempel` dengan `07:00\tUpacara\n07:15\tSholat Dhuha`.
     - Klik "Tambahkan ke daftar", biarkan hanya Senin tercentang (hapus centang Sel–Jum), lalu klik "Periksa".
     - Harapkan kartu "Baru" 1 dan "Bentrok" 1. Ubah select `Tindakan Senin 07:00` menjadi "Timpa", klik "Gabungkan (2 perubahan)", lalu konfirmasi "Timpa".
     - Harapkan notifikasi "1 ditambah, 1 ditimpa, 0 dilewati", dan lewat API Senin 07:00 berlabel "Upacara" serta 07:15 ada.
  2. **Bawaan pertahankan:** sama seperti di atas tanpa mengubah tindakan. Gabung (1 perubahan, tanpa konfirmasi), lalu Senin 07:00 tetap "Masuk".
  3. **Dari profil lain dengan filter hari:**
     - Profil sumber berisi Senin 08:00 dan Jumat 08:00. Profil tujuan kosong.
     - Tab "File/profil lain", pilih profil sumber di `#tb-profil`, lalu hapus centang Jumat di Ambil hari.
     - Periksa (Baru 1), Gabungkan, lalu lewat API tujuan hanya berisi Senin 08:00.
  4. **Jenis tak dikenal → pemetaan:**
     - Tempel `09:00\tLiterasi\tBel Literasi Uji` (jenis yang tidak ada), lalu Periksa.
     - Muncul select `#tb-peta-0` berlabel "Bel Literasi Uji" dengan nilai bawaan "Buat jenis bel baru". Pilih jenis "Masuk", lalu klik Lanjut.
     - Pratinjau tampil Baru 1, lalu Gabungkan. Lewat API bel 09:00 berjenis "Masuk", dan jenis "Bel Literasi Uji" **tidak** dibuat.
  - Semua tes jalan di proyek `hp` (Pixel 7). Tes 1 sekaligus memastikan pratinjau tampil sebagai kartu: tabel `md:block` tersembunyi, dan teks "Bentrok" terlihat di kartu.
- [ ] **Step 2: Jalankan**: `cd web && PORT=18093 npx playwright test e2e/tambah-banyak.spec.ts`, lalu seluruh suite `PORT=18093 npx playwright test`. Harapannya semua lulus, kecuali tes khusus Docker yang memang dilewati.
- [ ] **Step 3: Build produksi**: `cd web && env -u VITE_E2E npm run build && ! grep -rq __bel ../internal/webui/dist && cd .. && go test ./...`
- [ ] **Step 4: E2E Docker penuh**: `BEL_PORT=18081 ./scripts/e2e.sh` (harus lulus).
- [ ] **Step 5: Commit**

```bash
git add web/e2e/tambah-banyak.spec.ts internal/webui/dist README.md
git commit -m "test(e2e): alur tambah banyak; build produksi dan README"
```
