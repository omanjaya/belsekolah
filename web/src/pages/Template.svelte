<script lang="ts">
  import { onMount } from 'svelte'
  import { Download, Pencil, Play, Trash2 } from '@lucide/svelte'
  import { api, unduh, type FileJadwal, type Template } from '../lib/api'
  import { formatTanggal, NAMA_HARI } from '../lib/format'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import { pindah } from '../lib/router.svelte'
  import { kePratinjau } from '../lib/pembuat'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kartu from '../components/Kartu.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import PanelGeser from '../components/PanelGeser.svelte'
  import Tabel from '../components/Tabel.svelte'
  import BarAlat from '../components/BarAlat.svelte'
  import AlurImpor from '../components/AlurImpor.svelte'
  import JadwalMinggu from '../components/JadwalMinggu.svelte'
  import Medan from '../components/Medan.svelte'
  import FormPembuat from '../components/FormPembuat.svelte'
  import Tombol from '../components/Tombol.svelte'

  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const kelasIkon = 'inline-grid size-11 place-items-center rounded-lg text-slate-700 hover:bg-slate-100'

  // --- Pembuat jadwal ---
  let pratinjau = $state<FileJadwal | null>(null)
  let galatPratinjau = $state('')
  let menghitung = $state(false)
  let hariSekolah = $state<number[]>([1, 2, 3, 4, 5])

  const belPratinjau = $derived(pratinjau ? kePratinjau(pratinjau) : [])

  let bukaBuat = $state(false)
  let dataBuat = $state<FileJadwal | null>(null)

  function buatProfil() {
    if (!pratinjau) return
    dataBuat = pratinjau
    bukaBuat = true
  }

  const keJadwal = (id: number) => pindah(`/jadwal?profil=${id}`)

  // --- Template saya ---
  let memuat = $state(true)
  let daftar = $state<Template[]>([])
  let galatMuat = $state('')

  async function muat() {
    galatMuat = ''
    memuat = true
    try {
      daftar = await api.get<Template[]>('/template')
    } catch (e) {
      galatMuat = e instanceof Error ? e.message : String(e)
    } finally {
      memuat = false
    }
  }
  onMount(muat)

  function ringkasHari(hari: number[]): string {
    const h = [...hari].sort((a, b) => a - b)
    if (h.length === 0) return '—'
    const berurutan = h.every((x, i) => i === 0 || x === h[i - 1] + 1)
    const pendek = (x: number) => NAMA_HARI[x].slice(0, 3)
    return berurutan && h.length > 2 ? `${pendek(h[0])}–${pendek(h[h.length - 1])}` : h.map(pendek).join(', ')
  }

  let inputBerkas = $state<HTMLInputElement>()

  async function pilihBerkas(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const berkas = input.files?.[0]
    if (!berkas) return
    let data: FileJadwal
    try {
      const teks = await berkas.text()
      input.value = ''
      data = JSON.parse(teks) as FileJadwal
    } catch {
      input.value = ''
      beriTahu('File bukan file jadwal', 'galat')
      return
    }
    try {
      await api.post('/template', { nama: data.nama, data })
      beriTahu('Template diimpor')
      await muat()
    } catch (err) {
      galat(err)
    }
  }

  let bukaPakai = $state(false)
  let dataPakai = $state<FileJadwal | null>(null)

  async function pakai(t: Template) {
    try {
      const lengkap = await api.get<Template>(`/template/${t.id}`)
      if (!lengkap.data) throw new Error('Template tidak berisi data jadwal')
      dataPakai = lengkap.data
      bukaPakai = true
    } catch (err) {
      galat(err)
    }
  }

  async function ekspor(t: Template) {
    try {
      await unduh(`/template/${t.id}/ekspor`)
    } catch (err) {
      galat(err)
    }
  }

  let bukaNama = $state(false)
  let ubahId = $state(0)
  let ubahNama = $state('')
  let sibukNama = $state(false)
  let galatNama = $state('')

  function bukaUbahNama(t: Template) {
    ubahId = t.id
    ubahNama = t.nama
    galatNama = ''
    bukaNama = true
  }

  async function simpanNama(e: SubmitEvent) {
    e.preventDefault()
    galatNama = ''
    sibukNama = true
    try {
      await api.put(`/template/${ubahId}`, { nama: ubahNama })
      bukaNama = false
      beriTahu('Nama template disimpan')
      await muat()
    } catch (err) {
      galatNama = err instanceof Error ? err.message : String(err)
    } finally {
      sibukNama = false
    }
  }

  async function hapusTemplate(t: Template) {
    if (!(await konfirmasi('Hapus template?', `Template "${t.nama}" akan dihapus. Profil yang sudah dibuat darinya tidak ikut terhapus.`))) return
    try {
      await api.hapus(`/template/${t.id}`)
      beriTahu('Template dihapus')
      await muat()
    } catch (err) {
      galat(err)
    }
  }
