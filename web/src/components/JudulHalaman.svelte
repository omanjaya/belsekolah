<script lang="ts">
  import type { Snippet } from 'svelte'
  let { judul, keterangan = '', aksi }: { judul: string; keterangan?: string; aksi?: Snippet } = $props()
</script>

<header class="mb-5 flex flex-wrap items-end justify-between gap-3">
  <div>
    <h1 class="text-2xl font-bold tracking-tight text-slate-900">{judul}</h1>
    {#if keterangan}<p class="mt-1 text-slate-600">{keterangan}</p>{/if}
  </div>
  {#if aksi}<div class="flex flex-wrap gap-2">{@render aksi()}</div>{/if}
</header>
