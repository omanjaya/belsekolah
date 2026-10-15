# Pengumuman per Bel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Setiap baris jadwal boleh punya satu suara **pengumuman**. Suara ini diputar setelah suara jenis bel dan diisi otomatis dari keterangan (misalnya "Jam ke-3" → suara "Jam ke-3 dimulai").

**Architecture:**
- Kolom `jadwal.pengumuman_audio_id` (nullable FK, migrasi `ALTER TABLE` idempoten).
- `schedule.Jadwal` dan `schedule.Bel` membawa `PengumumanID`. Penjadwal menambahkan berkas pengumuman ke urutan putar.
- Pencocokan keterangan → suara adalah fungsi murni di paket baru `internal/pengumuman`. Fungsi ini dipakai store (impor, gabung, saran) dan API (`/pengumuman/cocok`).
- Frontend: komponen `PilihSuara` (pilih suara dengan pencarian) dan `IsiPengumuman` (panel saran), lalu dipasang di Jadwal, Beranda, alur impor, dan Tambah banyak.

**Tech Stack:** Go 1.27 (net/http, modernc SQLite satu koneksi), Svelte 5 (runes) + TypeScript + Tailwind 4, Vitest, Playwright (proyek `hp`).

**Spec:** `docs/superpowers/specs/2026-10-15-pengumuman-per-bel-design.md`

## Global Constraints

- Bahasa kode, komentar, pesan error, dan teks UI: **Bahasa Indonesia**, mengikuti gaya kode di sekitarnya.
- **Database satu koneksi:** di dalam transaksi jangan panggil method `Store` yang memakai `s.db`. Lakukan pembacaan sebelum `BeginTx`, atau query lewat `tx`.
- **Nilai pengumuman:** `PengumumanID` 0 di Go berarti tidak ada, disimpan sebagai `NULL`. Di JSON nilainya `pengumumanId: number | null`.
- **Pencocokan:** tingkat 1 `n == k`, tingkat 2 `n == k + " dimulai"`, tingkat 3 `n` diawali `k + " "`. Tingkat terbaik dengan tepat satu kandidat menang. Bila ambigu atau tidak ada kandidat, hasilnya tidak ada. Pencocokan mengikuti batas kata.
- **Normalisasi:** huruf kecil; `- _ . , : ( ) /` menjadi spasi; spasi dirapatkan; lalu di-trim.
- **Padanan bawaan:** `pulang`→`akhir pelajaran`, `upacara`→`upacara bendera`, `masuk`→`bel masuk sekolah`, `ujian selesai`→`ujian waktu telah habis`.
- Pilihan pengumuman manual tidak pernah ditimpa pengisian otomatis. Baris yang sudah punya pengumuman tidak pernah ditimpa oleh "terapkan".
- **Rute baru** didaftarkan lewat `rute(...)` (wajib login, cekOrigin, blokir restore).
- **UI:** masukan `min-h-11 rounded-xl`, tombol memakai `Tombol`, panel memakai `PanelGeser`, galat memakai `role="alert"`, status memuat memakai `role="status"`. Tabel dengan lebih dari 3 kolom tampil sebagai kartu di HP.
- Jangan commit build E2E: `internal/webui/dist` tidak boleh berisi `__bel`.
- **Tes:**
  - `go vet ./... && go test -race ./...`
  - `cd web && npm run check && npx vitest run`
  - Playwright dengan `PORT=<unik>` (port 8080, 18080, dan 18081 sudah dipakai)
- **Commit:** pesan diakhiri baris kosong lalu `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Gelombang

| Gelombang | Tugas | Catatan |
|---|---|---|
| G1 | 1, 2 | |
| G2 | 3, 5 | Tugas 5 hanya butuh kontrak API |
| G3 | 4, 6 | |
| G4 | 7 | |

---

### Task 1: Paket `internal/pengumuman` (pencocokan murni)

**Files:** Create `internal/pengumuman/pengumuman.go`, `internal/pengumuman/pengumuman_test.go`

**Interfaces:**
- Produces:
  - `type Kandidat struct{ ID int64; Nama string }`
  - `func Normal(s string) string`
  - `func Cocokkan(keterangan string, suara []Kandidat) (int64, bool)`

- [ ] **Step 1: Tes gagal**

```go
package pengumuman

import "testing"

