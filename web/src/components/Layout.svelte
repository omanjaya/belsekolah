<script lang="ts">
  import { onMount, type Snippet } from 'svelte'
  import { Bell, Ellipsis, LogOut, Menu, PanelLeftClose, PanelLeftOpen } from '@lucide/svelte'
  import { menu } from '../rute'
  import { rute, tautan } from '../lib/router.svelte'
  import { api, type Status } from '../lib/api'
  import { galat } from '../lib/notif.svelte'
  import { judulHalaman } from '../lib/judul.svelte'
  import { jamServer, mulaiDetak, perbaruiDariStatus, terakhirDiperbarui } from '../lib/jamServer.svelte'
  import { baca, tulis } from '../lib/simpan'

  let { children }: { children: Snippet } = $props()
  let lainnyaBuka = $state(false)
  let laciBuka = $state(false)
  let ciut = $state(baca<boolean>('sidebarCiut', false) === true)
  let keluarSibuk = $state(false)
  const utama = menu.filter((m) => m.utama)
  const lainnya = menu.filter((m) => !m.utama)
  const lainnyaAktif = $derived(lainnya.some((m) => m.path === rute.path))

  let tombolLainnya = $state<HTMLButtonElement>()
  let lembar = $state<HTMLElement>()
  let tombolMenu = $state<HTMLButtonElement>()
  let sidebar = $state<HTMLElement>()

  function klikLainnya(e: MouseEvent) {
    lainnyaBuka = false
    tautan(e)
  }

  // Tautan di sidebar/laci: di HP laci tertutup dan fokus kembali ke ☰ (juga bila rute sama).
  function klikLaci(e: MouseEvent) {
    tautan(e)
    if (laciBuka) {
      laciBuka = false
      tombolMenu?.focus()
    }
  }

  function aturCiut() {
    ciut = !ciut
    tulis('sidebarCiut', ciut)
  }

  // Lembar "Lainnya" dan laci tertutup setiap kali rute berubah (klik tautan, tombol kembali, dll).
  $effect(() => {
    void rute.path
    lainnyaBuka = false
    laciBuka = false
  })

  // Fokus ke tautan pertama saat lembar/laci dibuka.
  $effect(() => {
    if (lainnyaBuka) lembar?.querySelector('a')?.focus()
  })
  $effect(() => {
    if (laciBuka) sidebar?.querySelector('a')?.focus()
  })

  function tutupLaci() {
    laciBuka = false
    tombolMenu?.focus()
  }

  function tekanTombol(e: KeyboardEvent) {
    if (e.key !== 'Escape') return
    if (laciBuka) tutupLaci()
    else if (lainnyaBuka) {
      lainnyaBuka = false
      tombolLainnya?.focus()
    }
  }

  async function muatStatus() {
    try {
      perbaruiDariStatus(await api.get<Status>('/status'))
    } catch {
      /* jam & pil tetap seperti terakhir; dicoba lagi 30 detik kemudian */
    }
  }

  onMount(() => {
    void muatStatus()
    const henti = mulaiDetak()
    // Halaman lain (Beranda) boleh memperbarui status; muat di sini hanya bila sudah lama tidak ada yang memperbarui.
    const id = setInterval(() => {
      if (Date.now() - terakhirDiperbarui >= 29_000) void muatStatus()
    }, 30_000)
    return () => {
      henti()
      clearInterval(id)
    }
  })

  async function keluar() {
    keluarSibuk = true
    try {
      await api.post('/logout')
      location.reload()
    } catch (e) {
      galat(e)
      keluarSibuk = false
    }
  }

  const pil = $derived(
    jamServer.peringatan === null
      ? { teks: 'Memuat', kelas: 'bg-slate-100 text-slate-700' }
      : jamServer.peringatan.length > 0
        ? { teks: 'Perlu perhatian', kelas: 'bg-amber-100 text-amber-800' }
        : { teks: 'Berjalan', kelas: 'bg-green-100 text-green-800' },
  )
</script>

<svelte:window onkeydown={tekanTombol} />

