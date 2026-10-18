<script lang="ts">
  import { formatWaktu } from '../lib/format'
  import { pemutarSuara } from '../lib/pemutarSuara.svelte'

  // Bilah progres untuk suara yang sedang diputar; kosong bila suara ini tidak aktif.
  let { kunci }: { kunci: string } = $props()

  const aktif = $derived(pemutarSuara.sedang(kunci))
  const persen = $derived(
    pemutarSuara.durasi > 0 ? Math.min(100, (pemutarSuara.posisi / pemutarSuara.durasi) * 100) : 0,
  )
</script>

{#if aktif}
  <div class="mt-1.5 flex items-center gap-2 text-xs tabular-nums text-slate-600" aria-live="polite">
    <span class="whitespace-nowrap font-medium {pemutarSuara.aktif?.mode === 'speaker' ? 'text-amber-800' : 'text-blue-700'}">
      {pemutarSuara.aktif?.mode === 'speaker' ? 'Di speaker' : 'Diputar'}
    </span>
    <div class="h-1.5 min-w-16 flex-1 overflow-hidden rounded-full bg-slate-200" aria-hidden="true">
      <div
        class="h-full rounded-full transition-[width] duration-200 ease-linear {pemutarSuara.aktif?.mode === 'speaker' ? 'bg-amber-500' : 'bg-blue-600'}"
        style="width: {persen}%"
      ></div>
    </div>
    <span class="whitespace-nowrap">{formatWaktu(pemutarSuara.posisi)}{pemutarSuara.durasi > 0 ? ` / ${formatWaktu(pemutarSuara.durasi)}` : ''}</span>
  </div>
{/if}
