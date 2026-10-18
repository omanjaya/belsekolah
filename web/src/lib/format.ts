export const NAMA_HARI = ['', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu', 'Minggu']
const BULAN = ['Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni', 'Juli', 'Agustus',
  'September', 'Oktober', 'November', 'Desember']

function urai(tgl: string): [number, number, number] {
  const [y, m, d] = tgl.split('-').map(Number)
  return [y, m, d]
}

export function formatTanggal(tgl: string): string {
  const [y, m, d] = urai(tgl)
  return `${d} ${BULAN[m - 1]} ${y}`
}

/** 1 = Senin … 7 = Minggu, dihitung tanpa zona waktu. */
export function hariDariTanggal(tgl: string): number {
  const [y, m, d] = urai(tgl)
  const w = new Date(Date.UTC(y, m - 1, d)).getUTCDay()
  return w === 0 ? 7 : w
}

export function formatTanggalPanjang(tgl: string): string {
  return `${NAMA_HARI[hariDariTanggal(tgl)]}, ${formatTanggal(tgl)}`
}

export function formatRentang(mulai: string, selesai: string): string {
  return mulai === selesai ? formatTanggal(mulai) : `${formatTanggal(mulai)} – ${formatTanggal(selesai)}`
}

export function formatDurasi(ms: number): string {
  const detik = ms / 1000
  if (Math.round(detik * 10) / 10 < 60) {
    return `${detik.toFixed(detik % 1 === 0 ? 0 : 1).replace('.', ',')} dtk`
  }
  const total = Math.round(detik)
  return `${Math.floor(total / 60)} mnt ${total % 60} dtk`
}

export function hitungMundur(ms: number): string {
  if (ms <= 0) return 'sekarang'
  const total = Math.floor(ms / 1000)
  const jam = Math.floor(total / 3600)
  const menit = Math.floor((total % 3600) / 60)
  const detik = total % 60
  if (jam > 0) return `dalam ${jam} jam ${menit} menit`
  if (menit > 0) return `dalam ${menit} menit ${detik} detik`
  return `dalam ${detik} detik`
}

/** Offset zona waktu (menit) dari string ISO server, mis. "+07:00" → 420. */
export function offsetDariISO(iso: string): number {
  const m = iso.match(/([+-])(\d{2}):(\d{2})$/)
  if (!m) return 0
  const nilai = Number(m[2]) * 60 + Number(m[3])
  return m[1] === '-' ? -nilai : nilai
}

function geser(ms: number, offsetMenit: number): Date {
  return new Date(ms + offsetMenit * 60_000)
}

const dua = (n: number) => String(n).padStart(2, '0')

/** HH:MM:SS pada zona server. */
export function formatJam(ms: number, offsetMenit: number): string {
  const t = geser(ms, offsetMenit)
  return `${dua(t.getUTCHours())}:${dua(t.getUTCMinutes())}:${dua(t.getUTCSeconds())}`
}

/** YYYY-MM-DD pada zona server. */
export function tanggalHariIni(ms: number, offsetMenit: number): string {
  const t = geser(ms, offsetMenit)
  return `${t.getUTCFullYear()}-${dua(t.getUTCMonth() + 1)}-${dua(t.getUTCDate())}`
}

/** Posisi pemutaran dalam format m:ss, misalnya 0:07 atau 2:05. */
export function formatWaktu(detik: number): string {
  const total = Math.max(0, Math.floor(Number.isFinite(detik) ? detik : 0))
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`
}
