import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { jenisBelId, unik } from './bantu'

interface ProfilApi { id: number; nama: string; jumlahBel: number }

async function daftarProfil(request: APIRequestContext): Promise<ProfilApi[]> {
  return (await (await request.get('/api/profil')).json()) as ProfilApi[]
}

async function profilDefaultApi(request: APIRequestContext): Promise<ProfilApi> {
  const { profilDefaultId } = (await (await request.get('/api/pengaturan')).json()) as { profilDefaultId: number }
  const p = (await daftarProfil(request)).find((x) => x.id === profilDefaultId)
  if (!p) throw new Error('profil default tidak ditemukan')
  return p
}

/** Buat profil baru berisi 2 bel lewat API; mengembalikan profil dengan jumlahBel terbaru. */
async function buatProfilIsi(request: APIRequestContext, nama: string): Promise<ProfilApi> {
  const res = await request.post('/api/profil', { data: { nama } })
  expect(res.ok()).toBeTruthy()
  const p = (await daftarProfil(request)).find((x) => x.nama === nama)
  if (!p) throw new Error('profil uji tidak ditemukan')
  const jenis = await jenisBelId(request, 'Masuk')
  for (const jam of ['07:00', '09:30']) {
    const r = await request.post('/api/jadwal', { data: { profilId: p.id, hari: 1, jam, jenisBelId: jenis, label: `Uji ${jam}` } })
    expect(r.ok()).toBeTruthy()
  }
  const isi = (await daftarProfil(request)).find((x) => x.id === p.id)!
  expect(isi.jumlahBel).toBeGreaterThanOrEqual(2)
  return isi
}

/** Hapus profil yang namanya diawali awalan (bersih-bersih tes). */
async function hapusProfilBernama(request: APIRequestContext, ...nama: string[]) {
  for (const p of await daftarProfil(request)) {
    if (nama.includes(p.nama)) await request.delete(`/api/profil/${p.id}`)
  }
}

/** Setelah redirect ke /jadwal?profil=ID: halaman Jadwal tampil (bukan "tidak ditemukan") dan profil terpilih. */
async function harapJadwalTerpilih(page: Page, id: string | number) {
  await expect(page.getByRole('heading', { level: 1, name: 'Jadwal' })).toBeVisible()
  await expect(page.getByText('Halaman tidak ditemukan')).toHaveCount(0)
  await expect(page.locator('#jadwal-profil')).toHaveValue(String(id))
}

test('duplikat profil menghasilkan salinan dengan jumlah bel sama', async ({ page, request }) => {
  const namaAsal = unik('Asal Duplikat')
  const salinan = `${namaAsal} (salinan)`
  await hapusProfilBernama(request, namaAsal, salinan)
  try {
    const asal = await buatProfilIsi(request, namaAsal)
    await page.goto('/profil')
    await page.getByRole('button', { name: `Duplikat ${asal.nama}`, exact: true }).click()
    const panel = page.getByRole('dialog')
    await expect(panel.getByLabel('Nama salinan')).toHaveValue(salinan)
    await panel.getByRole('button', { name: 'Duplikat', exact: true }).click()

    const baris = page.getByRole('row').filter({ hasText: salinan })
    await expect(baris).toBeVisible()
    await expect(baris.getByRole('cell').nth(1)).toHaveText(String(asal.jumlahBel))
    const hasil = (await daftarProfil(request)).find((p) => p.nama === salinan)
    expect(hasil?.jumlahBel).toBe(asal.jumlahBel)
  } finally {
    await hapusProfilBernama(request, namaAsal, salinan)
  }
})

test('ekspor profil lalu impor file yang sama', async ({ page, request }, testInfo) => {
  const namaAsal = unik('Asal Ekspor')
  const hasilImpor = `${namaAsal} (impor)`
  await hapusProfilBernama(request, namaAsal, hasilImpor)
  try {
    const asal = await buatProfilIsi(request, namaAsal)
    await page.goto('/profil')
    const unduhan = page.waitForEvent('download')
    await page.getByRole('button', { name: `Ekspor ${asal.nama}`, exact: true }).click()
    const berkas = testInfo.outputPath('ekspor.json')
    await (await unduhan).saveAs(berkas)

    await page.locator('input[type=file]').setInputFiles(berkas)
    const panel = page.getByRole('dialog')
    await expect(panel.getByLabel('Nama profil')).toHaveValue(hasilImpor)
    await expect(panel.getByText(`${asal.jumlahBel} bel`)).toBeVisible()
    await panel.getByRole('button', { name: 'Impor', exact: true }).click()

    await expect(page).toHaveURL(/\/jadwal\?profil=\d+/)
    const baru = (await daftarProfil(request)).find((p) => p.nama === hasilImpor)
    expect(baru).toBeTruthy()
    expect(baru!.jumlahBel).toBe(asal.jumlahBel)
    expect(new URL(page.url()).searchParams.get('profil')).toBe(String(baru!.id))
    await harapJadwalTerpilih(page, baru!.id)
  } finally {
    await hapusProfilBernama(request, namaAsal, hasilImpor)
  }
})

