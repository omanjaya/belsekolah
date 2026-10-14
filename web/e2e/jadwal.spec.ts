import { expect, test } from '@playwright/test'
import { unik } from './bantu'

test.describe.configure({ mode: 'serial' })

test('kelola jadwal: tambah, duplikat ditolak, ubah, salin, hapus', async ({ page, request }) => {
  // Profil khusus agar tidak bentrok dengan tes lain.
  const namaProfil = unik('Profil Jadwal')
  const profil = await (await request.post('/api/profil', { data: { nama: namaProfil } })).json()

  await page.goto('/jadwal')
  await page.getByLabel('Profil').selectOption({ label: namaProfil })
  await page.getByRole('button', { name: 'Hari', exact: true }).click()
  await page.getByRole('tab', { name: /Sen/ }).click()
  await expect(page.getByText('Belum ada bel pada hari Senin')).toBeVisible()

  await page.getByRole('button', { name: '+ Tambah bel' }).click()
  const modal = page.getByRole('dialog')
  await modal.getByLabel('Jam').fill('07:00')
  await modal.getByLabel('Jenis bel').selectOption({ label: 'Masuk' })
  await modal.getByLabel('Label').fill('Jam ke-1')
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(page.getByText('Bel disimpan')).toBeVisible()
  await expect(page.getByText('Jam ke-1')).toBeVisible()
  await expect(page.getByRole('tab', { name: /Sen/ })).toContainText('1')

  await page.getByRole('button', { name: '+ Tambah bel' }).click()
  await modal.getByLabel('Jam').fill('07:00')
  await modal.getByLabel('Jenis bel').selectOption({ label: 'Istirahat' })
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(modal.getByRole('alert')).toContainText('sudah ada')
  await modal.getByRole('button', { name: 'Tutup' }).click()

  await page.getByRole('button', { name: 'Ubah 07:00' }).click()
  await modal.getByLabel('Jam').fill('07:15')
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(page.getByText('07:15')).toBeVisible()

  await page.getByRole('tab', { name: /Sel/ }).click()
  await page.getByRole('button', { name: 'Salin hari…' }).click()
  await modal.getByLabel('Salin dari').selectOption({ label: 'Senin' })
  await expect(modal.getByLabel('Salin ke')).toHaveValue('2')
  await modal.getByRole('button', { name: 'Salin' }).click()
  await expect(page.getByText('Jadwal disalin')).toBeVisible()
  await expect(page.getByText('Jam ke-1')).toBeVisible()

  await page.getByRole('button', { name: 'Hapus 07:15' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Hapus' }).click()
  await expect(page.getByText('Bel dihapus')).toBeVisible()
  await expect(page.getByText('Belum ada bel pada hari Selasa')).toBeVisible()

  const sisa = await (await request.get(`/api/jadwal?profil=${profil.id}`)).json()
  expect(sisa).toHaveLength(1)
})

test('tampilan Minggu: bel baru di kolom Senin, klik kartu membuka panel Ubah bel', async ({ page, request }) => {
  const namaProfil = unik('Profil Minggu')
  await request.post('/api/profil', { data: { nama: namaProfil } })

  await page.goto('/jadwal')
  await page.getByLabel('Profil').selectOption({ label: namaProfil })
  await expect(page.getByRole('button', { name: 'Minggu', exact: true })).toHaveAttribute('aria-pressed', 'true')

  await page.getByRole('button', { name: '+ Tambah bel' }).click()
  const panel = page.getByRole('dialog')
  await panel.getByLabel('Hari').selectOption({ label: 'Senin' })
  await panel.getByLabel('Jam').fill('06:45')
  await panel.getByLabel('Label').fill('Apel pagi')
  await panel.getByRole('button', { name: 'Simpan' }).click()
  await expect(page.getByText('Bel disimpan')).toBeVisible()

  const kolomSenin = page.getByRole('region', { name: 'Senin' })
  await expect(kolomSenin).toContainText('06:45')
  await expect(page.getByRole('region', { name: 'Selasa' })).toContainText('Tidak ada bel')

  await kolomSenin.getByRole('button', { name: /06:45/ }).click()
  await expect(page.getByRole('dialog').getByRole('heading', { name: 'Ubah bel' })).toBeVisible()
  await expect(page.getByRole('dialog').getByLabel('Label')).toHaveValue('Apel pagi')
})

test('pengalih tampilan Hari bertahan setelah reload', async ({ page }) => {
  await page.goto('/jadwal')
  await page.getByRole('button', { name: 'Hari', exact: true }).click()
  await expect(page.getByRole('tablist', { name: 'Hari' })).toBeVisible()
  await page.reload()
  await expect(page.getByRole('button', { name: 'Hari', exact: true })).toHaveAttribute('aria-pressed', 'true')
  await expect(page.getByRole('tablist', { name: 'Hari' })).toBeVisible()
})

test('Ekspor mengunduh file jadwal-*.json', async ({ page }) => {
  await page.goto('/jadwal')
  await expect(page.getByRole('button', { name: 'Ekspor' })).toBeEnabled()
  const unduh = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Ekspor' }).click()
  const berkas = await unduh
  expect(berkas.suggestedFilename()).toMatch(/^jadwal-.+\.json$/)
  const path = await berkas.path()
  const { readFileSync } = await import('node:fs')
  expect(JSON.parse(readFileSync(path, 'utf8')).format).toBe('belsekolah-jadwal')
  await expect(page.getByText('Jadwal diekspor')).toBeVisible()
})
