<script lang="ts">
  import type { Snippet } from 'svelte'
  import { Bell, Ellipsis } from '@lucide/svelte'
  import { menu } from '../rute'
  import { rute, tautan } from '../lib/router.svelte'

  let { children }: { children: Snippet } = $props()
  let lainnyaBuka = $state(false)
  const utama = menu.filter((m) => m.utama)
  const lainnya = menu.filter((m) => !m.utama)
  const lainnyaAktif = $derived(lainnya.some((m) => m.path === rute.path))

  function klikLainnya(e: MouseEvent) {
    lainnyaBuka = false
    tautan(e)
  }
</script>

<div class="min-h-dvh md:flex">
  <!-- Sidebar desktop -->
  <nav class="hidden w-60 shrink-0 border-r border-slate-200 bg-white md:block" aria-label="Menu utama">
    <div class="sticky top-0 p-4">
      <p class="mb-6 flex items-center gap-2 px-2 text-lg font-bold text-blue-800"><Bell class="size-6" /> Bel Sekolah</p>
      <ul class="flex flex-col gap-1">
        {#each menu as m (m.path)}
          <li>
            <a
              href={m.path}
              onclick={tautan}
              aria-current={rute.path === m.path ? 'page' : undefined}
              class="flex min-h-11 items-center gap-3 rounded-xl px-3 font-medium {rute.path === m.path ? 'bg-blue-50 text-blue-800' : 'text-slate-700 hover:bg-slate-100'}"
            >
              <m.ikon class="size-5" /> {m.label}
            </a>
          </li>
        {/each}
      </ul>
    </div>
  </nav>

  <main class="mx-auto w-full max-w-4xl px-4 pt-5 pb-28 md:px-8 md:pb-10">
    {@render children()}
  </main>

  <!-- Bar bawah HP -->
  <nav class="fixed inset-x-0 bottom-0 z-40 border-t border-slate-200 bg-white pb-[env(safe-area-inset-bottom)] md:hidden" aria-label="Menu utama">
    <ul class="grid grid-cols-5">
      {#each utama as m (m.path)}
        <li>
          <a
            href={m.path}
            onclick={tautan}
            aria-current={rute.path === m.path ? 'page' : undefined}
            class="flex min-h-14 flex-col items-center justify-center gap-0.5 text-xs font-medium {rute.path === m.path ? 'text-blue-700' : 'text-slate-600'}"
          >
            <m.ikon class="size-5" />{m.label}
          </a>
        </li>
      {/each}
      <li>
        <button
          type="button"
          onclick={() => (lainnyaBuka = !lainnyaBuka)}
          aria-expanded={lainnyaBuka}
          class="flex min-h-14 w-full flex-col items-center justify-center gap-0.5 text-xs font-medium {lainnyaAktif ? 'text-blue-700' : 'text-slate-600'}"
        >
          <Ellipsis class="size-5" />Lainnya
        </button>
      </li>
    </ul>
  </nav>

  {#if lainnyaBuka}
    <button type="button" class="fixed inset-0 z-30 bg-slate-900/40 md:hidden" aria-label="Tutup menu" onclick={() => (lainnyaBuka = false)}></button>
    <div class="fixed inset-x-0 bottom-14 z-40 rounded-t-2xl bg-white p-3 pb-[calc(env(safe-area-inset-bottom)+0.75rem)] shadow-xl md:hidden">
      <ul class="flex flex-col gap-1">
        {#each lainnya as m (m.path)}
          <li>
            <a href={m.path} onclick={klikLainnya} class="flex min-h-12 items-center gap-3 rounded-xl px-3 font-medium text-slate-800 hover:bg-slate-100">
              <m.ikon class="size-5" /> {m.label}
            </a>
          </li>
        {/each}
      </ul>
    </div>
  {/if}
</div>
