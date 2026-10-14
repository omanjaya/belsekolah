<script lang="ts">
  import { onMount } from 'svelte'
  import { ArrowDown, ArrowUp, X } from '@lucide/svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import BarAlat from '../components/BarAlat.svelte'
  import Tabel from '../components/Tabel.svelte'
  import Memuat from '../components/Memuat.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import PanelGeser from '../components/PanelGeser.svelte'
  import Medan from '../components/Medan.svelte'
  import Tombol from '../components/Tombol.svelte'
  import { api, type Audio, type JenisBel } from '../lib/api'
  import { formatDurasi } from '../lib/format'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import { beriTahu, galat } from '../lib/notif.svelte'

  const WARNA = [
    { nama: 'Biru', nilai: '#2563eb' },
    { nama: 'Hijau Tosca', nilai: '#0d9488' },
    { nama: 'Hijau', nilai: '#16a34a' },
    { nama: 'Ungu', nilai: '#9333ea' },
    { nama: 'Merah', nilai: '#dc2626' },
    { nama: 'Jingga', nilai: '#f59e0b' },
    { nama: 'Merah Muda', nilai: '#db2777' },
    { nama: 'Abu', nilai: '#475569' },
  ]
  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const kelasIkon =
    'inline-grid size-11 shrink-0 place-items-center rounded-lg text-slate-700 hover:bg-slate-100 disabled:opacity-40'

  let daftar = $state<JenisBel[]>([])
  let audio = $state<Audio[]>([])
  let memuat = $state(true)
  let galatMuat = $state('')

  let formBuka = $state(false)
  let idUbah = $state<number | null>(null)
  let nama = $state('')
  let warna = $state(WARNA[0].nilai)
  let urutan = $state<{ uid: number; id: number }[]>([])
  let uidTerakhir = 0
  const butir = (id: number) => ({ uid: ++uidTerakhir, id })
  let galatForm = $state('')
  let galatNama = $state('')
  let galatSuara = $state('')
  let menyimpan = $state(false)
  let tesId = $state<number | null>(null)

  const audioPer = $derived(new Map(audio.map((a) => [a.id, a])))
  const namaAudio = (id: number) => audioPer.get(id)?.nama ?? `Suara #${id}`

  const totalIds = (ids: number[]) => formatDurasi(ids.reduce((t, id) => t + (audioPer.get(id)?.durasiMs ?? 0), 0))

  async function muat() {
    galatMuat = ''
    memuat = true
    try {
      const [j, a] = await Promise.all([api.get<JenisBel[]>('/jenis-bel'), api.get<Audio[]>('/audio')])
      daftar = j
      audio = a
      galatMuat = ''
    } catch (e) {
      galatMuat = e instanceof Error ? e.message : String(e)
    } finally {
      memuat = false
    }
  }
  onMount(muat)

  function bukaForm(j?: JenisBel) {
    idUbah = j?.id ?? null
    nama = j?.nama ?? ''
    warna = j?.warna ?? WARNA[0].nilai
    urutan = j ? j.audioIds.map(butir) : []
    galatForm = ''
    galatNama = ''
    galatSuara = ''
    formBuka = true
  }

  function geser(i: number, arah: -1 | 1) {
    const k = i + arah
    if (k < 0 || k >= urutan.length) return
    const baru = [...urutan]
    ;[baru[i], baru[k]] = [baru[k], baru[i]]
    urutan = baru
  }

  function tambahSuara(e: Event) {
    const sel = e.currentTarget as HTMLSelectElement
    if (sel.value) {
      urutan = [...urutan, butir(Number(sel.value))]
      galatSuara = ''
    }
    sel.value = ''
  }

  async function simpan(e: SubmitEvent) {
    e.preventDefault()
    galatNama = nama.trim() ? '' : 'Nama wajib diisi'
    galatSuara = urutan.length ? '' : 'Pilih minimal satu suara'
    if (galatNama || galatSuara) return
    galatForm = ''
    menyimpan = true
    try {
      const badan = { nama: nama.trim(), warna, audioIds: urutan.map((u) => u.id) }
      if (idUbah === null) await api.post('/jenis-bel', badan)
      else await api.put(`/jenis-bel/${idUbah}`, badan)
      beriTahu(idUbah === null ? 'Jenis bel ditambahkan' : 'Jenis bel disimpan')
      formBuka = false
      await muat()
    } catch (err) {
      galatForm = err instanceof Error ? err.message : String(err)
    } finally {
      menyimpan = false
    }
  }

  async function tes(j: JenisBel) {
    tesId = j.id
    try {
      await api.post(`/jenis-bel/${j.id}/tes`)
      beriTahu('Suara diputar di speaker')
    } catch (e) {
      galat(e)
    } finally {
      tesId = null
    }
  }

  async function hapus(j: JenisBel) {
    if (!(await konfirmasi('Hapus jenis bel?', `Jenis bel "${j.nama}" akan dihapus.`))) return
    try {
      await api.hapus(`/jenis-bel/${j.id}`)
      beriTahu('Jenis bel dihapus')
      await muat()
    } catch (e) {
      galat(e)
    }
  }
