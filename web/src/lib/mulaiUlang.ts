import type { Sesi } from './api'

/** Ambil boot dari /api/sesi; null bila gagal, bukan 200, atau tanpa boot. */
export async function bacaBoot(): Promise<string | null> {
  const ac = new AbortController()
  const t = setTimeout(() => ac.abort(), 5000)
  try {
    const res = await fetch('/api/sesi', { signal: ac.signal })
    if (res.status !== 200) return null
    const j = (await res.json()) as Sesi
    return j.boot || null
  } catch {
    return null // galat jaringan wajar saat aplikasi memulai ulang
  } finally {
    clearTimeout(t)
  }
}

const tidur = (ms: number) => new Promise((r) => setTimeout(r, ms))

/**
 * Tunggu aplikasi menyala kembali setelah dimulai ulang (boot berubah), lalu muat ulang halaman.
 * true = boot berubah dan location.reload() dipanggil; false = habis waktu atau dibatalkan.
 */
export async function tungguMulaiUlang(
  bootLama: string | null,
  opsi: { batal: () => boolean; batasMs?: number },
): Promise<boolean> {
  const batas = Date.now() + (opsi.batasMs ?? 120_000)
  await tidur(2000)
  while (!opsi.batal() && Date.now() < batas) {
    const b = await bacaBoot()
    if (b && b !== bootLama) {
      if (opsi.batal()) return false
      location.reload()
      return true
    }
    await tidur(2000)
  }
  return false
}
