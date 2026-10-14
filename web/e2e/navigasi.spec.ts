import { expect, test } from '@playwright/test'

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
