<script lang="ts">
  import type { Snippet } from 'svelte'
  import { judulHalaman } from '../lib/judul.svelte'

  let { judul, keterangan = '', aksi }: { judul: string; keterangan?: string; aksi?: Snippet } = $props()

  // <h1> hanya ada di bar atas (Layout); komponen ini mengisinya dan merender keterangan + aksi halaman.
  $effect(() => {
    const ini = judul
    judulHalaman.teks = ini
    return () => {
      if (judulHalaman.teks === ini) judulHalaman.teks = ''
    }
  })
</script>

{#if keterangan || aksi}
  <header class="mb-4 flex flex-wrap items-center justify-between gap-3">
    {#if keterangan}<p class="text-slate-600">{keterangan}</p>{/if}
    {#if aksi}<div class="flex flex-wrap gap-2 md:ml-auto">{@render aksi()}</div>{/if}
  </header>
{/if}