func TestNormal(t *testing.T) {
	for in, mau := range map[string]string{
		"Jam ke-3":                 "jam ke 3",
		"  Akhir pelajaran (pulang) ": "akhir pelajaran pulang",
		"Sesi 1: Dimulai/IND":      "sesi 1 dimulai ind",
		"a__b..c":                  "a b c",
	} {
		if got := Normal(in); got != mau {
			t.Errorf("Normal(%q) = %q, mau %q", in, got, mau)
		}
	}
}

func TestCocokkan(t *testing.T) {
	suara := []Kandidat{
		{1, "Jam ke-1 dimulai"}, {10, "Jam ke-10 dimulai"}, {2, "Jam ke-2 dimulai"},
		{20, "Istirahat"}, {21, "Istirahat pertama"}, {22, "Istirahat kedua"},
		{30, "Akhir pelajaran (pulang)"}, {40, "Upacara bendera dimulai"},
		{50, "Bel masuk sekolah"}, {60, "Ujian (Waktu Telah Habis)"},
		{70, "Kegiatan Pramuka 5 menit"}, {71, "Kegiatan Pramuka akhir"},
	}
	kasus := []struct {
		ket  string
		id   int64
		ok   bool
	}{
		{"Jam ke-1", 1, true},   // tingkat 2, bukan "jam ke 10 dimulai" (batas kata)
		{"Jam ke-10", 10, true},
		{"jam ke 2", 2, true},
		{"Istirahat", 20, true}, // tingkat 1 mengalahkan tingkat 3
		{"Pulang", 30, true},    // padanan → tingkat 3
		{"Upacara", 40, true},   // padanan → tingkat 2
		{"Masuk", 50, true},     // padanan → tingkat 1
		{"Ujian selesai", 60, true},
		{"Kegiatan Pramuka", 0, false}, // dua kandidat di tingkat 3 → ambigu
		{"Jam ke-7", 0, false},
		{"", 0, false},
		{"   ", 0, false},
	}
	for _, k := range kasus {
		id, ok := Cocokkan(k.ket, suara)
		if id != k.id || ok != k.ok {
			t.Errorf("Cocokkan(%q) = (%d, %v), mau (%d, %v)", k.ket, id, ok, k.id, k.ok)
		}
	}
}

func TestCocokkan_AmbiguTingkatSama(t *testing.T) {
	_, ok := Cocokkan("Istirahat", []Kandidat{{1, "Istirahat"}, {2, "ISTIRAHAT"}})
	if ok {
		t.Fatal("dua kandidat tingkat 1 harus dianggap ambigu")
	}
}
```

Run: `go test ./internal/pengumuman/` → FAIL (simbol belum ada)

- [ ] **Step 2: Implementasi**

```go
// Package pengumuman mencocokkan keterangan bel dengan nama suara di pustaka
// untuk mengisi pengumuman secara otomatis. Murni, tanpa database.
package pengumuman

import "strings"

type Kandidat struct {
	ID   int64
	Nama string
}

var pemisah = strings.NewReplacer("-", " ", "_", " ", ".", " ", ",", " ", ":", " ", "(", " ", ")", " ", "/", " ")

// Normal: huruf kecil, tanda baca umum menjadi spasi, spasi dirapatkan.
func Normal(s string) string {
	return strings.Join(strings.Fields(pemisah.Replace(strings.ToLower(s))), " ")
}

// padanan menerjemahkan keterangan umum ke awal nama suara yang lazim.
var padanan = map[string]string{
	"pulang":        "akhir pelajaran",
	"upacara":       "upacara bendera",
	"masuk":         "bel masuk sekolah",
	"ujian selesai": "ujian waktu telah habis",
}

func tingkat(n, k string) int {
	switch {
	case n == k:
		return 1
	case n == k+" dimulai":
		return 2
	case strings.HasPrefix(n, k+" "):
		return 3
	}
	return 0
}

