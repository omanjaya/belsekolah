<script lang="ts">
  import type { Audio } from '../lib/api'
  import TombolDengar from './TombolDengar.svelte'

  let { audio, nilai = $bindable(null), id, label, onpilih }: {
    audio: Audio[]
    nilai: number | null
    id: string
    label: string
    onpilih?: () => void
  } = $props()

  let cari = $state('')

  const tersaring = $derived.by(() => {
    const k = cari.trim().toLowerCase()
    return k ? audio.filter((a) => a.nama.toLowerCase().includes(k)) : audio
  })
  const terpilih = $derived(audio.find((a) => a.id === nilai) ?? null)
  const hilang = $derived(nilai !== null && terpilih === null)

  function pilih(v: number | null) {
    nilai = v
    onpilih?.()
  }

  const kelasOpsi = 'flex min-h-11 w-full items-center rounded-lg px-3 text-left text-sm hover:bg-slate-100'
</script>

<div class="flex flex-col gap-2">
  <label for={id} class="text-sm font-medium text-slate-800">{label}</label>
  <input
    {id}
    type="search"
    bind:value={cari}
    placeholder="Cari suara…"
    autocomplete="off"
    class="min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20"
  />
  <div class="flex items-center justify-between gap-2 text-sm text-slate-700">
    <span>{terpilih ? terpilih.nama : hilang ? `Suara #${nilai} (tidak ada di daftar)` : 'Tanpa pengumuman'}</span>
    {#if terpilih}
      <div class="flex items-center gap-1">
        <TombolDengar kunci="a:{terpilih.id}" nama={terpilih.nama} mode="browser" src="/api/audio/{terpilih.id}/file" durasiMs={terpilih.durasiMs} />
        <TombolDengar kunci="a:{terpilih.id}" nama={terpilih.nama} mode="speaker" src="/audio/{terpilih.id}/tes" durasiMs={terpilih.durasiMs} ringkas />
      </div>
    {/if}
  </div>
  <div role="listbox" aria-label={label} class="max-h-64 overflow-y-auto rounded-xl border border-slate-200 p-1">
    <button type="button" role="option" aria-selected={nilai === null} class="{kelasOpsi} {nilai === null ? 'bg-blue-50 font-medium text-blue-800' : ''}" onclick={() => pilih(null)}>
      Tanpa pengumuman
    </button>
    {#each tersaring as a (a.id)}
      <button type="button" role="option" aria-selected={nilai === a.id} class="{kelasOpsi} {nilai === a.id ? 'bg-blue-50 font-medium text-blue-800' : ''}" onclick={() => pilih(a.id)}>
        {a.nama}
      </button>
    {/each}
  </div>
  {#if tersaring.length === 0}
    <p class="px-3 py-2 text-sm text-slate-500">Tidak ada suara yang cocok</p>
  {/if}
</div>
