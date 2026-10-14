import { describe, expect, it } from 'vitest'
import {
  formatDurasi, formatJam, formatRentang, formatTanggal, formatTanggalPanjang,
  hariDariTanggal, hitungMundur, offsetDariISO, tanggalHariIni,
} from './format'

describe('format', () => {
  it('formatTanggal & formatTanggalPanjang', () => {
    expect(formatTanggal('2026-10-15')).toBe('15 Oktober 2026')
    expect(formatTanggalPanjang('2026-10-15')).toBe('Kamis, 15 Oktober 2026')
    expect(formatTanggalPanjang('2026-10-18')).toBe('Minggu, 18 Oktober 2026')
  })
  it('hariDariTanggal 1=Senin..7=Minggu', () => {
    expect(hariDariTanggal('2026-10-12')).toBe(1)
    expect(hariDariTanggal('2026-10-18')).toBe(7)
  })
  it('formatRentang', () => {
    expect(formatRentang('2026-12-25', '2026-12-25')).toBe('25 Desember 2026')
    expect(formatRentang('2026-12-29', '2027-01-02')).toBe('29 Desember 2026 – 2 Januari 2027')
  })
  it('formatDurasi', () => {
    expect(formatDurasi(0)).toBe('0 dtk')
    expect(formatDurasi(1800)).toBe('1,8 dtk')
    expect(formatDurasi(65000)).toBe('1 mnt 5 dtk')
    expect(formatDurasi(59999)).toBe('1 mnt 0 dtk')
    expect(formatDurasi(119600)).toBe('2 mnt 0 dtk')
  })
  it('hitungMundur', () => {
    expect(hitungMundur(-5)).toBe('sekarang')
    expect(hitungMundur(45_000)).toBe('dalam 45 detik')
    expect(hitungMundur(12 * 60_000 + 5_000)).toBe('dalam 12 menit 5 detik')
    expect(hitungMundur(2 * 3_600_000 + 3 * 60_000 + 59_000)).toBe('dalam 2 jam 3 menit')
  })
  it('offsetDariISO & formatJam memakai zona server', () => {
    expect(offsetDariISO('2026-10-15T10:42:15.123456+07:00')).toBe(420)
    expect(offsetDariISO('2026-10-15T03:42:15Z')).toBe(0)
    expect(offsetDariISO('2026-10-15T10:42:15+09:00')).toBe(540)
    const ms = Date.parse('2026-10-15T10:42:15+07:00')
    expect(formatJam(ms, 420)).toBe('10:42:15')
    expect(formatJam(ms, 480)).toBe('11:42:15')
    expect(tanggalHariIni(ms, 420)).toBe('2026-10-15')
    expect(tanggalHariIni(Date.parse('2026-10-15T23:30:00Z'), 420)).toBe('2026-10-16')
  })
})
