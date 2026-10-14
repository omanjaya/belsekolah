import { expect, test, type APIRequestContext } from '@playwright/test'
import { unik } from './bantu'

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

/** Hapus profil yang namanya diawali awalan (bersih-bersih tes). */
async function hapusProfilBernama(request: APIRequestContext, ...nama: string[]) {
  for (const p of await daftarProfil(request)) {
    if (nama.includes(p.nama)) await request.delete(`/api/profil/${p.id}`)
  }
}

test('duplikat profil default menghasilkan salinan dengan jumlah bel sama', async ({ page, request }) => {
  const asal = await profilDefaultApi(request)
  const salinan = `${asal.nama} (salinan)`
  await hapusProfilBernama(request, salinan)
  try {
    await page.goto('/profil')
    await page.getByRole('button', { name: `Duplikat ${asal.nama}`, exact: true }).click()
    const panel = page.getByRole('dialog')
    await expect(panel.getByLabel('Nama salinan')).toHaveValue(salinan)
    await panel.getByRole('button', { name: 'Duplikat', exact: true }).click()

    const baris = page.getByRole('row').filter({ hasText: salinan })
    await expect(baris).toBeVisible()
    await expect(baris.getByRole('cell').nth(1)).toHaveText(String(asal.jumlahBel))
  } finally {
    await hapusProfilBernama(request, salinan)
  }
})

test('ekspor profil lalu impor file yang sama', async ({ page, request }, testInfo) => {
  const asal = await profilDefaultApi(request)
  const hasilImpor = `${asal.nama} (impor)`
  await hapusProfilBernama(request, hasilImpor)
  try {
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
  } finally {
    await hapusProfilBernama(request, hasilImpor)
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

    await page.goto('/jenis-bel')
    await expect(page.getByText(jenis).first()).toBeVisible()
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
