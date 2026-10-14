<script lang="ts">
  import { X } from '@lucide/svelte'
  import { daftarNotif, tutupNotif } from '../lib/notif.svelte'

  const gaya = {
    sukses: 'bg-emerald-600 text-white',
    galat: 'bg-red-600 text-white',
    info: 'bg-slate-800 text-white',
  }
</script>

<div class="pointer-events-none fixed inset-x-0 top-3 z-50 flex flex-col items-center gap-2 px-3" aria-live="polite" role="status">
  {#each daftarNotif as n (n.id)}
    <div class="pointer-events-auto flex w-full max-w-md items-start gap-3 rounded-xl px-4 py-3 shadow-lg {gaya[n.jenis]}">
      <p class="flex-1 text-sm font-medium">{n.pesan}</p>
      <button type="button" aria-label="Tutup notifikasi" onclick={() => tutupNotif(n.id)}><X class="size-4" /></button>
    </div>
  {/each}
</div>
