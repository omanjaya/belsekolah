import { expect, test } from '@playwright/test'
import { unik } from './bantu'

test('unggah, ganti nama, dengarkan, hapus suara', async ({ page, request }) => {
  const wav = await (await request.get('/api/audio/1/file')).body()
  const nama = unik('Mars')

  await page.goto('/suara')
  await page.getByLabel('File audio').setInputFiles({ name: 'mars.wav', mimeType: 'audio/wav', buffer: wav })
  await page.getByLabel('Nama (opsional)').first().fill(nama)
  await page.getByRole('button', { name: 'Unggah' }).click()
  await expect(page.getByText('Suara diunggah')).toBeVisible()

  await page.getByRole('tab', { name: 'Unggahan' }).click()
  const baris = page.getByRole('listitem').filter({ hasText: nama })
  await expect(baris).toBeVisible()
  await expect(page.getByRole('listitem').filter({ hasText: 'Ding Dong Panjang' })).toHaveCount(0)

  await baris.getByRole('button', { name: 'Ganti nama' }).click()
  await page.getByRole('dialog').getByLabel('Nama suara').fill(nama + ' Baru')
  await page.getByRole('dialog').getByRole('button', { name: 'Simpan' }).click()
  const barisBaru = page.getByRole('listitem').filter({ hasText: nama + ' Baru' })
  await expect(barisBaru).toBeVisible()

  await barisBaru.getByRole('button', { name: 'Tes di speaker' }).click()
  await expect(page.getByText('Suara diputar di speaker')).toBeVisible()

  await barisBaru.getByRole('button', { name: 'Hapus' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Hapus' }).click()
  await expect(barisBaru).toHaveCount(0)
})

test('file bukan audio ditolak', async ({ page }) => {
  await page.goto('/suara')
  await page.getByLabel('File audio').setInputFiles({ name: 'palsu.mp3', mimeType: 'audio/mpeg', buffer: Buffer.from('bukan audio') })
  await page.getByRole('button', { name: 'Unggah' }).click()
  await expect(page.getByText('file bukan audio yang valid')).toBeVisible()
})

test('suara bawaan tidak bisa dihapus atau diganti nama', async ({ page }) => {
  await page.goto('/suara')
  await page.getByRole('tab', { name: 'Bawaan' }).click()
  const baris = page.getByRole('listitem').filter({ hasText: 'Sirene' })
  await expect(baris.getByRole('button', { name: 'Hapus' })).toHaveCount(0)
  await expect(baris.getByRole('button', { name: 'Ganti nama' })).toHaveCount(0)
})

test('jenis bel: tambah dengan urutan, ubah urutan, tes, hapus', async ({ page }) => {
  const nama = unik('Upacara')
  await page.goto('/jenis-bel')
  await page.getByRole('button', { name: 'Tambah jenis bel' }).click()
  const modal = page.getByRole('dialog')
  await modal.getByLabel('Nama').fill(nama)
  await modal.getByRole('radio', { name: 'Jingga' }).click()
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(modal.getByText('Pilih minimal satu suara')).toBeVisible()

  await modal.getByLabel('Tambah suara').selectOption({ label: 'Ding Dong' })
  await modal.getByLabel('Tambah suara').selectOption({ label: 'Sirene' })
  await modal.getByRole('button', { name: 'Naikkan Sirene' }).click()
  await modal.getByRole('button', { name: 'Simpan' }).click()

  const kartu = page.getByRole('listitem').filter({ hasText: nama })
  await expect(kartu).toContainText('1. Sirene')
  await expect(kartu).toContainText('2. Ding Dong')

  await kartu.getByRole('button', { name: 'Tes di speaker' }).click()
  await expect(page.getByText('Suara diputar di speaker')).toBeVisible()

  await kartu.getByRole('button', { name: 'Hapus' }).click()
  await page.getByRole('dialog').getByRole('button', { name: 'Hapus' }).click()
  await expect(kartu).toHaveCount(0)
})

test('form TTS: penghitung karakter dan validasi', async ({ page }) => {
  await page.goto('/suara')
  const teks = page.getByLabel('Teks pengumuman')
  await teks.fill('Selamat pagi')
  await expect(page.getByText('12/1000')).toBeVisible()
  await expect(page.getByRole('radio', { name: 'Gadis (perempuan)' })).toBeChecked()
})

test('TTS tanpa edge-tts menampilkan galat yang jelas', async ({ page }) => {
  test.skip(!!process.env.BEL_URL, 'di Docker edge-tts terpasang')
  await page.goto('/suara')
  await page.getByLabel('Teks pengumuman').fill('Tes')
  await page.getByRole('button', { name: 'Dengarkan pratinjau' }).click()
  await expect(page.getByText(/edge-tts belum terpasang|terhubung ke internet/)).toBeVisible()
})
