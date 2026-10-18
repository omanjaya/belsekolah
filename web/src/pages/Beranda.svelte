<script lang="ts">
  import { onMount } from 'svelte'
  import { api, type JenisBel, type Status, type BelHariIni, type Sistem } from '../lib/api'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { hitungMundur } from '../lib/format'
  import { perbaruiDariStatus } from '../lib/jamServer.svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kartu from '../components/Kartu.svelte'
  import KartuStatistik from '../components/KartuStatistik.svelte'
  import Tabel from '../components/Tabel.svelte'
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

  let sistem = $state<Sistem | null>(null)
  let sibukManual = $state<number | null>(null)
  let sibukStop = $state(false)
  let sibukDarurat = $state(false)
  let progres = $state(0)
  let menahan = $state(false)
  let timerTahan: ReturnType<typeof setInterval> | undefined
  let mulaiTahan = 0

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
      perbaruiDariStatus(s)
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
    } catch (e) {
      galat(e)
    }
  }

  async function muatSistem() {
    try {
      sistem = await api.get<Sistem>('/sistem')
    } catch {
      // kartu status sistem menampilkan "—" bila gagal
    }
  }

  onMount(() => {
    muat()
    muatJenis()
    muatSistem()
    const poll = setInterval(() => {
      if (!document.hidden) muat()
    }, 5000)
    const pollSistem = setInterval(() => {
      if (!document.hidden) muatSistem()
    }, 30000)
    const detik = setInterval(() => {
      if (status) sekarangMs = dasarMs + (performance.now() - dasarPerf)
    }, 1000)
    const terlihat = () => {
      if (!document.hidden) muat()
    }
    document.addEventListener('visibilitychange', terlihat)
    return () => {
      clearInterval(poll)
      clearInterval(pollSistem)
      clearInterval(detik)
      batalTahan()
      document.removeEventListener('visibilitychange', terlihat)
    }
  })

  async function bunyikan(id: number) {
    sibukManual = id
    try {
      await api.post('/bel/manual', { jenisBelId: id })
      beriTahu('Bel dibunyikan')
    } catch (e) {
      galat(e)
    } finally {
      sibukManual = null
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
  const statusSistem = $derived.by(() => {
    const ntp = sistem?.ntp
    return {
      pemutar: !sistem?.alat ? '—' : sistem.alat.mpv ? 'mpv tersedia' : 'mpv tidak ada',
      nadaPemutar: !sistem?.alat ? 'biasa' : sistem.alat.mpv ? 'baik' : 'bahaya',
      ntp: ntp == null ? 'Tidak diketahui' : ntp ? 'Tersinkron' : 'Tidak sinkron',
      nadaNtp: ntp == null ? 'biasa' : ntp ? 'baik' : 'peringatan',
      disk: sistem?.ruangKosongMb == null ? '—' : `${sistem.ruangKosongMb.toLocaleString('id-ID')} MB`,
      backup: sistem?.backupTerakhir ? new Date(sistem.backupTerakhir).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' }) : 'Belum ada',
    } as const
  })
  const statusBaris = (b: BelHariIni) => (b.sudah ? 'Sudah' : b.jadwalId === berikutnyaId ? 'Berikutnya' : 'Menunggu')
</script>


<JudulHalaman judul="Beranda" />

{#if !status}
  {#if galatMuat}
    <GalatMuat pesan={galatMuat} oncoba={muat} />
  {:else}
    <Memuat />
  {/if}
{:else}
  <div class="space-y-4 md:space-y-6">
    {#if status.peringatan.length > 0}
      <div class="space-y-2">
        {#each status.peringatan as p, i (`${i}-${p}`)}
          <div role="alert" class="rounded-xl bg-amber-50 px-4 py-3 text-amber-900 ring-1 ring-amber-300">{p}</div>
        {/each}
      </div>
    {/if}
    {#if gagalBerturut >= 2}
      <p role="status" class="rounded-xl bg-amber-50 px-3 py-2 text-sm font-medium text-amber-900 ring-1 ring-amber-300">Koneksi ke server terputus — data mungkin tidak terbaru</p>
    {/if}

    <div data-testid="beranda-grid" class="grid grid-cols-1 gap-4 md:gap-6 lg:grid-cols-3">
      <Kartu class="lg:col-span-2">
        <h2 class="mb-2 text-sm font-semibold uppercase tracking-wide text-slate-500">Bel berikutnya</h2>
        {#if status.berikutnya}
          <p class="text-4xl font-bold tabular-nums text-slate-900">{status.berikutnya.jam}</p>
          <p class="mt-1 text-lg text-slate-800">{namaBel(status.berikutnya)}</p>
          {#if status.berikutnya.pengumuman}
            <p class="text-sm text-slate-600">+ {status.berikutnya.pengumuman.nama}</p>
          {/if}
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
        {#if status.libur}
          <p class="mt-3 rounded-xl bg-amber-50 px-3 py-2 font-medium text-amber-900 ring-1 ring-amber-300">Hari libur: {status.libur.keterangan}</p>
        {:else if status.profil}
          <p class="mt-3 inline-block rounded-full bg-blue-50 px-3 py-1 text-sm font-medium text-blue-800">Profil: {status.profil.nama}</p>
        {/if}
      </Kartu>

      <Kartu>
        <h2 class="mb-3 text-lg font-semibold text-slate-900">Bel manual</h2>
        <div class="space-y-3">
          <div class="grid grid-cols-2 gap-2">
            {#each jenisManual as j (j.id)}
              <Tombol varian="kedua" class="w-full" sibuk={sibukManual === j.id} onclick={() => bunyikan(j.id)}>
                <span class="size-3 shrink-0 rounded-full" style="background-color: {j.warna}" aria-hidden="true"></span>
                {j.nama}
              </Tombol>
            {/each}
          </div>

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

      <div class="lg:col-span-2">
        <h2 class="mb-2 text-lg font-semibold text-slate-900">Jadwal hari ini</h2>
        <Tabel label="Jadwal hari ini" jumlah={status.bel.length} kosong="Tidak ada jadwal bel hari ini">
          {#snippet kepala()}
            <th scope="col">Jam</th>
            <th scope="col">Keterangan</th>
            <th scope="col">Jenis bel</th>
            <th scope="col">Status</th>
          {/snippet}
          {#each status.bel as b (b.jadwalId)}
            <tr class={b.jadwalId === berikutnyaId ? 'bg-blue-50' : b.sudah ? 'text-slate-500' : ''}>
              <td class="font-semibold tabular-nums">{b.jam}</td>
              <td>{namaBel(b)}</td>
              <td>
                <span class="inline-flex items-center gap-2">
                  <span class="size-3 shrink-0 rounded-full" style="background-color: {b.jenisBel?.warna ?? '#94a3b8'}" aria-hidden="true"></span>
                  {b.jenisBel?.nama ?? '—'}
                </span>
              </td>
              <td class={b.jadwalId === berikutnyaId ? 'font-semibold text-blue-700' : ''}>{statusBaris(b)}</td>
            </tr>
          {/each}
        </Tabel>
      </div>

      <section aria-labelledby="judul-status-sistem">
        <h2 id="judul-status-sistem" class="mb-2 text-lg font-semibold text-slate-900">Status sistem</h2>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-1">
          <KartuStatistik label="Pemutar" nilai={statusSistem.pemutar} nada={statusSistem.nadaPemutar} />
          <KartuStatistik label="Sinkron jam (NTP)" nilai={statusSistem.ntp} nada={statusSistem.nadaNtp} />
          <KartuStatistik label="Ruang disk" nilai={statusSistem.disk} />
          <KartuStatistik label="Backup terakhir" nilai={statusSistem.backup} />
        </div>
      </section>
    </div>
  </div>
{/if}