</script>

<JudulHalaman
  judul="Template"
  keterangan="Buat jadwal dari pola bawaan, atau simpan dan pakai ulang jadwal Anda sendiri."
/>

<div class="flex flex-col gap-6">
  <Kartu>
    <h2 class="mb-4 text-lg font-semibold">Pembuat jadwal</h2>
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-[380px_1fr]">
      <FormPembuat bind:hasil={pratinjau} bind:menghitung bind:galat={galatPratinjau} bind:hariSekolah />

      <div class="flex min-w-0 flex-col gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <h3 class="text-base font-semibold">Pratinjau</h3>
          {#if menghitung}<span class="text-sm text-slate-600" role="status">Menghitung…</span>{/if}
          <Tombol class="md:ml-auto" onclick={buatProfil} disabled={!pratinjau || !!galatPratinjau || menghitung}>
            Buat profil dari jadwal ini
          </Tombol>
        </div>
        {#if galatPratinjau}
          <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatPratinjau}</p>
        {/if}
        {#if pratinjau}
          <div class={galatPratinjau ? 'opacity-50' : ''} data-testid="pratinjau">
            <JadwalMinggu bel={belPratinjau} {hariSekolah} padat />
          </div>
        {:else if !galatPratinjau}
          <p class="text-slate-600" role="status">Menyiapkan pratinjau…</p>
        {/if}
      </div>
    </div>
  </Kartu>

  <Kartu>
    <h2 class="mb-3 text-lg font-semibold">Template saya</h2>
    {#if galatMuat}
      <GalatMuat pesan={galatMuat} oncoba={muat} />
    {:else}
      <BarAlat>
        {#snippet kanan()}
          <input
            bind:this={inputBerkas}
            type="file"
            accept=".json,application/json"
            class="hidden"
            aria-label="Berkas template"
            onchange={pilihBerkas}
          />
          <Tombol varian="kedua" onclick={() => inputBerkas?.click()}>Impor template</Tombol>
        {/snippet}
      </BarAlat>
      <Tabel
        label="Daftar template"
        {memuat}
        jumlah={daftar.length}
        kosong="Belum ada template. Simpan profil sebagai template dari halaman Profil & Periode."
      >
        {#snippet kepala()}
          <th scope="col">Nama</th>
          <th scope="col" class="whitespace-nowrap">Jumlah bel</th>
          <th scope="col">Hari</th>
          <th scope="col">Dibuat</th>
          <th scope="col">Aksi</th>
        {/snippet}
        {#each daftar as t (t.id)}
          <tr>
            <td class="break-words font-medium">{t.nama}</td>
            <td>{t.jumlahBel}</td>
            <td class="whitespace-nowrap">{ringkasHari(t.hari)}</td>
            <td class="whitespace-nowrap">{formatTanggal(t.dibuat.slice(0, 10))}</td>
            <td>
              <div class="flex gap-1">
                <button type="button" class={kelasIkon} aria-label="Pakai {t.nama}" title="Pakai" onclick={() => pakai(t)}>
                  <Play class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class={kelasIkon} aria-label="Ekspor {t.nama}" title="Ekspor" onclick={() => ekspor(t)}>
                  <Download class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class={kelasIkon} aria-label="Ganti nama {t.nama}" title="Ganti nama" onclick={() => bukaUbahNama(t)}>
                  <Pencil class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class="{kelasIkon} text-red-700" aria-label="Hapus {t.nama}" title="Hapus" onclick={() => hapusTemplate(t)}>
                  <Trash2 class="size-5" aria-hidden="true" />
                </button>
              </div>
            </td>
          </tr>
        {/each}
      </Tabel>
    {/if}
  </Kartu>
</div>

<PanelGeser bind:buka={bukaNama} judul="Ganti nama template">
  <form id="form-nama-template" class="flex flex-col gap-3" onsubmit={simpanNama}>
    <Medan label="Nama template" untuk="nama-template">
      <input id="nama-template" class={kelasInput} bind:value={ubahNama} required maxlength="60" />
    </Medan>
    {#if galatNama}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatNama}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaNama = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-nama-template" sibuk={sibukNama}>Simpan</Tombol>
  {/snippet}
</PanelGeser>

<AlurImpor bind:buka={bukaBuat} data={dataBuat} judul="Buat profil dari jadwal" onselesai={keJadwal} />
<AlurImpor bind:buka={bukaPakai} data={dataPakai} judul="Pakai template" onselesai={keJadwal} />
