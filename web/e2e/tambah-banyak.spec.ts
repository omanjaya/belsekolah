import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { jenisBelId, unik } from './bantu'

interface ProfilApi { id: number; nama: string }
interface BelApi { hari: number; jam: string; label: string; jenisBelId: number }

async function buatProfil(request: APIRequestContext, nama: string): Promise<number> {
  expect((await request.post('/api/profil', { data: { nama } })).ok()).toBeTruthy()
  const daftar = (await (await request.get('/api/profil')).json()) as ProfilApi[]
  const p = daftar.find((x) => x.nama === nama)
  if (!p) throw new Error('profil uji tidak ditemukan')
  return p.id
}

async function hapusProfil(request: APIRequestContext, ...id: number[]) {
  for (const i of id) await request.delete(`/api/profil/${i}`)
}

async function tambahBel(request: APIRequestContext, profilId: number, hari: number, jam: string, label: string) {
  const jenis = await jenisBelId(request, 'Masuk')
  const r = await request.post('/api/jadwal', { data: { profilId, hari, jam, jenisBelId: jenis, label } })
  expect(r.ok()).toBeTruthy()
}

async function daftarBel(request: APIRequestContext, profilId: number): Promise<BelApi[]> {
  return (await (await request.get(`/api/jadwal?profil=${profilId}`)).json()) as BelApi[]
}

/** Buka panel Tambah banyak, tempel dua bel, hanya hari Senin, lalu Periksa. */
async function tempelDanPeriksa(page: Page, profilId: number) {
  await page.goto(`/jadwal?profil=${profilId}`)
  await expect(page.locator('#jadwal-profil')).toHaveValue(String(profilId))
  await page.getByRole('button', { name: 'Tambah banyak', exact: true }).click()
  const panel = page.getByRole('dialog')
  await panel.locator('#tb-tempel').fill('07:00\tUpacara\n07:15\tSholat Dhuha')
  await panel.getByRole('button', { name: 'Tambahkan ke daftar' }).click()
  for (const h of ['Selasa', 'Rabu', 'Kamis', 'Jumat']) await panel.getByLabel(h, { exact: true }).uncheck()
  await panel.getByRole('button', { name: 'Periksa', exact: true }).click()
  return panel
}

test('tempel: bentrok ditimpa; pratinjau tampil sebagai kartu', async ({ page, request }) => {
  const id = await buatProfil(request, unik('TB Timpa'))
  try {
    await tambahBel(request, id, 1, '07:00', 'Masuk')
    const panel = await tempelDanPeriksa(page, id)
    await expect(panel.getByText('Baru', { exact: true }).first()).toBeVisible()
    await expect(panel.getByText('Bentrok', { exact: true }).first()).toBeVisible()
    // HP: kartu, bukan tabel.
    await expect(panel.getByRole('table')).toHaveCount(0)
    await expect(panel.getByRole('listitem').filter({ hasText: 'Bentrok' })).toHaveCount(1)
    await expect(panel.getByRole('listitem')).toHaveCount(2)

    await panel.getByLabel('Tindakan Senin 07:00').selectOption('timpa')
    await panel.getByRole('button', { name: 'Gabungkan (2 perubahan)' }).click()
    await page.getByRole('dialog').filter({ hasText: 'Timpa bel lama?' }).getByRole('button', { name: 'Timpa', exact: true }).click()
    await expect(page.getByText('1 ditambah, 1 ditimpa, 0 dilewati')).toBeVisible()

    const bel = await daftarBel(request, id)
    expect(bel.find((b) => b.hari === 1 && b.jam === '07:00')?.label).toBe('Upacara')
    expect(bel.some((b) => b.hari === 1 && b.jam === '07:15')).toBeTruthy()
  } finally {
    await hapusProfil(request, id)
  }
})

