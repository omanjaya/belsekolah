# Rencana 7: Frontend Layar Penuh, Zona Waktu, Cadangan & Template

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merombak antarmuka menjadi panel admin layar penuh (sidebar gelap, bar atas berjam, tabel lebar, panel geser) dan menambah UI zona waktu, ekspor/impor/duplikat jadwal, serta halaman Template.

**Architecture:** Svelte 5 (runes) + Tailwind 4, SPA dengan router sendiri (`lib/router.svelte.ts`, `rute.ts`). Komponen bersama baru (`PanelGeser`, `Tabel`, `BarAlat`, `KartuStatistik`, `JadwalMinggu`, `AlurImpor`) dipakai semua halaman. Tugas halaman ditulis sebagai **spesifikasi perilaku + tes Playwright**, bukan markup kata per kata (keputusan yang sama dengan Rencana 4).

**Tech Stack:** Svelte 5, Vite, Tailwind CSS 4, `@lucide/svelte`, Vitest, Playwright.

**Spec:** `docs/superpowers/specs/2026-10-15-tampilan-zona-template-design.md` (§3–§6), API dari Rencana 6.

**Prasyarat:** Rencana 6 selesai (semua endpoint baru tersedia).

## Global Constraints

- Bahasa Indonesia untuk semua teks UI; gaya kode mengikuti file yang ada (runes `$state/$derived/$effect/$props`, nama Indonesia).
- **Tidak ada pembatas lebar isi** (`max-w-*` pada `<main>` atau pembungkus halaman dilarang). Pengecualian: halaman Login dan Ganti Password Awal, serta lebar panel geser/dialog.
- Breakpoint: HP `< md` (768px), laptop `≥ md`. Tidak boleh ada gulir horizontal halaman di lebar 375px; tabel lebar boleh bergulir di dalam kotaknya sendiri (`overflow-x-auto`).
- Warna: latar isi `bg-slate-100`, kartu `bg-white border-slate-200 rounded-xl`, sidebar `bg-slate-900`, aksen `blue-700`. Tanpa mode gelap.
- Form tambah/edit memakai `PanelGeser`; dialog tengah hanya untuk konfirmasi (`lib/konfirmasi.svelte.ts`) dan Login/Ganti Password Awal.
- `localStorage` selalu dibungkus `try/catch`; aplikasi harus berfungsi bila tidak tersedia.
- Aksesibilitas: target sentuh ≥ 44px; panel geser `role="dialog"` + `aria-modal="true"` + `aria-labelledby`, fokus terkunci, `Esc` menutup, fokus kembali ke pemicu.
- Semua perilaku yang sudah ada di v0.1.0 **dipertahankan** (baca halaman lama dan tes Playwright lama sebelum merombak); tes lama diperbarui, bukan dihapus, kecuali perilakunya memang diganti spec.
- Setiap tugas: `cd web && npm run check && npx vitest run` hijau, dan tes Playwright yang disentuh lulus secara lokal (`cd web && npx playwright test <file>` — memakai `go run` di port 18080; port 8080 terpakai proyek lain).
- Jangan commit `internal/webui/dist` hasil build `VITE_E2E`. Build produksi (`env -u VITE_E2E npm run build`) hanya di Task 8.
- Commit diakhiri baris kosong + `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Kontrak API yang dipakai (dari Rencana 6)

```ts
// tambahan/ubahan di web/src/lib/api.ts
export interface Profil { id: number; nama: string; jumlahBel?: number }
export interface Status { /* field lama */ zonaWaktu: string }
export interface Pengaturan { volume: number; profilDefaultId: number; zonaWaktu: 'WIB' | 'WITA' | 'WIT' | string; mulaiUlang?: boolean }
export interface Sistem { /* field lama; zonaWaktu kini 'WIB' dll */ zonaIana: string }
export interface FileJadwal {
  format: 'belsekolah-jadwal'; versi: 1; nama: string
  jenisBel: { nama: string; warna: string }[]
  jadwal: { hari: number; jam: string; jenisBel: string; label: string }[]
}
export interface HasilPeriksa { nama: string; jumlahBel: number; jenisBelTakDikenal: string[]; namaTersedia: string }
export type Pemetaan = Record<string, { jenisBelId: number } | { baru: true }>
export interface Template { id: number; nama: string; jumlahBel: number; hari: number[]; dibuat: string; data?: FileJadwal }
export interface ParameterPembuat {
  jenjang: 'SD' | 'SMP' | 'SMA'; varian: 'normal' | 'ramadan' | 'ujian'; hariSekolah: 5 | 6
  jamMasuk: string; lamaJP: number; jumlahJP: number; jumlahJPJumat: number; jumlahJPSabtu: number
  istirahat: { setelahJP: number; menit: number }[]
  upacaraSenin: { aktif: boolean; menit: number }
  kegiatanJumat: { aktif: boolean; menit: number; label: string }
  ujian: { jamMulai: string; jumlahSesi: number; lamaSesi: number; jeda: number }
}
```
Endpoint: `GET /profil` (dengan `jumlahBel`), `POST /profil/{id}/duplikat {nama}`, `GET /profil/{id}/ekspor` (file), `POST /impor/periksa` (body = FileJadwal), `POST /impor {data, nama, pemetaan}` → `{profilId}`, `GET/POST /template`, `GET/PUT/DELETE /template/{id}`, `GET /template/{id}/ekspor`, `POST /template/buat` (body = ParameterPembuat) → FileJadwal, `PUT /pengaturan` dengan `zonaWaktu` → `{..., mulaiUlang: true}` bila zona berubah.

## Struktur File

| File | Tanggung jawab |
|---|---|
| `web/src/lib/api.ts` | Tipe baru + `unduh(path)` |
| `web/src/lib/jam.ts` (+`jam.test.ts`) | Jam server: selisih & offset dari `sekarang`, format `HH:MM:SS` |
| `web/src/lib/jamServer.svelte.ts` | State jam server bersama (diisi dari `/api/status`, berdetik tiap detik) |
| `web/src/lib/mulaiUlang.ts` | `bacaBoot()`, `tungguMulaiUlang(bootLama, {batal})` — dipindah dari Pengaturan |
| `web/src/lib/simpan.ts` | `baca(kunci, bawaan)`/`tulis(kunci, nilai)` localStorage aman |
| `web/src/components/Layout.svelte` | Bingkai: sidebar, bar atas, laci HP, bar bawah |
| `web/src/components/PanelGeser.svelte` | Panel form dari kanan |
| `web/src/components/Tabel.svelte` | Kotak tabel (header lengket, kosong, memuat, gulir horizontal internal) |
| `web/src/components/BarAlat.svelte` | Baris filter/aksi di atas tabel |
| `web/src/components/KartuStatistik.svelte` | Kartu angka ringkas |
| `web/src/components/JadwalMinggu.svelte` | Tampilan mingguan (Jadwal & pratinjau Template) |
| `web/src/components/AlurImpor.svelte` | Panel periksa → pemetaan → impor |
| `web/src/pages/*.svelte` | Halaman dirombak; `Template.svelte` baru |
| `web/e2e/*.spec.ts` | Tes diperbarui + `tata-letak.spec.ts`, `cadangan.spec.ts`, `template.spec.ts`, `zona.spec.ts` baru |

---

### Task 1: Fondasi — lib, komponen bersama, dan bingkai aplikasi

**Files:**
- Create: `web/src/lib/jam.ts`, `web/src/lib/jam.test.ts`, `web/src/lib/jamServer.svelte.ts`, `web/src/lib/mulaiUlang.ts`, `web/src/lib/simpan.ts`, `web/src/components/PanelGeser.svelte`, `web/src/components/Tabel.svelte`, `web/src/components/BarAlat.svelte`, `web/src/components/KartuStatistik.svelte`, `web/e2e/tata-letak.spec.ts`
- Modify: `web/src/lib/api.ts`, `web/src/components/Layout.svelte`, `web/src/components/JudulHalaman.svelte`, `web/src/App.svelte` (bila perlu), `web/src/pages/Pengaturan.svelte` (pakai `lib/mulaiUlang.ts`), `web/e2e/navigasi.spec.ts`

**Interfaces:**
- Produces:
  - `jam.ts`: `offsetMenit(iso: string): number` (offset zona dari string RFC3339, mis. `+08:00` → 480, `Z` → 0); `selisihMs(iso: string, browserMs: number): number` (= `Date.parse(iso) - browserMs`); `formatJamServer(browserMs: number, selisih: number, offset: number): string` → `HH:MM:SS` waktu lokal server.
  - `jamServer.svelte.ts`: `export const jamServer = $state({ teks: '', zona: '', siap: false })`; `perbaruiDariStatus(s: Status): void`; `mulaiDetak(): () => void` (setInterval 1 dtk, mengembalikan penghenti). Layout memuat `/api/status` saat mount dan tiap 30 detik bila belum ada halaman lain yang memperbarui; Beranda boleh memanggil `perbaruiDariStatus` setiap kali memuat status.
  - `mulaiUlang.ts`: `bacaBoot(): Promise<string|null>`, `tungguMulaiUlang(bootLama: string|null, opsi: { batal: () => boolean; batasMs?: number }): Promise<boolean>` (true = boot berubah lalu `location.reload()` dipanggil; false = habis waktu/dibatalkan). Logika sama persis dengan versi lama di `Pengaturan.svelte` (tunggu 2 dtk, lalu cek tiap 2 dtk, batas bawaan 120 dtk).
  - `simpan.ts`: `baca<T>(kunci: string, bawaan: T): T`, `tulis(kunci: string, nilai: unknown): void` (JSON, try/catch).
  - `api.ts`: tipe pada bagian "Kontrak API"; `unduh(path: string): Promise<void>` — `fetch('/api'+path)`, error → `ApiError` seperti `minta`; nama file dari `Content-Disposition` (`filename="..."`), fallback `unduhan.json`; buat object URL lalu klik `<a download>`.
  - `PanelGeser.svelte` props: `buka` (bindable), `judul: string`, `children: Snippet`, `aksi?: Snippet` (tombol di kaki panel), `lebar?: 'md' | 'lg'` (440px / 640px). Di HP layar penuh.
  - `Tabel.svelte` props: `memuat?: boolean`, `kosong?: string` (pesan bila tidak ada baris; tampil bila `jumlah === 0`), `jumlah: number`, `kepala: Snippet` (isi `<tr>` header), `children: Snippet` (baris `<tr>`), `label: string` (aria-label tabel). Bungkus `overflow-x-auto`, `<thead>` lengket.
  - `BarAlat.svelte` props: `kiri?: Snippet`, `kanan?: Snippet` — flex wrap, `kanan` rata kanan di laptop.
  - `KartuStatistik.svelte` props: `label: string`, `nilai: string`, `keterangan?: string`, `nada?: 'biasa'|'baik'|'peringatan'|'bahaya'`.

**Perilaku bingkai (`Layout.svelte`):**
1. Laptop (`≥ md`): sidebar kiri `bg-slate-900` lebar 240px, menempel penuh tinggi layar (`sticky top-0 h-dvh`). Atas: logo lonceng + "Bel Sekolah". Daftar menu dari `rute.ts` (urutan: Beranda, Jadwal, Profil & Periode, Template*, Libur, Jenis Bel, Pustaka Suara, Riwayat, Pengaturan — *Template ditambahkan di Task 5). Item aktif `bg-blue-700 text-white`, lainnya `text-slate-300 hover:bg-slate-800`. Bawah sidebar: tombol **Ciutkan menu** (ikon) yang mengubah lebar menjadi 64px dengan ikon saja (label jadi `title`/`aria-label`); status disimpan di `simpan.ts` kunci `sidebarCiut`.
2. Bar atas (semua ukuran): putih, garis bawah, `sticky top-0 z-30`, tinggi 56px. Kiri: tombol ☰ (hanya HP, `aria-label="Buka menu"`) lalu judul halaman (`<h1>` — pindahkan judul halaman ke sini; `JudulHalaman` menjadi cara halaman memberi judul ke bar atas atau tetap merender `<h1>` di bar atas lewat snippet/state; yang penting **tepat satu `<h1>` per halaman** berisi nama halaman). Kanan: pil status (`● Berjalan` hijau bila `/api/status` tanpa peringatan; kuning `● Perlu perhatian` bila ada peringatan; abu `● Memuat`), jam `HH:MM:SS` + singkatan zona (mis. `07:42:10 WITA`, `data-testid="jam-server"`), tombol **Keluar** (laptop: ikon+teks; HP: ikon). Keluar memanggil `POST /logout` lalu ke Login seperti perilaku lama di Pengaturan.
3. Isi: `<main>` `flex-1 min-w-0 p-4 md:p-6` **tanpa max-width**; HP diberi `pb-24` agar tidak tertutup bar bawah.
4. HP: tombol ☰ membuka **laci kiri** (sidebar yang sama, lebar 280px, latar gelap transparan; `Esc`/klik latar/klik tautan menutup; fokus ke tautan pertama saat buka dan kembali ke ☰ saat tutup). Bar bawah lama (Beranda, Jadwal, Suara, Riwayat, Lainnya + lembar Lainnya) **tetap** seperti sekarang.
5. Satu `<nav aria-label="Menu utama">` untuk sidebar laptop/laci, dan bar bawah HP tetap `aria-label="Menu utama"` (tes lama memakai `.first()`/`.last()`); jangan menambah nav ketiga dengan nama yang sama.

**Perilaku `PanelGeser`:** muncul dari kanan dengan transisi singkat (≤200ms, hormati `prefers-reduced-motion`), latar `bg-slate-900/40` menutupi isi; kepala berisi judul + tombol Tutup (`aria-label="Tutup"`); isi bergulir; `aksi` menempel di kaki. Saat terbuka: fokus ke isian pertama (seperti `Modal`), `Tab` berputar di dalam panel, `Esc` dan klik latar menutup, fokus kembali ke elemen yang membuka. Implementasi boleh memakai `<dialog>` + `showModal()` (gaya `Modal.svelte`) yang diposisikan di kanan — itu sudah memberi kunci fokus dan `Esc`.

- [ ] **Step 1: `jam.test.ts` (gagal)** — kasus: `offsetMenit('2026-10-15T07:42:10+08:00') === 480`, `offsetMenit('2026-10-15T07:42:10.123+07:00') === 420`, `offsetMenit('2026-10-15T00:00:00Z') === 0`, `offsetMenit('2026-10-15T07:00:00-03:30') === -210`; `formatJamServer` dengan `selisih` dan `offset` menghasilkan jam lokal server walau zona browser berbeda (uji dengan `browserMs = Date.parse('2026-10-14T23:42:10Z')`, selisih 0, offset 480 → `'07:42:10'`; selisih +5000 → `'07:42:15'`; offset 540 → `'08:42:10'`).
- [ ] **Step 2:** `npx vitest run` → gagal; implementasi `jam.ts` (pakai getter UTC setelah menambah offset; jangan bergantung pada zona browser) → lulus.
- [ ] **Step 3:** Implementasi `simpan.ts`, `mulaiUlang.ts` (pindahkan dari Pengaturan dan ganti pemakaian di Pengaturan; perilaku restore tidak berubah), tipe & `unduh` di `api.ts`, `jamServer.svelte.ts`.
- [ ] **Step 4:** Komponen `PanelGeser`, `Tabel`, `BarAlat`, `KartuStatistik`.
- [ ] **Step 5:** Rombak `Layout.svelte` sesuai perilaku bingkai.
- [ ] **Step 6: Tes Playwright** — `web/e2e/tata-letak.spec.ts`:
  - laptop 1440×900 di `/riwayat`: sidebar terlihat; lebar `main` ≥ 1440 − 240 − 20 px (ambil `boundingBox`); `aria-current="page"` pada Riwayat; tombol "Ciutkan menu" membuat sidebar ≤ 80px dan tetap ciut setelah `page.reload()`; kembalikan.
  - jam bar atas (`data-testid="jam-server"`) cocok `/^\d{2}:\d{2}:\d{2} (WIB|WITA|WIT)$/` dan berubah dalam 2,5 detik.
  - HP (proyek bawaan Pixel 7): tombol "Buka menu" membuka laci berisi tautan "Pengaturan"; `Escape` menutup; klik "Libur" di laci pindah ke `/libur` dan laci tertutup.
  - lebar 375×800 di setiap rute menu: `document.documentElement.scrollWidth <= 375`.
  - Perbarui `navigasi.spec.ts` bila selektornya berubah (judul kini `<h1>` di bar atas).
- [ ] **Step 7:** `npm run check && npx vitest run && npx playwright test e2e/tata-letak.spec.ts e2e/navigasi.spec.ts` → hijau.
- [ ] **Step 8: Commit** — `feat(web): bingkai panel admin layar penuh, jam server di bar atas, komponen bersama`

---

### Task 2: Beranda

**Files:** Modify `web/src/pages/Beranda.svelte`, `web/e2e/beranda.spec.ts`

**Perilaku:**
1. Pita peringatan (bila `status.peringatan` tidak kosong) di atas grid, kuning, satu baris per peringatan.
2. Grid `grid-cols-1 lg:grid-cols-3 gap-4 md:gap-6`:
   - **Bel berikutnya** (`lg:col-span-2`): jam besar (`text-4xl`), label, jenis bel (titik warna), hitung mundur (`hitungMundur`), "Profil: <nama>" atau info libur. Bila tidak ada bel lagi: "Tidak ada bel lagi hari ini".
   - **Bel manual**: tombol per jenis bel (bukan Darurat) dalam grid 2 kolom, **Bel Darurat** (tahan 2 detik, perilaku lama dipertahankan termasuk teks/aria), **Hentikan suara**.
   - **Jadwal hari ini** (`lg:col-span-2`): `Tabel` kolom Jam, Keterangan, Jenis bel (titik warna + nama), Status (Sudah/Berikutnya/Menunggu, plus status riwayat bila ada seperti sekarang). Baris berikutnya disorot `bg-blue-50`.
   - **Status sistem**: `KartuStatistik`/daftar — Pemutar (mpv ada/tidak), Sinkron jam (NTP ya/tidak/tidak diketahui), Ruang disk (MB), Backup terakhir — dari `/api/sistem`.
3. Setiap memuat `/api/status`, panggil `perbaruiDariStatus` agar jam bar atas sinkron.
4. Semua perilaku lama (muat ulang berkala, notifikasi, darurat, stop) tetap.

- [ ] **Step 1:** Rombak halaman.
- [ ] **Step 2:** Perbarui `beranda.spec.ts`: semua tes lama lulus; tambah — di 1440×900 kartu "Bel berikutnya" dan "Bel manual" berada di baris yang sama (y kotak sama ±8px) dan lebar gabungan grid ≥ 1100px; heading kartu "Status sistem" terlihat.
- [ ] **Step 3:** `npm run check && npx playwright test e2e/beranda.spec.ts` → hijau.
- [ ] **Step 4: Commit** — `feat(web): beranda dasbor layar penuh`

---

### Task 3: Jadwal — tampilan Minggu/Hari, panel edit, ekspor

**Files:** Create `web/src/components/JadwalMinggu.svelte`; Modify `web/src/pages/Jadwal.svelte`, `web/e2e/jadwal.spec.ts`

**Interfaces:**
- `JadwalMinggu.svelte` props: `bel: { kunci: string|number; hari: number; jam: string; label: string; jenisNama: string; warna: string }[]`, `hariSekolah?: number[]` (hari yang selalu ditampilkan, bawaan `[1,2,3,4,5]`; hari 6/7 ditampilkan bila ada belnya), `onpilih?: (kunci) => void` (bila ada, kartu menjadi tombol), `padat?: boolean` (pratinjau). Kolom per hari (`grid` dengan kolom sejumlah hari tampil; di HP menjadi daftar bertumpuk per hari). Tiap bel = kartu kecil dengan garis kiri `border-l-4` berwarna jenis bel, isi `HH:MM` tebal + label (atau nama jenis bila label kosong). Kepala kolom: nama hari + jumlah bel. Kolom tanpa bel: teks "Tidak ada bel".

**Perilaku halaman Jadwal:**
1. `BarAlat`: kiri — pilih profil (`<select>`, label "Profil"), pengalih **Minggu | Hari** (dua tombol `aria-pressed`); kanan — **Salin hari…**, **Ekspor**, **+ Tambah bel**. Pilihan tampilan disimpan (`simpan.ts`, kunci `tampilanJadwal`), bawaan `minggu`.
2. **Minggu**: `JadwalMinggu` untuk profil terpilih; klik kartu membuka panel edit bel itu.
3. **Hari**: dua kolom di laptop — daftar hari kiri (Senin…Minggu, nama + "n bel", hari aktif disorot; `role="tablist"`/`tab` atau daftar tombol `aria-pressed`) dan `Tabel` kanan (Jam, Keterangan, Jenis bel, Suara = nama urutan suara jenis bel, Aksi Ubah/Hapus). Di HP daftar hari menjadi baris tombol yang bisa digulir horizontal di dalam kotaknya.
4. **Tambah/Ubah bel** di `PanelGeser` (isian lama: hari, jam, jenis bel, label; validasi & pesan error lama). Hapus memakai `konfirmasi`.
5. **Salin hari…** di `PanelGeser` (perilaku lama).
6. **Ekspor**: `unduh('/profil/<id>/ekspor')`; notifikasi "Jadwal diekspor".
7. Parameter `?profil=<id>` di URL (bila ada) memilih profil saat halaman dibuka (dipakai setelah impor/buat dari template).

- [ ] **Step 1:** `JadwalMinggu.svelte`, lalu rombak `Jadwal.svelte`.
- [ ] **Step 2:** Perbarui `jadwal.spec.ts` (tes lama lulus, disesuaikan dengan panel geser) + tambah: (a) tampilan Minggu menampilkan bel baru di kolom Senin, klik kartunya membuka panel "Ubah bel"; (b) pengalih ke "Hari" bertahan setelah reload; (c) tombol Ekspor mengunduh file `jadwal-*.json` (`page.waitForEvent('download')`) yang isinya `format: "belsekolah-jadwal"`.
- [ ] **Step 3:** `npm run check && npx playwright test e2e/jadwal.spec.ts` → hijau.
- [ ] **Step 4: Commit** — `feat(web): jadwal tampilan mingguan/harian, panel edit, ekspor`

---

### Task 4: Profil & Periode — duplikat, ekspor, simpan template, impor

**Files:** Create `web/src/components/AlurImpor.svelte`, `web/e2e/cadangan.spec.ts`; Modify `web/src/pages/ProfilPeriode.svelte`, `web/e2e/profil-libur.spec.ts`

**Interfaces:**
- `AlurImpor.svelte` props: `buka` (bindable), `data: FileJadwal | null` (isi file/template/pembuat), `namaAwal?: string` (bila diberikan, menimpa `namaTersedia`), `judul?: string` (bawaan "Impor jadwal"), `onselesai: (profilId: number) => void`. Perilaku: saat `buka && data`, panggil `POST /impor/periksa` dengan `data`; tampilkan di `PanelGeser`: isian **Nama profil** (awal = `namaAwal ?? namaTersedia`), ringkasan "n bel", dan bila `jenisBelTakDikenal` tidak kosong, tabel pemetaan: tiap nama → `<select>` berisi "Buat jenis bel baru" (bawaan) + semua jenis bel yang ada. Tombol **Impor** → `POST /impor` → notifikasi "Profil <nama> dibuat" → `onselesai(profilId)`. Error (400/409) tampil di panel tanpa menutupnya.

**Perilaku halaman:**
1. Laptop: dua kartu berdampingan (`grid lg:grid-cols-2 gap-6`); HP bertumpuk.
2. Kartu **Profil**: `BarAlat` kanan — **Impor jadwal** (input file `.json` tersembunyi; baca teks → `JSON.parse`; gagal parse → notifikasi "File bukan file jadwal") dan **+ Profil**. `Tabel` kolom: Nama (+ lencana "Default"), Jumlah bel, Aksi. Aksi per baris dalam menu/tombol ikon berlabel: **Ubah nama**, **Duplikat**, **Ekspor**, **Simpan sebagai template**, **Jadikan default**, **Hapus** (aturan lama: default tak bisa dihapus, dsb.).
   - **Duplikat** → `PanelGeser` isian nama (awal `"<nama> (salinan)"`) → `POST /profil/{id}/duplikat` → muat ulang daftar, notifikasi.
   - **Ekspor** → `unduh('/profil/{id}/ekspor')`.
   - **Simpan sebagai template** → `PanelGeser` isian nama template (awal = nama profil) → `POST /template {nama, profilId}` → notifikasi "Template disimpan" dengan tautan/teks ke halaman Template.
   - Impor selesai → pindah ke `/jadwal?profil=<id>`.
3. Kartu **Periode khusus**: `Tabel` (Rentang, Profil, Keterangan, Aksi) + `PanelGeser` tambah/ubah (perilaku lama).

- [ ] **Step 1:** `AlurImpor.svelte`, rombak halaman.
- [ ] **Step 2:** Perbarui `profil-libur.spec.ts` bagian profil/periode untuk panel geser.
- [ ] **Step 3:** `web/e2e/cadangan.spec.ts`:
  - duplikat profil default → profil "<nama> (salinan)" muncul dengan jumlah bel yang sama.
  - ekspor profil → simpan file download → impor file yang sama lewat tombol "Impor jadwal" (`setInputFiles`) → panel menampilkan nama "<nama> (impor)" → Impor → URL `/jadwal?profil=` dan profil terpilih.
  - impor file buatan tes berisi jenis bel "Bel Literasi Uji" → panel menampilkan baris pemetaan untuk nama itu → pilih "Buat jenis bel baru" → Impor sukses → halaman Jenis Bel memuat "Bel Literasi Uji".
  - file rusak (`{"a":1}`) → pesan galat "bukan file jadwal" terlihat.
  - "Simpan sebagai template" → notifikasi "Template disimpan".
  Pakai `unik()` untuk nama agar tes bisa diulang.
- [ ] **Step 4:** `npm run check && npx playwright test e2e/cadangan.spec.ts e2e/profil-libur.spec.ts` → hijau.
- [ ] **Step 5: Commit** — `feat(web): duplikat, ekspor/impor jadwal, simpan profil sebagai template`

---

### Task 5: Halaman Template

**Files:** Create `web/src/pages/Template.svelte`, `web/src/lib/pembuat.ts`, `web/src/lib/pembuat.test.ts`, `web/e2e/template.spec.ts`; Modify `web/src/rute.ts`

**Interfaces:**
- `pembuat.ts`: `nilaiAwal(jenjang, varian): ParameterPembuat` sesuai tabel spec §6.2 —

  | | SD | SMP | SMA |
  |---|---|---|---|
  | `lamaJP` normal/ujian · ramadan | 35 · 25 | 40 · 30 | 45 · 30 |
  | `jumlahJP` / `jumlahJPJumat` / `jumlahJPSabtu` | 6/4/4 | 7/5/5 | 8/5/5 |
  | `istirahat` | `[{3,15},{5,15}]` | `[{3,15},{6,15}]` | `[{4,15},{6,15}]` |

  Bersama: `hariSekolah: 5`, `jamMasuk: '07:00'` (ramadan `'07:30'`), `upacaraSenin: {aktif: true, menit: 40}`, `kegiatanJumat: {aktif: false, menit: 30, label: 'Kegiatan Jumat'}`, `ujian: {jamMulai: '07:30', jumlahSesi: 2, lamaSesi: 120, jeda: 30}`.
  `kePratinjau(d: FileJadwal): JadwalMinggu['bel']` — memetakan bel file ke props `JadwalMinggu` (warna dari `d.jenisBel`).
- Rute: `{ path: '/template', label: 'Template', ikon: LayoutTemplate (lucide), utama: false, halaman: Template }` disisipkan setelah Profil & Periode.

**Perilaku:**
1. Bagian **Pembuat jadwal** (kartu, `grid lg:grid-cols-[380px_1fr] gap-6`):
   - Kiri (form): pilihan **Jenjang** (SD/SMP/SMA-SMK, tombol segmen) dan **Varian** (Normal/Ramadan/Ujian) — mengganti keduanya mengisi ulang semua isian dari `nilaiAwal`; **Hari sekolah** (5 hari Sen–Jum / 6 hari Sen–Sab); untuk normal/ramadan: Jam masuk, Lama 1 JP (menit), JP Senin–Kamis, JP Jumat, JP Sabtu (hanya bila 6 hari), Istirahat 1 & 2 (centang aktif, "setelah JP ke-", menit), Upacara Senin (centang, menit), Kegiatan pagi Jumat (centang, menit, nama); untuk ujian: Jam mulai, Jumlah sesi, Lama sesi, Jeda.
   - Kanan: **Pratinjau** `JadwalMinggu` (padat) dari `POST /template/buat`, dipanggil ulang 300ms setelah isian terakhir berubah (debounce; abaikan respons yang sudah usang). Error 400 tampil sebagai pesan merah di atas pratinjau (pratinjau terakhir yang valid tetap terlihat, diberi opasitas).
   - Tombol **Buat profil dari jadwal ini** → `AlurImpor` dengan `data` = hasil terakhir → selesai → `/jadwal?profil=<id>`.
2. Bagian **Template saya**: `BarAlat` kanan **Impor template** (file `.json` → `POST /template {nama: data.nama, data}`); `Tabel` kolom Nama, Jumlah bel, Hari (mis. "Sen–Jum" / daftar singkat), Dibuat (tanggal), Aksi: **Pakai** (`GET /template/{id}` → `AlurImpor` dengan `data`), **Ekspor** (`unduh`), **Ganti nama** (`PanelGeser`), **Hapus** (konfirmasi). Kosong: "Belum ada template. Simpan profil sebagai template dari halaman Profil & Periode."

- [ ] **Step 1:** `pembuat.test.ts` (gagal) — `nilaiAwal('SMP','normal')` sama dengan objek lengkap di tabel; `nilaiAwal('SD','ramadan').lamaJP === 25` dan `jamMasuk === '07:30'`; `kePratinjau` memberi warna dari `jenisBel`. Lalu implementasi → lulus.
- [ ] **Step 2:** Halaman + rute.
- [ ] **Step 3:** `web/e2e/template.spec.ts`:
  - buka `/template`, pilih SMP + Normal → pratinjau kolom Senin memuat "07:00" dan "Upacara"; ubah "Lama 1 JP" ke 45 → dalam 3 dtk pratinjau memuat jam yang berubah (mis. kartu "Pulang" tidak lagi "12:50").
  - pilih 6 hari → kolom Sabtu muncul.
  - isi Lama 1 JP = 5 → pesan galat "lama jam pelajaran" terlihat.
  - Buat profil → nama awal "SMP Normal 5 hari" (atau dengan akhiran bila sudah ada) → Impor → di `/jadwal?profil=`.
  - Impor template dari file (hasil ekspor profil) → muncul di tabel → Pakai → Impor → profil baru; Ganti nama; Hapus.
- [ ] **Step 4:** `npm run check && npx vitest run && npx playwright test e2e/template.spec.ts e2e/tata-letak.spec.ts` → hijau.
- [ ] **Step 5: Commit** — `feat(web): halaman Template (pembuat jadwal bawaan dan template saya)`

---

### Task 6: Libur, Jenis Bel, Pustaka Suara, Riwayat

**Files:** Modify `web/src/pages/Libur.svelte`, `web/src/pages/JenisBel.svelte`, `web/src/pages/PustakaSuara.svelte`, `web/src/pages/Riwayat.svelte`, `web/e2e/profil-libur.spec.ts`, `web/e2e/suara.spec.ts`, `web/e2e/riwayat-pengaturan.spec.ts`

**Perilaku (pola sama untuk semua):** `BarAlat` di atas (`kiri` = filter/tab, `kanan` = tombol tambah), `Tabel` lebar di laptop, kartu bertumpuk di HP (boleh tetap tabel dengan gulir internal bila kolom ≤ 3), form di `PanelGeser`, hapus dengan `konfirmasi`.
- **Libur**: kolom Tanggal/rentang, Keterangan, Aksi.
- **Jenis Bel**: kolom Warna (titik), Nama, Urutan suara (nama-nama dipisah "→"), Aksi (**Tes** ke speaker, Ubah, Hapus). Panel ubah berisi nama, warna, dan penyusun urutan suara (tambah/naik/turun/hapus — perilaku lama).
- **Pustaka Suara**: tab Bawaan/Upload/TTS di `BarAlat` kiri (`role="tablist"`), satu `Tabel` (Nama, Sumber, Durasi, Putar di browser, Tes, Aksi). Unggah & buat TTS di `PanelGeser` (perilaku lama termasuk pratinjau TTS).
- **Riwayat**: `BarAlat` kiri filter tanggal + status; `Tabel` (Waktu, Keterangan, Jenis bel, Sumber, Status berwarna, Pesan) memenuhi lebar.

- [ ] **Step 1:** Rombak keempat halaman satu per satu; jalankan tes Playwright terkait setelah masing-masing.
- [ ] **Step 2:** Perbarui tes lama untuk panel geser; tambah satu tes 1440×900 per halaman: kotak tabel lebarnya ≥ 1100px.
- [ ] **Step 3:** `npm run check && npx playwright test e2e/profil-libur.spec.ts e2e/suara.spec.ts e2e/riwayat-pengaturan.spec.ts` → hijau.
- [ ] **Step 4: Commit** — `feat(web): tabel lebar dan panel geser untuk libur, jenis bel, suara, riwayat`

---

### Task 7: Pengaturan — grid kartu dan zona waktu

**Files:** Modify `web/src/pages/Pengaturan.svelte`, `web/e2e/riwayat-pengaturan.spec.ts`; Create `web/e2e/zona.spec.ts`

**Perilaku:**
1. Grid `grid md:grid-cols-2 xl:grid-cols-3 gap-6` berisi kartu: **Volume**, **Zona waktu**, **Profil default** (bila memang ada di halaman lama), **Ganti password**, **Info sistem**, **Backup & restore** (perilaku lama; logika tunggu restart memakai `lib/mulaiUlang.ts`).
2. Kartu **Zona waktu**: tiga pilihan radio — "WIB — Waktu Indonesia Barat (Jawa, Sumatra, Kalimantan Barat & Tengah)", "WITA — Waktu Indonesia Tengah (Bali, NTB, NTT, Sulawesi, Kalimantan Selatan, Timur & Utara)", "WIT — Waktu Indonesia Timur (Maluku, Papua)". Yang aktif terpilih dari `pengaturan.zonaWaktu`. Tombol **Simpan zona** aktif hanya bila pilihan berbeda. Klik → `konfirmasi('Ganti zona waktu?', 'Aplikasi akan dimulai ulang (±2 detik). Bel yang jatuh tepat saat itu bisa terlewat.', 'Ganti zona')` → `bacaBoot()` → `PUT /pengaturan` (kirim volume & profil default saat ini + `zonaWaktu`) → bila `mulaiUlang`: tampilkan layar/overlay "Menerapkan zona waktu…" lalu `tungguMulaiUlang(bootLama, …)`; habis waktu → pesan "Aplikasi belum kembali. Muat ulang halaman sebentar lagi." (seperti restore).
3. Info sistem menampilkan "Zona waktu: WITA (Asia/Makassar)" dari `zonaWaktu` + `zonaIana`.

- [ ] **Step 1:** Rombak halaman.
- [ ] **Step 2:** Perbarui `riwayat-pengaturan.spec.ts` (selektor kartu; teks zona di info sistem kini `WIB (Asia/Jakarta)` atau zona aktif).
- [ ] **Step 3:** `web/e2e/zona.spec.ts` — `test.skip(!process.env.BEL_URL, 'restart otomatis hanya ada di Docker')`; jalankan **serial**: pilih WITA → konfirmasi → halaman memuat ulang sendiri → jam bar atas berakhiran `WITA` dan Info sistem "Asia/Makassar"; lalu kembalikan ke WIB dengan cara yang sama (agar tes lain tidak terpengaruh). Tes lokal (tanpa `BEL_URL`): hanya pastikan tombol Simpan zona nonaktif saat pilihan sama dan aktif saat berbeda, tanpa menyimpan.
- [ ] **Step 4:** `npm run check && npx playwright test e2e/riwayat-pengaturan.spec.ts e2e/zona.spec.ts` → hijau (zona yang butuh Docker dilewati).
- [ ] **Step 5: Commit** — `feat(web): pengaturan berbentuk grid kartu dan pilihan zona waktu WIB/WITA/WIT`

---

### Task 8: Rapikan, build produksi, uji penuh di Docker

**Files:** Modify sesuai temuan; `internal/webui/dist/*` (build produksi); `README.md` (bagian tampilan bila perlu)

- [ ] **Step 1:** Cari sisa `max-w-` pada halaman (kecuali Login/Ganti Password Awal/panel/dialog) dan sisa pemakaian `Modal` untuk form; ganti ke `PanelGeser`. `grep -rn "max-w-\|<Modal" web/src` lalu tinjau setiap hasil.
- [ ] **Step 2:** `cd web && npm run check && npx vitest run && npx playwright test` (lokal, go run) → semua lulus kecuali yang sengaja dilewati.
- [ ] **Step 3:** Build produksi: `cd web && env -u VITE_E2E npm run build`; pastikan `grep -rq __bel ../internal/webui/dist` **tidak** menemukan apa pun.
- [ ] **Step 4:** Uji penuh di Docker: `BEL_PORT=18080 ./scripts/e2e.sh` → Playwright (termasuk `zona.spec.ts`) dan e2e Go lulus.
- [ ] **Step 5:** `go vet ./... && go test -race ./...` → hijau.
- [ ] **Step 6: Commit** — `build(web): build produksi tampilan layar penuh`
