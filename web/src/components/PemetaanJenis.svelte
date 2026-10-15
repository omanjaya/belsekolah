<script lang="ts">
  import type { JenisBel } from '../lib/api'
  import { BARU_JENIS } from '../lib/tambahBanyak'
  import Medan from './Medan.svelte'

  let { nama, jenisBel, pilihan = $bindable(), awalanId }: {
    nama: string[]
    jenisBel: JenisBel[]
    // nama jenis bel tak dikenal -> 'baru' | id jenis bel (string)
    pilihan: Record<string, string>
    awalanId: string
  } = $props()

  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
</script>

<div class="flex flex-col gap-3">
  <p class="text-sm text-slate-700">
    Jenis bel berikut belum ada. Pilih jenis bel yang dipakai untuk masing-masing.
  </p>
  {#each nama as n, i (n)}
    <Medan label={n} untuk="{awalanId}-{i}">
      <select id="{awalanId}-{i}" class={kelasInput} bind:value={pilihan[n]}>
        <option value={BARU_JENIS}>Buat jenis bel baru</option>
        {#each jenisBel as j (j.id)}<option value={String(j.id)}>{j.nama}</option>{/each}
      </select>
    </Medan>
  {/each}
</div>