// Cocokkan mengembalikan suara yang paling cocok dengan keterangan.
// Tidak menebak: bila tingkat terbaik punya lebih dari satu kandidat, hasilnya (0, false).
func Cocokkan(keterangan string, suara []Kandidat) (int64, bool) {
	k := Normal(keterangan)
	if k == "" {
		return 0, false
	}
	if p, ok := padanan[k]; ok {
		k = p
	}
	terbaik, jumlah := 0, 0
	var id int64
	for _, s := range suara {
		t := tingkat(Normal(s.Nama), k)
		switch {
		case t == 0:
		case terbaik == 0 || t < terbaik:
			terbaik, jumlah, id = t, 1, s.ID
		case t == terbaik:
			jumlah++
		}
	}
	if jumlah != 1 {
		return 0, false
	}
	return id, true
}
```

- [ ] **Step 3:** `go vet ./internal/pengumuman/ && go test -race ./internal/pengumuman/` → PASS
- [ ] **Step 4:** Commit `feat(pengumuman): pencocokan keterangan bel dengan nama suara`

---

### Task 2: Kolom pengumuman — skema, migrasi, CRUD jadwal, salin/duplikat, hapus suara, penjadwal

**Files:**
- Modify: `internal/store/schema.sql`, `internal/store/store.go`, `internal/store/jadwal.go`, `internal/store/audio.go`, `internal/store/cadangan.go` (hanya SQL di `DuplikatProfil`)
- Modify: `internal/schedule/schedule.go`, `internal/scheduler/scheduler.go`
- Test: `internal/store/pengumuman_test.go` (baru), tes scheduler yang ada (tambah kasus)

**Interfaces:**
- Produces:
  - `schedule.Jadwal.PengumumanID int64`, `schedule.Bel.PengumumanID int64` (0 = tidak ada)
  - `store.ListJadwal`, `GetJadwal`, `CreateJadwal`, `UpdateJadwal`, dan `LoadDataJadwal` membawa kolom ini
  - Interface `scheduler.Sumber` mendapat `GetAudio(ctx context.Context, id int64) (store.Audio, error)`. Method ini sudah ada di `*store.Store`, jadi semua implementasi palsu di tes scheduler harus menambahkannya.

**Langkah:**

- [ ] **Step 1: Skema.** Di `schema.sql`, tabel `jadwal` tambahkan baris `pengumuman_audio_id INTEGER REFERENCES audio(id) ON DELETE RESTRICT,` sebelum `UNIQUE (...)`.
- [ ] **Step 2: Migrasi di `Open`.** Setelah `db.Exec(skema)`, panggil `migrasi(db)`:

```go
// migrasi menambah kolom yang belum ada pada database lama. Idempoten.
func migrasi(db *sql.DB) error {
	ada, err := adaKolom(db, "jadwal", "pengumuman_audio_id")
	if err != nil || ada {
		return err
	}
	_, err = db.Exec(`ALTER TABLE jadwal ADD COLUMN pengumuman_audio_id INTEGER REFERENCES audio(id) ON DELETE RESTRICT`)
	return err
}

