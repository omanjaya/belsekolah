import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { jenisBelId, unik } from './bantu'

interface BelApi { id: number; hari: number; jam: string; label: string; pengumumanId: number | null }

/** WAV PCM 16-bit mono 8 kHz, hening 0,3 detik. */
function wavHening(): Buffer {
  const n = Math.round(8000 * 0.3)
  const data = n * 2
  const b = Buffer.alloc(44 + data)
  b.write('RIFF', 0)
  b.writeUInt32LE(36 + data, 4)
  b.write('WAVEfmt ', 8)
  b.writeUInt32LE(16, 16)
  b.writeUInt16LE(1, 20)
  b.writeUInt16LE(1, 22)
  b.writeUInt32LE(8000, 24)
  b.writeUInt32LE(16000, 28)
  b.writeUInt16LE(2, 32)
  b.writeUInt16LE(16, 34)
  b.write('data', 36)
  b.writeUInt32LE(data, 40)
  return b
}

async function unggahSuara(request: APIRequestContext, nama: string): Promise<number> {
  const r = await request.post('/api/audio', {
    multipart: { nama, file: { name: 'x.wav', mimeType: 'audio/wav', buffer: wavHening() } },
  })
  expect(r.ok(), await r.text()).toBeTruthy()
  return ((await r.json()) as { id: number }).id
}

async function buatProfil(request: APIRequestContext, nama: string): Promise<number> {
  expect((await request.post('/api/profil', { data: { nama } })).ok()).toBeTruthy()
  const daftar = (await (await request.get('/api/profil')).json()) as { id: number; nama: string }[]
  const p = daftar.find((x) => x.nama === nama)
  if (!p) throw new Error('profil uji tidak ditemukan')
  return p.id
}

async function tambahBel(request: APIRequestContext, profilId: number, jam: string, label: string) {
  const jenis = await jenisBelId(request, 'Masuk')
  const r = await request.post('/api/jadwal', { data: { profilId, hari: 1, jam, jenisBelId: jenis, label } })
  expect(r.ok()).toBeTruthy()
}

async function daftarBel(request: APIRequestContext, profilId: number): Promise<BelApi[]> {
  return (await (await request.get(`/api/jadwal?profil=${profilId}`)).json()) as BelApi[]
}

async function bukaJadwalHari(page: Page, profilId: number) {
  await page.goto(`/jadwal?profil=${profilId}`)
  await expect(page.locator('#jadwal-profil')).toHaveValue(String(profilId))
  await page.getByRole('button', { name: 'Hari', exact: true }).click()
  await page.getByRole('tab', { name: /Senin/ }).click()
}

function sufiks(): string {
  return unik('').trim().replace(/\s+/g, '')
}

test('keterangan mengisi pengumuman otomatis dan tersimpan', async ({ page, request }) => {
  const ket = `Jam ke-3 ${sufiks()}`
  const namaSuara = `${ket} dimulai`
  const suaraId = await unggahSuara(request, namaSuara)
  const profilId = await buatProfil(request, unik('Pgm Otomatis'))
  try {
    await bukaJadwalHari(page, profilId)
    await page.getByRole('button', { name: '+ Tambah bel' }).click()
    const panel = page.getByRole('dialog', { name: 'Tambah bel' })
    await panel.getByLabel('Label', { exact: true }).fill(ket)
    await expect(panel.getByText(namaSuara, { exact: true }).first()).toBeVisible()
    await expect(panel.getByText('Diisi otomatis dari keterangan')).toBeVisible()
    await panel.getByLabel('Jam', { exact: true }).fill('08:30')
    await panel.getByRole('button', { name: 'Simpan' }).click()

    const baris = page.getByRole('row').filter({ hasText: ket })
    await expect(baris).toContainText(namaSuara)
    const bel = await daftarBel(request, profilId)
    expect(bel.find((b) => b.label === ket)?.pengumumanId).toBe(suaraId)
  } finally {
    await request.delete(`/api/profil/${profilId}`)
    await request.delete(`/api/audio/${suaraId}`)
  }
})

