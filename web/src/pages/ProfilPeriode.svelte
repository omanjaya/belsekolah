<script lang="ts">
  import { onMount } from 'svelte'
  import { Pencil, Trash2 } from '@lucide/svelte'
  import { api, type Pengaturan, type Periode, type Profil, type Status } from '../lib/api'
  import { formatRentang } from '../lib/format'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kartu from '../components/Kartu.svelte'
  import Kosong from '../components/Kosong.svelte'
  import Memuat from '../components/Memuat.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import Modal from '../components/Modal.svelte'
  import Medan from '../components/Medan.svelte'
  import Tombol from '../components/Tombol.svelte'

  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const kelasIkon = 'inline-grid size-11 place-items-center rounded-lg text-slate-700 hover:bg-slate-100'

  let memuat = $state(true)
  let profil = $state<Profil[]>([])
  let periode = $state<Periode[]>([])
  let pengaturan = $state<Pengaturan>({ volume: 100, profilDefaultId: 0, zonaWaktu: '' })
  let hariIni = $state('')
  let galatMuat = $state('')

  async function muat() {
    galatMuat = ''
    memuat = true
    try {
      const [p, per, pg, st] = await Promise.all([
        api.get<Profil[]>('/profil'),
        api.get<Periode[]>('/periode'),
        api.get<Pengaturan>('/pengaturan'),
        api.get<Status>('/status'),
      ])
      profil = p
      periode = [...per].sort((a, b) => a.mulai.localeCompare(b.mulai))
      pengaturan = pg
      hariIni = st.tanggal
    } catch (e) {
      galatMuat = e instanceof Error ? e.message : String(e)
    } finally {
      memuat = false
    }
  }
  onMount(muat)

  const namaProfil = (id: number) => profil.find((p) => p.id === id)?.nama ?? '—'

  // --- Profil ---
  let namaBaru = $state('')
  let sibukTambah = $state(false)

  async function tambahProfil(e: SubmitEvent) {
    e.preventDefault()
    sibukTambah = true
    try {
      await api.post('/profil', { nama: namaBaru })
      namaBaru = ''
      beriTahu('Profil ditambahkan')
      await muat()
    } catch (err) {
      galat(err)
    } finally {
      sibukTambah = false
    }
  }

  async function jadikanDefault(p: Profil) {
    try {
      const terbaru = await api.get<Pengaturan>('/pengaturan')
      await api.put('/pengaturan', { volume: terbaru.volume, profilDefaultId: p.id })
      beriTahu('Profil default diubah')
      await muat()
    } catch (err) {
      galat(err)
    }
  }

  async function hapusProfil(p: Profil) {
    if (!(await konfirmasi('Hapus profil?', `Profil "${p.nama}" beserta jadwalnya akan dihapus.`))) return
    try {
      await api.hapus(`/profil/${p.id}`)
      beriTahu('Profil dihapus')
      await muat()
    } catch (err) {
      galat(err)
    }
  }

  let bukaNama = $state(false)
  let ubahId = $state(0)
  let ubahNama = $state('')
  let sibukNama = $state(false)
  let galatNama = $state('')

  function bukaUbahNama(p: Profil) {
    ubahId = p.id
    ubahNama = p.nama
    galatNama = ''
    bukaNama = true
  }

  async function simpanNama(e: SubmitEvent) {
    e.preventDefault()
    galatNama = ''
    sibukNama = true
    try {
      await api.put(`/profil/${ubahId}`, { nama: ubahNama })
      bukaNama = false
      beriTahu('Nama profil disimpan')
      await muat()
    } catch (err) {
      galatNama = err instanceof Error ? err.message : String(err)
    } finally {
      sibukNama = false
    }
  }

  // --- Periode ---
  let bukaPeriode = $state(false)
  let periodeId = $state(0)
  let fMulai = $state('')
  let fSelesai = $state('')
  let fProfil = $state(0)
  let fKet = $state('')
  let galatPeriode = $state('')
  let sibukPeriode = $state(false)

  function bukaFormPeriode(p?: Periode) {
    periodeId = p?.id ?? 0
    fMulai = p?.mulai ?? hariIni
    fSelesai = p?.selesai ?? hariIni
    const adaDefault = profil.some((x) => x.id === pengaturan.profilDefaultId)
    fProfil = p?.profilId ?? (adaDefault ? pengaturan.profilDefaultId : (profil[0]?.id ?? 0))
    fKet = p?.keterangan ?? ''
    galatPeriode = ''
    bukaPeriode = true
  }

  async function simpanPeriode(e: SubmitEvent) {
    e.preventDefault()
    galatPeriode = ''
    sibukPeriode = true
    const data = { mulai: fMulai, selesai: fSelesai, profilId: Number(fProfil), keterangan: fKet }
    try {
      if (periodeId) await api.put(`/periode/${periodeId}`, data)
      else await api.post('/periode', data)
      bukaPeriode = false
      beriTahu('Periode disimpan')
      await muat()
    } catch (err) {
      galatPeriode = err instanceof Error ? err.message : String(err)
    } finally {
      sibukPeriode = false
    }
  }

  async function hapusPeriode(p: Periode) {
    if (!(await konfirmasi('Hapus periode?', `Periode ${formatRentang(p.mulai, p.selesai)} akan dihapus.`))) return
    try {
      await api.hapus(`/periode/${p.id}`)
      beriTahu('Periode dihapus')
      await muat()
    } catch (err) {
      galat(err)
    }
  }
