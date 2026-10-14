<script lang="ts">
  import { Bell } from '@lucide/svelte'
  import Tombol from '../components/Tombol.svelte'
  import Medan from '../components/Medan.svelte'
  import { api, ApiError } from '../lib/api'

  let { onmasuk }: { onmasuk: () => void } = $props()
  let password = $state('')
  let sibuk = $state(false)
  let pesanGalat = $state('')

  async function kirim(e: SubmitEvent) {
    e.preventDefault()
    sibuk = true
    pesanGalat = ''
    try {
      await api.post('/login', { password })
      onmasuk()
    } catch (err) {
      pesanGalat = err instanceof ApiError && err.status === 401 ? 'Password salah' : (err as Error).message
    } finally {
      sibuk = false
    }
  }
</script>

<main class="flex min-h-dvh items-center justify-center px-4">
  <form onsubmit={kirim} class="w-full max-w-sm rounded-2xl bg-white p-6 shadow-sm ring-1 ring-slate-200">
    <p class="mb-1 flex items-center gap-2 text-xl font-bold text-blue-800"><Bell class="size-6" /> Bel Sekolah</p>
    <p class="mb-6 text-slate-600">Masuk sebagai admin untuk mengatur bel.</p>
    <Medan label="Password" untuk="password" galat={pesanGalat}
      bantuan="Password awal ada di file password-awal.txt pada mini PC.">
      <input id="password" type="password" autocomplete="current-password" required bind:value={password}
        class="min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20" />
    </Medan>
    <Tombol type="submit" {sibuk} class="mt-5 w-full">Masuk</Tombol>
  </form>
</main>