func adaKolom(db *sql.DB, tabel, kolom string) (bool, error) {
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, tabel)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return false, err
		}
		if n == kolom {
			return true, nil
		}
	}
	return false, rows.Err()
}
```

  Bila migrasi gagal, tutup db dan kembalikan `fmt.Errorf("migrasi skema: %w", err)`, seperti penanganan skema.

- [ ] **Step 3: Store jadwal** (`jadwal.go`):
  - `kolomJadwal` menjadi `id, profil_id, hari, jam, jenis_bel_id, label, pengumuman_audio_id`.
  - `scanJadwal` memindai kolom itu ke `sql.NullInt64` lalu menyalinnya ke `j.PengumumanID`.
  - `CreateJadwal` dan `UpdateJadwal` menulis kolom itu lewat helper `nullID(id int64) any`, yang mengembalikan `nil` bila id 0.
  - `semuaJadwal` (sekitar baris 378) ikut membaca kolom itu.
  - **Validasi:** sebelum menulis, bila `PengumumanID != 0` dan `s.GetAudio` mengembalikan `ErrTidakAda`, kembalikan `validasi("suara pengumuman tidak ditemukan")`. Pembacaan ini terjadi di luar transaksi, jadi aman.
  - `SalinHari` menyalin `pengumuman_audio_id` pada `INSERT ... SELECT`.
- [ ] **Step 4: `DuplikatProfil`** (`cadangan.go` baris 63-65): sertakan `pengumuman_audio_id` di `INSERT ... SELECT`. Fungsi lain di berkas ini **jangan disentuh**, karena Tugas 3 yang mengubahnya.
- [ ] **Step 5: Hapus suara** (`audio.go` `DeleteAudio`). Sebelum `DELETE`, hitung `SELECT COUNT(*) FROM jadwal WHERE pengumuman_audio_id = ?`. Bila lebih dari 0, kembalikan `dipakai(fmt.Sprintf("suara masih dipakai oleh %d bel di jadwal", n))`. Pesan FK yang sudah ada untuk jenis bel tetap dipakai.
- [ ] **Step 6: `schedule`.** Tambah field `PengumumanID int64` di `Jadwal` (setelah `Label`) dan di `Bel`. `Bel` diisi dari `j.PengumumanID` (sekitar baris 115).
- [ ] **Step 7: Scheduler.**
  - `putar` menerima `pengumumanID int64` tambahan.
  - Bel jadwal (`bunyikan`) mengirim `b.PengumumanID`. Bel manual mengirim 0.
  - Berkas yang diputar = `s.FileUrutan(ctx, jenis)` lalu, bila id ≠ 0, berkas suara dari `s.sumber.GetAudio(ctx, id)` dengan path `filepath.Join(s.audioDir, filepath.Base(a.File))`.
  - Bila `GetAudio` gagal: log error dan putar tanpa pengumuman.
- [ ] **Step 8: Tes** di `internal/store/pengumuman_test.go`:
  - **migrasi:**
    - buat db dengan tabel `jadwal` versi lama (tanpa kolom) lewat `sql.Open` + DDL lama
    - `Open` dua kali → kolom ada dan tidak error
    - (salin DDL lama dari `git show HEAD:internal/store/schema.sql` bagian `jadwal`, beserta tabel yang dirujuk, atau buat db lewat `Open`, lalu `ALTER TABLE jadwal DROP COLUMN pengumuman_audio_id`, lalu buka ulang)
  - **CRUD:** Create/Get/List/Update membawa `PengumumanID`; id 0 → NULL; id tak ada → `harapValidasi`.
  - **SalinHari dan DuplikatProfil** membawa kolom.
  - **DeleteAudio:** suara yang dipakai pengumuman → `errors.Is(err, ErrDipakai)` dan pesannya memuat "1 bel".
  - **LoadDataJadwal** membawa kolom.
  - **Tes scheduler:** satu kasus memastikan berkas putar = urutan jenis lalu pengumuman, dan kasus `GetAudio` error → hanya urutan jenis.
- [ ] **Step 9:** `go vet ./... && go test -race ./...` → PASS. Commit `feat(store): kolom pengumuman per bel, migrasi, dan diputar setelah suara jenis bel`.

---

### Task 3: Store — pengumuman di ekspor/impor/gabung, saran & terapkan

**Files:**
- Modify: `internal/formatjadwal/formatjadwal.go` (+ tes)
- Modify: `internal/store/cadangan.go` (`EksporProfil`, `ImporJadwal`)
- Modify: `internal/store/gabung.go`, `internal/store/jenisimpor.go` (bila perlu helper)
- Create: `internal/store/saranpengumuman.go`
- Test: `internal/store/saranpengumuman_test.go`, tambah kasus di `cadangan_test.go` dan `gabung_test.go`

**Interfaces:**
- Consumes: Tugas 1 (`pengumuman.Cocokkan`, `Kandidat`), Tugas 2 (kolom, `PengumumanID`).
- Produces (dipakai Tugas 4):
  - `formatjadwal.Bel.Pengumuman string \`json:"pengumuman,omitempty"\``. `Validasi`: maksimal 100 rune setelah trim, dengan pesan `"bel ke-%d: pengumuman maksimal 100 karakter"`.
  - `func (s *Store) CocokkanPengumuman(ctx context.Context, keterangan string) (*store.SuaraRingkas, error)`. `type SuaraRingkas struct{ ID int64 \`json:"id"\`; Nama string \`json:"nama"\` }`. Bila tidak ada kecocokan, kembalikan `nil, nil`.
  - `type SaranPengumuman struct{ JadwalID int64 \`json:"jadwalId"\`; Hari int \`json:"hari"\`; Jam string \`json:"jam"\`; Label string \`json:"label"\`; Suara SuaraRingkas \`json:"suara"\` }`
  - `func (s *Store) SaranPengumuman(ctx context.Context, profilID int64) ([]SaranPengumuman, error)`. Hanya baris dengan pengumuman NULL dan yang punya kecocokan, urut hari lalu jam. Profil tidak ada → `ErrTidakAda`. Slice kosong non-nil.
  - `type IsiPengumuman struct{ JadwalID int64 \`json:"jadwalId"\`; AudioID int64 \`json:"audioId"\` }`
  - `func (s *Store) TerapkanPengumuman(ctx context.Context, profilID int64, isi []IsiPengumuman) (int, error)`
    - `UPDATE jadwal SET pengumuman_audio_id = ? WHERE id = ? AND profil_id = ? AND pengumuman_audio_id IS NULL`, semuanya dalam satu transaksi. Hasilnya jumlah baris yang berubah.
    - Audio yang tidak ada → `validasi("suara pengumuman tidak ditemukan")`. Periksa semua audio id sebelum transaksi.
    - Profil tidak ada → `ErrTidakAda`.
  - `HasilPeriksaGabung` mendapat `PengumumanTakDitemukan []string \`json:"pengumumanTakDitemukan"\``.
  - `func (s *Store) PengumumanTakDikenal(ctx context.Context, d formatjadwal.Data) ([]string, error)` dipakai periksa impor. Hasilnya nama unik berurutan pertama muncul, slice non-nil.

