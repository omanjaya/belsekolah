<script lang="ts">
  import { Volume2 } from '@lucide/svelte'
  import { NAMA_HARI } from '../lib/format'

  type BelMinggu = { kunci: string | number; hari: number; jam: string; label: string; jenisNama: string; warna: string; pengumuman?: string }

  let { bel, hariSekolah = [1, 2, 3, 4, 5], onpilih, padat = false }: {
    bel: BelMinggu[]
    hariSekolah?: number[]
    onpilih?: (kunci: string | number) => void
    padat?: boolean
  } = $props()

  const jam5 = (j: string) => j.slice(0, 5)

  // Hari yang selalu tampil, ditambah hari 6/7 (dan hari lain) bila punya bel.
  const hariTampil = $derived(
    [1, 2, 3, 4, 5, 6, 7].filter((h) => hariSekolah.includes(h) || bel.some((b) => b.hari === h)),
  )
  const perHari = (h: number) => bel.filter((b) => b.hari === h).sort((a, b) => jam5(a.jam).localeCompare(jam5(b.jam)))
</script>

<div class="grid grid-cols-1 gap-3 md:[grid-template-columns:repeat(var(--kolom),minmax(0,1fr))]" style="--kolom: {hariTampil.length}">
  {#each hariTampil as h (h)}
    {@const daftar = perHari(h)}
    <section class="min-w-0 rounded-xl border border-slate-200 bg-white {padat ? 'p-2' : 'p-3'}" aria-label={NAMA_HARI[h]}>
      <header class="mb-2 flex items-baseline justify-between gap-2">
        <h3 class="font-semibold {padat ? 'text-sm' : ''}">{NAMA_HARI[h]}</h3>
        <span class="text-xs text-slate-600">{daftar.length} bel</span>
      </header>
      {#if daftar.length === 0}
        <p class="text-sm text-slate-600">Tidak ada bel</p>
      {:else}
        <ul class="flex flex-col {padat ? 'gap-1' : 'gap-2'}">
          {#each daftar as b (b.kunci)}
            {@const kelas = `block w-full min-w-0 rounded-lg border-l-4 bg-slate-50 text-left ${padat ? 'px-2 py-1 text-xs' : 'px-3 py-2 text-sm'}`}
            <li>
              {#if onpilih}
                <button
                  type="button"
                  class="{kelas} hover:bg-slate-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700 {padat ? '' : 'min-h-11'}"
                  style="border-left-color: {b.warna}"
                  onclick={() => onpilih(b.kunci)}
                >
                  <span class="font-bold tabular-nums">{jam5(b.jam)}</span>
                  {#if b.pengumuman}<Volume2 class="ml-1 inline size-3.5 text-slate-500" aria-label="dengan pengumuman {b.pengumuman}" />{/if}
                  <span class="block truncate text-slate-700">{b.label || b.jenisNama}</span>
                </button>
              {:else}
                <div class={kelas} style="border-left-color: {b.warna}">
                  <span class="font-bold tabular-nums">{jam5(b.jam)}</span>
                  {#if b.pengumuman}<Volume2 class="ml-1 inline size-3.5 text-slate-500" aria-label="dengan pengumuman {b.pengumuman}" />{/if}
                  <span class="block truncate text-slate-700">{b.label || b.jenisNama}</span>
                </div>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {/each}
</div>
