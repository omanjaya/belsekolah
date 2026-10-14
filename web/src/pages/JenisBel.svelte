<script lang="ts">
  import { onMount } from 'svelte'
  import { ArrowDown, ArrowUp, Pencil, Plus, Trash2, Volume2, X } from '@lucide/svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kosong from '../components/Kosong.svelte'
  import Memuat from '../components/Memuat.svelte'
  import Modal from '../components/Modal.svelte'
  import Medan from '../components/Medan.svelte'
  import Tombol from '../components/Tombol.svelte'
  import { api, ApiError, type Audio, type JenisBel } from '../lib/api'
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

  async function muat() {
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

  function total(j: JenisBel) {
    return formatDurasi(j.audioIds.reduce((s, id) => s + (audioPer.get(id)?.durasiMs ?? 0), 0))
  }

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
      if (e instanceof ApiError && e.status === 409) beriTahu('jenis bel masih dipakai di jadwal', 'galat')
      else galat(e)
    }
  }
</script>

<JudulHalaman judul="Jenis Bel" keterangan="Setiap jenis bel adalah urutan suara yang diputar berurutan.">
  {#snippet aksi()}
    <Tombol onclick={() => bukaForm()}><Plus class="size-5" /> Tambah jenis bel</Tombol>
  {/snippet}
</JudulHalaman>

{#if memuat}
  <Memuat />
{:else if galatMuat}
  <p role="alert" class="rounded-xl bg-red-50 px-4 py-3 text-red-700">{galatMuat}</p>
{:else if daftar.length === 0}
  <Kosong pesan="Belum ada jenis bel. Tambahkan yang pertama." />
{:else}
  <ul class="grid gap-3 md:grid-cols-2">
    {#each daftar as j (j.id)}
      <li class="flex overflow-hidden rounded-2xl bg-white shadow-sm ring-1 ring-slate-200">
        <span class="w-2 shrink-0" style:background-color={j.warna} aria-hidden="true"></span>
        <div class="min-w-0 flex-1 p-4">
          <h2 class="break-words text-lg font-semibold text-slate-900">{j.nama}</h2>
          <p class="mt-1 break-words text-sm text-slate-600">
            {j.audioIds.map((id, i) => `${i + 1}. ${namaAudio(id)}`).join(', ')}
          </p>
          <p class="mt-1 text-sm text-slate-500">Total {total(j)}</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <Tombol varian="kedua" sibuk={tesId === j.id} onclick={() => tes(j)} aria-label="Tes di speaker {j.nama}">
              <Volume2 class="size-4" /> Tes di speaker
            </Tombol>
            <Tombol varian="kedua" onclick={() => bukaForm(j)} aria-label="Ubah {j.nama}">
              <Pencil class="size-4" /> Ubah
            </Tombol>
            <Tombol varian="hantu" onclick={() => hapus(j)} aria-label="Hapus {j.nama}">
              <Trash2 class="size-4" /> Hapus
            </Tombol>
          </div>
        </div>
      </li>
    {/each}
  </ul>
{/if}

<Modal bind:buka={formBuka} judul={idUbah === null ? 'Tambah jenis bel' : 'Ubah jenis bel'}>
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
      {#if galatSuara}<p class="text-sm text-red-600" role="alert">{galatSuara}</p>{/if}
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
</Modal>
