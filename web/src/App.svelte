<script lang="ts">
  import { onMount } from 'svelte'
  import Layout from './components/Layout.svelte'
  import Notifikasi from './components/Notifikasi.svelte'
  import KonfirmasiHost from './components/KonfirmasiHost.svelte'
  import Memuat from './components/Memuat.svelte'
  import Tombol from './components/Tombol.svelte'
  import Login from './pages/Login.svelte'
  import GantiPasswordAwal from './pages/GantiPasswordAwal.svelte'
  import { api, aturSaatTidakMasuk, type Sesi } from './lib/api'
  import { rute } from './lib/router.svelte'
  import { halamanUntuk } from './rute'

  let sesi = $state<Sesi | null>(null)
  let galatMuat = $state('')

  async function muatSesi() {
    galatMuat = ''
    try {
      sesi = await api.get<Sesi>('/sesi')
    } catch (e) {
      galatMuat = (e as Error).message
    }
  }

  aturSaatTidakMasuk(() => {
    sesi = { masuk: false, wajibGantiPassword: false }
  })
  onMount(muatSesi)

  const Halaman = $derived(halamanUntuk(rute.path))
</script>

{#if galatMuat}
  <main class="flex min-h-dvh flex-col items-center justify-center gap-4 px-4 text-center">
    <p class="text-slate-700">{galatMuat}</p>
    <Tombol onclick={muatSesi}>Coba lagi</Tombol>
  </main>
{:else if !sesi}
  <Memuat />
{:else if !sesi.masuk}
  <Login onmasuk={muatSesi} />
{:else if sesi.wajibGantiPassword}
  <GantiPasswordAwal onselesai={muatSesi} />
{:else}
  <Layout><Halaman /></Layout>
{/if}

<Notifikasi />
<KonfirmasiHost />
