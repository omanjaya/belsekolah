<script lang="ts">
  import { onMount } from 'svelte'
  import { api, type JenisBel, type Status, type BelHariIni } from '../lib/api'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { formatJam, formatTanggalPanjang, hitungMundur, offsetDariISO } from '../lib/format'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kartu from '../components/Kartu.svelte'
  import Kosong from '../components/Kosong.svelte'
  import Memuat from '../components/Memuat.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import Tombol from '../components/Tombol.svelte'

  const DURASI_TAHAN = 2000

  let status = $state<Status | null>(null)
  let jenisBel = $state<JenisBel[]>([])
  let galatMuat = $state('')
  let dasarMs = 0
  let dasarPerf = 0
  let sekarangMs = $state(0)

  let jenisDipilih = $state<number | ''>('')
  let sibukManual = $state(false)
  let sibukStop = $state(false)
  let sibukDarurat = $state(false)
  let progres = $state(0)
  let menahan = $state(false)
  let timerTahan: ReturnType<typeof setInterval> | undefined
  let mulaiTahan = 0

  const offset = $derived(status ? offsetDariISO(status.sekarang) : 0)
  const jenisManual = $derived(jenisBel.filter((j) => j.nama !== 'Darurat'))
  const jenisDarurat = $derived(jenisBel.find((j) => j.nama === 'Darurat'))

  let urutMuat = 0
  let gagalBerturut = $state(0)

  async function muat() {
    const urut = ++urutMuat
    try {
      const s = await api.get<Status>('/status')
      if (urut !== urutMuat) return
      status = s
      gagalBerturut = 0
      if (jenisBel.length === 0) muatJenis()
      dasarMs = Date.parse(s.sekarang)
      dasarPerf = performance.now()
      sekarangMs = dasarMs
      galatMuat = ''
    } catch (e) {
      if (urut !== urutMuat) return
      gagalBerturut++
      if (!status) galatMuat = e instanceof Error ? e.message : String(e)
    }
  }

  async function muatJenis() {
    try {
      jenisBel = await api.get<JenisBel[]>('/jenis-bel')
      if (jenisDipilih === '' && jenisManual.length > 0) jenisDipilih = jenisManual[0].id
    } catch (e) {
      galat(e)
    }
  }

  onMount(() => {
    muat()
    muatJenis()
    const poll = setInterval(() => {
      if (!document.hidden) muat()
    }, 5000)
    const detik = setInterval(() => {
      if (status) sekarangMs = dasarMs + (performance.now() - dasarPerf)
    }, 1000)
    const terlihat = () => {
      if (!document.hidden) muat()
    }
    document.addEventListener('visibilitychange', terlihat)
    return () => {
      clearInterval(poll)
      clearInterval(detik)
      batalTahan()
      document.removeEventListener('visibilitychange', terlihat)
    }
  })

  async function bunyikan() {
    if (jenisDipilih === '') return
    sibukManual = true
    try {
      await api.post('/bel/manual', { jenisBelId: jenisDipilih })
      beriTahu('Bel dibunyikan')
    } catch (e) {
      galat(e)
    } finally {
      sibukManual = false
    }
  }

  async function hentikan() {
    sibukStop = true
    try {
      await api.post('/bel/stop')
      beriTahu('Suara dihentikan')
    } catch (e) {
      galat(e)
    } finally {
      sibukStop = false
    }
  }

  async function darurat() {
    if (!jenisDarurat) return
    sibukDarurat = true
    try {
      await api.post('/bel/darurat', { jenisBelId: jenisDarurat.id })
      beriTahu('Bel darurat dibunyikan')
    } catch (e) {
      galat(e)
    } finally {
      sibukDarurat = false
    }
  }

  function mulaiMenahan(e?: PointerEvent) {
    if (e && (e.button !== 0 || !e.isPrimary)) return
    if (menahan || sibukDarurat) return
    menahan = true
    progres = 0
    mulaiTahan = performance.now()
    timerTahan = setInterval(() => {
      progres = Math.min(1, (performance.now() - mulaiTahan) / DURASI_TAHAN)
      if (progres >= 1) {
        batalTahan()
        darurat()
      }
    }, 40)
  }

  function batalTahan() {
    if (timerTahan) clearInterval(timerTahan)
    timerTahan = undefined
    menahan = false
    progres = 0
  }

  function tombolTahan(e: KeyboardEvent) {
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault()
      if (!e.repeat) mulaiMenahan()
    }
  }
  function lepasTombol(e: KeyboardEvent) {
    if (e.key === ' ' || e.key === 'Enter') {
      e.preventDefault()
      batalTahan()
    }
  }

  const berikutnyaId = $derived(status?.berikutnya?.jadwalId)
  const sisa = $derived(status?.berikutnya ? Date.parse(status.berikutnya.waktu) - sekarangMs : 0)
  const namaBel = (b: BelHariIni) => b.label || b.jenisBel?.nama || 'Bel'
</script>

<JudulHalaman judul="Beranda" />

