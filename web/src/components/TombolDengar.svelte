<script lang="ts">
  import { LoaderCircle, Play, Square, Volume2 } from '@lucide/svelte'
  import { pemutarSuara, type Mode } from '../lib/pemutarSuara.svelte'

  // Tombol putar/hentikan untuk satu suara. mode 'browser' memutar di perangkat
  // ini, 'speaker' memutar lewat speaker mini PC.
  let { kunci, nama, mode, src, durasiMs, ringkas = false }: {
    kunci: string
    nama: string
    mode: Mode
    src: string // URL file (browser) atau path API tes (speaker)
    durasiMs: number
    ringkas?: boolean
  } = $props()

  const aktif = $derived(pemutarSuara.sedang(kunci, mode))
  const memuat = $derived(aktif && pemutarSuara.memuat)

  function klik() {
    if (aktif) pemutarSuara.henti()
    else if (mode === 'browser') pemutarSuara.putarBrowser(kunci, src, durasiMs, nama)
    else pemutarSuara.putarSpeaker(kunci, src, durasiMs, nama)
  }
</script>

{#if mode === 'browser'}
  <button
    type="button"
    class="inline-flex size-11 shrink-0 items-center justify-center rounded-full transition-colors {aktif ? 'bg-blue-600 text-white hover:bg-blue-700' : 'bg-slate-100 text-slate-700 hover:bg-slate-200'}"
    aria-label="{aktif ? 'Berhenti' : 'Putar'} {nama}"
    aria-pressed={aktif}
    title={aktif ? 'Berhenti' : 'Dengarkan di perangkat ini'}
    onclick={klik}
  >
    {#if memuat}<LoaderCircle class="size-5 animate-spin" aria-hidden="true" />
    {:else if aktif}<Square class="size-4 fill-current" aria-hidden="true" />
    {:else}<Play class="ml-0.5 size-5 fill-current" aria-hidden="true" />{/if}
  </button>
{:else}
  <button
    type="button"
    class="inline-flex min-h-11 items-center gap-1.5 whitespace-nowrap rounded-lg px-3 text-sm font-medium transition-colors {aktif ? 'bg-amber-100 text-amber-900 hover:bg-amber-200' : 'text-slate-700 hover:bg-slate-100'}"
    aria-label="{aktif ? 'Hentikan speaker' : 'Tes di speaker'} {nama}"
    aria-pressed={aktif}
    title={aktif ? 'Hentikan suara di speaker' : 'Putar lewat speaker sekolah'}
    onclick={klik}
  >
    {#if memuat}<LoaderCircle class="size-4 animate-spin" aria-hidden="true" />
    {:else if aktif}<Square class="size-3.5 fill-current" aria-hidden="true" />
    {:else}<Volume2 class="size-4" aria-hidden="true" />{/if}
    {#if !ringkas}{aktif ? 'Hentikan' : 'Speaker'}{/if}
  </button>
{/if}