**Aturan resolusi (helper bersama di `saranpengumuman.go`):**

```go
// petaSuara memotret pustaka suara untuk mengisi pengumuman.
type petaSuara struct {
	kandidat []pengumuman.Kandidat
	idNama   map[string]int64 // kunciNama(nama) -> id; nama ganda: id terkecil
}
func (s *Store) bacaPetaSuara(ctx context.Context) (petaSuara, error) // dari ListAudio
// resolusi: nama dari file bila ada (tak ditemukan → 0, false=ditemukan), selain itu Cocokkan(label).
func (p petaSuara) resolusi(namaFile, label string) (id int64, takDitemukan bool)
```

- `resolusi`:
  - Bila `TrimSpace(namaFile) != ""`, cari di `idNama` dengan `kunciNama`. Bila ada, kembalikan `(id, false)`. Bila tidak ada, kembalikan `(0, true)`.
  - Selain itu, `Cocokkan(label)` → `(id atau 0, false)`.
- **`ImporJadwal`:** baca `petaSuara` **sebelum** transaksi. `INSERT` jadwal menyertakan `pengumuman_audio_id` (`nullID`) hasil `resolusi(b.Pengumuman, b.Label)`.
- **`GabungJadwal`:** baca `petaSuara` sebelum transaksi.
  - `Tambah`: `INSERT` dengan pengumuman hasil resolusi untuk bel masukan `d.Jadwal[b.Indeks]`.
  - `Timpa`: `UPDATE` juga menyetel `pengumuman_audio_id`.
  - Kembar tidak disentuh.
- **`PeriksaGabung`:** isi `PengumumanTakDitemukan`. Nama unik, urut kemunculan, slice non-nil.
- **`EksporProfil`:** isi `Bel.Pengumuman` dengan nama suara dari `ListAudio` (map id → nama). Lakukan sebelum `Validasi` yang sudah ada.

- [ ] **Step 1: Tes gagal**
  - `formatjadwal`:
    - field `pengumuman` terbaca dan ditulis `omitempty`
    - lebih dari 100 karakter → galat
  - Store:
    - ekspor lalu impor membawa nama pengumuman (round trip)
    - impor dengan nama yang tidak ada → kolom NULL, dan `PengumumanTakDikenal` melaporkan nama itu
    - impor tanpa pengumuman dengan label "Jam ke-1" ketika pustaka punya suara "Jam ke-1 dimulai" → terisi otomatis
    - gabung: baris baru terisi otomatis; timpa mengganti pengumuman; kembar tidak berubah; `PengumumanTakDitemukan` terisi
    - `SaranPengumuman`: melewati baris yang sudah terisi dan baris tanpa kecocokan; urutan benar; profil tidak ada → `ErrTidakAda`
    - `TerapkanPengumuman`: tidak menimpa baris yang sudah terisi, mengabaikan baris dari profil lain, audio tidak ada → validasi, dan mengembalikan jumlah
    - `CocokkanPengumuman`: kasus cocok dan tidak cocok
  - Buat suara uji dengan `s.CreateAudio(ctx, Audio{Nama: "Jam ke-1 dimulai", Sumber: "upload", File: "x.mp3", Dibuat: time.Now()})`.
