import type { Status } from './api'
import { formatJamServer, offsetMenit, selisihMs } from './jam'

/** Jam server bersama: `teks` = HH:MM:SS waktu server, `zona` = WIB/WITA/WIT. `peringatan` null sampai status termuat. */
export const jamServer = $state<{ teks: string; zona: string; siap: boolean; peringatan: string[] | null }>({
  teks: '',
  zona: '',
  siap: false,
  peringatan: null,
})

let selisih = 0
let offset = 0
/** Waktu browser (ms) saat status terakhir diterima; dipakai Layout untuk melewati muat ulang yang tak perlu. */
export let terakhirDiperbarui = 0

function hitung() {
  jamServer.teks = formatJamServer(Date.now(), selisih, offset)
}

export function perbaruiDariStatus(s: Status): void {
  const kini = Date.now()
  selisih = selisihMs(s.sekarang, kini)
  offset = offsetMenit(s.sekarang)
  jamServer.zona = s.zonaWaktu
  jamServer.peringatan = s.peringatan
  jamServer.siap = true
  terakhirDiperbarui = kini
  hitung()
}

/** Mulai detak 1 detik; mengembalikan fungsi penghenti. */
export function mulaiDetak(): () => void {
  const id = setInterval(() => {
    if (jamServer.siap) hitung()
  }, 1000)
  return () => clearInterval(id)
}
