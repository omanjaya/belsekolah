import { writeFileSync } from 'node:fs'
import { expect, test, type APIRequestContext } from '@playwright/test'
import { unik } from './bantu'

async function hapusProfil(request: APIRequestContext, nama: string) {
  const daftar = (await (await request.get('/api/profil')).json()) as { id: number; nama: string }[]
  for (const p of daftar.filter((x) => x.nama === nama)) await request.delete(`/api/profil/${p.id}`)
}

async function hapusTemplate(request: APIRequestContext, nama: string) {
  const daftar = (await (await request.get('/api/template')).json()) as { id: number; nama: string }[]
  for (const t of daftar.filter((x) => x.nama === nama)) await request.delete(`/api/template/${t.id}`)
}

test('pembuat: pratinjau mengikuti isian, 6 hari, dan galat validasi', async ({ page }) => {
  await page.goto('/template')
  await page.getByRole('button', { name: 'SMP', exact: true }).click()
  await page.getByRole('button', { name: 'Normal', exact: true }).click()

  const pratinjau = page.getByTestId('pratinjau')
  const senin = pratinjau.getByRole('region', { name: 'Senin' })
  await expect(senin).toContainText('07:00')
  await expect(senin).toContainText('Upacara')
  await expect(senin).toContainText('12:50')

  await page.getByLabel('Lama 1 JP (menit)').fill('45')
  await expect(senin).not.toContainText('12:50', { timeout: 3000 })

  await expect(pratinjau.getByRole('region', { name: 'Sabtu' })).toHaveCount(0)
  await page.getByRole('button', { name: /6 hari/ }).click()
  await expect(pratinjau.getByRole('region', { name: 'Sabtu' })).toBeVisible()

  await page.getByLabel('Lama 1 JP (menit)').fill('5')
  await expect(page.getByRole('alert').filter({ hasText: 'lama jam pelajaran' })).toBeVisible()
  // pratinjau terakhir yang valid tetap terlihat
  await expect(pratinjau.getByRole('region', { name: 'Sabtu' })).toBeVisible()

  await page.getByRole('button', { name: 'SD', exact: true }).click()
  await expect(page.getByLabel('Lama 1 JP (menit)')).toHaveValue('35')
  await page.getByRole('button', { name: 'Ramadan', exact: true }).click()
  await expect(page.getByLabel('Lama 1 JP (menit)')).toHaveValue('25')
  await expect(page.getByLabel('Jam masuk')).toHaveValue('07:30')
  await page.getByRole('button', { name: 'Ujian', exact: true }).click()
  await expect(page.getByLabel('Jumlah sesi')).toHaveValue('2')
  await expect(page.getByRole('alert').filter({ hasText: 'lama jam pelajaran' })).toHaveCount(0)
})

test('buat profil dari pembuat, lalu simpan, impor, pakai, ganti nama, hapus template', async ({ page, request }, testInfo) => {
  const nama = unik('Alur Template')
  const nama2 = `${nama} dua`
  const nama3 = `${nama} baru`
  try {
    await page.goto('/template')
    await page.getByRole('button', { name: 'SMP', exact: true }).click()
    await page.getByRole('button', { name: 'Normal', exact: true }).click()
    await expect(page.getByTestId('pratinjau').getByRole('region', { name: 'Senin' })).toContainText('Upacara')
    await page.getByRole('button', { name: 'Buat profil dari jadwal ini' }).click()
    const panel = page.getByRole('dialog')
    await expect(panel.getByLabel('Nama profil')).toHaveValue(/^SMP Normal 5 hari/)
    await panel.getByLabel('Nama profil').fill(nama)
    await panel.getByRole('button', { name: 'Impor', exact: true }).click()
    await expect(page).toHaveURL(/\/jadwal\?profil=\d+/)
    const id = new URL(page.url()).searchParams.get('profil')

    // ekspor profil -> impor sebagai template
    const berkas = testInfo.outputPath('profil.json')
    writeFileSync(berkas, await (await request.get(`/api/profil/${id}/ekspor`)).body())
    await page.goto('/template')
    await page.locator('input[type=file]').setInputFiles(berkas)
    const baris = page.getByRole('row').filter({ hasText: nama })
    await expect(baris).toBeVisible()
    await expect(baris).toContainText('Sen–Jum')

    // pakai template
    await baris.getByRole('button', { name: `Pakai ${nama}` }).click()
    await expect(panel.getByLabel('Nama profil')).toBeVisible()
    await panel.getByLabel('Nama profil').fill(nama2)
    await panel.getByRole('button', { name: 'Impor', exact: true }).click()
    await expect(page).toHaveURL(/\/jadwal\?profil=\d+/)
    expect(new URL(page.url()).searchParams.get('profil')).not.toBe(id)

    // ganti nama
    await page.goto('/template')
    await page.getByRole('row').filter({ hasText: nama }).getByRole('button', { name: `Ganti nama ${nama}` }).click()
    await panel.getByLabel('Nama template').fill(nama3)
    await panel.getByRole('button', { name: 'Simpan' }).click()
    await expect(page.getByRole('row').filter({ hasText: nama3 })).toBeVisible()

    // hapus
    await page.getByRole('row').filter({ hasText: nama3 }).getByRole('button', { name: `Hapus ${nama3}` }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Hapus' }).click()
    await expect(page.getByText('Template dihapus')).toBeVisible()
    await expect(page.getByRole('row').filter({ hasText: nama3 })).toHaveCount(0)
  } finally {
    await hapusProfil(request, nama)
    await hapusProfil(request, nama2)
    await hapusTemplate(request, nama)
    await hapusTemplate(request, nama3)
  }
})

test('impor template: file bukan jadwal ditolak', async ({ page }, testInfo) => {
  const berkas = testInfo.outputPath('rusak.json')
  writeFileSync(berkas, 'bukan json')
  await page.goto('/template')
  await page.locator('input[type=file]').setInputFiles(berkas)
  await expect(page.getByText('File bukan file jadwal')).toBeVisible()
})
