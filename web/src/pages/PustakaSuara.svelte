<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { Play, Square } from '@lucide/svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import BarAlat from '../components/BarAlat.svelte'
  import Tabel from '../components/Tabel.svelte'
  import Memuat from '../components/Memuat.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import PanelGeser from '../components/PanelGeser.svelte'
  import Medan from '../components/Medan.svelte'
  import Tombol from '../components/Tombol.svelte'
  import { api, type Audio, type SuaraTTS } from '../lib/api'
  import { formatDurasi } from '../lib/format'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import { beriTahu, galat } from '../lib/notif.svelte'

  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const MAKS_BYTE = 20 * 1024 * 1024
  const MAKS_TEKS = 1000
  const TAB = [
    { kunci: 'semua', label: 'Semua' },
    { kunci: 'bawaan', label: 'Bawaan' },
    { kunci: 'upload', label: 'Unggahan' },
    { kunci: 'tts', label: 'TTS' },
  ] as const
  const NAMA_SUMBER = { bawaan: 'Bawaan', upload: 'Unggahan', tts: 'TTS' }
  const GAYA_SUMBER = {
    bawaan: 'bg-slate-100 text-slate-700',
    upload: 'bg-blue-50 text-blue-800',
    tts: 'bg-violet-50 text-violet-800',
  }

  let daftar = $state<Audio[]>([])
  let suara = $state<SuaraTTS[]>([])
  let galatSuara = $state('')
  let memuat = $state(true)
  let galatMuat = $state('')
  let tab = $state<(typeof TAB)[number]['kunci']>('semua')
  const tampil = $derived(tab === 'semua' ? daftar : daftar.filter((a) => a.sumber === tab))

  async function muat() {
    galatMuat = ''
    memuat = true
    try {
      daftar = await api.get<Audio[]>('/audio')
      galatMuat = ''
    } catch (e) {
      galatMuat = e instanceof Error ? e.message : String(e)
    } finally {
      memuat = false
    }
  }
  onMount(() => {
    muat()
    api.get<SuaraTTS[]>('/tts/suara').then(
      (s) => {
        suara = s
        if (!suaraTts && s.length) suaraTts = (s.find((v) => v.id === 'id-ID-GadisNeural') ?? s[0]).id
      },
      () => (galatSuara = 'Daftar suara TTS gagal dimuat'),
    )
  })

  // ---- Pemutar browser bersama ----
  let pemutar = $state<HTMLAudioElement>()
  let memutar = $state<string | null>(null) // 'a:<id>' atau 'pratinjau'
  let urlPratinjau = ''

  function berhenti() {
    pemutar?.pause()
    memutar = null
  }
  async function putar(kunci: string, src: string) {
    if (!pemutar) return
    pemutar.src = src
    memutar = kunci
    try {
      await pemutar.play()
    } catch (e) {
      if (e instanceof DOMException && e.name === 'AbortError') return
      if (memutar === kunci) memutar = null
      beriTahu('Browser tidak bisa memutar suara ini', 'galat')
    }
  }
  const dengarkan = (a: Audio) => (memutar === `a:${a.id}` ? berhenti() : putar(`a:${a.id}`, `/api/audio/${a.id}/file`))
  function cabutPratinjau() {
    if (urlPratinjau) URL.revokeObjectURL(urlPratinjau)
    urlPratinjau = ''
  }
  onDestroy(() => {
    pemutar?.pause()
    cabutPratinjau()
  })

  // ---- Tes di speaker, hapus ----
  let tesId = $state<number | null>(null)
  async function tes(a: Audio) {
    tesId = a.id
    try {
      await api.post(`/audio/${a.id}/tes`)
      beriTahu('Suara diputar di speaker')
    } catch (e) {
      galat(e)
    } finally {
      tesId = null
    }
  }
  async function hapus(a: Audio) {
    if (!(await konfirmasi('Hapus suara?', `Suara "${a.nama}" akan dihapus.`))) return
    if (memutar === `a:${a.id}`) berhenti()
    try {
      await api.hapus(`/audio/${a.id}`)
      beriTahu('Suara dihapus')
      await muat()
    } catch (e) {
      galat(e)
    }
  }

  // ---- Unggah ----
  let unggahBuka = $state(false)
  function bukaUnggah() {
    berkas = null
    namaUnggah = ''
    galatBerkas = ''
    unggahBuka = true
  }
  let berkas = $state<File | null>(null)
  let namaUnggah = $state('')
  let mengunggah = $state(false)
  let galatBerkas = $state('')
  let masukanBerkas = $state<HTMLInputElement>()
  async function unggah(e: SubmitEvent) {
    e.preventDefault()
    if (!berkas) {
      galatBerkas = 'Pilih file audio terlebih dahulu'
      return
    }
    if (berkas.size > MAKS_BYTE) {
      galatBerkas = 'File terlalu besar (maksimal 20 MB)'
      return
    }
    galatBerkas = ''
    mengunggah = true
    try {
      const f = new FormData()
      if (namaUnggah.trim()) f.append('nama', namaUnggah.trim())
      f.append('file', berkas)
      await api.post('/audio', f)
      beriTahu('Suara diunggah')
      berkas = null
      namaUnggah = ''
      unggahBuka = false
      await muat()
    } catch (err) {
      galat(err)
    } finally {
      mengunggah = false
    }
  }

  // ---- Ganti nama ----
  let gnBuka = $state(false)
  let gnAudio = $state<Audio | null>(null)
  let gnNama = $state('')
  let gnGalat = $state('')
  let gnSimpan = $state(false)
  let gnGalatForm = $state('')
  function bukaGantiNama(a: Audio) {
    gnAudio = a
    gnNama = a.nama
    gnGalat = ''
    gnGalatForm = ''
    gnBuka = true
  }
  async function simpanNama(e: SubmitEvent) {
    e.preventDefault()
    if (!gnAudio) return
    if (!gnNama.trim()) {
      gnGalat = 'Nama wajib diisi'
      return
    }
    gnSimpan = true
    gnGalatForm = ''
    try {
      await api.put(`/audio/${gnAudio.id}`, { nama: gnNama.trim() })
      beriTahu('Nama suara disimpan')
      gnBuka = false
      await muat()
    } catch (err) {
      gnGalatForm = err instanceof Error ? err.message : String(err)
    } finally {
      gnSimpan = false
    }
  }

  // ---- Ubah teks TTS (modal) ----
  let utBuka = $state(false)
  let utAudio = $state<Audio | null>(null)
  let utTeks = $state('')
  let utSuara = $state('')
  let utGalat = $state('')
  let utSimpan = $state(false)
  let utGalatForm = $state('')
  function bukaUbahTeks(a: Audio) {
    utAudio = a
    utTeks = a.teksTts ?? ''
    utSuara = a.suaraTts ?? suara[0]?.id ?? ''
    utGalat = ''
    utGalatForm = ''
    utBuka = true
  }
  async function simpanTeks(e: SubmitEvent) {
    e.preventDefault()
    if (!utAudio) return
    if (!utTeks.trim()) {
      utGalat = 'Teks wajib diisi'
      return
    }
    utSimpan = true
    utGalatForm = ''
    try {
      await api.put(`/audio/${utAudio.id}/tts`, { teks: utTeks.trim(), suara: utSuara })
      beriTahu('Teks suara disimpan')
      utBuka = false
      await muat()
    } catch (err) {
      utGalatForm = err instanceof Error ? err.message : String(err)
    } finally {
      utSimpan = false
    }
  }

  // ---- Form TTS ----
  let ttsBuka = $state(false)
  function bukaTts() {
    teks = ''
    namaTts = ''
    galatTeks = ''
    ttsBuka = true
  }
  $effect(() => {
    if (!ttsBuka && memutar === 'pratinjau') berhenti()
  })
  let teks = $state('')
  let suaraTts = $state('')
  let namaTts = $state('')
  let galatTeks = $state('')
  let praBusy = $state(false)
  let simpanBusy = $state(false)
  function teksValid() {
    galatTeks = teks.trim() ? '' : 'Teks pengumuman wajib diisi'
    return !galatTeks
  }
  async function pratinjau() {
    if (!teksValid()) return
    praBusy = true
    try {
      const blob = await api.post<Blob>('/tts/pratinjau', { teks: teks.trim(), suara: suaraTts })
      cabutPratinjau()
      if (!ttsBuka) return
      urlPratinjau = URL.createObjectURL(blob)
      await putar('pratinjau', urlPratinjau)
    } catch (e) {
      galat(e)
    } finally {
      praBusy = false
    }
  }
  async function simpanTts() {
    if (!teksValid()) return
    simpanBusy = true
    try {
      await api.post('/audio/tts', { nama: namaTts.trim(), teks: teks.trim(), suara: suaraTts })
      beriTahu('Suara TTS disimpan')
      teks = ''
      namaTts = ''
      ttsBuka = false
      await muat()
    } catch (e) {
      galat(e)
    } finally {
      simpanBusy = false
    }
  }