test('pilihan manual "Tanpa pengumuman" tidak terisi ulang saat keterangan berubah', async ({ page, request }) => {
  const ket = `Jam ke-3 ${sufiks()}`
  const suaraId = await unggahSuara(request, `${ket} dimulai`)
  const profilId = await buatProfil(request, unik('Pgm Manual'))
  try {
    await bukaJadwalHari(page, profilId)
    await page.getByRole('button', { name: '+ Tambah bel' }).click()
    const panel = page.getByRole('dialog', { name: 'Tambah bel' })
    await panel.getByLabel('Label', { exact: true }).fill(ket)
    await expect(panel.getByText('Diisi otomatis dari keterangan')).toBeVisible()

    await panel.getByRole('option', { name: 'Tanpa pengumuman' }).click()
    await expect(panel.getByText('Diisi otomatis dari keterangan')).toHaveCount(0)
    // Ubah keterangan ke yang cocok lagi; tidak boleh terisi ulang.
    await panel.getByLabel('Label', { exact: true }).fill(ket + ' x')
    await panel.getByLabel('Label', { exact: true }).fill(ket)
    await page.waitForTimeout(800) // lewati debounce 300 ms
    await expect(panel.getByRole('option', { name: 'Tanpa pengumuman' })).toHaveAttribute('aria-selected', 'true')
    await expect(panel.getByText('Diisi otomatis dari keterangan')).toHaveCount(0)

    await panel.getByLabel('Jam', { exact: true }).fill('08:31')
    await panel.getByRole('button', { name: 'Simpan' }).click()
    await expect(page.getByRole('row').filter({ hasText: ket })).toBeVisible()
    const bel = await daftarBel(request, profilId)
    expect(bel.find((b) => b.label === ket)?.pengumumanId ?? null).toBeNull()
  } finally {
    await request.delete(`/api/profil/${profilId}`)
    await request.delete(`/api/audio/${suaraId}`)
  }
})

test('isi pengumuman otomatis: kurangi satu centang, terapkan satu', async ({ page, request }) => {
  const sfx = sufiks()
  const ketA = `Jam ke-3 ${sfx}`
  const ketB = `Jam ke-4 ${sfx}`
  const idA = await unggahSuara(request, `${ketA} dimulai`)
  const idB = await unggahSuara(request, `${ketB} dimulai`)
  const profilId = await buatProfil(request, unik('Pgm Massal'))
  try {
    await tambahBel(request, profilId, '09:00', ketA)
    await tambahBel(request, profilId, '10:00', ketB)
    await bukaJadwalHari(page, profilId)
    await page.getByRole('button', { name: 'Isi pengumuman otomatis' }).click()
    const panel = page.getByRole('dialog', { name: 'Isi pengumuman otomatis' })
    await expect(panel.getByRole('checkbox')).toHaveCount(2)
    await expect(panel.getByRole('button', { name: 'Terapkan (2)' })).toBeVisible()

    await panel.getByRole('checkbox').last().uncheck()
    await panel.getByRole('button', { name: 'Terapkan (1)' }).click()
    await expect(page.getByText('1 bel diberi pengumuman')).toBeVisible()

    const bel = await daftarBel(request, profilId)
    expect(bel.filter((b) => b.pengumumanId !== null)).toHaveLength(1)
    expect(bel.filter((b) => b.pengumumanId === null)).toHaveLength(1)
  } finally {
    await request.delete(`/api/profil/${profilId}`)
    await request.delete(`/api/audio/${idA}`)
    await request.delete(`/api/audio/${idB}`)
  }
})

test('suara yang dipakai bel tidak bisa dihapus (409)', async ({ request }) => {
  const ket = `Jam ke-3 ${sufiks()}`
  const suaraId = await unggahSuara(request, `${ket} dimulai`)
  const profilId = await buatProfil(request, unik('Pgm Hapus'))
  try {
    const jenis = await jenisBelId(request, 'Masuk')
    const r = await request.post('/api/jadwal', { data: { profilId, hari: 1, jam: '11:00', jenisBelId: jenis, label: ket, pengumumanId: suaraId } })
    expect(r.ok(), await r.text()).toBeTruthy()

    const res = await request.delete(`/api/audio/${suaraId}`)
    expect(res.status()).toBe(409)
    expect(await res.text()).toContain('bel di jadwal')
  } finally {
    await request.delete(`/api/profil/${profilId}`)
    await request.delete(`/api/audio/${suaraId}`)
  }
})
