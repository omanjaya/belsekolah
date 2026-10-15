import { describe, expect, it } from 'vitest'
import { formatJamServer, offsetMenit, selisihMs } from './jam'

describe('jam server', () => {
  it('offsetMenit membaca offset RFC3339', () => {
    expect(offsetMenit('2026-10-15T07:42:10+08:00')).toBe(480)
    expect(offsetMenit('2026-10-15T07:42:10.123+07:00')).toBe(420)
    expect(offsetMenit('2026-10-15T00:00:00Z')).toBe(0)
    expect(offsetMenit('2026-10-15T07:00:00-03:30')).toBe(-210)
  })
  it('selisihMs = waktu server - waktu browser', () => {
    expect(selisihMs('2026-10-15T00:00:05Z', Date.parse('2026-10-15T00:00:00Z'))).toBe(5000)
  })
  it('formatJamServer memakai zona server, bukan zona browser', () => {
    const browser = Date.parse('2026-10-14T23:42:10Z')
    expect(formatJamServer(browser, 0, 480)).toBe('07:42:10')
    expect(formatJamServer(browser, 5000, 480)).toBe('07:42:15')
    expect(formatJamServer(browser, 0, 540)).toBe('08:42:10')
  })
})