</script>

<JudulHalaman
  judul="Profil & Periode"
  keterangan="Profil adalah kumpulan jadwal; periode khusus memakai profil tertentu pada tanggal tertentu."
/>

{#if memuat}
  <Memuat />
{:else if galatMuat}
  <GalatMuat pesan={galatMuat} oncoba={muat} />
{:else}
  <div class="grid gap-5 lg:grid-cols-2 lg:items-start">
    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Profil</h2>
      <form class="mb-4 flex flex-col gap-2 sm:flex-row sm:items-end" onsubmit={tambahProfil}>
        <div class="flex-1">
          <Medan label="Nama profil baru" untuk="profil-baru">
            <input id="profil-baru" class={kelasInput} bind:value={namaBaru} required maxlength="60" />
          </Medan>
        </div>
        <Tombol type="submit" sibuk={sibukTambah}>Tambah profil</Tombol>
      </form>
      {#if profil.length === 0}
        <Kosong pesan="Belum ada profil" />
      {:else}
        <ul class="divide-y divide-slate-200">
          {#each profil as p (p.id)}
            {@const isDefault = p.id === pengaturan.profilDefaultId}
            <li class="flex flex-wrap items-center gap-x-3 gap-y-1 py-2">
              <span class="min-w-0 flex-1 break-words font-medium">{p.nama}</span>
              {#if isDefault}
                <span class="rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-semibold text-blue-800">Default</span>
              {:else}
                <Tombol varian="kedua" class="text-sm" aria-label="Jadikan default {p.nama}" onclick={() => jadikanDefault(p)}>
                  Jadikan default
                </Tombol>
              {/if}
              <button type="button" class={kelasIkon} aria-label="Ubah nama {p.nama}" onclick={() => bukaUbahNama(p)}>
                <Pencil class="size-5" aria-hidden="true" />
              </button>
              <button type="button" class="{kelasIkon} text-red-700" aria-label="Hapus {p.nama}" onclick={() => hapusProfil(p)}>
                <Trash2 class="size-5" aria-hidden="true" />
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </Kartu>

    <Kartu>
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h2 class="text-lg font-semibold">Periode khusus</h2>
        <Tombol onclick={() => bukaFormPeriode()}>Tambah periode</Tombol>
      </div>
      {#if periode.length === 0}
        <Kosong pesan="Belum ada periode khusus" />
      {:else}
        <ul class="divide-y divide-slate-200">
          {#each periode as p (p.id)}
            <li class="flex flex-wrap items-center gap-x-3 gap-y-1 py-2">
              <div class="min-w-0 flex-1">
                <p class="font-medium">{formatRentang(p.mulai, p.selesai)}</p>
                <p class="break-words text-sm text-slate-600">
                  {namaProfil(p.profilId)}{p.keterangan ? ` · ${p.keterangan}` : ''}
                </p>
              </div>
              {#if hariIni >= p.mulai && hariIni <= p.selesai}
                <span class="rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-semibold text-green-800">Aktif</span>
              {:else if p.selesai < hariIni}
                <span class="rounded-full bg-slate-200 px-2.5 py-0.5 text-xs font-semibold text-slate-700">Selesai</span>
              {/if}
              <button type="button" class={kelasIkon} aria-label="Ubah periode {p.keterangan || formatRentang(p.mulai, p.selesai)}" onclick={() => bukaFormPeriode(p)}>
                <Pencil class="size-5" aria-hidden="true" />
              </button>
              <button type="button" class="{kelasIkon} text-red-700" aria-label="Hapus periode {p.keterangan || formatRentang(p.mulai, p.selesai)}" onclick={() => hapusPeriode(p)}>
                <Trash2 class="size-5" aria-hidden="true" />
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </Kartu>
  </div>
{/if}

<Modal bind:buka={bukaNama} judul="Ubah nama profil">
  <form id="form-nama" class="flex flex-col gap-3" onsubmit={simpanNama}>
    <Medan label="Nama profil" untuk="nama-profil">
      <input id="nama-profil" class={kelasInput} bind:value={ubahNama} required maxlength="60" />
    </Medan>
    {#if galatNama}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatNama}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaNama = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-nama" sibuk={sibukNama}>Simpan</Tombol>
  {/snippet}
</Modal>

<Modal bind:buka={bukaPeriode} judul={periodeId ? 'Ubah periode' : 'Tambah periode'}>
  <form id="form-periode" class="flex flex-col gap-3" onsubmit={simpanPeriode}>
    <Medan label="Tanggal mulai" untuk="per-mulai">
      <input id="per-mulai" type="date" class={kelasInput} bind:value={fMulai} required />
    </Medan>
    <Medan label="Tanggal selesai" untuk="per-selesai">
      <input id="per-selesai" type="date" class={kelasInput} bind:value={fSelesai} required min={fMulai} />
    </Medan>
    <Medan label="Profil" untuk="per-profil">
      <select id="per-profil" class={kelasInput} bind:value={fProfil} required>
        {#each profil as p (p.id)}<option value={p.id}>{p.nama}</option>{/each}
      </select>
    </Medan>
    <Medan label="Keterangan" untuk="per-ket">
      <input id="per-ket" class={kelasInput} bind:value={fKet} maxlength="100" />
    </Medan>
    {#if galatPeriode}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatPeriode}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaPeriode = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-periode" sibuk={sibukPeriode}>Simpan</Tombol>
  {/snippet}
</Modal>