- [ ] **Step 2: Implementasi** sesuai Interfaces dan aturan resolusi di atas.
- [ ] **Step 3:** `go vet ./... && go test -race ./...` → PASS. Commit `feat(store): pengumuman di ekspor/impor/gabung, saran dan terapkan otomatis`.

---

### Task 4: API — `pengumumanId`, cocok, saran, terapkan, status, periksa

**Files:**
- Modify: `internal/api/dto.go`, `internal/api/jadwal.go`, `internal/api/cadangan.go` (DTO periksa impor), `internal/api/status.go` (atau berkas yang membentuk `belHariIniDTO`), `internal/api/rute_data.go`
- Create: `internal/api/pengumuman.go`, `internal/api/pengumuman_test.go`

**Interfaces:**
- Consumes: Tugas 2 dan 3.
- Produces (kontrak JSON spec §8, dipakai Tugas 5 dan 6):
  - `jadwalDTO.PengumumanID *int64 \`json:"pengumumanId"\``. `null` ↔ 0. `keJadwalDTO` dan `keJadwal` dikonversi dua arah.
  - `GET /api/pengumuman/cocok?keterangan=...` → `{"suara": SuaraRingkas|null}`
  - `GET /api/profil/{id}/pengumuman/saran` → `[]SaranPengumuman` (bila kosong `[]`)
  - `POST /api/profil/{id}/pengumuman/terapkan` dengan body `{"isi":[{"jadwalId","audioId"}]}` → `{"diisi": N}`. Lalu `s.ubahData()`.
  - `belHariIniDTO.Pengumuman *store.SuaraRingkas \`json:"pengumuman"\``. Isi dari `BelHariIni`. Pada status, nama suara diambil lewat `GetAudio`; bila gagal, nilainya `null`. Bila status dibangun dari `schedule.Bel`, `PengumumanID` sudah tersedia (Tugas 2). Tambahkan field ke struktur perantara bila perlu.
  - `periksaDTO` (impor) mendapat `PengumumanTakDitemukan []string \`json:"pengumumanTakDitemukan"\`` dari `Store.PengumumanTakDikenal`.
  - Periksa gabung sudah membawa field ini lewat `HasilPeriksaGabung` (Tugas 3).

- [ ] **Step 1: Tes gagal** di `pengumuman_test.go`:
  - buat dan ubah jadwal dengan `pengumumanId`, lalu baca kembali; `null` tetap `null`
  - `pengumumanId` yang tidak ada → 400
  - cocok: ada dan tidak ada
  - saran → terapkan → saran kosong, dan penjadwal dimuat ulang (`l.penjadwal.jumlahReload()`)
  - saran untuk profil 9999 → 404
  - tanpa login → 401 (klien `l.klien()` tanpa masuk; lihat `gabung_test.go`)
  - status memuat `pengumuman` untuk bel hari ini
  - hapus suara yang dipakai → 409 dengan pesan "bel di jadwal"
  - Suara uji dibuat lewat `l.st.CreateAudio`. Status berbasis hari ini: buat bel di hari `schedule.Hari(time.Now())` pada profil default (pola di tes status yang ada).
- [ ] **Step 2: Implementasi.** Rute di `rute_data.go`:

```go
	rute("GET /api/pengumuman/cocok", s.cocokPengumuman)
	rute("GET /api/profil/{id}/pengumuman/saran", s.saranPengumuman)
	rute("POST /api/profil/{id}/pengumuman/terapkan", s.terapkanPengumuman)
```

- [ ] **Step 3:** `go vet ./... && go test -race ./...` → PASS. Commit `feat(api): pengumuman per bel — cocok, saran, terapkan, status`.

---

### Task 5: Frontend — tipe API, `PilihSuara`, `IsiPengumuman`

**Files:**
- Modify: `web/src/lib/api.ts`
- Create: `web/src/components/PilihSuara.svelte`, `web/src/components/IsiPengumuman.svelte`

