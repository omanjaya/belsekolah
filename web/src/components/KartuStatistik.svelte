<script lang="ts">
  let { label, nilai, keterangan = '', nada = 'biasa' }: {
    label: string
    nilai: string
    keterangan?: string
    nada?: 'biasa' | 'baik' | 'peringatan' | 'bahaya'
  } = $props()

  const warna = {
    biasa: 'text-slate-900',
    baik: 'text-green-700',
    peringatan: 'text-amber-700',
    bahaya: 'text-red-700',
  }
</script>

<div class="rounded-xl border border-slate-200 bg-white p-4">
  <p class="text-sm font-medium text-slate-600">{label}</p>
  <p class="mt-1 text-2xl font-bold tabular-nums {warna[nada]}">{nilai}</p>
  {#if keterangan}<p class="mt-1 text-sm text-slate-600">{keterangan}</p>{/if}
</div>