{#if !status}
  {#if galatMuat}
    <GalatMuat pesan={galatMuat} oncoba={muat} />
  {:else}
    <Memuat />
  {/if}
{:else}
  <div class="grid gap-4 lg:grid-cols-2">
    <div class="space-y-4">
      <Kartu>
        <p class="text-5xl font-bold tabular-nums tracking-tight text-slate-900 sm:text-6xl">{formatJam(sekarangMs, offset)}</p>
        <p class="mt-1 text-slate-600">{formatTanggalPanjang(status.tanggal)}</p>
        {#if status.libur}
          <p class="mt-3 rounded-xl bg-amber-50 px-3 py-2 font-medium text-amber-900 ring-1 ring-amber-300">Hari libur: {status.libur.keterangan}</p>
        {:else if status.profil}
          <p class="mt-3 inline-block rounded-full bg-blue-50 px-3 py-1 text-sm font-medium text-blue-800">Profil: {status.profil.nama}</p>
        {/if}
        {#if gagalBerturut >= 2}
          <p role="status" class="mt-3 rounded-xl bg-amber-50 px-3 py-2 text-sm font-medium text-amber-900 ring-1 ring-amber-300">Koneksi ke server terputus — data mungkin tidak terbaru</p>
        {/if}
      </Kartu>

      {#each status.peringatan as p, i (`${i}-${p}`)}
        <div role="alert" class="rounded-xl bg-amber-50 px-4 py-3 text-amber-900 ring-1 ring-amber-300">{p}</div>
      {/each}

      <Kartu>
        <h2 class="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-500">Bel berikutnya</h2>
        {#if status.berikutnya}
          <p class="text-4xl font-bold tabular-nums text-slate-900">{status.berikutnya.jam}</p>
          <p class="mt-1 text-lg text-slate-800">{namaBel(status.berikutnya)}</p>
          {#if status.berikutnya.jenisBel}
            <p class="mt-1 flex items-center gap-2 text-slate-600">
              <span class="size-3 rounded-full" style="background-color: {status.berikutnya.jenisBel.warna}" aria-hidden="true"></span>
              {status.berikutnya.jenisBel.nama}
            </p>
          {/if}
          <p class="mt-2 font-medium text-blue-700">{hitungMundur(sisa)}</p>
        {:else}
          <p class="text-slate-600">Tidak ada bel lagi hari ini</p>
        {/if}
      </Kartu>

      <Kartu>
        <h2 class="mb-3 text-lg font-semibold text-slate-900">Bunyikan bel</h2>
        <div class="space-y-3">
          <div>
            <label for="jenis-bel" class="mb-1 block text-sm font-medium text-slate-700">Jenis bel</label>
            <select
              id="jenis-bel"
              bind:value={jenisDipilih}
              class="min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20"
            >
              {#each jenisManual as j (j.id)}
                <option value={j.id}>{j.nama}</option>
              {/each}
            </select>
          </div>
          <Tombol class="w-full" sibuk={sibukManual} disabled={jenisDipilih === ''} onclick={bunyikan}>Bunyikan</Tombol>

          <button
            type="button"
            aria-label="Tahan 2 detik untuk bel darurat"
            disabled={sibukDarurat || !jenisDarurat}
            class="relative inline-flex min-h-14 w-full touch-none select-none [-webkit-touch-callout:none] items-center justify-center overflow-hidden rounded-xl bg-red-600 px-4 py-3 font-semibold text-white hover:bg-red-700 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-red-600 disabled:opacity-50"
            onpointerdown={(e) => mulaiMenahan(e)}
            onpointerup={batalTahan}
            onpointerleave={batalTahan}
            onpointercancel={batalTahan}
            onkeydown={tombolTahan}
            onkeyup={lepasTombol}
            onblur={batalTahan}
            oncontextmenu={(e) => e.preventDefault()}
          >
            <span class="absolute inset-y-0 left-0 bg-red-900/60" style="width: {progres * 100}%" aria-hidden="true"></span>
            <span class="relative">{menahan ? 'Terus tahan…' : 'Tahan untuk bel darurat'}</span>
          </button>
          {#if jenisBel.length > 0 && !jenisDarurat}
            <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="status">Jenis bel "Darurat" tidak ada. Buat di menu Jenis Bel.</p>
          {/if}

          <Tombol varian="kedua" class="w-full" sibuk={sibukStop} onclick={hentikan}>Hentikan suara</Tombol>
        </div>
      </Kartu>
    </div>

    <Kartu>
      <h2 class="mb-3 text-lg font-semibold text-slate-900">Jadwal hari ini</h2>
      {#if status.bel.length === 0}
        <Kosong pesan="Tidak ada jadwal bel hari ini" />
      {:else}
        <ul class="space-y-2">
          {#each status.bel as b (b.jadwalId)}
            <li
              class="flex min-h-11 items-center gap-3 rounded-xl px-3 py-2 {b.jadwalId === berikutnyaId ? 'ring-2 ring-blue-600' : 'ring-1 ring-slate-200'} {b.sudah ? 'opacity-50' : ''}"
            >
              <span class="size-3 shrink-0 rounded-full" style="background-color: {b.jenisBel?.warna ?? '#94a3b8'}" aria-hidden="true"></span>
              <span class="font-semibold tabular-nums">{b.jam}</span>
              <span class="min-w-0 flex-1 truncate">{namaBel(b)}</span>
              {#if b.sudah}
                <svg class="size-5 shrink-0 text-emerald-600" viewBox="0 0 20 20" fill="currentColor" role="img" aria-label="sudah berbunyi">
                  <path fill-rule="evenodd" d="M16.7 5.3a1 1 0 0 1 0 1.4l-7.5 7.5a1 1 0 0 1-1.4 0L3.3 9.7a1 1 0 1 1 1.4-1.4l3.8 3.8 6.8-6.8a1 1 0 0 1 1.4 0Z" clip-rule="evenodd" />
                </svg>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </Kartu>
  </div>
{/if}
