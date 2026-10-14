import { inflateSync } from 'node:zlib'
import { expect, test } from '@playwright/test'

/** Ambil data chunk IDAT dari PNG 1x1 (tanpa filter/interlace). */
function idat(png: Buffer): Buffer[] {
  const hasil: Buffer[] = []
  let o = 8
  while (o < png.length) {
    const len = png.readUInt32BE(o)
    const jenis = png.toString('ascii', o + 4, o + 8)
    if (jenis === 'IDAT') hasil.push(png.subarray(o + 8, o + 8 + len))
    o += 12 + len
  }
  return hasil
}

test('navigasi bar bawah dan menu Lainnya di HP', async ({ page }) => {
  await page.goto('/')
  const nav = page.getByRole('navigation', { name: 'Menu utama' }).last()
  await nav.getByRole('link', { name: 'Jadwal' }).click()
  await expect(page).toHaveURL(/\/jadwal$/)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Jadwal')

  await nav.getByRole('button', { name: 'Lainnya' }).click()
  await page.getByRole('link', { name: 'Pengaturan' }).click()
  await expect(page).toHaveURL(/\/pengaturan$/)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Pengaturan')

  await page.goBack()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Jadwal')
})

test('sidebar di desktop', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 })
  await page.goto('/riwayat')
  const sidebar = page.getByRole('navigation', { name: 'Menu utama' }).first()
  await expect(sidebar).toBeVisible()
  await expect(sidebar.getByRole('link', { name: 'Riwayat' })).toHaveAttribute('aria-current', 'page')
})

test('path tak dikenal menampilkan halaman tidak ditemukan', async ({ page }) => {
  await page.goto('/tidak-ada')
  await expect(page.getByRole('heading', { name: 'Halaman tidak ditemukan' })).toBeVisible()
})

test('lembar Lainnya: Escape menutup dan fokus kembali ke tombol', async ({ page }) => {
  await page.goto('/')
  const nav = page.getByRole('navigation', { name: 'Menu utama' }).last()
  const tombol = nav.getByRole('button', { name: 'Lainnya' })
  await tombol.click()
  const lembar = page.locator('#lembar-lainnya')
  await expect(lembar).toBeVisible()
  await expect(lembar.getByRole('link').first()).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(lembar).toBeHidden()
  await expect(tombol).toBeFocused()
})

test('lembar Lainnya tertutup setelah pindah lewat bar bawah', async ({ page }) => {
  await page.goto('/')
  const nav = page.getByRole('navigation', { name: 'Menu utama' }).last()
  await nav.getByRole('button', { name: 'Lainnya' }).click()
  await expect(page.locator('#lembar-lainnya')).toBeVisible()
  await nav.getByRole('link', { name: 'Jadwal' }).click()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Jadwal')
  await expect(page.locator('#lembar-lainnya')).toBeHidden()
})

test('toast tampil di atas dialog modal', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('heading', { name: 'Beranda' }).waitFor()
  const ada = await page.evaluate(() => '__bel' in window)
  test.skip(!ada, 'build tanpa VITE_E2E')

  await page.evaluate(() => {
    void (window as any).__bel.konfirmasi('Hapus contoh', 'Yakin?')
  })
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.evaluate(() => (window as any).__bel.beriTahu('Halo dari toast'))
  const toast = page.getByText('Halo dari toast')
  await expect(toast).toBeVisible()
  // Dialog modal membuat sisa halaman inert (elementFromPoint menembus toast), jadi urutan
  // tumpukan diperiksa lewat warna piksel: toast berwarna emerald-700 penuh (0,122,85),
  // bukan gelap tertutup backdrop.
  const r = (await toast.evaluate((el) => {
    const k = el.parentElement!.getBoundingClientRect()
    return { x: k.x + 8, y: k.y + k.height / 2 }
  }))
  const png = await page.screenshot({ clip: { x: Math.floor(r.x), y: Math.floor(r.y), width: 1, height: 1 } })
  const dataPng = Buffer.concat(idat(png))
  const mentah = inflateSync(dataPng)
  const [merah, hijau, biru] = [mentah[1], mentah[2], mentah[3]]
  expect({ merah, hijau, biru }).toEqual({ merah: 0, hijau: 122, biru: 85 })
  // tombol tutup toast & X modal berukuran minimal 44px
  const kotak = await page.getByRole('button', { name: 'Tutup notifikasi' }).boundingBox()
  expect(kotak!.width).toBeGreaterThanOrEqual(44)
  expect(kotak!.height).toBeGreaterThanOrEqual(44)
  const x = await page.getByRole('button', { name: 'Tutup', exact: true }).boundingBox()
  expect(x!.width).toBeGreaterThanOrEqual(44)
  expect(x!.height).toBeGreaterThanOrEqual(44)
})
