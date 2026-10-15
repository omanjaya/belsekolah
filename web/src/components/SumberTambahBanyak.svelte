<script lang="ts">
  import { untrack } from 'svelte'
  import { api, type FileJadwal, type JenisBel, type Profil, type Template } from '../lib/api'
  import { NAMA_HARI } from '../lib/format'
  import {
    hariBerisi,
    jamGanda,
    keFileJadwal,
    saringHari,
    uraiTempel,
    type BarisKetik,
    type GalatTempel,
  } from '../lib/tambahBanyak'
  import FormPembuat from './FormPembuat.svelte'
  import Medan from './Medan.svelte'
  import Tombol from './Tombol.svelte'

  let { profilId, jenisBel, data = $bindable(), galat = $bindable() }: {
    profilId: number
    jenisBel: JenisBel[]
    data: FileJadwal | null
    galat: string
  } = $props()

  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const kelasCentang = 'size-5 accent-blue-700'
  const kelasTab = (aktif: boolean) =>
    `min-h-11 flex-1 rounded-lg px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-700 ${aktif ? 'bg-white text-blue-800 shadow-sm ring-1 ring-slate-200' : 'text-slate-600 hover:text-slate-900'}`

  type Tab = 'ketik' | 'template' | 'file'
  const TAB: { id: Tab; label: string }[] = [
    { id: 'ketik', label: 'Ketik/tempel' },
    { id: 'template', label: 'Template' },
    { id: 'file', label: 'File/profil lain' },
  ]
  let tab = $state<Tab>('ketik')

  // --- Ketik/tempel ---
  const namaJenis = $derived(jenisBel.map((j) => j.nama))
  let jenisBawaan = $state('')
  let tempel = $state('')
  let galatTempel = $state<GalatTempel[]>([])
  let hariTujuan = $state<number[]>([1, 2, 3, 4, 5])
  // Id unik per baris agar penghapusan tidak mengacak isian.
  let kunciBerikut = 0
  type Isian = BarisKetik & { k: number }
  const barisKosong = (jenis: string): Isian => ({ k: kunciBerikut++, jam: '', label: '', jenis })
  let isian = $state<Isian[]>([])

  // Jenis bawaan = jenis pertama selama belum dipilih / sudah tidak ada.
  $effect(() => {
    if (!namaJenis.includes(jenisBawaan)) jenisBawaan = namaJenis[0] ?? ''
  })
  $effect(() => {
    if (untrack(() => isian.length) === 0) isian = [barisKosong(untrack(() => jenisBawaan))]
  })

  const opsiJenis = (b: Isian) => (b.jenis && !namaJenis.includes(b.jenis) ? [...namaJenis, b.jenis] : namaJenis)

  function tambahkan() {
    const { baris, galat: g } = uraiTempel(tempel, jenisBawaan)
    galatTempel = g
    if (baris.length) {
      const sisa = isian.filter((b) => b.jam || b.label)
      isian = [...sisa, ...baris.map((b) => ({ ...b, k: kunciBerikut++ }))]
    }
    if (g.length === 0) tempel = ''
  }
  const hapus = (k: number) => (isian = isian.filter((b) => b.k !== k))
  const tambahBaris = () => (isian = [...isian, barisKosong(jenisBawaan)])
  const alihHari = (h: number, ada: boolean, daftar: number[]) =>
    ada ? [...new Set([...daftar, h])].sort((a, b) => a - b) : daftar.filter((x) => x !== h)

  const terisi = $derived(isian.filter((b) => b.jam))
  const ganda = $derived(jamGanda(terisi))
  const dataKetik = $derived(
    terisi.length && hariTujuan.length && !ganda.length
      ? keFileJadwal(
          terisi.map(({ jam, label, jenis }) => ({ jam, label, jenis })),
          hariTujuan,
          jenisBel,
        )
      : null,
  )
  const galatKetik = $derived(ganda.length ? `Jam ${ganda.join(', ')} ganda` : '')

  // --- Template ---
  let modeTemplate = $state<'pembuat' | 'saya'>('pembuat')
  let hasilPembuat = $state<FileJadwal | null>(null)
  let menghitung = $state(false)
  let galatPembuat = $state('')
  let hariSekolah = $state<number[]>([1, 2, 3, 4, 5])
  let daftarTemplate = $state<Template[]>([])
  let templateId = $state('')
  let sumberTemplate = $state<FileJadwal | null>(null)
  let galatMuatTemplate = $state('')
  let urutanTemplate = 0

  // --- File/profil lain ---
  let daftarProfil = $state<Profil[]>([])
  let profilLain = $state('')
  let sumberFile = $state<FileJadwal | null>(null)
  let galatFile = $state('')
  let galatMuatFile = $state('')
  let urutanFile = 0
  let galatDaftarProfil = $state('')
  let urutanDaftarProfil = 0

  const pesan = (e: unknown) => (e instanceof Error ? e.message : 'Gagal memuat data')

  async function muatDaftarTemplate() {
    const saya = ++urutanTemplate
    galatMuatTemplate = ''
    try {
      const r = await api.get<Template[]>('/template')
      if (saya !== urutanTemplate) return
      daftarTemplate = r ?? []
    } catch (e) {
      if (saya !== urutanTemplate) return
      galatMuatTemplate = pesan(e)
    }
  }
  async function pilihTemplate(id: string) {
    templateId = id
    sumberTemplate = null
    const saya = ++urutanTemplate
    galatMuatTemplate = ''
    if (!id) return
    try {
      const r = await api.get<Template>(`/template/${id}`)
      if (saya !== urutanTemplate) return
      sumberTemplate = r.data ?? null
    } catch (e) {
      if (saya !== urutanTemplate) return
      galatMuatTemplate = pesan(e)
    }
  }

  async function muatDaftarProfil() {
    const saya = ++urutanDaftarProfil
    galatDaftarProfil = ''
    try {
      const r = await api.get<Profil[]>('/profil')
      if (saya !== urutanDaftarProfil) return
      daftarProfil = (r ?? []).filter((p) => p.id !== profilId)
    } catch (e) {
      if (saya !== urutanDaftarProfil) return
      galatDaftarProfil = pesan(e)
    }
  }
  async function pilihProfil(id: string) {
    profilLain = id
    sumberFile = null
    galatFile = ''
    const saya = ++urutanFile
    galatMuatFile = ''
    if (!id) return
    try {
      const r = await api.get<FileJadwal>(`/profil/${id}/ekspor`)
      if (saya !== urutanFile) return
      sumberFile = r
    } catch (e) {
      if (saya !== urutanFile) return
      galatMuatFile = pesan(e)
    }
  }
  async function pilihBerkas(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const f = input.files?.[0]
    if (!f) return
    const saya = ++urutanFile
    profilLain = ''
    sumberFile = null
    galatFile = ''
    galatMuatFile = ''
    try {
      const teks = await f.text()
      const d = JSON.parse(teks) as FileJadwal
      if (!d || !Array.isArray(d.jadwal) || !Array.isArray(d.jenisBel)) throw new Error('bukan file jadwal')
      if (saya !== urutanFile) return
      sumberFile = d
    } catch {
      if (saya !== urutanFile) return
      galatFile = 'File tidak bisa dibaca sebagai file jadwal'
    }
    input.value = ''
  }

  // Daftar template / profil dimuat saat tab-nya pertama dibuka.
  let templateDimuat = false
  let profilDimuat = false
  $effect(() => {
    if (tab === 'template' && !templateDimuat) {
      templateDimuat = true
      void muatDaftarTemplate()
    }
    if (tab === 'file' && !profilDimuat) {
      profilDimuat = true
      void muatDaftarProfil()
    }
  })

  // --- Ambil hari (Template dan File/profil lain) ---
  const sumberAktif = $derived<FileJadwal | null>(
    tab === 'template' ? (modeTemplate === 'pembuat' ? hasilPembuat : sumberTemplate) : tab === 'file' ? sumberFile : null,
  )
  const hariTersedia = $derived(sumberAktif ? hariBerisi(sumberAktif) : [])
  let hariTerpilih = $state<number[]>([])
  $effect(() => {
    const s = sumberAktif
    untrack(() => (hariTerpilih = s ? hariBerisi(s) : []))
  })
  const dataSumber = $derived.by(() => {
    if (!sumberAktif) return null
    const d = saringHari(sumberAktif, hariTerpilih)
    return d.jadwal.length ? d : null
  })

  // --- Keluaran ke induk ---
  $effect(() => {
    if (tab === 'ketik') {
      data = dataKetik
      galat = galatKetik
    } else {
      data = dataSumber
      galat = tab === 'template' ? (modeTemplate === 'pembuat' ? galatPembuat : '') : galatFile
    }
  })

  const ringkasan = $derived(
    data
      ? `${data.jadwal.length} bel dari ${new Set(data.jadwal.map((b) => b.hari)).size} hari`
      : 'Belum ada bel',
  )
