/** Offset zona (menit) dari string RFC3339, mis. "+08:00" → 480, "Z" → 0. */
export function offsetMenit(iso: string): number {
  const m = iso.match(/([+-])(\d{2}):(\d{2})$/)
  if (!m) return 0
  const nilai = Number(m[2]) * 60 + Number(m[3])
  return m[1] === '-' ? -nilai : nilai
}

/** Selisih (ms) waktu server terhadap jam browser; tambahkan ke jam browser untuk mendapat waktu server. */
export function selisihMs(iso: string, browserMs: number): number {
  return Date.parse(iso) - browserMs
}

const dua = (n: number) => String(n).padStart(2, '0')

/** HH:MM:SS waktu lokal server, tidak bergantung pada zona browser. */
export function formatJamServer(browserMs: number, selisih: number, offset: number): string {
  const t = new Date(browserMs + selisih + offset * 60_000)
  return `${dua(t.getUTCHours())}:${dua(t.getUTCMinutes())}:${dua(t.getUTCSeconds())}`
}
