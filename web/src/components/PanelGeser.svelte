<script lang="ts">
  import type { Snippet } from 'svelte'
  import { X } from '@lucide/svelte'

  let { buka = $bindable(false), judul, children, aksi, lebar = 'md' }: {
    buka?: boolean
    judul: string
    children: Snippet
    aksi?: Snippet
    lebar?: 'md' | 'lg'
  } = $props()

  const uid = $props.id()
  let dialog = $state<HTMLDialogElement>()
  let pemicu: HTMLElement | null = null

  $effect(() => {
    if (!dialog) return
    if (buka && !dialog.open) {
      pemicu = document.activeElement instanceof HTMLElement ? document.activeElement : null
      dialog.showModal()
      // Fokus ke isian pertama bila ada; selain itu biarkan bawaan dialog.
      dialog.querySelector<HTMLElement>('input:not([type=hidden]):not([disabled]), select:not([disabled]), textarea:not([disabled])')?.focus()
    } else if (!buka && dialog.open) {
      dialog.close()
    }
  })

  function tertutup() {
    buka = false
    pemicu?.focus()
    pemicu = null
  }

  // Klik pada latar (di luar isi panel) mengenai <dialog> itu sendiri.
  // Hanya menutup bila tekan dan lepas keduanya di latar (seret seleksi teks yang berakhir di latar tidak menutup).
  let tekanDiLatar = false
  function tekanLatar(e: MouseEvent) {
    tekanDiLatar = e.target === dialog
  }
  function klikLatar(e: MouseEvent) {
    if (e.target === dialog && tekanDiLatar) buka = false
    tekanDiLatar = false
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_noninteractive_element_interactions -->
<dialog
  bind:this={dialog}
  onclose={tertutup}
  onmousedown={tekanLatar}
  onclick={klikLatar}
  aria-modal="true"
  aria-labelledby="{uid}-judul"
  class="panel-geser m-0 ml-auto h-dvh max-h-none w-full max-w-none bg-white p-0 text-slate-900 shadow-xl backdrop:bg-slate-900/40 {lebar === 'lg' ? 'md:w-[640px]' : 'md:w-[440px]'}"
>
  {#if buka}
    <div class="flex h-full flex-col">
      <div class="flex shrink-0 items-center justify-between border-b border-slate-200 px-5 py-3">
        <h2 id="{uid}-judul" class="text-lg font-semibold">{judul}</h2>
        <button type="button" class="-my-1 -mr-2 inline-grid size-11 place-items-center rounded-lg hover:bg-slate-100" aria-label="Tutup" onclick={() => (buka = false)}>
          <X class="size-5" />
        </button>
      </div>
      <div class="min-h-0 flex-1 overflow-y-auto px-5 py-4">{@render children()}</div>
      {#if aksi}
        <div class="flex shrink-0 flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-3">{@render aksi()}</div>
      {/if}
    </div>
  {/if}
</dialog>
