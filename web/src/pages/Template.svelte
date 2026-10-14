<script lang="ts">
  import { onMount } from 'svelte'
  import { Download, Pencil, Play, Trash2 } from '@lucide/svelte'
  import { api, unduh, type FileJadwal, type ParameterPembuat, type Template } from '../lib/api'
  import { formatTanggal, NAMA_HARI } from '../lib/format'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import { pindah } from '../lib/router.svelte'
  import { kePratinjau, nilaiAwal } from '../lib/pembuat'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kartu from '../components/Kartu.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import PanelGeser from '../components/PanelGeser.svelte'
  import Tabel from '../components/Tabel.svelte'
  import BarAlat from '../components/BarAlat.svelte'
  import AlurImpor from '../components/AlurImpor.svelte'
  import JadwalMinggu from '../components/JadwalMinggu.svelte'
  import Medan from '../components/Medan.svelte'
  import Tombol from '../components/Tombol.svelte'

  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const kelasIkon = 'inline-grid size-11 place-items-center rounded-lg text-slate-700 hover:bg-slate-100'
  const kelasCentang = 'size-5 accent-blue-700'

  const JENJANG = [
    { nilai: 'SD', label: 'SD' },
    { nilai: 'SMP', label: 'SMP' },
    { nilai: 'SMA', label: 'SMA/SMK' },
  ] as const
  const VARIAN = [
    { nilai: 'normal', label: 'Normal' },
    { nilai: 'ramadan', label: 'Ramadan' },
    { nilai: 'ujian', label: 'Ujian' },
  ] as const

  // --- Pembuat jadwal ---
  let param = $state<ParameterPembuat>(nilaiAwal('SMP', 'normal'))
  let istirahatAktif = $state([true, true])
  let pratinjau = $state<FileJadwal | null>(null)
  let galatPratinjau = $state('')
  let menghitung = $state(false)
  let urutan = 0

  function ganti(jenjang: ParameterPembuat['jenjang'], varian: ParameterPembuat['varian']) {
    param = nilaiAwal(jenjang, varian)
    istirahatAktif = param.istirahat.map(() => true)
  }

  // Angka dari input kosong (null) dikirim 0 agar server yang menolaknya.
  const angka = (n: number | null | undefined) => (typeof n === 'number' && Number.isFinite(n) ? n : 0)

  const badan = $derived<ParameterPembuat>({
    ...$state.snapshot(param),
    hariSekolah: param.hariSekolah,
    lamaJP: angka(param.lamaJP),
    jumlahJP: angka(param.jumlahJP),
    jumlahJPJumat: angka(param.jumlahJPJumat),
    jumlahJPSabtu: angka(param.jumlahJPSabtu),
    istirahat: param.istirahat
      .map((s, i) => ({ setelahJP: angka(s.setelahJP), menit: angka(s.menit), aktif: istirahatAktif[i] }))
      .filter((s) => s.aktif)
      .map(({ setelahJP, menit }) => ({ setelahJP, menit })),
    upacaraSenin: { aktif: param.upacaraSenin.aktif, menit: angka(param.upacaraSenin.menit) },
    kegiatanJumat: { ...param.kegiatanJumat, menit: angka(param.kegiatanJumat.menit) },
    ujian: {
      jamMulai: param.ujian.jamMulai,
      jumlahSesi: angka(param.ujian.jumlahSesi),
      lamaSesi: angka(param.ujian.lamaSesi),
      jeda: angka(param.ujian.jeda),
    },
  })

  // Debounce 300 ms; respons yang sudah usang diabaikan.
  $effect(() => {
    const kirim = JSON.stringify(badan)
    const timer = setTimeout(async () => {
      const saya = ++urutan
      menghitung = true
      try {
        const hasil = await api.post<FileJadwal>('/template/buat', JSON.parse(kirim))
        if (saya !== urutan) return
        pratinjau = hasil
        galatPratinjau = ''
      } catch (e) {
        if (saya !== urutan) return
        galatPratinjau = e instanceof Error ? e.message : String(e)
      } finally {
        if (saya === urutan) menghitung = false
      }
    }, 300)
    return () => clearTimeout(timer)
  })

  const belPratinjau = $derived(pratinjau ? kePratinjau(pratinjau) : [])
  const hariSekolah = $derived(param.hariSekolah === 6 ? [1, 2, 3, 4, 5, 6] : [1, 2, 3, 4, 5])

  let bukaBuat = $state(false)
  let dataBuat = $state<FileJadwal | null>(null)

  function buatProfil() {
    if (!pratinjau) return
    dataBuat = pratinjau
    bukaBuat = true
  }

  const keJadwal = (id: number) => pindah(`/jadwal?profil=${id}`)

  // --- Template saya ---
  let memuat = $state(true)
  let daftar = $state<Template[]>([])
  let galatMuat = $state('')

  async function muat() {
    galatMuat = ''
    memuat = true
    try {
      daftar = await api.get<Template[]>('/template')
    } catch (e) {
      galatMuat = e instanceof Error ? e.message : String(e)
    } finally {
      memuat = false
    }
  }
  onMount(muat)

  function ringkasHari(hari: number[]): string {
    const h = [...hari].sort((a, b) => a - b)
    if (h.length === 0) return '—'
    const berurutan = h.every((x, i) => i === 0 || x === h[i - 1] + 1)
    const pendek = (x: number) => NAMA_HARI[x].slice(0, 3)
    return berurutan && h.length > 2 ? `${pendek(h[0])}–${pendek(h[h.length - 1])}` : h.map(pendek).join(', ')
  }

  let inputBerkas = $state<HTMLInputElement>()

  async function pilihBerkas(e: Event) {
    const input = e.currentTarget as HTMLInputElement
    const berkas = input.files?.[0]
    if (!berkas) return
    const teks = await berkas.text()
    input.value = ''
    let data: FileJadwal
    try {
      data = JSON.parse(teks) as FileJadwal
    } catch {
      beriTahu('File bukan file jadwal', 'galat')
      return
    }
    try {
      await api.post('/template', { nama: data.nama, data })
      beriTahu('Template diimpor')
      await muat()
    } catch (err) {
      galat(err)
    }
  }

  let bukaPakai = $state(false)
  let dataPakai = $state<FileJadwal | null>(null)

  async function pakai(t: Template) {
    try {
      const lengkap = await api.get<Template>(`/template/${t.id}`)
      if (!lengkap.data) throw new Error('Template tidak berisi data jadwal')
      dataPakai = lengkap.data
      bukaPakai = true
    } catch (err) {
      galat(err)
    }
  }

  async function ekspor(t: Template) {
    try {
      await unduh(`/template/${t.id}/ekspor`)
    } catch (err) {
      galat(err)
    }
  }

  let bukaNama = $state(false)
  let ubahId = $state(0)
  let ubahNama = $state('')
  let sibukNama = $state(false)
  let galatNama = $state('')

  function bukaUbahNama(t: Template) {
    ubahId = t.id
    ubahNama = t.nama
    galatNama = ''
    bukaNama = true
  }

  async function simpanNama(e: SubmitEvent) {
    e.preventDefault()
    galatNama = ''
    sibukNama = true
    try {
      await api.put(`/template/${ubahId}`, { nama: ubahNama })
      bukaNama = false
      beriTahu('Nama template disimpan')
      await muat()
    } catch (err) {
      galatNama = err instanceof Error ? err.message : String(err)
    } finally {
      sibukNama = false
    }
  }

  async function hapusTemplate(t: Template) {
    if (!(await konfirmasi('Hapus template?', `Template "${t.nama}" akan dihapus. Profil yang sudah dibuat darinya tidak ikut terhapus.`))) return
    try {
      await api.hapus(`/template/${t.id}`)
      beriTahu('Template dihapus')
      await muat()
    } catch (err) {
      galat(err)
    }
  }
