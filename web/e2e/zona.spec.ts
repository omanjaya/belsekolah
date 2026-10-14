import { expect, test, type Page } from '@playwright/test'
import { PASSWORD_E2E } from './bantu'

test.describe('pilihan zona waktu (lokal)', () => {
  test('tombol Simpan zona hanya aktif bila pilihan berbeda', async ({ page }) => {
    await page.goto('/pengaturan')
    const simpan = page.getByRole('button', { name: 'Simpan zona' })
    await expect(page.getByRole('radio', { name: /^WIB —/ })).toBeChecked()
    await expect(simpan).toBeDisabled()
    await page.getByRole('radio', { name: /^WITA —/ }).check()
    await expect(simpan).toBeEnabled()
    await page.getByRole('radio', { name: /^WIB —/ }).check()
    await expect(simpan).toBeDisabled()
  })
})

test.describe('ganti zona waktu (Docker)', () => {
  test.skip(!process.env.BEL_URL, 'restart otomatis hanya ada di Docker')
  test.describe.configure({ mode: 'serial' })

  async function gantiZona(page: Page, label: RegExp, akhiran: string) {
    await page.goto('/pengaturan')
    await page.getByRole('radio', { name: label }).check()
    await page.getByRole('button', { name: 'Simpan zona' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Ganti zona' }).click()
    await expect(page.getByText('Menerapkan zona waktu…')).toBeVisible()
    // Sesi di memori hilang setelah restart: masuk lagi dan simpan sesi untuk tes berikutnya.
    await expect(page.getByRole('button', { name: 'Masuk' })).toBeVisible({ timeout: 120_000 })
    await page.getByLabel('Password').fill(PASSWORD_E2E)
    await page.getByRole('button', { name: 'Masuk' }).click()
    await expect(page.getByRole('heading', { name: 'Pengaturan' })).toBeVisible()
    await page.context().storageState({ path: 'e2e/.auth.json' })
    await expect(page.getByTestId('jam-server')).toHaveText(new RegExp(`${akhiran}$`))
  }

  test('WITA lalu kembali ke WIB', async ({ page }) => {
    test.setTimeout(300_000)
    await gantiZona(page, /^WITA —/, 'WITA')
    await expect(page.getByText('WITA (Asia/Makassar)')).toBeVisible()
    await gantiZona(page, /^WIB —/, 'WIB')
    await expect(page.getByText('WIB (Asia/Jakarta)')).toBeVisible()
  })
})
