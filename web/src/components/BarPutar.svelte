<script lang="ts">
  import { Square, Volume2, Headphones } from '@lucide/svelte'
  import { pemutarSuara } from '../lib/pemutarSuara.svelte'
  import ProgresPutar from './ProgresPutar.svelte'

  // Bar melayang yang selalu terlihat selama ada suara diputar, di halaman mana pun.
  const aktif = $derived(pemutarSuara.aktif)
</script>

{#if aktif}
  <div
    class="fixed inset-x-3 bottom-[calc(env(safe-area-inset-bottom)+4.25rem)] z-30 rounded-2xl border bg-white p-3 shadow-lg md:inset-x-auto md:right-6 md:bottom-6 md:w-96 {aktif.mode === 'speaker' ? 'border-amber-300' : 'border-blue-300'}"
    role="region"
    aria-label="Suara yang sedang diputar"
  >
    <div class="flex items-center gap-3">
      <span class="inline-flex size-9 shrink-0 items-center justify-center rounded-full {aktif.mode === 'speaker' ? 'bg-amber-100 text-amber-800' : 'bg-blue-100 text-blue-700'}" aria-hidden="true">
        {#if aktif.mode === 'speaker'}<Volume2 class="size-5" />{:else}<Headphones class="size-5" />{/if}
      </span>
      <div class="min-w-0 flex-1">
        <p class="truncate text-sm font-semibold text-slate-900">{aktif.nama || 'Suara'}</p>
        <ProgresPutar kunci={aktif.kunci} />
      </div>
      <button
        type="button"
        class="inline-flex min-h-11 items-center gap-1.5 rounded-lg bg-slate-900 px-3 text-sm font-medium text-white hover:bg-slate-700"
        onclick={() => pemutarSuara.henti()}
      >
        <Square class="size-3.5 fill-current" aria-hidden="true" /> Hentikan
      </button>
    </div>
  </div>
{/if}
