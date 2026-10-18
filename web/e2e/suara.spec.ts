import { expect, test, type Page } from '@playwright/test'
import { unik } from './bantu'

// Tabel di laptop, daftar kartu di HP; hanya salah satunya yang terlihat.
const barisAtauKartu = (page: Page) => page.getByRole('row').or(page.getByRole('listitem'))

test('unggah, ganti nama, dengarkan, hapus suara', async ({ page, request }) => {
  const wav = await (await request.get('/api/audio/1/file')).body()
  const nama = unik('Mars')

  await page.goto('/suara')
  await page.getByRole('button', { name: 'Unggah suara' }).click()
  const panel = page.getByRole('dialog', { name: 'Unggah suara' })
  await panel.getByLabel('File audio').setInputFiles({ name: 'mars.wav', mimeType: 'audio/wav', buffer: wav })
  await panel.getByLabel('Nama (opsional)').fill(nama)
  await panel.getByRole('button', { name: 'Unggah' }).click()
  await expect(page.getByText('Suara diunggah')).toBeVisible()

  await page.getByRole('tab', { name: 'Unggahan' }).click()
  const baris = barisAtauKartu(page).filter({ hasText: nama })
  await expect(baris).toBeVisible()
  await expect(barisAtauKartu(page).filter({ hasText: 'Ding Dong Panjang' })).toHaveCount(0)

  await baris.getByRole('button', { name: 'Ganti nama' }).click()
  await page.getByRole('dialog', { name: 'Ganti nama suara' }).getByLabel('Nama suara').fill(nama + ' Baru')
  await page.getByRole('dialog', { name: 'Ganti nama suara' }).getByRole('button', { name: 'Simpan' }).click()
  const barisBaru = barisAtauKartu(page).filter({ hasText: nama + ' Baru' })
  await expect(barisBaru).toBeVisible()

  // Putar di browser, lalu suara lain: yang pertama otomatis berhenti.
  await barisBaru.getByRole('button', { name: 'Putar ' + nama + ' Baru' }).click()
  await expect(barisBaru.getByRole('button', { name: 'Berhenti ' + nama + ' Baru' })).toBeVisible()
  const dingDong = barisAtauKartu(page).filter({ hasText: 'Ding Dong Panjang' })
  await page.getByRole('tab', { name: 'Semua' }).click()
  await dingDong.getByRole('button', { name: 'Putar Ding Dong Panjang' }).click()
  await expect(page.getByRole('button', { name: /^Berhenti / })).toHaveCount(1)
  await expect(barisBaru.getByRole('button', { name: 'Putar ' + nama + ' Baru' })).toBeVisible()

  // Tes di speaker menggantikan pemutaran browser dan bisa dihentikan dari bar.
  await barisBaru.getByRole('button', { name: 'Tes di speaker' }).click()
  await expect(barisBaru.getByRole('button', { name: 'Hentikan speaker ' + nama + ' Baru' })).toBeVisible()
  await expect(page.getByRole('button', { name: /^Berhenti / })).toHaveCount(0)
  const bar = page.getByRole('region', { name: 'Suara yang sedang diputar' })
  await expect(bar).toContainText(nama + ' Baru')
  const stop = page.waitForRequest((r) => r.url().endsWith('/api/tes/stop') && r.method() === 'POST')
  await bar.getByRole('button', { name: 'Hentikan' }).click()
  await stop
  await expect(bar).toHaveCount(0)

  await barisBaru.getByRole('button', { name: 'Hapus' }).click()
  await page.getByRole('dialog', { name: 'Hapus suara?' }).getByRole('button', { name: 'Hapus' }).click()
  await expect(barisBaru).toHaveCount(0)
})

test('file bukan audio ditolak', async ({ page }) => {
  await page.goto('/suara')
  await page.getByRole('button', { name: 'Unggah suara' }).click()
  const panel = page.getByRole('dialog', { name: 'Unggah suara' })
  await panel.getByLabel('File audio').setInputFiles({ name: 'palsu.mp3', mimeType: 'audio/mpeg', buffer: Buffer.from('bukan audio') })
  await panel.getByRole('button', { name: 'Unggah' }).click()
  await expect(page.getByText('file bukan audio yang valid')).toBeVisible()
})

