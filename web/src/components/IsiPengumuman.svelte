<script lang="ts">
  import { untrack } from 'svelte'
  import { api, type Profil, type SaranPengumuman } from '../lib/api'
  import { NAMA_HARI } from '../lib/format'
  import { beriTahu } from '../lib/notif.svelte'
  import PanelGeser from './PanelGeser.svelte'
  import Tombol from './Tombol.svelte'

  let { buka = $bindable(false), profil, onselesai }: {
    buka?: boolean
    profil: Profil
    onselesai: () => void
  } = $props()

  let memuat = $state(false)
  let saran = $state<SaranPengumuman[]>([])
  let dicentang = $state<Record<number, boolean>>({})
  let pesanGalat = $state('')
  let sibuk = $state(false)
  let urutan = 0

  $effect(() => {
    if (!buka) return
    const id = profil.id
    untrack(() => muat(id))
  })

  async function muat(id: number) {
    const saya = ++urutan
    memuat = true
    saran = []
    dicentang = {}
    pesanGalat = ''
    try {
      const s = await api.get<SaranPengumuman[]>(`/profil/${id}/pengumuman/saran`)
      if (saya !== urutan) return
      saran = s
      dicentang = Object.fromEntries(s.map((r) => [r.jadwalId, true]))
    } catch (e) {
      if (saya !== urutan) return
      pesanGalat = e instanceof Error ? e.message : String(e)
    } finally {
      if (saya === urutan) memuat = false
    }
  }

  const terpilih = $derived(saran.filter((r) => dicentang[r.jadwalId]))
  const semua = $derived(saran.length > 0 && terpilih.length === saran.length)
  const perHari = $derived.by(() => {
    const m = new Map<number, SaranPengumuman[]>()
    for (const r of saran) m.set(r.hari, [...(m.get(r.hari) ?? []), r])
    return [...m.entries()]
  })

  function gantiSemua() {
    const nilai = !semua
    dicentang = Object.fromEntries(saran.map((r) => [r.jadwalId, nilai]))
  }

  async function terapkan() {
    pesanGalat = ''
    sibuk = true
    try {
      const res = await api.post<{ diisi: number }>(`/profil/${profil.id}/pengumuman/terapkan`, {
        isi: terpilih.map((r) => ({ jadwalId: r.jadwalId, audioId: r.suara.id })),
      })
      beriTahu(`${res.diisi} bel diberi pengumuman`)
      buka = false
      onselesai()
    } catch (e) {
      pesanGalat = e instanceof Error ? e.message : String(e)
    } finally {
      sibuk = false
    }
  }
</script>

<PanelGeser bind:buka judul="Isi pengumuman otomatis">
  <div class="flex flex-col gap-4">
    {#if memuat}
      <p class="text-slate-600" role="status">Memuat saran…</p>
    {:else if saran.length === 0 && !pesanGalat}
      <p class="text-slate-600">Semua bel sudah punya pengumuman atau tidak ada suara yang cocok</p>
    {:else if saran.length > 0}
      <button type="button" class="min-h-11 self-start rounded-lg px-3 text-sm font-medium text-blue-700 hover:bg-blue-50" onclick={gantiSemua}>
        {semua ? 'Kosongkan' : 'Pilih semua'}
      </button>
      {#each perHari as [hari, baris] (hari)}
        <fieldset class="flex flex-col gap-1">
          <legend class="mb-1 text-sm font-semibold text-slate-800">{NAMA_HARI[hari]}</legend>
          {#each baris as r (r.jadwalId)}
            <label class="flex min-h-11 items-center gap-3 rounded-lg px-2 text-sm hover:bg-slate-50">
              <input type="checkbox" class="size-5" bind:checked={dicentang[r.jadwalId]} />
              <span>{NAMA_HARI[r.hari]} {r.jam} · {r.label || '—'} → {r.suara.nama}</span>
            </label>
          {/each}
        </fieldset>
      {/each}
    {/if}
    {#if pesanGalat}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{pesanGalat}</p>
    {/if}
  </div>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (buka = false)}>Batal</Tombol>
    <Tombol {sibuk} disabled={terpilih.length === 0} onclick={terapkan}>Terapkan ({terpilih.length})</Tombol>
  {/snippet}
</PanelGeser>
