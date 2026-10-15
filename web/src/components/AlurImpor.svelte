<script lang="ts">
  import { untrack } from 'svelte'
  import { api, type FileJadwal, type HasilPeriksa, type JenisBel, type Pemetaan } from '../lib/api'
  import { beriTahu } from '../lib/notif.svelte'
  import PanelGeser from './PanelGeser.svelte'
  import Medan from './Medan.svelte'
  import Tombol from './Tombol.svelte'

  let { buka = $bindable(false), data, namaAwal, judul = 'Impor jadwal', onselesai }: {
    buka?: boolean
    data: FileJadwal | null
    namaAwal?: string
    judul?: string
    onselesai: (profilId: number) => void
  } = $props()

  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const BARU = 'baru'

  let memeriksa = $state(false)
  let hasil = $state<HasilPeriksa | null>(null)
  let jenisBel = $state<JenisBel[]>([])
  let nama = $state('')
  // nama jenis bel tak dikenal -> 'baru' | id jenis bel (string)
  let pilihan = $state<Record<string, string>>({})
  let pesanGalat = $state('')
  let sibuk = $state(false)
  let urutan = 0

  $effect(() => {
    if (!(buka && data)) return
    const d = data
    untrack(() => periksa(d))
  })

  async function periksa(d: FileJadwal) {
    const saya = ++urutan
    memeriksa = true
    hasil = null
    pesanGalat = ''
    try {
      const [h, jb] = await Promise.all([
        api.post<HasilPeriksa>('/impor/periksa', d),
        api.get<JenisBel[]>('/jenis-bel'),
      ])
      if (saya !== urutan) return
      hasil = h
      jenisBel = jb
      nama = namaAwal ?? h.namaTersedia
      pilihan = Object.fromEntries(h.jenisBelTakDikenal.map((n) => [n, BARU]))
    } catch (e) {
      if (saya !== urutan) return
      pesanGalat = e instanceof Error ? e.message : String(e)
    } finally {
      if (saya === urutan) memeriksa = false
    }
  }

  async function impor(e: SubmitEvent) {
    e.preventDefault()
    if (!data || !hasil) return
    pesanGalat = ''
    sibuk = true
    const pemetaan: Pemetaan = {}
    for (const [n, v] of Object.entries(pilihan)) {
      pemetaan[n] = v === BARU ? { baru: true } : { jenisBelId: Number(v) }
    }
    try {
      const res = await api.post<{ profilId: number }>('/impor', { data, nama, pemetaan })
      buka = false
      beriTahu(`Profil ${nama} dibuat`)
      onselesai(res.profilId)
    } catch (err) {
      pesanGalat = err instanceof Error ? err.message : String(err)
    } finally {
      sibuk = false
    }
  }
</script>

<PanelGeser bind:buka {judul}>
  <form id="form-impor" class="flex flex-col gap-4" onsubmit={impor}>
    {#if memeriksa}
      <p class="text-slate-600" role="status">Memeriksa file…</p>
    {:else if hasil}
      <Medan label="Nama profil" untuk="impor-nama">
        <input id="impor-nama" class={kelasInput} bind:value={nama} required maxlength="60" />
      </Medan>
      <p class="text-sm text-slate-700">{hasil.jumlahBel} bel</p>
      {#if hasil.jenisBelTakDikenal.length > 0}
        <div class="flex flex-col gap-3">
          <p class="text-sm text-slate-700">
            Jenis bel berikut belum ada. Pilih jenis bel yang dipakai untuk masing-masing.
          </p>
          {#each hasil.jenisBelTakDikenal as n, i (n)}
            <Medan label={n} untuk="impor-peta-{i}">
              <select id="impor-peta-{i}" class={kelasInput} bind:value={pilihan[n]}>
                <option value={BARU}>Buat jenis bel baru</option>
                {#each jenisBel as j (j.id)}<option value={String(j.id)}>{j.nama}</option>{/each}
              </select>
            </Medan>
          {/each}
        </div>
      {/if}
    {/if}
    {#if pesanGalat}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{pesanGalat}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (buka = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-impor" {sibuk} disabled={!hasil}>Impor</Tombol>
  {/snippet}
</PanelGeser>
