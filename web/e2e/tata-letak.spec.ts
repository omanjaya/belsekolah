import { expect, test } from '@playwright/test'

const RUTE = ['/', '/jadwal', '/profil', '/template', '/libur', '/jenis-bel', '/suara', '/riwayat', '/pengaturan']

test('laptop: sidebar penuh, isi selebar layar, dan sidebar bisa diciutkan', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto('/riwayat')
  const sidebar = page.getByRole('navigation', { name: 'Menu utama' }).first()
  await expect(sidebar).toBeVisible()
  await expect(sidebar.getByRole('link', { name: 'Riwayat' })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Riwayat')
  await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1)

  const main = await page.locator('main').boundingBox()
  expect(main!.width).toBeGreaterThanOrEqual(1440 - 240 - 20)

  const aside = page.locator('aside')
  expect((await aside.boundingBox())!.width).toBe(240)

  await page.getByRole('button', { name: 'Ciutkan menu' }).click()
  await expect.poll(async () => (await aside.boundingBox())!.width).toBeLessThanOrEqual(80)
  await page.reload()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Riwayat')
  expect((await aside.boundingBox())!.width).toBeLessThanOrEqual(80)

  await page.getByRole('button', { name: 'Perluas menu' }).click()
  await expect.poll(async () => (await aside.boundingBox())!.width).toBe(240)
  await page.reload()
  expect((await aside.boundingBox())!.width).toBe(240)
})

test('jam server di bar atas berformat dan berdetak', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 })
  await page.goto('/')
  const jam = page.getByTestId('jam-server')
  await expect(jam).toHaveText(/^\d{2}:\d{2}:\d{2} (WIB|WITA|WIT)$/)
  const awal = await jam.textContent()
  await expect.poll(async () => jam.textContent(), { timeout: 2500 }).not.toBe(awal)
})

test('HP: laci menu dibuka dengan tombol, Escape menutup, tautan pindah halaman', async ({ page }) => {
  await page.goto('/')
  const buka = page.getByRole('button', { name: 'Buka menu' })
  await buka.click()
  const laci = page.getByRole('navigation', { name: 'Menu utama' }).first()
  await expect(laci.getByRole('link', { name: 'Pengaturan' })).toBeVisible()
  await expect(laci.getByRole('link').first()).toBeFocused()
  await page.keyboard.press('Escape')
  await expect(laci.getByRole('link', { name: 'Pengaturan' })).toBeHidden()
  await expect(buka).toBeFocused()

  await buka.click()
  await laci.getByRole('link', { name: 'Libur' }).click()
  await expect(page).toHaveURL(/\/libur$/)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Libur')
  await expect(page.getByRole('navigation', { name: 'Menu utama' }).first().getByRole('link', { name: 'Pengaturan' })).toBeHidden()
})

test('HP: klik tautan halaman saat ini menutup laci dan fokus kembali ke tombol menu', async ({ page }) => {
  await page.goto('/riwayat')
  const buka = page.getByRole('button', { name: 'Buka menu' })
  await buka.click()
  const laci = page.locator('aside')
  await laci.getByRole('link', { name: 'Pustaka Suara' }).waitFor()
  await laci.getByRole('link', { name: 'Riwayat' }).click()
  await expect(page).toHaveURL(/\/riwayat$/)
  await expect(laci).toBeHidden()
  await expect(buka).toBeFocused()
})

test('tidak ada gulir horizontal di 375px pada setiap rute menu', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 800 })
  for (const rute of RUTE) {
    await page.goto(rute)
    await expect(page.getByRole('heading', { level: 1 })).not.toBeEmpty()
    await page.waitForLoadState('networkidle')
    const lebar = await page.evaluate(() => document.documentElement.scrollWidth)
    expect(lebar, rute).toBeLessThanOrEqual(375)
  }
})