test('tempel: bawaan mempertahankan bel lama tanpa konfirmasi', async ({ page, request }) => {
  const id = await buatProfil(request, unik('TB Tahan'))
  try {
    await tambahBel(request, id, 1, '07:00', 'Masuk')
    const panel = await tempelDanPeriksa(page, id)
    await panel.getByRole('button', { name: 'Gabungkan (1 perubahan)' }).click()
    await expect(page.getByText('1 ditambah, 0 ditimpa, 1 dilewati')).toBeVisible()
    await expect(page.getByText('Timpa bel lama?')).toHaveCount(0)

    const bel = await daftarBel(request, id)
    expect(bel.find((b) => b.hari === 1 && b.jam === '07:00')?.label).toBe('Masuk')
    expect(bel.some((b) => b.hari === 1 && b.jam === '07:15')).toBeTruthy()
  } finally {
    await hapusProfil(request, id)
  }
})

test('dari profil lain dengan filter hari', async ({ page, request }) => {
  const sumber = await buatProfil(request, unik('TB Sumber'))
  const tujuan = await buatProfil(request, unik('TB Tujuan'))
  try {
    await tambahBel(request, sumber, 1, '08:00', 'Senin delapan')
    await tambahBel(request, sumber, 5, '08:00', 'Jumat delapan')
    await page.goto(`/jadwal?profil=${tujuan}`)
    await expect(page.locator('#jadwal-profil')).toHaveValue(String(tujuan))
    await page.getByRole('button', { name: 'Tambah banyak', exact: true }).click()
    const panel = page.getByRole('dialog')
    await panel.getByRole('tab', { name: 'File/profil lain' }).click()
    await expect(panel.locator(`#tb-profil option[value="${sumber}"]`)).toHaveCount(1)
    await panel.locator('#tb-profil').selectOption(String(sumber))
    await panel.getByLabel('Jumat', { exact: true }).uncheck()
    await panel.getByRole('button', { name: 'Periksa', exact: true }).click()
    await expect(panel.getByRole('button', { name: 'Gabungkan (1 perubahan)' })).toBeVisible()
    await panel.getByRole('button', { name: 'Gabungkan (1 perubahan)' }).click()
    await expect(page.getByText('1 ditambah, 0 ditimpa, 0 dilewati')).toBeVisible()

    const bel = await daftarBel(request, tujuan)
    expect(bel.map((b) => `${b.hari} ${b.jam}`)).toEqual(['1 08:00'])
  } finally {
    await hapusProfil(request, sumber, tujuan)
  }
})

test('jenis tak dikenal dipetakan ke jenis yang ada', async ({ page, request }) => {
  const id = await buatProfil(request, unik('TB Peta'))
  const jenisBaru = unik('Bel Literasi Uji')
  try {
    await page.goto(`/jadwal?profil=${id}`)
    await expect(page.locator('#jadwal-profil')).toHaveValue(String(id))
    await page.getByRole('button', { name: 'Tambah banyak', exact: true }).click()
    const panel = page.getByRole('dialog')
    await panel.locator('#tb-tempel').fill(`09:00\tLiterasi\t${jenisBaru}`)
    await panel.getByRole('button', { name: 'Tambahkan ke daftar' }).click()
    for (const h of ['Selasa', 'Rabu', 'Kamis', 'Jumat']) await panel.getByLabel(h, { exact: true }).uncheck()
    await panel.getByRole('button', { name: 'Periksa', exact: true }).click()

    const peta = panel.locator('#tb-peta-0')
    await expect(peta).toBeVisible()
    // Langkah Sumber tetap ter-mount (tersembunyi), jadi batasi ke label pemetaan.
    await expect(panel.locator('label[for="tb-peta-0"]')).toHaveText(jenisBaru)
    await expect(peta.locator('option:checked')).toHaveText('Buat jenis bel baru')
    await peta.selectOption({ label: 'Masuk' })
    await panel.getByRole('button', { name: 'Lanjut', exact: true }).click()
    await expect(panel.getByText('Baru', { exact: true }).first()).toBeVisible()
    await panel.getByRole('button', { name: 'Gabungkan (1 perubahan)' }).click()
    await expect(page.getByText('1 ditambah, 0 ditimpa, 0 dilewati')).toBeVisible()

    const masuk = await jenisBelId(request, 'Masuk')
    const bel = await daftarBel(request, id)
    expect(bel.find((b) => b.hari === 1 && b.jam === '09:00')?.jenisBelId).toBe(masuk)
    const jenis = (await (await request.get('/api/jenis-bel')).json()) as { nama: string }[]
    expect(jenis.some((j) => j.nama === jenisBaru)).toBeFalsy()
  } finally {
    await hapusProfil(request, id)
  }
})

