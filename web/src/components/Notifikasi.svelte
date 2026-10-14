<script lang="ts">
  import { X } from '@lucide/svelte'
  import { daftarNotif, tutupNotif } from '../lib/notif.svelte'

  let wadah = $state<HTMLElement>()

  // Popover manual berada di top layer; menampilkannya ulang menaikkannya di atas dialog yang dibuka lebih dulu.
  $effect(() => {
    if (!wadah) return
    if (daftarNotif.length > 0) {
      if (wadah.matches(':popover-open')) wadah.hidePopover()
      wadah.showPopover()
    } else if (wadah.matches(':popover-open')) {
      wadah.hidePopover()
    }
  })

  const gaya = {
    sukses: 'bg-emerald-700 text-white',
    galat: 'bg-red-600 text-white',
    info: 'bg-slate-800 text-white',
  }
</script>

<div
  bind:this={wadah}
  popover="manual"
  aria-live="polite"
  role="status"
  class="pointer-events-none fixed inset-auto top-3 left-0 m-0 w-full flex-col items-center gap-2 overflow-visible border-0 bg-transparent p-0 px-3 open:flex"
>
  {#each daftarNotif as n (n.id)}
    <div class="pointer-events-auto flex w-full max-w-md items-start gap-3 rounded-xl px-4 py-3 shadow-lg {gaya[n.jenis]}">
      <p class="flex-1 text-sm font-medium">{n.pesan}</p>
      <button type="button" class="-my-2 -mr-3 inline-grid size-11 shrink-0 place-items-center" aria-label="Tutup notifikasi" onclick={() => tutupNotif(n.id)}><X class="size-4" /></button>
    </div>
  {/each}
</div>