</script>

<JudulHalaman
  judul="Template"
  keterangan="Buat jadwal dari pola bawaan, atau simpan dan pakai ulang jadwal Anda sendiri."
/>

<div class="flex flex-col gap-6">
  <Kartu>
    <h2 class="mb-4 text-lg font-semibold">Pembuat jadwal</h2>
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-[380px_1fr]">
      <form class="flex min-w-0 flex-col gap-4" aria-label="Pembuat jadwal" onsubmit={(e) => e.preventDefault()}>
        <fieldset class="flex flex-col gap-1.5">
          <legend class="mb-1.5 text-sm font-medium text-slate-800">Jenjang</legend>
          <div class="grid grid-cols-3 gap-2">
            {#each JENJANG as j (j.nilai)}
              <Tombol
                varian={param.jenjang === j.nilai ? 'utama' : 'kedua'}
                aria-pressed={param.jenjang === j.nilai}
                onclick={() => ganti(j.nilai, param.varian)}>{j.label}</Tombol
              >
            {/each}
          </div>
        </fieldset>
        <fieldset class="flex flex-col gap-1.5">
          <legend class="mb-1.5 text-sm font-medium text-slate-800">Varian</legend>
          <div class="grid grid-cols-3 gap-2">
            {#each VARIAN as v (v.nilai)}
              <Tombol
                varian={param.varian === v.nilai ? 'utama' : 'kedua'}
                aria-pressed={param.varian === v.nilai}
                onclick={() => ganti(param.jenjang, v.nilai)}>{v.label}</Tombol
              >
            {/each}
          </div>
        </fieldset>
        <fieldset class="flex flex-col gap-1.5">
          <legend class="mb-1.5 text-sm font-medium text-slate-800">Hari sekolah</legend>
          <div class="grid grid-cols-2 gap-2">
            <Tombol
              varian={param.hariSekolah === 5 ? 'utama' : 'kedua'}
              aria-pressed={param.hariSekolah === 5}
              onclick={() => (param.hariSekolah = 5)}>5 hari (Sen–Jum)</Tombol
            >
            <Tombol
              varian={param.hariSekolah === 6 ? 'utama' : 'kedua'}
              aria-pressed={param.hariSekolah === 6}
              onclick={() => (param.hariSekolah = 6)}>6 hari (Sen–Sab)</Tombol
            >
          </div>
        </fieldset>

        {#if param.varian === 'ujian'}
          <Medan label="Jam mulai" untuk="pb-ujian-mulai">
            <input id="pb-ujian-mulai" type="time" class={kelasInput} bind:value={param.ujian.jamMulai} />
          </Medan>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-3 lg:grid-cols-1 xl:grid-cols-3">
            <Medan label="Jumlah sesi" untuk="pb-ujian-sesi">
              <input id="pb-ujian-sesi" type="number" inputmode="numeric" class={kelasInput} bind:value={param.ujian.jumlahSesi} />
            </Medan>
            <Medan label="Lama sesi (menit)" untuk="pb-ujian-lama">
              <input id="pb-ujian-lama" type="number" inputmode="numeric" class={kelasInput} bind:value={param.ujian.lamaSesi} />
            </Medan>
            <Medan label="Jeda (menit)" untuk="pb-ujian-jeda">
              <input id="pb-ujian-jeda" type="number" inputmode="numeric" class={kelasInput} bind:value={param.ujian.jeda} />
            </Medan>
          </div>
        {:else}
          <div class="grid grid-cols-2 gap-3">
            <Medan label="Jam masuk" untuk="pb-masuk">
              <input id="pb-masuk" type="time" class={kelasInput} bind:value={param.jamMasuk} />
            </Medan>
            <Medan label="Lama 1 JP (menit)" untuk="pb-lama">
              <input id="pb-lama" type="number" inputmode="numeric" class={kelasInput} bind:value={param.lamaJP} />
            </Medan>
            <Medan label="JP Senin–Kamis" untuk="pb-jp">
              <input id="pb-jp" type="number" inputmode="numeric" class={kelasInput} bind:value={param.jumlahJP} />
            </Medan>
            <Medan label="JP Jumat" untuk="pb-jp-jumat">
              <input id="pb-jp-jumat" type="number" inputmode="numeric" class={kelasInput} bind:value={param.jumlahJPJumat} />
            </Medan>
            {#if param.hariSekolah === 6}
              <Medan label="JP Sabtu" untuk="pb-jp-sabtu">
                <input id="pb-jp-sabtu" type="number" inputmode="numeric" class={kelasInput} bind:value={param.jumlahJPSabtu} />
              </Medan>
            {/if}
          </div>

          {#each param.istirahat as s, i (i)}
            <fieldset class="flex flex-col gap-2 rounded-xl border border-slate-200 p-3">
              <legend class="px-1 text-sm font-medium text-slate-800">
                <label class="flex min-h-11 items-center gap-2">
                  <input type="checkbox" class={kelasCentang} bind:checked={istirahatAktif[i]} />
                  Istirahat {i + 1}
                </label>
              </legend>
              <div class="grid grid-cols-2 gap-3">
                <Medan label="Setelah JP ke-" untuk="pb-ist-{i}-jp">
                  <input id="pb-ist-{i}-jp" type="number" inputmode="numeric" class={kelasInput} disabled={!istirahatAktif[i]} bind:value={s.setelahJP} />
                </Medan>
                <Medan label="Lama (menit)" untuk="pb-ist-{i}-menit">
                  <input id="pb-ist-{i}-menit" type="number" inputmode="numeric" class={kelasInput} disabled={!istirahatAktif[i]} bind:value={s.menit} />
                </Medan>
              </div>
            </fieldset>
          {/each}

          <fieldset class="flex flex-col gap-2 rounded-xl border border-slate-200 p-3">
            <legend class="px-1 text-sm font-medium text-slate-800">
              <label class="flex min-h-11 items-center gap-2">
                <input type="checkbox" class={kelasCentang} bind:checked={param.upacaraSenin.aktif} />
                Upacara Senin
              </label>
            </legend>
            <Medan label="Lama upacara (menit)" untuk="pb-upacara">
              <input id="pb-upacara" type="number" inputmode="numeric" class={kelasInput} disabled={!param.upacaraSenin.aktif} bind:value={param.upacaraSenin.menit} />
            </Medan>
          </fieldset>

          <fieldset class="flex flex-col gap-2 rounded-xl border border-slate-200 p-3">
            <legend class="px-1 text-sm font-medium text-slate-800">
              <label class="flex min-h-11 items-center gap-2">
                <input type="checkbox" class={kelasCentang} bind:checked={param.kegiatanJumat.aktif} />
                Kegiatan pagi Jumat
              </label>
            </legend>
            <div class="grid grid-cols-2 gap-3">
              <Medan label="Lama kegiatan (menit)" untuk="pb-jumat-menit">
                <input id="pb-jumat-menit" type="number" inputmode="numeric" class={kelasInput} disabled={!param.kegiatanJumat.aktif} bind:value={param.kegiatanJumat.menit} />
              </Medan>
              <Medan label="Nama kegiatan" untuk="pb-jumat-label">
                <input id="pb-jumat-label" class={kelasInput} maxlength="100" disabled={!param.kegiatanJumat.aktif} bind:value={param.kegiatanJumat.label} />
              </Medan>
            </div>
          </fieldset>
        {/if}
      </form>

      <div class="flex min-w-0 flex-col gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <h3 class="text-base font-semibold">Pratinjau</h3>
          {#if menghitung}<span class="text-sm text-slate-600" role="status">Menghitung…</span>{/if}
          <Tombol class="md:ml-auto" onclick={buatProfil} disabled={!pratinjau || !!galatPratinjau}>
            Buat profil dari jadwal ini
          </Tombol>
        </div>
        {#if galatPratinjau}
          <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatPratinjau}</p>
        {/if}
        {#if pratinjau}
          <div class={galatPratinjau ? 'opacity-50' : ''} data-testid="pratinjau">
            <JadwalMinggu bel={belPratinjau} {hariSekolah} padat />
          </div>
        {:else if !galatPratinjau}
          <p class="text-slate-600" role="status">Menyiapkan pratinjau…</p>
        {/if}
      </div>
    </div>
  </Kartu>

  <Kartu>
    <h2 class="mb-3 text-lg font-semibold">Template saya</h2>
    {#if galatMuat}
      <GalatMuat pesan={galatMuat} oncoba={muat} />
    {:else}
      <BarAlat>
        {#snippet kanan()}
          <input
            bind:this={inputBerkas}
            type="file"
            accept=".json,application/json"
            class="hidden"
            aria-label="Berkas template"
            onchange={pilihBerkas}
          />
          <Tombol varian="kedua" onclick={() => inputBerkas?.click()}>Impor template</Tombol>
        {/snippet}
      </BarAlat>
      <Tabel
        label="Daftar template"
        {memuat}
        jumlah={daftar.length}
        kosong="Belum ada template. Simpan profil sebagai template dari halaman Profil & Periode."
      >
        {#snippet kepala()}
          <th scope="col">Nama</th>
          <th scope="col" class="whitespace-nowrap">Jumlah bel</th>
          <th scope="col">Hari</th>
          <th scope="col">Dibuat</th>
          <th scope="col">Aksi</th>
        {/snippet}
        {#each daftar as t (t.id)}
          <tr>
            <td class="break-words font-medium">{t.nama}</td>
            <td>{t.jumlahBel}</td>
            <td class="whitespace-nowrap">{ringkasHari(t.hari)}</td>
            <td class="whitespace-nowrap">{formatTanggal(t.dibuat.slice(0, 10))}</td>
            <td>
              <div class="flex gap-1">
                <button type="button" class={kelasIkon} aria-label="Pakai {t.nama}" title="Pakai" onclick={() => pakai(t)}>
                  <Play class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class={kelasIkon} aria-label="Ekspor {t.nama}" title="Ekspor" onclick={() => ekspor(t)}>
                  <Download class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class={kelasIkon} aria-label="Ganti nama {t.nama}" title="Ganti nama" onclick={() => bukaUbahNama(t)}>
                  <Pencil class="size-5" aria-hidden="true" />
                </button>
                <button type="button" class="{kelasIkon} text-red-700" aria-label="Hapus {t.nama}" title="Hapus" onclick={() => hapusTemplate(t)}>
                  <Trash2 class="size-5" aria-hidden="true" />
                </button>
              </div>
            </td>
          </tr>
        {/each}
      </Tabel>
    {/if}
  </Kartu>
</div>

<PanelGeser bind:buka={bukaNama} judul="Ganti nama template">
  <form id="form-nama-template" class="flex flex-col gap-3" onsubmit={simpanNama}>
    <Medan label="Nama template" untuk="nama-template">
      <input id="nama-template" class={kelasInput} bind:value={ubahNama} required maxlength="60" />
    </Medan>
    {#if galatNama}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatNama}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaNama = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-nama-template" sibuk={sibukNama}>Simpan</Tombol>
  {/snippet}
</PanelGeser>

<AlurImpor bind:buka={bukaBuat} data={dataBuat} judul="Buat profil dari jadwal" onselesai={keJadwal} />
<AlurImpor bind:buka={bukaPakai} data={dataPakai} judul="Pakai template" onselesai={keJadwal} />
