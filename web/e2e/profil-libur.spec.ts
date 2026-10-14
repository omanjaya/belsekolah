import { expect, test } from '@playwright/test'
import { unik } from './bantu'

test('profil: tambah, ubah nama, jadikan default, hapus', async ({ page, request }) => {
  const awal = await (await request.get('/api/pengaturan')).json()
  const nama = unik('Ujian')
  try {
    await page.goto('/profil')
    await page.getByRole('button', { name: '+ Profil' }).click()
    await page.getByRole('dialog').getByLabel('Nama profil baru').fill(nama)
    await page.getByRole('dialog').getByRole('button', { name: 'Simpan' }).click()
    await expect(page.getByText('Profil ditambahkan')).toBeVisible()

    const baris = page.getByRole('row').filter({ hasText: nama })
    await baris.getByRole('button', { name: 'Ubah nama' }).click()
    await page.getByRole('dialog').getByLabel('Nama profil').fill(nama + ' Akhir')
    await page.getByRole('dialog').getByRole('button', { name: 'Simpan' }).click()
    await expect(page.getByText(nama + ' Akhir')).toBeVisible()

    const barisBaru = page.getByRole('row').filter({ hasText: nama + ' Akhir' })
    await barisBaru.getByRole('button', { name: 'Jadikan default' }).click()
    await expect(page.getByText('Profil default diubah')).toBeVisible()
    await expect(barisBaru.getByText('Default')).toBeVisible()

    await barisBaru.getByRole('button', { name: 'Hapus' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Hapus' }).click()
    await expect(page.getByText('profil default tidak bisa dihapus')).toBeVisible()
  } finally {
    await request.put('/api/pengaturan', { data: awal })
  }
})

test('periode khusus: tambah, tumpang tindih ditolak, hapus', async ({ page }) => {
  await page.goto('/profil')
  await page.getByRole('button', { name: 'Tambah periode' }).click()
  const modal = page.getByRole('dialog')
  await modal.getByLabel('Tanggal mulai').fill('2031-03-02')
  await modal.getByLabel('Tanggal selesai').fill('2031-03-06')
  await modal.getByLabel('Profil').selectOption({ label: 'Normal' })
  await modal.getByLabel('Keterangan').fill('PTS Uji')
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(page.getByText('2 Maret 2031 – 6 Maret 2031')).toBeVisible()

  await page.getByRole('button', { name: 'Tambah periode' }).click()
  await modal.getByLabel('Tanggal mulai').fill('2031-03-05')
  await modal.getByLabel('Tanggal selesai').fill('2031-03-09')
  await modal.getByLabel('Profil').selectOption({ label: 'Normal' })
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(modal.getByRole('alert')).toContainText('bertabrakan')
  await modal.getByRole('button', { name: 'Tutup' }).click()

  const baris = page.getByRole('row').filter({ hasText: 'PTS Uji' })
  await baris.getByRole('button', { name: /Hapus/ }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Hapus' }).click()
  await expect(page.getByText('PTS Uji')).toHaveCount(0)
})

test('libur satu hari dan rentang', async ({ page }) => {
  await page.goto('/libur')
  await page.getByRole('button', { name: 'Tambah libur' }).click()
  const modal = page.getByRole('dialog')
  await expect(modal.getByLabel('Hanya satu hari')).toBeChecked()
  await modal.getByLabel('Tanggal mulai').fill('2031-12-25')
  await modal.getByLabel('Keterangan').fill('Natal Uji')
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(page.getByText('Libur disimpan')).toBeVisible()
  await expect(page.getByText('25 Desember 2031')).toBeVisible()

  await page.getByRole('button', { name: 'Tambah libur' }).click()
  await modal.getByLabel('Hanya satu hari').uncheck()
  await modal.getByLabel('Tanggal mulai').fill('2031-12-29')
  await modal.getByLabel('Tanggal selesai').fill('2032-01-02')
  await modal.getByLabel('Keterangan').fill('Akhir Tahun Uji')
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(page.getByText('29 Desember 2031 – 2 Januari 2032')).toBeVisible()

  for (const ket of ['Natal Uji', 'Akhir Tahun Uji']) {
    await page.getByRole('button', { name: `Hapus ${ket}` }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Hapus' }).click()
    await expect(page.getByText(ket)).toHaveCount(0)
  }
})

test('ubah nama profil ganda menampilkan galat di dalam dialog', async ({ page }) => {
  await page.goto('/profil')
  await page.getByRole('button', { name: /^Ubah nama/ }).first().click()
  const modal = page.getByRole('dialog')
  const nama = await modal.getByLabel('Nama profil').inputValue()
  await page.getByRole('button', { name: 'Tutup' }).click()
  await page.getByRole('button', { name: '+ Profil' }).click()
  await modal.getByLabel('Nama profil baru').fill(nama + ' Dua')
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(page.getByText('Profil ditambahkan')).toBeVisible()
  const baris = page.getByRole('row').filter({ hasText: nama + ' Dua' })
  await baris.getByRole('button', { name: 'Ubah nama' }).click()
  await modal.getByLabel('Nama profil').fill(nama)
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(modal.getByRole('alert')).toContainText('sudah ada')
  await modal.getByRole('button', { name: 'Tutup' }).click()
  await baris.getByRole('button', { name: /^Hapus/ }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Hapus' }).click()
  await expect(page.getByText(nama + ' Dua')).toHaveCount(0)
})
