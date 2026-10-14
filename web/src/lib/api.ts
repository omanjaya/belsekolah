export class ApiError extends Error {
  constructor(public status: number, pesan: string) {
    super(pesan)
  }
}

let saatTidakMasuk: () => void = () => {}
/** Dipanggil saat API menjawab 401 (sesi habis). */
export function aturSaatTidakMasuk(f: () => void) {
  saatTidakMasuk = f
}

async function minta<T>(metode: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  const init: RequestInit = { method: metode, headers }
  if (body instanceof FormData) {
    init.body = body
  } else if (body !== undefined) {
    init.body = JSON.stringify(body)
    headers['Content-Type'] = 'application/json'
  }
  let res: Response
  try {
    res = await fetch('/api' + path, init)
  } catch {
    throw new ApiError(0, 'Tidak bisa terhubung ke server bel. Periksa jaringan WiFi.')
  }
  if (res.status === 401 && path !== '/login') saatTidakMasuk()
  if (!res.ok) {
    let pesan = `Terjadi kesalahan (${res.status})`
    try {
      const j = await res.json()
      if (j && typeof j.error === 'string') pesan = j.error
    } catch {
      /* bukan JSON */
    }
    throw new ApiError(res.status, pesan)
  }
  if (res.status === 204) return undefined as T
  const ct = res.headers.get('Content-Type') ?? ''
  if (ct.includes('application/json')) return (await res.json()) as T
  return (await res.blob()) as T
}

export const api = {
  get: <T>(path: string) => minta<T>('GET', path),
  post: <T>(path: string, body?: unknown) => minta<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => minta<T>('PUT', path, body),
  hapus: (path: string) => minta<void>('DELETE', path),
}

export interface Sesi { masuk: boolean; wajibGantiPassword: boolean; boot?: string }
export interface Profil { id: number; nama: string }
export interface JenisBel { id: number; nama: string; warna: string; audioIds: number[] }
export interface Audio {
  id: number; nama: string; sumber: 'bawaan' | 'upload' | 'tts'; durasiMs: number
  teksTts?: string; suaraTts?: string; dibuat: string
}
export interface Jadwal { id: number; profilId: number; hari: number; jam: string; jenisBelId: number; label: string }
export interface Periode { id: number; mulai: string; selesai: string; profilId: number; keterangan: string }
export interface Libur { id: number; mulai: string; selesai: string; keterangan: string }
export interface Riwayat {
  id: number; waktu: string; jadwalId?: number; jenisBelId?: number; label: string
  sumber: 'jadwal' | 'manual' | 'darurat'
  status: 'berhasil' | 'gagal' | 'terlewat' | 'cadangan' | 'dihentikan'; pesan?: string
}
export interface BelHariIni {
  jadwalId: number; jam: string; waktu: string; label: string
  jenisBel: { id: number; nama: string; warna: string } | null; sudah: boolean
}
export interface Status {
  sekarang: string; tanggal: string; hari: number; profil: Profil | null; libur: Libur | null
  bel: BelHariIni[]; berikutnya: BelHariIni | null; peringatan: string[]
}
export interface Pengaturan { volume: number; profilDefaultId: number }
export interface Sistem {
  versi: string; zonaWaktu: string; sekarang: string; ip: string[]; ntp: boolean | null
  boot?: string
  alat?: { mpv: boolean; ffprobe: boolean; edgeTts: boolean }
  ruangKosongMb?: number
  backupTerakhir?: string
}
export interface SuaraTTS { id: string; nama: string }
