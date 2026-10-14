<script lang="ts">
  import { onMount } from 'svelte'
  import { Copy, Download, FileDown, Pencil, Star, Trash2 } from '@lucide/svelte'
  import { api, unduh, type FileJadwal, type Pengaturan, type Periode, type Profil, type Status } from '../lib/api'
  import { formatRentang } from '../lib/format'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import { pindah } from '../lib/router.svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kartu from '../components/Kartu.svelte'
  import Memuat from '../components/Memuat.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import PanelGeser from '../components/PanelGeser.svelte'
  import Tabel from '../components/Tabel.svelte'
  import BarAlat from '../components/BarAlat.svelte'
  import AlurImpor from '../components/AlurImpor.svelte'
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
  let bukaTambah = $state(false)
  let namaBaru = $state('')
  let sibukTambah = $state(false)
  let galatTambah = $state('')

  function bukaFormTambah() {
    namaBaru = ''
    galatTambah = ''
    bukaTambah = true
  }

  async function tambahProfil(e: SubmitEvent) {
    e.preventDefault()
    galatTambah = ''
    sibukTambah = true
    try {
      await api.post('/profil', { nama: namaBaru })
      bukaTambah = false
      beriTahu('Profil ditambahkan')
      await muat()
    } catch (err) {
      galatTambah = err instanceof Error ? err.message : String(err)
    } finally {
      sibukTambah = false
    }
  }

  // --- Duplikat ---
  let bukaDuplikat = $state(false)
  let duplikatId = $state(0)
  let duplikatNama = $state('')
  let sibukDuplikat = $state(false)
  let galatDuplikat = $state('')

  function bukaFormDuplikat(p: Profil) {
    duplikatId = p.id
    duplikatNama = `${p.nama} (salinan)`
    galatDuplikat = ''
    bukaDuplikat = true
  }

  async function simpanDuplikat(e: SubmitEvent) {
    e.preventDefault()
    galatDuplikat = ''
    sibukDuplikat = true
    try {
      await api.post(`/profil/${duplikatId}/duplikat`, { nama: duplikatNama })
      bukaDuplikat = false
      beriTahu(`Profil ${duplikatNama} dibuat`)
      await muat()
    } catch (err) {
      galatDuplikat = err instanceof Error ? err.message : String(err)
    } finally {
      sibukDuplikat = false
    }
  }

  // --- Ekspor & template ---
  async function ekspor(p: Profil) {
    try {
      await unduh(`/profil/${p.id}/ekspor`)
    } catch (err) {
      galat(err)
    }
  }

  let bukaTemplate = $state(false)
  let templateProfilId = $state(0)
  let templateNama = $state('')
  let sibukTemplate = $state(false)
  let galatTemplate = $state('')

  function bukaFormTemplate(p: Profil) {
    templateProfilId = p.id
    templateNama = p.nama
    galatTemplate = ''
    bukaTemplate = true
  }

  async function simpanTemplate(e: SubmitEvent) {
    e.preventDefault()
    galatTemplate = ''
    sibukTemplate = true
    try {
      await api.post('/template', { nama: templateNama, profilId: templateProfilId })
      bukaTemplate = false
      beriTahu('Template disimpan. Lihat di halaman Template.')
    } catch (err) {
      galatTemplate = err instanceof Error ? err.message : String(err)
    } finally {
      sibukTemplate = false
    }
  }

  // --- Impor ---
  let inputBerkas = $state<HTMLInputElement>()
  let bukaImpor = $state(false)
  let dataImpor = $state<FileJadwal | null>(null)

  async function pilihBerkas(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const berkas = input.files?.[0]
    if (!berkas) return
    const teks = await berkas.text()
    input.value = ''
    try {
      dataImpor = JSON.parse(teks) as FileJadwal
    } catch {
      beriTahu('File bukan file jadwal', 'galat')
      return
    }
    bukaImpor = true
  }

  function imporSelesai(id: number) {
    pindah(`/jadwal?profil=${id}`)
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

{#if galatMuat}
  <GalatMuat pesan={galatMuat} oncoba={muat} />
{:else}
  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2 lg:items-start">
    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Profil</h2>
      <BarAlat>
        {#snippet kanan()}
          <input
            bind:this={inputBerkas}
            type="file"
            accept=".json,application/json"
            class="hidden"
            aria-label="Berkas jadwal"
            onchange={pilihBerkas}
          />
          <Tombol varian="kedua" onclick={() => inputBerkas?.click()}>Impor jadwal</Tombol>
          <Tombol onclick={bukaFormTambah}>+ Profil</Tombol>
        {/snippet}
      </BarAlat>
      <Tabel label="Daftar profil" {memuat} jumlah={profil.length} kosong="Belum ada profil">
        {#snippet kepala()}
          <th scope="col">Nama</th>
          <th scope="col" class="whitespace-nowrap">Jumlah bel</th>
          <th scope="col">Aksi</th>
        {/snippet}
        {#each profil as p (p.id)}
          {@const isDefault = p.id === pengaturan.profilDefaultId}
          <tr>
            <td class="break-words font-medium">
              {p.nama}
              {#if isDefault}
                <span class="ml-2 rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-semibold text-blue-800">Default</span>
              {/if}
            </td>
            <td>{p.jumlahBel ?? 0}</td>
            <td>
              <div class="flex gap-1">
                <button type="button" class={kelasIkon} aria-label="Ubah nama {p.nama}" title="Ubah nama" onclick={() => bukaUbahNama(p)}>
                  <Pencil class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class={kelasIkon} aria-label="Duplikat {p.nama}" title="Duplikat" onclick={() => bukaFormDuplikat(p)}>
                  <Copy class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class={kelasIkon} aria-label="Ekspor {p.nama}" title="Ekspor" onclick={() => ekspor(p)}>
                  <Download class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class={kelasIkon} aria-label="Simpan {p.nama} sebagai template" title="Simpan sebagai template" onclick={() => bukaFormTemplate(p)}>
                  <FileDown class="size-5" aria-hidden="true" />
                </button>
                {#if !isDefault}
                  <button type="button" class={kelasIkon} aria-label="Jadikan default {p.nama}" title="Jadikan default" onclick={() => jadikanDefault(p)}>
                    <Star class="size-5" aria-hidden="true" />
                  </button>
                {/if}
                <button type="button" class="{kelasIkon} text-red-700" aria-label="Hapus {p.nama}" title="Hapus" onclick={() => hapusProfil(p)}>
                  <Trash2 class="size-5" aria-hidden="true" />
                </button>
              </div>
            </td>
          </tr>
        {/each}
      </Tabel>
    </Kartu>

    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Periode khusus</h2>
      <BarAlat>
        {#snippet kanan()}
          <Tombol onclick={() => bukaFormPeriode()}>Tambah periode</Tombol>
        {/snippet}
      </BarAlat>
      <Tabel label="Daftar periode khusus" {memuat} jumlah={periode.length} kosong="Belum ada periode khusus">
        {#snippet kepala()}
          <th scope="col">Rentang</th>
          <th scope="col">Profil</th>
          <th scope="col">Keterangan</th>
          <th scope="col">Aksi</th>
        {/snippet}
        {#each periode as p (p.id)}
          <tr>
            <td class="whitespace-nowrap font-medium">
              {formatRentang(p.mulai, p.selesai)}
              {#if hariIni >= p.mulai && hariIni <= p.selesai}
                <span class="ml-2 rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-semibold text-green-800">Aktif</span>
              {:else if p.selesai < hariIni}
                <span class="ml-2 rounded-full bg-slate-200 px-2.5 py-0.5 text-xs font-semibold text-slate-700">Selesai</span>
              {/if}
            </td>
            <td class="break-words">{namaProfil(p.profilId)}</td>
            <td class="break-words">{p.keterangan}</td>
            <td>
              <div class="flex gap-1">
                <button type="button" class={kelasIkon} aria-label="Ubah periode {p.keterangan || formatRentang(p.mulai, p.selesai)}" title="Ubah" onclick={() => bukaFormPeriode(p)}>
                  <Pencil class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class="{kelasIkon} text-red-700" aria-label="Hapus periode {p.keterangan || formatRentang(p.mulai, p.selesai)}" title="Hapus" onclick={() => hapusPeriode(p)}>
                  <Trash2 class="size-5" aria-hidden="true" />
                </button>
              </div>
            </td>
          </tr>
        {/each}
      </Tabel>
    </Kartu>
  </div>
{/if}

<PanelGeser bind:buka={bukaTambah} judul="Tambah profil">
  <form id="form-tambah" class="flex flex-col gap-3" onsubmit={tambahProfil}>
    <Medan label="Nama profil baru" untuk="profil-baru">
      <input id="profil-baru" class={kelasInput} bind:value={namaBaru} required maxlength="60" />
    </Medan>
    {#if galatTambah}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatTambah}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaTambah = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-tambah" sibuk={sibukTambah}>Simpan</Tombol>
  {/snippet}
</PanelGeser>

<PanelGeser bind:buka={bukaNama} judul="Ubah nama profil">
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
</PanelGeser>

<PanelGeser bind:buka={bukaDuplikat} judul="Duplikat profil">
  <form id="form-duplikat" class="flex flex-col gap-3" onsubmit={simpanDuplikat}>
    <Medan label="Nama salinan" untuk="duplikat-nama">
      <input id="duplikat-nama" class={kelasInput} bind:value={duplikatNama} required maxlength="60" />
    </Medan>
    {#if galatDuplikat}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatDuplikat}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaDuplikat = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-duplikat" sibuk={sibukDuplikat}>Duplikat</Tombol>
  {/snippet}
</PanelGeser>

<PanelGeser bind:buka={bukaTemplate} judul="Simpan sebagai template">
  <form id="form-template" class="flex flex-col gap-3" onsubmit={simpanTemplate}>
    <Medan label="Nama template" untuk="template-nama">
      <input id="template-nama" class={kelasInput} bind:value={templateNama} required maxlength="60" />
    </Medan>
    {#if galatTemplate}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatTemplate}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaTemplate = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-template" sibuk={sibukTemplate}>Simpan</Tombol>
  {/snippet}
</PanelGeser>

<AlurImpor bind:buka={bukaImpor} data={dataImpor} onselesai={imporSelesai} />

<PanelGeser bind:buka={bukaPeriode} judul={periodeId ? 'Ubah periode' : 'Tambah periode'}>
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
</PanelGeser>
