import { expect, test } from '@playwright/test'
import { profilDefault, jenisBelId } from './bantu'

test('jam, profil, dan bel berikutnya tampil', async ({ page, request }) => {
  // Buat jadwal 23:59 hari ini agar selalu ada "bel berikutnya" (kecuali tepat 23:59).
  const status = await (await request.get('/api/status')).json()
  const profil = await profilDefault(request)
  const masuk = await jenisBelId(request, 'Masuk')
  await request.post('/api/jadwal', { data: { profilId: profil, hari: status.hari, jam: '23:59', jenisBelId: masuk, label: 'Bel Uji Malam' } })

  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Beranda' })).toBeVisible()
  await expect(page.getByText(/^\d{2}:\d{2}:\d{2}$/).first()).toBeVisible()
  await expect(page.getByText('Profil: Normal')).toBeVisible()
  await expect(page.getByText('Bel Uji Malam').first()).toBeVisible()
  await expect(page.getByText(/^dalam /)).toBeVisible()
})

test('bunyikan bel manual dan hentikan', async ({ page, request }) => {
  await page.goto('/')
  await page.getByLabel('Jenis bel').selectOption({ label: 'Istirahat' })
  await page.getByRole('button', { name: 'Bunyikan', exact: true }).click()
  await expect(page.getByText('Bel dibunyikan')).toBeVisible()
  await page.getByRole('button', { name: 'Hentikan suara' }).click()
  await expect(page.getByText('Suara dihentikan')).toBeVisible()

  const riwayat = await (await request.get('/api/riwayat?batas=20')).json()
  expect(riwayat.some((r: { label: string }) => r.label === 'Manual: Istirahat')).toBe(true)
})

test('bel darurat harus ditahan 2 detik', async ({ page }) => {
  await page.goto('/')
  const tombol = page.getByRole('button', { name: 'Tahan 2 detik untuk bel darurat' })

  await tombol.hover()
  await page.mouse.down()
  await page.waitForTimeout(500)
  await page.mouse.up()
  await page.waitForTimeout(300)
  await expect(page.getByText('Bel darurat dibunyikan')).toHaveCount(0)

  await tombol.hover()
  await page.mouse.down()
  await page.waitForTimeout(2300)
  await page.mouse.up()
  await expect(page.getByText('Bel darurat dibunyikan')).toBeVisible()
  await page.getByRole('button', { name: 'Hentikan suara' }).click()
})

test('banner libur', async ({ page, request }) => {
  const status = await (await request.get('/api/status')).json()
  const res = await request.post('/api/libur', { data: { mulai: status.tanggal, selesai: status.tanggal, keterangan: 'Libur Uji' } })
  const libur = await res.json()
  try {
    await expect.poll(async () => (await (await request.get('/api/status')).json()).libur?.keterangan, { timeout: 5000 }).toBe('Libur Uji')
    await page.goto('/')
    await expect(page.getByText('Hari libur: Libur Uji')).toBeVisible()
  } finally {
    await request.delete(`/api/libur/${libur.id}`)
  }
})
