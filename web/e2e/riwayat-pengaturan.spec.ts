import { expect, test } from '@playwright/test'
import { PASSWORD_E2E, jenisBelId } from './bantu'

test('riwayat menampilkan bel manual dan filter status', async ({ page, request }) => {
  await request.post('/api/bel/manual', { data: { jenisBelId: await jenisBelId(request, 'Pulang') } })
  await expect.poll(async () => {
    const r = await (await request.get('/api/riwayat?batas=20')).json()
    return r.some((x: { label: string }) => x.label === 'Manual: Pulang')
  }, { timeout: 20_000 }).toBe(true)

  await page.goto('/riwayat')
  await expect(page.getByRole('listitem').filter({ hasText: 'Manual: Pulang' }).first()).toBeVisible()
  await expect(page.getByText('Manual', { exact: true }).first()).toBeVisible()

  await page.getByLabel('Status').selectOption({ label: 'Terlewat' })
  await expect(page.getByRole('listitem').filter({ hasText: 'Manual: Pulang' })).toHaveCount(0)
})

test('simpan volume', async ({ page, request }) => {
  const awal = await (await request.get('/api/pengaturan')).json()
  try {
    await page.goto('/pengaturan')
    await page.getByLabel(/Volume/).fill('65')
    await expect(page.getByText('Volume: 65%')).toBeVisible()
    await page.getByRole('button', { name: 'Simpan pengaturan' }).click()
    await expect(page.getByText('Pengaturan disimpan')).toBeVisible()
    expect((await (await request.get('/api/pengaturan')).json()).volume).toBe(65)
  } finally {
    await request.put('/api/pengaturan', { data: awal })
  }
})

test('ganti password lalu kembalikan', async ({ page, request }) => {
  await page.goto('/pengaturan')
  await page.getByLabel('Password lama').fill('salah-salah')
  await page.getByLabel('Password baru', { exact: true }).fill('password-baru-1')
  await page.getByLabel('Ulangi password baru').fill('password-baru-1')
  await page.getByRole('button', { name: 'Ganti password' }).click()
  await expect(page.getByRole('alert')).toContainText('password lama salah')

  await page.getByLabel('Password lama').fill(PASSWORD_E2E)
  await page.getByRole('button', { name: 'Ganti password' }).click()
  await expect(page.getByText('Password berhasil diganti')).toBeVisible()
  const res = await request.post('/api/password', { data: { lama: 'password-baru-1', baru: PASSWORD_E2E } })
  expect(res.ok()).toBe(true)
})

test('info sistem dan tautan backup', async ({ page }) => {
  await page.goto('/pengaturan')
  await expect(page.getByRole('heading', { name: 'Info sistem' })).toBeVisible()
  await expect(page.getByText('WIB (Asia/Jakarta)')).toBeVisible()
  await expect(page.getByText(/^http:\/\/\d+\.\d+\.\d+\.\d+$/).first()).toBeVisible()
  const unduh = page.waitForEvent('download')
  await page.getByRole('link', { name: 'Unduh backup' }).click()
  expect((await unduh).suggestedFilename()).toMatch(/^bel-sekolah-\d{4}-\d{2}-\d{2}\.zip$/)
})

test('restore backup memulai ulang dan kembali', async ({ page, request }) => {
  test.skip(!process.env.BEL_URL, 'restart otomatis hanya ada di Docker (restart: unless-stopped)')
  const zip = await (await request.get('/api/backup')).body()
  await page.goto('/pengaturan')
  await page.getByLabel('File backup (.zip)').setInputFiles({ name: 'backup.zip', mimeType: 'application/zip', buffer: zip })
  await page.getByRole('button', { name: 'Pulihkan dari backup' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Pulihkan' }).click()
  await expect(page.getByText('Memulai ulang aplikasi…')).toBeVisible()
  // Sesi di memori hilang setelah restart, jadi layar login muncul. (Label 'Password' ambigu dengan form ganti password.)
  await expect(page.getByRole('button', { name: 'Masuk' })).toBeVisible({ timeout: 120_000 })
  // Masuk lagi dan simpan sesi baru agar tes berikutnya (yang memakai e2e/.auth.json) tetap masuk.
  await page.getByLabel('Password').fill(PASSWORD_E2E)
  await page.getByRole('button', { name: 'Masuk' }).click()
  await expect(page.getByRole('heading', { name: 'Pengaturan' })).toBeVisible()
  await page.context().storageState({ path: 'e2e/.auth.json' })
})

test('keluar kembali ke layar login', async ({ browser }) => {
  // Konteks terpisah agar sesi utama tes lain tidak ikut keluar.
  const ctx = await browser.newContext({ storageState: { cookies: [], origins: [] } })
  const page = await ctx.newPage()
  await page.goto('/')
  await page.getByLabel('Password').fill(PASSWORD_E2E)
  await page.getByRole('button', { name: 'Masuk' }).click()
  // Tunggu login selesai; tanpa ini goto membatalkan permintaan login yang masih berjalan.
  await expect(page.getByRole('button', { name: 'Masuk' })).toHaveCount(0)
  await page.goto('/pengaturan')
  await page.getByRole('button', { name: 'Keluar' }).click()
  await expect(page.getByRole('button', { name: 'Masuk' })).toBeVisible()
  await ctx.close()
})