test('impor file dengan jenis bel baru lewat pemetaan', async ({ page, request }, testInfo) => {
  const jenis = unik('Bel Literasi Uji')
  const nama = unik('Jadwal Literasi')
  const berkas = testInfo.outputPath('literasi.json')
  const { writeFileSync } = await import('node:fs')
  writeFileSync(berkas, JSON.stringify({
    format: 'belsekolah-jadwal', versi: 1, nama,
    jenisBel: [{ nama: jenis, warna: '#16a34a' }],
    jadwal: [{ hari: 1, jam: '07:00', jenisBel: jenis, label: 'Literasi pagi' }],
  }))
  try {
    await page.goto('/profil')
    await page.locator('input[type=file]').setInputFiles(berkas)
    const panel = page.getByRole('dialog')
    await expect(panel.getByLabel('Nama profil')).toHaveValue(nama)
    const peta = panel.getByLabel(jenis)
    await expect(peta).toBeVisible()
    await expect(peta.locator('option:checked')).toHaveText('Buat jenis bel baru')
    await peta.selectOption({ label: 'Buat jenis bel baru' })
    await panel.getByRole('button', { name: 'Impor', exact: true }).click()
    await expect(page).toHaveURL(/\/jadwal\?profil=\d+/)
    const baru = (await daftarProfil(request)).find((p) => p.nama === nama)
    expect(baru).toBeTruthy()
    await harapJadwalTerpilih(page, baru!.id)

    await page.goto('/jenis-bel')
    await expect(page.getByText(jenis).and(page.locator(':visible')).first()).toBeVisible()
  } finally {
    await hapusProfilBernama(request, nama)
    const daftar = (await (await request.get('/api/jenis-bel')).json()) as { id: number; nama: string }[]
    for (const j of daftar.filter((x) => x.nama === jenis)) await request.delete(`/api/jenis-bel/${j.id}`)
  }
})

test('file rusak menampilkan galat', async ({ page }, testInfo) => {
  const { writeFileSync } = await import('node:fs')
  const bukanJadwal = testInfo.outputPath('rusak.json')
  writeFileSync(bukanJadwal, '{"a":1}')
  const bukanJson = testInfo.outputPath('rusak.txt')
  writeFileSync(bukanJson, 'ini bukan json')

  await page.goto('/profil')
  await page.locator('input[type=file]').setInputFiles(bukanJadwal)
  await expect(page.getByRole('dialog').getByRole('alert')).toContainText('bukan file jadwal')
  await page.getByRole('dialog').getByRole('button', { name: 'Batal' }).click()

  await page.locator('input[type=file]').setInputFiles(bukanJson)
  await expect(page.getByText('File bukan file jadwal')).toBeVisible()
})

test('simpan profil sebagai template', async ({ page, request }) => {
  const asal = await profilDefaultApi(request)
  const namaTemplate = unik('Template Uji')
  try {
    await page.goto('/profil')
    await page.getByRole('button', { name: `Simpan ${asal.nama} sebagai template`, exact: true }).click()
    const panel = page.getByRole('dialog')
    await expect(panel.getByLabel('Nama template')).toHaveValue(asal.nama)
    await panel.getByLabel('Nama template').fill(namaTemplate)
    await panel.getByRole('button', { name: 'Simpan', exact: true }).click()
    await expect(page.getByText('Template disimpan')).toBeVisible()
  } finally {
    const daftar = (await (await request.get('/api/template')).json()) as { id: number; nama: string }[]
    for (const t of daftar.filter((x) => x.nama === namaTemplate)) await request.delete(`/api/template/${t.id}`)
  }
})
