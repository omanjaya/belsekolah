<script lang="ts">
  import { onMount } from 'svelte'
  import { Pencil, Trash2 } from '@lucide/svelte'
  import { api, ApiError, type Jadwal, type JenisBel, type Pengaturan, type Profil, type Status } from '../lib/api'
  import { NAMA_HARI } from '../lib/format'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Tombol from '../components/Tombol.svelte'
  import Modal from '../components/Modal.svelte'
  import Medan from '../components/Medan.svelte'
  import Kartu from '../components/Kartu.svelte'
  import Kosong from '../components/Kosong.svelte'
  import Memuat from '../components/Memuat.svelte'

  const masukan =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const HARI = [1, 2, 3, 4, 5, 6, 7]

  let profil = $state<Profil[]>([])
  let jenis = $state<JenisBel[]>([])
  let defaultId = $state(0)
  let profilId = $state(0)
  let hari = $state(1)
  let semua = $state<Jadwal[]>([])
  let memuat = $state(true)
  let galatMuat = $state('')
  let memuatDaftar = $state(false)
  let galatDaftar = $state(false)
  let urutMuat = 0

  const jam5 = (j: string) => j.slice(0, 5)
  const hariIni = $derived(semua.filter((j) => j.hari === hari).sort((a, b) => jam5(a.jam).localeCompare(jam5(b.jam))))
  const jumlah = (h: number) => semua.filter((j) => j.hari === h).length
  const jenisPer = (id: number) => jenis.find((j) => j.id === id)

  async function muatJadwal(kosongkan = false) {
    if (!profilId) return
    const no = ++urutMuat
    if (kosongkan) {
      semua = []
      memuatDaftar = true
    }
    galatDaftar = false
    try {
      const data = await api.get<Jadwal[]>(`/jadwal?profil=${profilId}`)
      if (no !== urutMuat) return
      semua = data
    } catch (e) {
      if (no !== urutMuat) return
      galatDaftar = true
      galat(e)
    } finally {
      if (no === urutMuat) memuatDaftar = false
    }
  }

  onMount(async () => {
    try {
      const stP = api.get<Status>('/status').catch(() => null)
      const [p, j, s, st] = await Promise.all([
        api.get<Profil[]>('/profil'),
        api.get<JenisBel[]>('/jenis-bel'),
        api.get<Pengaturan>('/pengaturan'),
        stP,
      ])
      profil = p
      jenis = j
      defaultId = s.profilDefaultId
      profilId = p.some((x) => x.id === s.profilDefaultId) ? s.profilDefaultId : (p[0]?.id ?? 0)
      if (st && st.hari >= 1 && st.hari <= 7) hari = st.hari
      await muatJadwal(true)
    } catch (e) {
      galatMuat = e instanceof Error ? e.message : String(e)
    } finally {
      memuat = false
    }
  })

  // Modal tambah/ubah
  let bukaForm = $state(false)
  let ubahId = $state<number | null>(null)
  let fJam = $state('')
  let fJenis = $state(0)
  let fLabel = $state('')
  let galatForm = $state('')
  let simpan = $state(false)

  function bukaTambah() {
    ubahId = null
    fJam = ''
    fJenis = jenis[0]?.id ?? 0
    fLabel = ''
    galatForm = ''
    bukaForm = true
  }

  function bukaUbah(j: Jadwal) {
    ubahId = j.id
    fJam = jam5(j.jam)
    fJenis = j.jenisBelId
    fLabel = j.label
    galatForm = ''
    bukaForm = true
  }

  async function kirimForm(e: SubmitEvent) {
    e.preventDefault()
    galatForm = ''
    if (!fJam) return (galatForm = 'Jam wajib diisi.')
    if (!fJenis) return (galatForm = 'Jenis bel wajib dipilih.')
    simpan = true
    try {
      const data = { profilId, hari, jam: fJam.slice(0, 5), jenisBelId: fJenis, label: fLabel.trim() }
      if (ubahId === null) await api.post('/jadwal', data)
      else await api.put(`/jadwal/${ubahId}`, data)
      bukaForm = false
      beriTahu('Bel disimpan')
      await muatJadwal()
    } catch (err) {
      galatForm = err instanceof ApiError ? err.message : String(err)
    } finally {
      simpan = false
    }
  }

  async function hapus(j: Jadwal) {
    const ya = await konfirmasi('Hapus bel?', `Bel ${jam5(j.jam)} pada hari ${NAMA_HARI[hari]} akan dihapus.`)
    if (!ya) return
    try {
      await api.hapus(`/jadwal/${j.id}`)
      beriTahu('Bel dihapus')
      await muatJadwal()
    } catch (e) {
      galat(e)
    }
  }

  // Modal salin
  let bukaSalin = $state(false)
  let dariHari = $state(2)
  let salin = $state(false)
  let galatSalin = $state('')

  function bukaSalinan() {
    dariHari = HARI.find((h) => h !== hari) ?? 1
    galatSalin = ''
    bukaSalin = true
  }

  async function kirimSalin(e: SubmitEvent) {
    e.preventDefault()
    salin = true
    galatSalin = ''
    try {
      await api.post('/jadwal/salin', { profilId, dariHari, keHari: hari })
      bukaSalin = false
      beriTahu('Jadwal disalin')
      await muatJadwal()
    } catch (err) {
      galatSalin = err instanceof Error ? err.message : String(err)
    } finally {
      salin = false
    }
  }
</script>

