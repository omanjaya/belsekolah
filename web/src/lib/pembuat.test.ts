import { describe, expect, it } from 'vitest'
import type { FileJadwal } from './api'
import { kePratinjau, nilaiAwal } from './pembuat'

describe('nilaiAwal', () => {
  it('SMP normal sesuai tabel spec', () => {
    expect(nilaiAwal('SMP', 'normal')).toEqual({
      jenjang: 'SMP',
      varian: 'normal',
      hariSekolah: 5,
      jamMasuk: '07:00',
      lamaJP: 40,
      jumlahJP: 7,
      jumlahJPJumat: 5,
      jumlahJPSabtu: 5,
      istirahat: [
        { setelahJP: 3, menit: 15 },
        { setelahJP: 6, menit: 15 },
      ],
      upacaraSenin: { aktif: true, menit: 40 },
      kegiatanJumat: { aktif: false, menit: 30, label: 'Kegiatan Jumat' },
      ujian: { jamMulai: '07:30', jumlahSesi: 2, lamaSesi: 120, jeda: 30 },
    })
  })

  it('ramadan: JP lebih singkat dan masuk 07:30', () => {
    const d = nilaiAwal('SD', 'ramadan')
    expect(d.lamaJP).toBe(25)
    expect(d.jamMasuk).toBe('07:30')
    expect(d.jumlahJP).toBe(6)
  })

  it('SMA: 45 menit, 8 JP, istirahat setelah JP 4 dan 6', () => {
    const d = nilaiAwal('SMA', 'normal')
    expect(d.lamaJP).toBe(45)
    expect(d.jumlahJP).toBe(8)
    expect(d.istirahat.map((i) => i.setelahJP)).toEqual([4, 6])
    expect(nilaiAwal('SMA', 'ramadan').lamaJP).toBe(30)
  })

  it('ujian memakai lama JP normal', () => {
    expect(nilaiAwal('SMP', 'ujian').lamaJP).toBe(40)
  })

  it('mengembalikan objek baru tiap panggilan', () => {
    const a = nilaiAwal('SD', 'normal')
    a.istirahat[0].menit = 99
    expect(nilaiAwal('SD', 'normal').istirahat[0].menit).toBe(15)
  })
})

describe('kePratinjau', () => {
  const d: FileJadwal = {
    format: 'belsekolah-jadwal',
    versi: 1,
    nama: 'X',
    jenisBel: [{ nama: 'Masuk', warna: '#2563eb' }],
    jadwal: [
      { hari: 1, jam: '07:00', jenisBel: 'Masuk', label: 'Upacara' },
      { hari: 2, jam: '07:00', jenisBel: 'Asing', label: '' },
    ],
  }
  it('memetakan warna dari jenisBel dan memberi kunci unik', () => {
    const p = kePratinjau(d)
    expect(p).toHaveLength(2)
    expect(p[0]).toMatchObject({ hari: 1, jam: '07:00', label: 'Upacara', jenisNama: 'Masuk', warna: '#2563eb' })
    expect(p[1].warna).toBe('#64748b')
    expect(p[0].kunci).not.toBe(p[1].kunci)
  })
})
