import { expect, test } from '@playwright/test'
import { unik } from './bantu'

test.describe.configure({ mode: 'serial' })

test('kelola jadwal: tambah, duplikat ditolak, ubah, salin, hapus', async ({ page, request }) => {
  // Profil khusus agar tidak bentrok dengan tes lain.
  const namaProfil = unik('Profil Jadwal')
  const profil = await (await request.post('/api/profil', { data: { nama: namaProfil } })).json()

  await page.goto('/jadwal')
  await page.getByLabel('Profil').selectOption({ label: namaProfil })
  await page.getByRole('tab', { name: /Sen/ }).click()
  await expect(page.getByText('Belum ada bel pada hari Senin')).toBeVisible()

  await page.getByRole('button', { name: 'Tambah bel' }).click()
  const modal = page.getByRole('dialog')
  await modal.getByLabel('Jam').fill('07:00')
  await modal.getByLabel('Jenis bel').selectOption({ label: 'Masuk' })
  await modal.getByLabel('Label').fill('Jam ke-1')
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(page.getByText('Bel disimpan')).toBeVisible()
  await expect(page.getByText('Jam ke-1')).toBeVisible()
  await expect(page.getByRole('tab', { name: /Sen/ })).toContainText('1')

  await page.getByRole('button', { name: 'Tambah bel' }).click()
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
  await page.getByRole('button', { name: 'Salin dari hari lain' }).click()
  await modal.getByLabel('Salin dari').selectOption({ label: 'Senin' })
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
