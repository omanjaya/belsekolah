<script lang="ts">
  import type { Snippet } from 'svelte'

  let { memuat = false, kosong = 'Belum ada data.', jumlah, kepala, children, label }: {
    memuat?: boolean
    kosong?: string
    jumlah: number
    kepala: Snippet
    children: Snippet
    label: string
  } = $props()
</script>

<div class="overflow-hidden rounded-xl border border-slate-200 bg-white">
  <div class="overflow-x-auto" aria-busy={memuat}>
    <table class="w-full border-collapse text-left text-sm" aria-label={label}>
      <thead class="sticky top-0 z-10 bg-slate-50 text-xs font-semibold uppercase tracking-wide text-slate-600 [&_th]:px-4 [&_th]:py-3">
        <tr>{@render kepala()}</tr>
      </thead>
      <tbody class="divide-y divide-slate-200 [&_td]:px-4 [&_td]:py-3">
        {@render children()}
      </tbody>
    </table>
  </div>
  {#if memuat}
    <p class="px-4 py-6 text-center text-slate-600" role="status">Memuat…</p>
  {:else if jumlah === 0}
    <p class="px-4 py-6 text-center text-slate-600">{kosong}</p>
  {/if}
</div>
