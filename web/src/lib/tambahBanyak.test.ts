import { describe, expect, it } from 'vitest'
import { hariBerisi,
  hariDipilih, jamGanda, keFileJadwal, kePemetaan, kunciSlot, normalJam, saringHari, uraiTempel } from './tambahBanyak'

describe('normalJam', () => {
  it('menerima titik, titik dua, dan satu digit jam', () => {
    expect(normalJam('7.15')).toBe('07:15')
    expect(normalJam('7:15')).toBe('07:15')
    expect(normalJam(' 07:15 ')).toBe('07:15')
    expect(normalJam('23.59')).toBe('23:59')
  })
  it('menolak yang tidak valid', () => {
    for (const s of ['', '24:00', '7:60', '7', 'jam', '07:5', '07:155']) expect(normalJam(s)).toBeNull()
  })
})

describe('uraiTempel', () => {
  it('memisah dengan TAB, titik koma, atau koma; kolom jenis kosong memakai bawaan', () => {
    const teks = '07:15\tSholat Dhuha\tMasuk\n7.30;Literasi;\n\n  \n08:00, Jam ke-1 ,Ganti Jam'
    const h = uraiTempel(teks, 'Bawaan')
    expect(h.galat).toEqual([])
    expect(h.baris).toEqual([
      { jam: '07:15', label: 'Sholat Dhuha', jenis: 'Masuk' },
      { jam: '07:30', label: 'Literasi', jenis: 'Bawaan' },
      { jam: '08:00', label: 'Jam ke-1', jenis: 'Ganti Jam' },
    ])
  })
  it('hanya jam juga boleh', () => {
    expect(uraiTempel('09:00', 'Masuk').baris).toEqual([{ jam: '09:00', label: '', jenis: 'Masuk' }])
  })
  it('baris rusak dilaporkan dengan nomor baris dan tidak ikut', () => {
    const h = uraiTempel('jam\tX\n07:00\tOk\n1;2;3;4\n08:00\t' + 'a'.repeat(101), 'Masuk')
    expect(h.baris).toEqual([{ jam: '07:00', label: 'Ok', jenis: 'Masuk' }])
    expect(h.galat.map((g) => g.nomor)).toEqual([1, 3, 4])
    expect(h.galat[0].alasan).toContain('jam')
  })
})

describe('jamGanda', () => {
  it('mengembalikan jam yang muncul lebih dari sekali, urut', () => {
    expect(jamGanda([{ jam: '08:00', label: '', jenis: 'A' }, { jam: '07:00', label: '', jenis: 'A' }, { jam: '08:00', label: 'x', jenis: 'B' }])).toEqual(['08:00'])
  })
})

describe('keFileJadwal', () => {
  it('memperluas baris × hari dan mengisi warna jenis tanpa beda huruf', () => {
    const d = keFileJadwal(
      [{ jam: '07:15', label: 'Dhuha', jenis: 'masuk' }, { jam: '09:00', label: 'Ist', jenis: 'Literasi' }],
      [3, 1],
      [{ nama: 'Masuk', warna: '#2563eb' }],
    )
    expect(d.format).toBe('belsekolah-jadwal')
    expect(d.versi).toBe(1)
    expect(d.nama).toBe('Tambah banyak')
    expect(d.jenisBel).toEqual([{ nama: 'masuk', warna: '#2563eb' }, { nama: 'Literasi', warna: '#64748b' }])
    expect(d.jadwal).toEqual([
      { hari: 1, jam: '07:15', jenisBel: 'masuk', label: 'Dhuha' },
      { hari: 1, jam: '09:00', jenisBel: 'Literasi', label: 'Ist' },
      { hari: 3, jam: '07:15', jenisBel: 'masuk', label: 'Dhuha' },
      { hari: 3, jam: '09:00', jenisBel: 'Literasi', label: 'Ist' },
    ])
  })
  it('nama jenis yang hanya beda huruf dicatat sekali', () => {
    const d = keFileJadwal([{ jam: '07:00', label: '', jenis: 'Masuk' }, { jam: '08:00', label: '', jenis: 'MASUK' }], [1], [])
    expect(d.jenisBel).toHaveLength(1)
  })
})

describe('saringHari & hariBerisi', () => {
  const d = keFileJadwal([{ jam: '07:00', label: '', jenis: 'A' }], [1, 2, 5], [])
  it('hariBerisi urut naik', () => expect(hariBerisi(d)).toEqual([1, 2, 5]))
  it('saringHari hanya menyisakan hari terpilih tanpa mengubah aslinya', () => {
    const s = saringHari(d, [5, 1])
    expect(s.jadwal.map((b) => b.hari)).toEqual([1, 5])
    expect(d.jadwal).toHaveLength(3)
  })
})

describe('hariDipilih', () => {
  it('mengiriskan hari tersedia dengan yang sengaja dimatikan', () => {
    expect(hariDipilih([1, 2, 3], [1])).toEqual([2, 3])
  })
  it('hari yang hilang dari sumber diabaikan, hari baru tercentang', () => {
    expect(hariDipilih([2, 3, 4], [1])).toEqual([2, 3, 4])
  })
})

describe('kePemetaan & kunciSlot', () => {
  it('mengubah pilihan select ke pemetaan API', () => {
    expect(kePemetaan({ A: 'baru', B: '7' })).toEqual({ A: { baru: true }, B: { jenisBelId: 7 } })
  })
  it('kunciSlot', () => expect(kunciSlot(1, '07:00')).toBe('1 07:00'))
})