test('suara bawaan tidak bisa dihapus atau diganti nama', async ({ page }) => {
  await page.goto('/suara')
  await page.getByRole('tab', { name: 'Bawaan' }).click()
  const baris = barisAtauKartu(page).filter({ hasText: 'Sirene' })
  await expect(baris.getByRole('button', { name: 'Hapus' })).toHaveCount(0)
  await expect(baris.getByRole('button', { name: 'Ganti nama' })).toHaveCount(0)
})

test('jenis bel: tambah dengan urutan, ubah urutan, tes, hapus', async ({ page }) => {
  const nama = unik('Upacara')
  await page.goto('/jenis-bel')
  await page.getByRole('button', { name: 'Tambah jenis bel' }).click()
  const modal = page.getByRole('dialog', { name: /jenis bel/ })
  await modal.getByLabel('Nama').fill(nama)
  await modal.getByRole('radio', { name: 'Jingga' }).click()
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(modal.getByText('Pilih minimal satu suara')).toBeVisible()

  await modal.getByLabel('Tambah suara').selectOption({ label: 'Ding Dong' })
  await modal.getByLabel('Tambah suara').selectOption({ label: 'Sirene' })
  await modal.getByRole('button', { name: 'Naikkan Sirene' }).click()
  await modal.getByRole('button', { name: 'Simpan' }).click()

  const kartu = barisAtauKartu(page).filter({ hasText: nama })
  await expect(kartu).toContainText('Sirene → Ding Dong')

  await kartu.getByRole('button', { name: 'Tes di speaker' }).click()
  await expect(page.getByText('Suara diputar di speaker')).toBeVisible()

  await kartu.getByRole('button', { name: 'Hapus' }).click()
  await page.getByRole('dialog', { name: 'Hapus jenis bel?' }).getByRole('button', { name: 'Hapus' }).click()
  await expect(kartu).toHaveCount(0)
})

test('form TTS: penghitung karakter dan validasi', async ({ page }) => {
  await page.goto('/suara')
  await page.getByRole('button', { name: 'Buat suara TTS' }).click()
  const teks = page.getByLabel('Teks pengumuman')
  await teks.fill('Selamat pagi')
  await expect(page.getByText('12/1000')).toBeVisible()
  await expect(page.getByRole('radio', { name: 'Gadis (perempuan)' })).toBeChecked()
})

test('TTS tanpa edge-tts menampilkan galat yang jelas', async ({ page }) => {
  test.skip(!!process.env.BEL_URL, 'di Docker edge-tts terpasang')
  await page.goto('/suara')
  await page.getByRole('button', { name: 'Buat suara TTS' }).click()
  await page.getByLabel('Teks pengumuman').fill('Tes')
  await page.getByRole('button', { name: 'Dengarkan pratinjau' }).click()
  await expect(page.getByText(/edge-tts belum terpasang|terhubung ke internet/)).toBeVisible()
})

test('jenis bel: nama kembar menampilkan galat di dalam dialog', async ({ page }) => {
  await page.goto('/jenis-bel')
  await page.getByRole('button', { name: 'Tambah jenis bel' }).click()
  const modal = page.getByRole('dialog', { name: /jenis bel/ })
  await modal.getByLabel('Nama').fill('Masuk')
  await modal.getByLabel('Tambah suara').selectOption({ label: 'Ding Dong' })
  await modal.getByRole('button', { name: 'Simpan' }).click()
  await expect(modal.getByRole('alert').filter({ hasText: 'sudah ada' })).toBeVisible()
})

test('lebar 1440x900: tabel jenis bel dan pustaka suara memenuhi lebar', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 })
  for (const [url, label] of [['/jenis-bel', 'Daftar jenis bel'], ['/suara', 'Daftar suara']]) {
    await page.goto(url)
    const tabel = page.getByRole('table', { name: label })
    await expect(tabel.getByRole('columnheader', { name: 'Nama' })).toBeVisible()
    expect((await tabel.boundingBox())!.width).toBeGreaterThanOrEqual(1100)
  }
})