**Interfaces:**
- Produces (dipakai Tugas 6):
  - `api.ts`:
    - `Jadwal` mendapat `pengumumanId: number | null`
    - `export interface SuaraRingkas { id: number; nama: string }`
    - `BelHariIni` mendapat `pengumuman: SuaraRingkas | null`
    - `HasilPeriksa` (impor) dan `HasilPeriksaGabung` mendapat `pengumumanTakDitemukan: string[]`
    - `export interface SaranPengumuman { jadwalId: number; hari: number; jam: string; label: string; suara: SuaraRingkas }`
  - `PilihSuara.svelte` props `{ audio: Audio[]; nilai: number | null ($bindable); id: string; label: string; onpilih?: () => void }`:
    - `<label for={id}>` dan input cari (`id`) yang menyaring nama (Normal sederhana: huruf kecil + includes).
    - Daftar tombol `role="listbox"` / `role="option"` (`aria-selected`), tinggi maksimal sekitar 16rem dan bisa digulir. Opsi pertama "Tanpa pengumuman" (null).
    - Nama terpilih ditampilkan di atas daftar, disertai tombol ▶ `aria-label="Coba {nama}"` yang memanggil `POST /audio/{id}/tes`.
    - Memilih opsi mengatur `nilai` dan memanggil `onpilih?.()`. Komponen induk memakai `onpilih` untuk menandai pilihan manual.
  - `IsiPengumuman.svelte` props `{ buka: boolean ($bindable); profil: Profil; onselesai: () => void }`:
    - `PanelGeser` dengan judul "Isi pengumuman otomatis".
    - Saat dibuka, memuat `GET /profil/{id}/pengumuman/saran` (pola nomor-urut, `role="status"` "Memuat saran…").
    - Daftar per baris berisi checkbox (tercentang sebagai bawaan) dengan label `"{NAMA_HARI[hari]} {jam} · {label || '—'} → {suara.nama}"`, dikelompokkan per hari.
    - Ada tautan atau tombol "Pilih semua / Kosongkan".
    - Bila kosong: "Semua bel sudah punya pengumuman atau tidak ada suara yang cocok".
    - Aksi: Batal dan **"Terapkan (N)"** (nonaktif bila N = 0). Tombol ini memanggil `POST /profil/{id}/pengumuman/terapkan` dengan baris yang dicentang, lalu `beriTahu('{diisi} bel diberi pengumuman')`, `buka = false`, dan `onselesai()`. Galat ditampilkan dengan `role="alert"`.
- Bila backend belum di-merge di worktree ini, komponen tetap dikompilasi karena hanya bergantung pada tipe.

- [ ] **Step 1:** Implementasi.
- [ ] **Step 2:** `cd web && npm run check && npx vitest run` → 0 error/0 warning.
- [ ] **Step 3:** Commit `feat(web): komponen PilihSuara dan IsiPengumuman`.

---

### Task 6: Frontend — pasang di Jadwal, Minggu, Beranda, impor & Tambah banyak

**Files:**
- Modify: `web/src/pages/Jadwal.svelte`, `web/src/components/JadwalMinggu.svelte`, `web/src/pages/Beranda.svelte`
- Modify: `web/src/components/AlurImpor.svelte`, `web/src/components/TambahBanyak.svelte` (atau `PratinjauGabung.svelte`) untuk peringatan

**Interfaces:** Consumes Tugas 4 (API) dan Tugas 5 (komponen dan tipe).

**Perilaku (spec §6, §9):**
- **Jadwal.svelte — form tambah/ubah** (panel geser yang sudah ada):
  - Tambah state `fPengumuman: number | null` dan `pengumumanManual: boolean`.
  - Saat membuka "+ Tambah bel": `fPengumuman = null`, `pengumumanManual = false`.
  - Saat membuka ubah: `fPengumuman = j.pengumumanId`, `pengumumanManual = j.pengumumanId !== null`.
  - Medan `<PilihSuara audio={audio} bind:nilai={fPengumuman} id="f-pengumuman" label="Pengumuman" onpilih={() => (pengumumanManual = true)} />`.
  - `$effect` pada `fLabel`: bila `!pengumumanManual`, debounce 300 ms lalu panggil `GET /pengumuman/cocok?keterangan=${encodeURIComponent(fLabel)}`. Respons yang usang diabaikan. Hasilnya: `fPengumuman = suara?.id ?? null`, dan `otomatis = !!suara`.
  - Bila nilai berasal dari pengisian otomatis, tampilkan teks kecil "Diisi otomatis dari keterangan".
  - Simpan mengirim `pengumumanId: fPengumuman`.