<JudulHalaman judul="Jadwal" keterangan="Atur jam bel untuk setiap hari.">
  {#snippet aksi()}
    <Tombol onclick={bukaTambah} disabled={memuat || !profilId}>Tambah bel</Tombol>
    <Tombol varian="kedua" onclick={bukaSalinan} disabled={memuat || !profilId}>Salin dari hari lain</Tombol>
  {/snippet}
</JudulHalaman>

{#if memuat}
  <Memuat />
{:else if galatMuat}
  <p class="rounded-xl bg-red-50 px-4 py-3 text-red-700" role="alert">{galatMuat}</p>
{:else}
  <div class="mb-4 max-w-sm">
    <Medan label="Profil" untuk="jadwal-profil">
      <select id="jadwal-profil" class={masukan} bind:value={profilId} onchange={() => muatJadwal(true)}>
        {#each profil as p (p.id)}
          <option value={p.id}>{p.nama}{p.id === defaultId ? ' (default)' : ''}</option>
        {/each}
      </select>
    </Medan>
  </div>

  <div class="-mx-4 mb-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
    <div role="tablist" aria-label="Hari" class="flex w-max gap-2">
      {#each HARI as h (h)}
        <button
          type="button"
          role="tab"
          id="tab-hari-{h}"
          aria-selected={hari === h}
          aria-controls="panel-hari"
          class="min-h-11 shrink-0 rounded-xl px-4 py-2 font-medium transition {hari === h
            ? 'bg-blue-700 text-white'
            : 'bg-white text-slate-700 ring-1 ring-slate-300 hover:bg-slate-50'}"
          onclick={() => (hari = h)}
        >
          {NAMA_HARI[h].slice(0, 3)} {jumlah(h)}
        </button>
      {/each}
    </div>
  </div>

  <Kartu>
    <div id="panel-hari" role="tabpanel" aria-labelledby="tab-hari-{hari}">
      <h2 class="mb-3 text-lg font-semibold">{NAMA_HARI[hari]}</h2>
      {#if memuatDaftar}
        <Memuat />
      {:else if galatDaftar}
        <div class="flex flex-col items-center gap-3 py-6">
          <p class="text-slate-600">Daftar bel gagal dimuat.</p>
          <Tombol varian="kedua" onclick={() => muatJadwal(true)}>Coba lagi</Tombol>
        </div>
      {:else if hariIni.length === 0}
        <Kosong pesan="Belum ada bel pada hari {NAMA_HARI[hari]}" />
      {:else}
        <ul class="divide-y divide-slate-200">
          {#each hariIni as j (j.id)}
            {@const jb = jenisPer(j.jenisBelId)}
            <li class="flex items-center gap-3 py-2">
              <span class="w-14 shrink-0 text-lg font-bold tabular-nums">{jam5(j.jam)}</span>
              <div class="min-w-0 flex-1">
                <p class="truncate">{j.label || jb?.nama || 'Bel'}</p>
                {#if jb}
                  <span
                    class="mt-0.5 inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium text-slate-800 ring-1 ring-slate-300"
                  >
                    <span class="size-2.5 rounded-full" style="background-color: {jb.warna}" aria-hidden="true"></span>
                    {jb.nama}
                  </span>
                {/if}
              </div>
              <button
                type="button"
                class="inline-grid size-11 shrink-0 place-items-center rounded-lg text-slate-700 hover:bg-slate-100"
                aria-label="Ubah {jam5(j.jam)}"
                onclick={() => bukaUbah(j)}
              >
                <Pencil class="size-5" />
              </button>
              <button
                type="button"
                class="inline-grid size-11 shrink-0 place-items-center rounded-lg text-red-600 hover:bg-red-50"
                aria-label="Hapus {jam5(j.jam)}"
                onclick={() => hapus(j)}
              >
                <Trash2 class="size-5" />
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  </Kartu>
{/if}

<Modal bind:buka={bukaForm} judul={ubahId === null ? 'Tambah bel' : 'Ubah bel'}>
  <form id="form-bel" class="flex flex-col gap-4" onsubmit={kirimForm}>
    <p class="text-sm text-slate-600">Hari {NAMA_HARI[hari]}</p>
    <Medan label="Jam" untuk="bel-jam">
      <input id="bel-jam" type="time" required class={masukan} bind:value={fJam} />
    </Medan>
    <Medan label="Jenis bel" untuk="bel-jenis">
      <select id="bel-jenis" required class={masukan} bind:value={fJenis}>
        {#each jenis as j (j.id)}
          <option value={j.id}>{j.nama}</option>
        {/each}
      </select>
    </Medan>
    <Medan label="Label" untuk="bel-label" bantuan="Opsional, maksimal 100 karakter.">
      <input id="bel-label" type="text" maxlength={100} class={masukan} bind:value={fLabel} />
    </Medan>
    {#if galatForm}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatForm}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaForm = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-bel" sibuk={simpan}>Simpan</Tombol>
  {/snippet}
</Modal>

<Modal bind:buka={bukaSalin} judul="Salin dari hari lain">
  <form id="form-salin" class="flex flex-col gap-4" onsubmit={kirimSalin}>
    <Medan label="Salin dari" untuk="salin-dari">
      <select id="salin-dari" class={masukan} bind:value={dariHari}>
        {#each HARI.filter((h) => h !== hari) as h (h)}
          <option value={h}>{NAMA_HARI[h]}</option>
        {/each}
      </select>
    </Medan>
    <p class="rounded-xl bg-amber-50 px-3 py-2 text-sm text-amber-800">
      Semua bel pada hari {NAMA_HARI[hari]} akan diganti.{#if jumlah(dariHari) === 0}
        Hari {NAMA_HARI[dariHari]} belum punya bel; semua bel hari {NAMA_HARI[hari]} akan dihapus.{/if}
    </p>
    {#if galatSalin}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatSalin}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaSalin = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-salin" sibuk={salin}>Salin</Tombol>
  {/snippet}
</Modal>
