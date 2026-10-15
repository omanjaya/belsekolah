import type { FileJadwal, ParameterPembuat } from './api'

type Jenjang = ParameterPembuat['jenjang']
type Varian = ParameterPembuat['varian']

// Nilai awal pembuat jadwal (spec §6.2).
const TABEL: Record<Jenjang, { lama: number; lamaRamadan: number; jp: number; jpJumat: number; jpSabtu: number; istirahat: [number, number] }> = {
  SD: { lama: 35, lamaRamadan: 25, jp: 6, jpJumat: 4, jpSabtu: 4, istirahat: [3, 5] },
  SMP: { lama: 40, lamaRamadan: 30, jp: 7, jpJumat: 5, jpSabtu: 5, istirahat: [3, 6] },
  SMA: { lama: 45, lamaRamadan: 30, jp: 8, jpJumat: 5, jpSabtu: 5, istirahat: [4, 6] },
}

export function nilaiAwal(jenjang: Jenjang, varian: Varian): ParameterPembuat {
  const t = TABEL[jenjang]
  const ramadan = varian === 'ramadan'
  return {
    jenjang,
    varian,
    hariSekolah: 5,
    jamMasuk: ramadan ? '07:30' : '07:00',
    lamaJP: ramadan ? t.lamaRamadan : t.lama,
    jumlahJP: t.jp,
    jumlahJPJumat: t.jpJumat,
    jumlahJPSabtu: t.jpSabtu,
    istirahat: t.istirahat.map((setelahJP) => ({ setelahJP, menit: 15 })),
    upacaraSenin: { aktif: true, menit: 40 },
    kegiatanJumat: { aktif: false, menit: 30, label: 'Kegiatan Jumat' },
    ujian: { jamMulai: '07:30', jumlahSesi: 2, lamaSesi: 120, jeda: 30 },
  }
}

const WARNA_CADANGAN = '#64748b'

export interface BelPratinjau {
  kunci: number
  hari: number
  jam: string
  label: string
  jenisNama: string
  warna: string
}

/** Bel pada file jadwal -> props `bel` milik JadwalMinggu. */
export function kePratinjau(d: FileJadwal): BelPratinjau[] {
  const warna = new Map(d.jenisBel.map((j) => [j.nama, j.warna]))
  return d.jadwal.map((b, i) => ({
    kunci: i,
    hari: b.hari,
    jam: b.jam,
    label: b.label,
    jenisNama: b.jenisBel,
    warna: warna.get(b.jenisBel) ?? WARNA_CADANGAN,
  }))
}
