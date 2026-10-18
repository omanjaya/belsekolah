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
export interface Profil { id: number; nama: string; jumlahBel?: number }
export interface JenisBel { id: number; nama: string; warna: string; audioIds: number[] }
export interface Audio {
  id: number; nama: string; sumber: 'bawaan' | 'upload' | 'tts'; durasiMs: number
  teksTts?: string; suaraTts?: string; dibuat: string
}
export interface Jadwal { id: number; profilId: number; hari: number; jam: string; jenisBelId: number; label: string; pengumumanId: number | null }
export interface SuaraRingkas { id: number; nama: string }
export interface SaranPengumuman { jadwalId: number; hari: number; jam: string; label: string; suara: SuaraRingkas }
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
  pengumuman: SuaraRingkas | null
}
export interface Status {
  sekarang: string; tanggal: string; hari: number; profil: Profil | null; libur: Libur | null
  bel: BelHariIni[]; berikutnya: BelHariIni | null; peringatan: string[]
  zonaWaktu: string
}
export interface Pengaturan {
  volume: number; profilDefaultId: number; zonaWaktu: 'WIB' | 'WITA' | 'WIT' | string; mulaiUlang?: boolean
}
export interface Sistem {
  versi: string; zonaWaktu: string; zonaIana: string; sekarang: string; ip: string[]; ntp: boolean | null
  boot?: string
  alat?: { mpv: boolean; ffprobe: boolean; edgeTts: boolean }
  ruangKosongMb?: number
  backupTerakhir?: string
}
export interface SuaraTTS { id: string; nama: string }

export interface FileJadwal {
  format: 'belsekolah-jadwal'; versi: 1; nama: string
  jenisBel: { nama: string; warna: string }[]
  jadwal: { hari: number; jam: string; jenisBel: string; label: string }[]
}
export interface HasilPeriksa { nama: string; jumlahBel: number; jenisBelTakDikenal: string[]; namaTersedia: string; pengumumanTakDitemukan: string[] }
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

export type StatusGabung = 'baru' | 'kembar' | 'bentrok'
export interface IsiGabung { jenisBel: string; label: string }
export interface BarisGabung { hari: number; jam: string; status: StatusGabung; lama: IsiGabung | null; baru: IsiGabung }
export interface HasilPeriksaGabung {
  jenisBelTakDikenal: string[]
  pengumumanTakDitemukan: string[]
  ringkasan: { baru: number; kembar: number; bentrok: number }
  baris: BarisGabung[]
}
export interface HasilGabung { ditambah: number; ditimpa: number; dilewati: number }

/** Unduh file dari API (mis. ekspor jadwal); nama file dari Content-Disposition. */
export async function unduh(path: string): Promise<void> {
  let res: Response
  try {
    res = await fetch('/api' + path)
  } catch {
    throw new ApiError(0, 'Tidak bisa terhubung ke server bel. Periksa jaringan WiFi.')
  }
  if (res.status === 401) saatTidakMasuk()
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
  const cd = res.headers.get('Content-Disposition') ?? ''
  const nama = cd.match(/filename="([^"]+)"/)?.[1] ?? 'unduhan.json'
  const url = URL.createObjectURL(await res.blob())
  const a = document.createElement('a')
  a.href = url
  a.download = nama
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
