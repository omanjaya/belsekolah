<script lang="ts">
  import type { Snippet } from 'svelte'
  import { X } from '@lucide/svelte'

  let { buka = $bindable(false), judul, children, aksi }: {
    buka?: boolean
    judul: string
    children: Snippet
    aksi?: Snippet
  } = $props()

  const uid = $props.id()
  let dialog = $state<HTMLDialogElement>()

  $effect(() => {
    if (!dialog) return
    if (buka && !dialog.open) dialog.showModal()
    else if (!buka && dialog.open) dialog.close()
  })
</script>

<dialog
  bind:this={dialog}
  onclose={() => (buka = false)}
  aria-labelledby="{uid}-judul"
  class="m-auto w-[min(32rem,calc(100vw-2rem))] rounded-2xl bg-white p-0 text-slate-900 shadow-xl backdrop:bg-slate-900/50"
>
  {#if buka}
    <div class="flex items-center justify-between border-b border-slate-200 px-5 py-4">
      <h2 id="{uid}-judul" class="text-lg font-semibold">{judul}</h2>
      <button type="button" class="-my-2 -mr-2 inline-grid size-11 place-items-center rounded-lg hover:bg-slate-100" aria-label="Tutup" onclick={() => (buka = false)}>
        <X class="size-5" />
      </button>
    </div>
    <div class="max-h-[70vh] overflow-y-auto px-5 py-4">{@render children()}</div>
    {#if aksi}
      <div class="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-3">{@render aksi()}</div>
    {/if}
  {/if}
</dialog>