</script>

<div class="flex flex-col gap-4">
  <div role="tablist" aria-label="Sumber bel" class="flex gap-1 rounded-xl bg-slate-100 p-1">
    {#each TAB as t (t.id)}
      <button
        type="button"
        role="tab"
        id="tb-tab-{t.id}"
        aria-selected={tab === t.id}
        aria-controls="tb-panel-{t.id}"
        class={kelasTab(tab === t.id)}
        onclick={() => (tab = t.id)}
      >
        {t.label}
      </button>
    {/each}
  </div>

  {#if tab === 'ketik'}
    <div role="tabpanel" id="tb-panel-ketik" aria-labelledby="tb-tab-ketik" class="flex flex-col gap-4">
      <Medan label="Jenis bel bawaan" untuk="tb-jenis-bawaan">
        <select id="tb-jenis-bawaan" class={kelasInput} bind:value={jenisBawaan}>
          {#each namaJenis as n (n)}
            <option value={n}>{n}</option>
          {/each}
        </select>
      </Medan>

      <Medan label="Tempel dari Excel/teks" untuk="tb-tempel" bantuan="Satu bel per baris: jam, keterangan, jenis bel.">
        <textarea
          id="tb-tempel"
          rows="4"
          class={kelasInput}
          placeholder={'07:15\tSholat Dhuha\tMasuk'}
          bind:value={tempel}
        ></textarea>
      </Medan>
      <div>
        <Tombol varian="kedua" onclick={tambahkan}>Tambahkan ke daftar</Tombol>
      </div>
      {#if galatTempel.length}
        <ul class="flex flex-col gap-1">
          {#each galatTempel as g (g.nomor)}
            <li class="text-sm text-red-700" role="alert">Baris {g.nomor}: {g.alasan}</li>
          {/each}
        </ul>
      {/if}

      <ul class="flex flex-col gap-3" aria-label="Daftar bel">
        {#each isian as b, i (b.k)}
          <li class="grid grid-cols-[auto_1fr_auto] items-end gap-2 sm:grid-cols-[8rem_1fr_10rem_auto]">
            <input type="time" aria-label="Jam baris {i + 1}" class={kelasInput} bind:value={b.jam} />
            <input
              type="text"
              maxlength="100"
              aria-label="Keterangan baris {i + 1}"
              placeholder="Keterangan"
              class={kelasInput}
              bind:value={b.label}
            />
            <select aria-label="Jenis bel baris {i + 1}" class="{kelasInput} col-span-2 sm:col-span-1" bind:value={b.jenis}>
              {#each opsiJenis(b) as n (n)}
                <option value={n}>{n}</option>
              {/each}
            </select>
            <button
              type="button"
              aria-label="Hapus baris {i + 1}"
              class="inline-flex size-11 items-center justify-center rounded-xl text-red-700 hover:bg-red-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-red-600"
              onclick={() => hapus(b.k)}
            >
              <svg viewBox="0 0 20 20" class="size-5" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
                <path d="M4 6h12M8 6V4h4v2M6 6l1 10h6l1-10" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>
          </li>
        {/each}
      </ul>
      <div>
        <Tombol varian="hantu" onclick={tambahBaris}>+ Baris</Tombol>
      </div>
      {#if galatKetik}
        <p class="text-sm text-red-700" role="alert">{galatKetik}</p>
      {/if}

      <fieldset class="flex flex-col gap-2">
        <legend class="mb-1 text-sm font-medium text-slate-800">Hari tujuan</legend>
        <div class="flex flex-wrap gap-x-4 gap-y-1">
          {#each NAMA_HARI as nama, h (h)}
            {#if h >= 1}
              <label class="inline-flex min-h-11 items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  class={kelasCentang}
                  checked={hariTujuan.includes(h)}
                  onchange={(e) => (hariTujuan = alihHari(h, e.currentTarget.checked, hariTujuan))}
                />
                {nama}
              </label>
            {/if}
          {/each}
        </div>
      </fieldset>
    </div>
  {:else if tab === 'template'}
    <div role="tabpanel" id="tb-panel-template" aria-labelledby="tb-tab-template" class="flex flex-col gap-4">
      <div class="flex gap-2">
        <Tombol varian={modeTemplate === 'pembuat' ? 'utama' : 'kedua'} aria-pressed={modeTemplate === 'pembuat'} onclick={() => (modeTemplate = 'pembuat')}>
          Pembuat jadwal
        </Tombol>
        <Tombol varian={modeTemplate === 'saya' ? 'utama' : 'kedua'} aria-pressed={modeTemplate === 'saya'} onclick={() => (modeTemplate = 'saya')}>
          Template saya
        </Tombol>
      </div>
      {#if modeTemplate === 'pembuat'}
        <FormPembuat bind:hasil={hasilPembuat} bind:menghitung bind:galat={galatPembuat} bind:hariSekolah />
      {:else}
        <Medan label="Template saya" untuk="tb-template">
          <select id="tb-template" class={kelasInput} value={templateId} onchange={(e) => pilihTemplate(e.currentTarget.value)}>
            <option value="">Pilih template</option>
            {#each daftarTemplate as t (t.id)}
              <option value={String(t.id)}>{t.nama} ({t.jumlahBel} bel)</option>
            {/each}
          </select>
        </Medan>
        {#if galatMuatTemplate}
          <p class="text-sm text-red-700" role="alert">{galatMuatTemplate}</p>
        {/if}
      {/if}
    </div>
  {:else}
    <div role="tabpanel" id="tb-panel-file" aria-labelledby="tb-tab-file" class="flex flex-col gap-4">
      <Medan label="File jadwal (.json)" untuk="tb-berkas">
        <input id="tb-berkas" type="file" accept=".json,application/json" class={kelasInput} onchange={pilihBerkas} />
      </Medan>
      <p class="text-sm text-slate-500">atau</p>
      <Medan label="Profil lain" untuk="tb-profil">
        <select id="tb-profil" class={kelasInput} value={profilLain} onchange={(e) => pilihProfil(e.currentTarget.value)}>
          <option value="">Pilih profil</option>
          {#each daftarProfil as p (p.id)}
            <option value={String(p.id)}>{p.nama}</option>
          {/each}
        </select>
      </Medan>
      {#if galatFile}
        <p class="text-sm text-red-700" role="alert">{galatFile}</p>
      {/if}
      {#if galatDaftarProfil}
        <p class="text-sm text-red-700" role="alert">{galatDaftarProfil}</p>
      {/if}
      {#if galatMuatFile}
        <p class="text-sm text-red-700" role="alert">{galatMuatFile}</p>
      {/if}
    </div>
  {/if}

  {#if tab !== 'ketik' && hariTersedia.length}
    <fieldset class="flex flex-col gap-2">
      <legend class="mb-1 text-sm font-medium text-slate-800">Ambil hari</legend>
      <div class="flex flex-wrap gap-x-4 gap-y-1">
        {#each hariTersedia as h (h)}
          <label class="inline-flex min-h-11 items-center gap-2 text-sm">
            <input
              type="checkbox"
              class={kelasCentang}
              checked={hariTerpilih.includes(h)}
              onchange={(e) => (hariTerpilih = alihHari(h, e.currentTarget.checked, hariTerpilih))}
            />
            {NAMA_HARI[h]}
          </label>
        {/each}
      </div>
    </fieldset>
  {/if}

  <p class="rounded-xl bg-slate-50 px-3 py-2 text-sm font-medium text-slate-800" data-testid="ringkasan-sumber">{ringkasan}</p>
</div>