</script>

<JudulHalaman judul="Pustaka Suara" keterangan="Suara yang bisa dipakai di jenis bel: bawaan, unggahan, dan hasil teks ke suara." />

<audio bind:this={pemutar} onended={() => (memutar = null)} onerror={() => (memutar = null)} onpause={() => (memutar = null)} class="hidden"></audio>

<BarAlat>
  {#snippet kiri()}
    <div role="tablist" aria-label="Sumber suara" class="flex gap-1 overflow-x-auto rounded-xl bg-slate-200/60 p-1">
      {#each TAB as t (t.kunci)}
        <button
          type="button"
          role="tab"
          aria-selected={tab === t.kunci}
          onclick={() => (tab = t.kunci)}
          class="min-h-11 whitespace-nowrap rounded-lg px-4 text-sm font-medium {tab === t.kunci ? 'bg-white text-blue-800 shadow-sm' : 'text-slate-600 hover:text-slate-900'}"
        >
          {t.label}
        </button>
      {/each}
    </div>
  {/snippet}
  {#snippet kanan()}
    <Tombol varian="kedua" onclick={bukaUnggah}>Unggah suara</Tombol>
    <Tombol onclick={bukaTts}>+ Buat suara TTS</Tombol>
  {/snippet}
</BarAlat>

{#if memuat}
  <Memuat />
{:else if galatMuat}
  <GalatMuat pesan={galatMuat} oncoba={muat} />
{:else}
  <div class="hidden md:block">
  <Tabel label="Daftar suara" jumlah={tampil.length} kosong="Belum ada suara di kategori ini.">
    {#snippet kepala()}
      <th scope="col">Nama</th>
      <th scope="col">Sumber</th>
      <th scope="col">Durasi</th>
      <th scope="col">Putar di browser</th>
      <th scope="col">Tes</th>
      <th scope="col" class="text-right">Aksi</th>
    {/snippet}
    {#each tampil as a (a.id)}
      {@const sedangMemutar = memutar === `a:${a.id}`}
      <tr>
        <td class="font-semibold break-words">{a.nama}</td>
        <td><span class="rounded-full px-2 py-0.5 text-xs font-medium {GAYA_SUMBER[a.sumber]}">{NAMA_SUMBER[a.sumber]}</span></td>
        <td class="whitespace-nowrap tabular-nums text-slate-600">{formatDurasi(a.durasiMs)}</td>
        <td>
          <button type="button" class="inline-flex min-h-11 items-center gap-1.5 rounded-lg px-3 text-slate-700 hover:bg-slate-100" onclick={() => dengarkan(a)} aria-label="{sedangMemutar ? 'Berhenti' : 'Putar'} {a.nama}">
            {#if sedangMemutar}<Square class="size-4" /> Berhenti{:else}<Play class="size-4" /> Putar{/if}
          </button>
        </td>
        <td>
          <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100 disabled:opacity-50" disabled={tesId === a.id} onclick={() => tes(a)} aria-label="Tes di speaker {a.nama}">Tes</button>
        </td>
        <td class="text-right whitespace-nowrap">
          {#if a.sumber !== 'bawaan'}
            <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100" onclick={() => bukaGantiNama(a)} aria-label="Ganti nama {a.nama}">Ganti nama</button>
          {/if}
          {#if a.sumber === 'tts'}
            <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100" onclick={() => bukaUbahTeks(a)} aria-label="Ubah teks {a.nama}">Ubah teks</button>
          {/if}
          {#if a.sumber !== 'bawaan'}
            <button type="button" class="min-h-11 rounded-lg px-3 text-red-700 hover:bg-red-50" onclick={() => hapus(a)} aria-label="Hapus {a.nama}">Hapus</button>
          {/if}
        </td>
      </tr>
    {/each}
  </Tabel>
  </div>

  <ul class="flex flex-col gap-2 md:hidden" aria-label="Daftar suara">
    {#each tampil as a (a.id)}
      {@const sedangMemutar = memutar === `a:${a.id}`}
      <li class="rounded-xl border border-slate-200 bg-white p-3">
        <p class="break-words font-semibold text-slate-900">{a.nama}</p>
        <div class="mt-1 flex flex-wrap items-center gap-2 text-sm">
          <span class="rounded-full px-2 py-0.5 text-xs font-medium {GAYA_SUMBER[a.sumber]}">{NAMA_SUMBER[a.sumber]}</span>
          <span class="tabular-nums text-slate-500">{formatDurasi(a.durasiMs)}</span>
        </div>
        <div class="mt-2 flex flex-wrap gap-1">
          <button type="button" class="inline-flex min-h-11 items-center gap-1.5 rounded-lg px-3 text-slate-700 hover:bg-slate-100" onclick={() => dengarkan(a)} aria-label="{sedangMemutar ? 'Berhenti' : 'Putar'} {a.nama}">
            {#if sedangMemutar}<Square class="size-4" /> Berhenti{:else}<Play class="size-4" /> Putar{/if}
          </button>
          <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100 disabled:opacity-50" disabled={tesId === a.id} onclick={() => tes(a)} aria-label="Tes di speaker {a.nama}">Tes</button>
          {#if a.sumber !== 'bawaan'}
            <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100" onclick={() => bukaGantiNama(a)} aria-label="Ganti nama {a.nama}">Ganti nama</button>
          {/if}
          {#if a.sumber === 'tts'}
            <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100" onclick={() => bukaUbahTeks(a)} aria-label="Ubah teks {a.nama}">Ubah teks</button>
          {/if}
          {#if a.sumber !== 'bawaan'}
            <button type="button" class="min-h-11 rounded-lg px-3 text-red-700 hover:bg-red-50" onclick={() => hapus(a)} aria-label="Hapus {a.nama}">Hapus</button>
          {/if}
        </div>
      </li>
    {:else}
      <li class="rounded-xl border border-slate-200 bg-white px-4 py-6 text-center text-slate-600">Belum ada suara di kategori ini.</li>
    {/each}
  </ul>
{/if}

<PanelGeser bind:buka={unggahBuka} judul="Unggah suara">
  <form id="form-unggah" onsubmit={unggah} class="flex flex-col gap-4">
    <Medan label="File audio" untuk="up-file" galat={galatBerkas} bantuan="Format MP3 atau WAV, maksimal 20 MB.">
      <input
        id="up-file"
        type="file"
        accept=".mp3,.wav,audio/mpeg,audio/wav"
        bind:this={masukanBerkas}
        onchange={(e) => {
          berkas = e.currentTarget.files?.[0] ?? null
          galatBerkas = ''
        }}
        aria-invalid={!!galatBerkas}
        aria-describedby={galatBerkas ? 'up-file-galat' : undefined}
        class="{kelasInput} file:mr-3 file:rounded-lg file:border-0 file:bg-slate-100 file:px-3 file:py-1.5"
      />
    </Medan>
    <Medan label="Nama (opsional)" untuk="up-nama">
      <input id="up-nama" class={kelasInput} bind:value={namaUnggah} autocomplete="off" />
    </Medan>
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (unggahBuka = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-unggah" sibuk={mengunggah}>Unggah</Tombol>
  {/snippet}
</PanelGeser>

<PanelGeser bind:buka={ttsBuka} judul="Buat suara dari teks (TTS)">
  <div class="flex flex-col gap-4">
    <Medan label="Teks pengumuman" untuk="tts-teks" galat={galatTeks}>
      <textarea
        id="tts-teks"
        rows="4"
        maxlength={MAKS_TEKS}
        class={kelasInput}
        bind:value={teks}
        aria-invalid={!!galatTeks}
        aria-describedby={galatTeks ? 'tts-teks-galat' : undefined}
      ></textarea>
      <p class="text-right text-xs text-slate-500" aria-live="polite">{teks.length}/{MAKS_TEKS}</p>
    </Medan>
    <fieldset class="flex flex-col gap-1">
      <legend class="mb-1 text-sm font-medium text-slate-800">Suara</legend>
      {#if galatSuara}<p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatSuara}</p>{/if}
      {#each suara as s (s.id)}
        <label class="flex min-h-11 items-center gap-3 rounded-xl px-2 hover:bg-slate-50">
          <input type="radio" name="suara-tts" value={s.id} bind:group={suaraTts} class="size-5" />
          {s.nama}
        </label>
      {/each}
    </fieldset>
    <Medan label="Nama (opsional)" untuk="tts-nama">
      <input id="tts-nama" class={kelasInput} bind:value={namaTts} autocomplete="off" />
    </Medan>
  </div>
  {#snippet aksi()}
    <Tombol varian="kedua" sibuk={praBusy} disabled={simpanBusy || !!galatSuara} onclick={() => (memutar === 'pratinjau' ? berhenti() : pratinjau())}>
      {#if memutar === 'pratinjau'}<Square class="size-4" /> Hentikan pratinjau{:else}<Play class="size-4" /> Dengarkan pratinjau{/if}
    </Tombol>
    <Tombol sibuk={simpanBusy} disabled={praBusy || !!galatSuara} onclick={simpanTts}>Simpan suara</Tombol>
  {/snippet}
</PanelGeser>

<PanelGeser bind:buka={gnBuka} judul="Ganti nama suara">
  <form id="form-nama" onsubmit={simpanNama} novalidate>
    <Medan label="Nama suara" untuk="gn-nama" galat={gnGalat}>
      <input id="gn-nama" class={kelasInput} bind:value={gnNama} autocomplete="off" aria-invalid={!!gnGalat} aria-describedby={gnGalat ? 'gn-nama-galat' : undefined} />
    </Medan>
    {#if gnGalatForm}<p class="mt-3 rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{gnGalatForm}</p>{/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (gnBuka = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-nama" sibuk={gnSimpan}>Simpan</Tombol>
  {/snippet}
</PanelGeser>

<PanelGeser bind:buka={utBuka} judul="Ubah teks suara">
  <form id="form-teks" onsubmit={simpanTeks} class="flex flex-col gap-3" novalidate>
    <Medan label="Teks pengumuman" untuk="ut-teks" galat={utGalat}>
      <textarea id="ut-teks" rows="4" maxlength={MAKS_TEKS} class={kelasInput} bind:value={utTeks} aria-invalid={!!utGalat} aria-describedby={utGalat ? 'ut-teks-galat' : undefined}></textarea>
      <p class="text-right text-xs text-slate-500">{utTeks.length}/{MAKS_TEKS}</p>
    </Medan>
    <fieldset class="flex flex-col gap-1">
      <legend class="mb-1 text-sm font-medium text-slate-800">Suara</legend>
      {#each suara as s (s.id)}
        <label class="flex min-h-11 items-center gap-3 rounded-xl px-2 hover:bg-slate-50">
          <input type="radio" name="suara-ubah" value={s.id} bind:group={utSuara} class="size-5" />
          {s.nama}
        </label>
      {/each}
    </fieldset>
    {#if utGalatForm}<p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{utGalatForm}</p>{/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (utBuka = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-teks" sibuk={utSimpan}>Simpan</Tombol>
  {/snippet}
</PanelGeser>
