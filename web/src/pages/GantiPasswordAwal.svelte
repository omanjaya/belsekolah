<script lang="ts">
  import Tombol from '../components/Tombol.svelte'
  import Medan from '../components/Medan.svelte'
  import { api } from '../lib/api'
  import { beriTahu } from '../lib/notif.svelte'

  let { onselesai }: { onselesai: () => void } = $props()
  let lama = $state('')
  let baru = $state('')
  let ulangi = $state('')
  let sibuk = $state(false)
  let pesanGalat = $state('')

  const galatUlang = $derived(ulangi && ulangi !== baru ? 'Password tidak sama' : '')

  async function kirim(e: SubmitEvent) {
    e.preventDefault()
    if (baru.length < 8) { pesanGalat = 'Password baru minimal 8 karakter'; return }
    if (baru !== ulangi) { pesanGalat = 'Password tidak sama'; return }
    sibuk = true
    pesanGalat = ''
    try {
      await api.post('/password', { lama, baru })
      beriTahu('Password berhasil diganti')
      onselesai()
    } catch (err) {
      pesanGalat = (err as Error).message
    } finally {
      sibuk = false
    }
  }
</script>

<main class="flex min-h-dvh items-center justify-center px-4">
  <form onsubmit={kirim} class="flex w-full max-w-sm flex-col gap-4 rounded-2xl bg-white p-6 shadow-sm ring-1 ring-slate-200">
    <div>
      <h1 class="text-xl font-bold">Ganti password awal</h1>
      <p class="mt-1 text-slate-600">Demi keamanan, ganti password awal sebelum memakai aplikasi.</p>
    </div>
    <Medan label="Password awal" untuk="lama">
      <input id="lama" type="password" autocomplete="current-password" required bind:value={lama}
        class="min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20" />
    </Medan>
    <Medan label="Password baru" untuk="baru" bantuan="Minimal 8 karakter.">
      <input id="baru" type="password" autocomplete="new-password" required minlength="8" bind:value={baru}
        class="min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20" />
    </Medan>
    <Medan label="Ulangi password baru" untuk="ulangi" galat={galatUlang}>
      <input id="ulangi" type="password" autocomplete="new-password" required bind:value={ulangi}
        class="min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20" />
    </Medan>
    {#if pesanGalat}<p class="text-sm text-red-600" role="alert">{pesanGalat}</p>{/if}
    <Tombol type="submit" {sibuk}>Simpan password</Tombol>
  </form>
</main>