</script>

<JudulHalaman judul="Jenis Bel" keterangan="Setiap jenis bel adalah urutan suara yang diputar berurutan." />

{#if memuat}
  <Memuat />
{:else if galatMuat}
  <GalatMuat pesan={galatMuat} oncoba={muat} />
{:else}
  <BarAlat>
    {#snippet kanan()}
      <Tombol onclick={() => bukaForm()}>+ Tambah jenis bel</Tombol>
    {/snippet}
  </BarAlat>

  <div class="hidden md:block">
  <Tabel label="Daftar jenis bel" jumlah={daftar.length} kosong="Belum ada jenis bel. Tambahkan yang pertama.">
    {#snippet kepala()}
      <th class="w-10"><span class="sr-only">Warna</span></th>
      <th>Nama</th>
      <th>Urutan suara</th>
      <th class="text-right">Aksi</th>
    {/snippet}
    {#each daftar as j (j.id)}
      <tr>
        <td><span class="block size-4 rounded-full" style:background-color={j.warna} aria-hidden="true"></span></td>
        <td class="font-semibold break-words">{j.nama}</td>
        <td class="break-words text-slate-600">{j.audioIds.map((id) => namaAudio(id)).join(' → ') || '-'}</td>
        <td class="text-right whitespace-nowrap">
          <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100 disabled:opacity-50" disabled={tesId === j.id} aria-label="Tes di speaker {j.nama}" onclick={() => tes(j)}>Tes</button>
          <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100" aria-label="Ubah {j.nama}" onclick={() => bukaForm(j)}>Ubah</button>
          <button type="button" class="min-h-11 rounded-lg px-3 text-red-700 hover:bg-red-50" aria-label="Hapus {j.nama}" onclick={() => hapus(j)}>Hapus</button>
        </td>
      </tr>
    {/each}
  </Tabel>
  </div>

  <ul class="flex flex-col gap-2 md:hidden" aria-label="Daftar jenis bel">
    {#each daftar as j (j.id)}
      <li class="flex overflow-hidden rounded-xl border border-slate-200 bg-white">
        <span class="w-2 shrink-0" style:background-color={j.warna} aria-hidden="true"></span>
        <div class="min-w-0 flex-1 p-3">
          <p class="break-words font-semibold text-slate-900">{j.nama}</p>
          <p class="mt-0.5 break-words text-sm text-slate-600">{j.audioIds.map((id) => namaAudio(id)).join(' → ') || '-'}</p>
          <p class="mt-0.5 text-xs text-slate-500">Total {totalIds(j.audioIds)}</p>
          <div class="mt-2 flex flex-wrap gap-1">
            <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100 disabled:opacity-50" disabled={tesId === j.id} aria-label="Tes di speaker {j.nama}" onclick={() => tes(j)}>Tes</button>
            <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100" aria-label="Ubah {j.nama}" onclick={() => bukaForm(j)}>Ubah</button>
            <button type="button" class="min-h-11 rounded-lg px-3 text-red-700 hover:bg-red-50" aria-label="Hapus {j.nama}" onclick={() => hapus(j)}>Hapus</button>
          </div>
        </div>
      </li>
    {:else}
      <li class="rounded-xl border border-slate-200 bg-white px-4 py-6 text-center text-slate-600">Belum ada jenis bel. Tambahkan yang pertama.</li>
    {/each}
  </ul>
{/if}

<PanelGeser bind:buka={formBuka} judul={idUbah === null ? 'Tambah jenis bel' : 'Ubah jenis bel'}>
  <form id="form-jenis" onsubmit={simpan} class="flex flex-col gap-4" novalidate>
    <Medan label="Nama" untuk="jb-nama" galat={galatNama}>
      <input id="jb-nama" class={kelasInput} bind:value={nama} autocomplete="off" aria-invalid={!!galatNama} aria-describedby={galatNama ? 'jb-nama-galat' : undefined} />
    </Medan>

    <div class="flex flex-col gap-1.5">
      <span id="jb-warna" class="text-sm font-medium text-slate-800">Warna</span>
      <div role="radiogroup" aria-labelledby="jb-warna" class="flex flex-wrap gap-1">
        {#each WARNA as w (w.nilai)}
          <button
            type="button"
            role="radio"
            aria-checked={warna === w.nilai}
            aria-label={w.nama}
            onclick={() => (warna = w.nilai)}
            class="grid size-11 place-items-center rounded-full focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-blue-700"
          >
            <span
              class="size-8 rounded-full ring-offset-2 {warna === w.nilai ? 'ring-2 ring-slate-900' : 'ring-1 ring-slate-200'}"
              style:background-color={w.nilai}
            ></span>
          </button>
        {/each}
      </div>
    </div>

    <div class="flex flex-col gap-2">
      <span class="text-sm font-medium text-slate-800">Urutan suara</span>
      {#if urutan.length === 0}
        <p class="text-sm text-slate-500">Belum ada suara.</p>
      {:else}
        <ol class="flex flex-col gap-1">
          {#each urutan as { uid, id }, i (uid)}
            <li class="flex items-center gap-1 rounded-xl bg-slate-50 pl-3 ring-1 ring-slate-200">
              <span class="min-w-0 flex-1 break-words text-sm">{i + 1}. {namaAudio(id)}</span>
              <button type="button" class={kelasIkon} disabled={i === 0} onclick={() => geser(i, -1)} aria-label="Naikkan {namaAudio(id)}"><ArrowUp class="size-5" /></button>
              <button type="button" class={kelasIkon} disabled={i === urutan.length - 1} onclick={() => geser(i, 1)} aria-label="Turunkan {namaAudio(id)}"><ArrowDown class="size-5" /></button>
              <button type="button" class={kelasIkon} onclick={() => (urutan = urutan.filter((_, k) => k !== i))} aria-label="Buang {namaAudio(id)}"><X class="size-5" /></button>
            </li>
          {/each}
        </ol>
      {/if}
      {#if urutan.length}<p class="text-sm text-slate-600">Total durasi {totalIds(urutan.map((u) => u.id))}</p>{/if}
      {#if galatSuara}<p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatSuara}</p>{/if}
      <Medan label="Tambah suara" untuk="jb-tambah">
        <select id="jb-tambah" class={kelasInput} onchange={tambahSuara}>
          <option value="">Pilih suara…</option>
          {#each audio as a (a.id)}<option value={a.id}>{a.nama}</option>{/each}
        </select>
      </Medan>
    </div>
    {#if galatForm}<p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatForm}</p>{/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (formBuka = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-jenis" sibuk={menyimpan}>Simpan</Tombol>
  {/snippet}
</PanelGeser>
