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
