import { execSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import type { APIRequestContext } from '@playwright/test'

export const PASSWORD_E2E = 'rahasia-e2e-123'

/** Password awal: dari file data lokal, atau dari container Docker jika BEL_URL dipakai. */
export function passwordAwal(): string | null {
  const port = Number(process.env.PORT ?? 18080)
  const file = `.e2e-data-${port}/password-awal.txt`
  if (!process.env.BEL_URL) {
    return existsSync(file) ? readFileSync(file, 'utf8').trim() : null
  }
  try {
    return execSync('docker compose exec -T belsekolah cat /var/lib/belsekolah/password-awal.txt', { cwd: '..' })
      .toString().trim()
  } catch {
    return null
  }
}

export async function apiMasuk(request: APIRequestContext) {
  const res = await request.post('/api/login', { data: { password: PASSWORD_E2E } })
  if (!res.ok()) throw new Error(`login API gagal: ${res.status()}`)
}

export async function jenisBelId(request: APIRequestContext, nama: string): Promise<number> {
  const res = await request.get('/api/jenis-bel')
  const daftar = (await res.json()) as { id: number; nama: string }[]
  const j = daftar.find((x) => x.nama === nama)
  if (!j) throw new Error(`jenis bel ${nama} tidak ada`)
  return j.id
}

export async function profilDefault(request: APIRequestContext): Promise<number> {
  const res = await request.get('/api/pengaturan')
  return ((await res.json()) as { profilDefaultId: number }).profilDefaultId
}

/** Nama unik per jalankan tes, agar data antar-tes tidak bentrok. */
export function unik(awalan: string): string {
  return `${awalan} ${Date.now().toString(36).slice(-5)}`
}
