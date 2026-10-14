import { expect, test } from '@playwright/test'
import { PASSWORD_E2E } from './bantu'

test.use({ storageState: { cookies: [], origins: [] } })

test('password salah ditolak, password benar masuk', async ({ page }) => {
  await page.goto('/jadwal')
  await page.getByLabel('Password').fill('salah-sekali')
  await page.getByRole('button', { name: 'Masuk' }).click()
  await expect(page.getByRole('alert')).toHaveText('Password salah')

  await page.getByLabel('Password').fill(PASSWORD_E2E)
  await page.getByRole('button', { name: 'Masuk' }).click()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Jadwal')
})
