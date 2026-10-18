import { api } from './api'
import { beriTahu, galat } from './notif.svelte'

export type Mode = 'browser' | 'speaker'

/**
 * Satu pemutar suara untuk seluruh aplikasi: hanya satu suara yang aktif,
 * baik di browser maupun di speaker. Memutar suara lain menghentikan yang lama.
 */
class PemutarSuara {
  aktif = $state<{ kunci: string; mode: Mode; nama: string } | null>(null)
  memuat = $state(false)
  posisi = $state(0) // detik
  durasi = $state(0) // detik, 0 bila belum diketahui

  #audio: HTMLAudioElement | null = null
  #jam: ReturnType<typeof setInterval> | undefined
  #urut = 0

  sedang(kunci: string, mode?: Mode) {
    return this.aktif?.kunci === kunci && (!mode || this.aktif.mode === mode)
  }

  #elemen() {
    if (this.#audio) return this.#audio
    const a = new Audio()
    a.preload = 'auto'
    a.addEventListener('playing', () => (this.memuat = false))
    a.addEventListener('timeupdate', () => {
      if (this.aktif?.mode === 'browser') this.posisi = a.currentTime
    })
    a.addEventListener('loadedmetadata', () => {
      if (this.aktif?.mode === 'browser' && Number.isFinite(a.duration)) this.durasi = a.duration
    })
    a.addEventListener('ended', () => {
      if (this.aktif?.mode === 'browser') this.#bersihkan()
    })
    this.#audio = a
    return a
  }

  #bersihkan() {
    clearInterval(this.#jam)
    this.#jam = undefined
    this.aktif = null
    this.memuat = false
    this.posisi = 0
    this.durasi = 0
  }

  /** Menghentikan suara aktif (browser maupun speaker). */
  async henti() {
    const mode = this.aktif?.mode
    this.#urut++
    this.#audio?.pause()
    this.#bersihkan()
    if (mode === 'speaker') {
      try {
        await api.post('/tes/stop')
      } catch (e) {
        galat(e)
      }
    }
  }

  /** Memutar di browser (HP/laptop yang membuka halaman). */
  async putarBrowser(kunci: string, src: string, durasiMs = 0, nama = '') {
    if (this.aktif?.mode === 'speaker') void this.henti()
    const urut = ++this.#urut
    clearInterval(this.#jam)
    const a = this.#elemen()
    a.pause()
    this.aktif = { kunci, mode: 'browser', nama }
    this.memuat = true
    this.posisi = 0
    this.durasi = durasiMs / 1000
    a.src = src
    try {
      await a.play()
    } catch (e) {
      if (urut !== this.#urut) return
      if (e instanceof DOMException && e.name === 'AbortError') return
      this.#bersihkan()
      beriTahu('Browser tidak bisa memutar suara ini', 'galat')
    }
  }

  /** Memutar lewat speaker mini PC. Tes baru menggantikan tes lama di server. */
  async putarSpeaker(kunci: string, path: string, durasiMs: number, nama = '') {
    if (this.aktif?.mode === 'browser') this.#audio?.pause()
    const urut = ++this.#urut
    clearInterval(this.#jam)
    this.aktif = { kunci, mode: 'speaker', nama }
    this.memuat = true
    this.posisi = 0
    this.durasi = durasiMs / 1000
    try {
      await api.post(path)
    } catch (e) {
      if (urut === this.#urut) this.#bersihkan()
      galat(e)
      return
    }
    if (urut !== this.#urut) return
    this.memuat = false
    const mulai = Date.now()
    // Server tidak mengabarkan kapan tes selesai; perkirakan dari durasi.
    this.#jam = setInterval(() => {
      this.posisi = (Date.now() - mulai) / 1000
      if (this.posisi >= this.durasi + 0.5) this.#bersihkan()
    }, 250)
  }

  /** Menghentikan pemutaran browser saja (misalnya saat meninggalkan halaman). */
  hentiBrowser() {
    if (this.aktif?.mode === 'browser') {
      this.#urut++
      this.#audio?.pause()
      this.#bersihkan()
    }
  }
}

export const pemutarSuara = new PemutarSuara()
