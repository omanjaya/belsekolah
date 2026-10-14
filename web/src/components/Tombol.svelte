<script lang="ts">
  import type { Snippet } from 'svelte'
  import type { HTMLButtonAttributes } from 'svelte/elements'

  type Props = HTMLButtonAttributes & {
    varian?: 'utama' | 'kedua' | 'bahaya' | 'hantu'
    sibuk?: boolean
    children: Snippet
  }
  let { varian = 'utama', sibuk = false, disabled, class: kelas = '', type = 'button', children, ...rest }: Props = $props()

  const gaya = {
    utama: 'bg-blue-700 text-white hover:bg-blue-800 focus-visible:outline-blue-700',
    kedua: 'bg-white text-slate-800 ring-1 ring-slate-300 hover:bg-slate-50 focus-visible:outline-slate-500',
    bahaya: 'bg-red-600 text-white hover:bg-red-700 focus-visible:outline-red-600',
    hantu: 'text-slate-700 hover:bg-slate-100 focus-visible:outline-slate-500',
  }
</script>

<button
  {type}
  disabled={disabled || sibuk}
  aria-busy={sibuk}
  class="inline-flex min-h-11 items-center justify-center gap-2 rounded-xl px-4 py-2 font-medium transition focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {gaya[varian]} {kelas}"
  {...rest}
>
  {#if sibuk}
    <span class="size-4 animate-spin rounded-full border-2 border-current border-t-transparent" aria-hidden="true"></span>
  {/if}
  {@render children()}
</button>