- **Tampilan Hari:** kolom "Pengumuman" berisi nama suara (`audio.find`) atau "—". Di HP, isinya menjadi baris tambahan di kartu.
- **Bar alat:** tombol `<Tombol varian="kedua">Isi pengumuman otomatis</Tombol>` (nonaktif bila tidak ada profil) membuka `<IsiPengumuman bind:buka profil={profilAktif} onselesai={() => muatJadwal()} />`.
- **JadwalMinggu:** props bel mendapat `pengumuman?: string` (nama). Bila ada, tampilkan ikon speaker kecil (`Volume2` dari `@lucide/svelte`, `aria-label="dengan pengumuman {nama}"`) di kartu. Jadwal.svelte mengisi `pengumuman` di `belMinggu`. Pratinjau Template dan Tambah banyak tidak mengirim `pengumuman`, jadi tidak berubah.
- **Beranda:** bila `berikutnya.pengumuman` ada, tampilkan "+ {nama}" di bawah nama bel berikutnya.
- **AlurImpor dan TambahBanyak (pratinjau):** bila `pengumumanTakDitemukan.length > 0`, tampilkan kotak kuning `rounded-xl bg-amber-50 text-amber-800`: "Pengumuman tidak ditemukan di pustaka: {nama.join(', ')} — dikosongkan."

- [ ] **Step 1:** Implementasi.
- [ ] **Step 2:** `cd web && npm run check && npx vitest run`, lalu `PORT=18095 npx playwright test e2e/jadwal.spec.ts e2e/cadangan.spec.ts e2e/tambah-banyak.spec.ts e2e/beranda.spec.ts` → lulus. Kembalikan `internal/webui/dist` sebelum commit.
- [ ] **Step 3:** Commit `feat(web): pengumuman per bel di Jadwal, Minggu, Beranda, impor dan tambah banyak`.

---

### Task 7: E2E, build produksi, README

**Files:** Create `web/e2e/pengumuman.spec.ts`; Modify `internal/webui/dist/**`, `README.md`

- [ ] **Step 1: Tes Playwright** (proyek `hp`; setiap tes membuat profil uji sendiri dan menghapusnya di `finally`):
  - **Suara uji:**
    - buat berkas WAV hening 0,3 detik di tes (`Buffer` header RIFF 44 byte + data nol)
    - unggah lewat `request.post('/api/audio', { multipart: { nama: unik('Jam ke-3 dimulai'), file: { name: 'x.wav', mimeType: 'audio/wav', buffer } } })`
    - **perhatian:** pencocokan memerlukan nama yang diawali keterangan, jadi beri nama unik dengan akhiran, misalnya `Jam ke-3 dimulai` + sufiks; dan pakai keterangan yang unik pula, misalnya `Jam ke-3 {sufiks}` dengan suara `Jam ke-3 {sufiks} dimulai`, agar tidak bentrok dengan suara lain di data uji
    - hapus suara di `finally` setelah profil dihapus
  1. Tambah bel dengan keterangan `K` → medan Pengumuman otomatis menunjukkan suara `K dimulai` dan teks "Diisi otomatis dari keterangan" → simpan → kolom atau kartu Pengumuman menampilkan nama itu → lewat API `pengumumanId` sesuai.
  2. Pilih manual "Tanpa pengumuman" lalu ubah keterangan → tidak terisi ulang.
  3. Profil berisi 2 bel tanpa pengumuman (dibuat lewat API, keterangan cocok dengan suara uji) → "Isi pengumuman otomatis" → 2 saran → hapus satu centang → "Terapkan (1)" → notifikasi "1 bel diberi pengumuman" → lewat API hanya satu yang terisi.
  4. Hapus suara yang dipakai (Pustaka Suara atau API `DELETE /api/audio/{id}`) → 409 dan pesan memuat "bel di jadwal".
- [ ] **Step 2:** `cd web && PORT=18096 npx playwright test` (seluruh suite).
- [ ] **Step 3:** README: satu kalimat di Ringkasan tentang pengumuman per bel yang diisi otomatis dari keterangan.
- [ ] **Step 4 (terakhir):** `cd web && env -u VITE_E2E npm run build && ! grep -rq __bel ../internal/webui/dist && cd .. && go test ./...`, lalu `BEL_PORT=18081 ./scripts/e2e.sh` (exit 0), lalu periksa `__bel` lagi.
- [ ] **Step 5:** Commit `test(e2e): pengumuman per bel; build produksi dan README`.