test('pembuat: parameter tidak valid menampilkan galat dan Periksa nonaktif', async ({ page, request }) => {
  const id = await buatProfil(request, unik('TB Galat'))
  try {
    await page.goto(`/jadwal?profil=${id}`)
    await expect(page.locator('#jadwal-profil')).toHaveValue(String(id))
    await page.getByRole('button', { name: 'Tambah banyak', exact: true }).click()
    const panel = page.getByRole('dialog')
    await panel.getByRole('tab', { name: 'Template' }).click()
    await expect(panel.getByTestId('ringkasan-sumber')).toContainText(/\d+ bel dari/)
    await panel.locator('#pb-jp').fill('')
    await expect(panel.getByRole('alert').filter({ hasText: /\S/ })).toBeVisible()
    await expect(panel.getByTestId('ringkasan-sumber')).toHaveText('Belum ada bel')
    await expect(panel.getByRole('button', { name: 'Periksa', exact: true })).toBeDisabled()
  } finally {
    await hapusProfil(request, id)
  }
})

test('Kembali dari pratinjau mempertahankan isian tab Ketik', async ({ page, request }) => {
  const id = await buatProfil(request, unik('TB Kembali'))
  try {
    const panel = await tempelDanPeriksa(page, id)
    await expect(panel.getByRole('button', { name: 'Gabungkan (2 perubahan)' })).toBeVisible()
    await panel.getByRole('button', { name: 'Kembali', exact: true }).click()
    await expect(panel.getByLabel('Jam baris 1')).toHaveValue('07:00')
    await expect(panel.getByLabel('Keterangan baris 2')).toHaveValue('Sholat Dhuha')
    await expect(panel.getByLabel('Senin', { exact: true })).toBeChecked()
    await expect(panel.getByLabel('Selasa', { exact: true })).not.toBeChecked()
    await expect(panel.getByRole('button', { name: 'Periksa', exact: true })).toBeEnabled()
  } finally {
    await hapusProfil(request, id)
  }
})

test('radio global mereset pilihan per baris yang campuran', async ({ page, request }) => {
  const id = await buatProfil(request, unik('TB Radio'))
  try {
    await tambahBel(request, id, 1, '07:00', 'Masuk')
    await tambahBel(request, id, 1, '07:15', 'Masuk')
    const panel = await tempelDanPeriksa(page, id)
    const a = panel.getByLabel('Tindakan Senin 07:00')
    const b = panel.getByLabel('Tindakan Senin 07:15')
    await a.selectOption('timpa')
    await expect(panel.getByRole('radio', { name: 'Pertahankan yang lama' })).not.toBeChecked()
    await panel.getByRole('radio', { name: 'Pertahankan yang lama' }).click()
    await expect(a).toHaveValue('pertahankan')
    await expect(b).toHaveValue('pertahankan')
    await b.selectOption('timpa')
    await panel.getByRole('radio', { name: 'Timpa dengan yang baru' }).click()
    await expect(a).toHaveValue('timpa')
    await expect(b).toHaveValue('timpa')
  } finally {
    await hapusProfil(request, id)
  }
})
