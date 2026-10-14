import { expect, test as siapkan } from '@playwright/test'
import { PASSWORD_E2E, passwordAwal } from './bantu'

siapkan('login dan ganti password awal', async ({ page }) => {
  await page.goto('/')
  const awal = passwordAwal()
  await page.getByLabel('Password').fill(awal ?? PASSWORD_E2E)
  await page.getByRole('button', { name: 'Masuk' }).click()

  const judulGanti = page.getByRole('heading', { name: 'Ganti password awal' })
  const beranda = page.getByRole('heading', { name: 'Beranda' })
  await expect(judulGanti.or(beranda)).toBeVisible()
  if (await judulGanti.isVisible()) {
    await page.getByLabel('Password awal').fill(awal!)
    await page.getByLabel('Password baru', { exact: true }).fill(PASSWORD_E2E)
    await page.getByLabel('Ulangi password baru').fill(PASSWORD_E2E)
    await page.getByRole('button', { name: 'Simpan password' }).click()
  }
  await expect(beranda).toBeVisible()
  await page.context().storageState({ path: 'e2e/.auth.json' })
})