<div class="min-h-dvh bg-slate-100 md:flex">
  <!-- Sidebar laptop; di HP menjadi laci kiri yang sama -->
  <aside
    bind:this={sidebar}
    class="flex-col bg-slate-900 text-slate-300 md:sticky md:top-0 md:z-auto md:flex md:h-dvh md:shrink-0 {ciut ? 'md:w-16' : 'md:w-60'} {laciBuka ? 'fixed inset-y-0 left-0 z-50 flex w-[280px]' : 'hidden'}"
  >
    <div class="flex h-14 shrink-0 items-center gap-2 px-4 text-lg font-bold text-white {ciut ? 'md:justify-center md:px-0' : ''}">
      <Bell class="size-6 shrink-0" />
      <span class={ciut ? 'md:hidden' : ''}>Bel Sekolah</span>
    </div>
    <nav class="min-h-0 flex-1 overflow-y-auto px-2 py-2" aria-label="Menu utama">
      <ul class="flex flex-col gap-1">
        {#each menu as m (m.path)}
          {@const nama = m.labelPanjang ?? m.label}
          <li>
            <a
              href={m.path}
              onclick={klikLaci}
              title={ciut ? nama : undefined}
              aria-current={rute.path === m.path ? 'page' : undefined}
              class="flex min-h-11 items-center gap-3 rounded-xl px-3 font-medium {ciut ? 'md:justify-center md:px-0' : ''} {rute.path === m.path ? 'bg-blue-700 text-white' : 'text-slate-300 hover:bg-slate-800'}"
            >
              <m.ikon class="size-5 shrink-0" /><span class={ciut ? 'md:sr-only' : ''}>{nama}</span>
            </a>
          </li>
        {/each}
      </ul>
    </nav>
    <div class="hidden shrink-0 border-t border-slate-800 p-2 md:block">
      <button
        type="button"
        onclick={aturCiut}
        aria-label={ciut ? 'Perluas menu' : 'Ciutkan menu'}
        title={ciut ? 'Perluas menu' : 'Ciutkan menu'}
        class="flex min-h-11 w-full items-center gap-3 rounded-xl px-3 font-medium text-slate-300 hover:bg-slate-800 {ciut ? 'justify-center px-0' : ''}"
      >
        {#if ciut}<PanelLeftOpen class="size-5 shrink-0" />{:else}<PanelLeftClose class="size-5 shrink-0" /><span>Ciutkan menu</span>{/if}
      </button>
    </div>
  </aside>

  {#if laciBuka}
    <button type="button" class="fixed inset-0 z-[45] bg-slate-900/50 md:hidden" aria-label="Tutup menu" tabindex="-1" onclick={tutupLaci}></button>
  {/if}

  <div class="flex min-w-0 flex-1 flex-col">
    <header class="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-2 border-b border-slate-200 bg-white px-3 md:px-6">
      <button
        type="button"
        bind:this={tombolMenu}
        onclick={() => (laciBuka = true)}
        aria-label="Buka menu"
        aria-expanded={laciBuka}
        class="-ml-1 inline-grid size-11 shrink-0 place-items-center rounded-lg text-slate-700 hover:bg-slate-100 md:hidden"
      >
        <Menu class="size-6" />
      </button>
      <h1 class="min-w-0 flex-1 truncate text-lg font-bold tracking-tight text-slate-900 md:text-xl">{judulHalaman.teks}</h1>
      <span class="inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1 text-sm font-medium {pil.kelas}" title={pil.teks}>
        <span aria-hidden="true">●</span><span class="max-sm:sr-only">{pil.teks}</span>
      </span>
      <span class="shrink-0 text-sm font-medium tabular-nums text-slate-700" data-testid="jam-server">{jamServer.siap ? `${jamServer.teks} ${jamServer.zona}` : '--:--:--'}</span>
      <button
        type="button"
        onclick={keluar}
        disabled={keluarSibuk}
        aria-label="Keluar"
        class="inline-flex min-h-11 min-w-11 shrink-0 items-center justify-center gap-2 rounded-xl px-2 font-medium text-slate-700 hover:bg-slate-100 disabled:opacity-50 md:px-3"
      >
        <LogOut class="size-5" /><span class="max-md:sr-only">Keluar</span>
      </button>
    </header>

    <main class="min-w-0 flex-1 p-4 pb-24 md:p-6">
      {@render children()}
    </main>
  </div>

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
          bind:this={tombolLainnya}
          onclick={() => (lainnyaBuka = !lainnyaBuka)}
          aria-expanded={lainnyaBuka}
          aria-controls="lembar-lainnya"
          class="flex min-h-14 w-full flex-col items-center justify-center gap-0.5 text-xs font-medium {lainnyaAktif ? 'text-blue-700' : 'text-slate-600'}"
        >
          <Ellipsis class="size-5" />Lainnya
        </button>
      </li>
    </ul>
  </nav>

  {#if lainnyaBuka}
    <button type="button" class="fixed inset-0 z-30 bg-slate-900/40 md:hidden" aria-label="Tutup menu" onclick={() => (lainnyaBuka = false)}></button>
    <div id="lembar-lainnya" bind:this={lembar} class="fixed inset-x-0 bottom-14 z-40 rounded-t-2xl bg-white p-3 pb-[calc(env(safe-area-inset-bottom)+0.75rem)] shadow-xl md:hidden">
      <ul class="flex flex-col gap-1">
        {#each lainnya as m (m.path)}
          <li>
            <a href={m.path} onclick={klikLainnya} aria-current={rute.path === m.path ? 'page' : undefined} class="flex min-h-12 items-center gap-3 rounded-xl px-3 font-medium text-slate-800 hover:bg-slate-100">
              <m.ikon class="size-5" /> {m.label}
            </a>
          </li>
        {/each}
      </ul>
    </div>
  {/if}
</div>
