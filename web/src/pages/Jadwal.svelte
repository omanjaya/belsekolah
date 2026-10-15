<script lang="ts">
  import { onMount, untrack } from 'svelte'
  import { api, ApiError, unduh, type Audio, type Jadwal, type JenisBel, type Pengaturan, type Profil, type Status, type SuaraRingkas } from '../lib/api'
  import { NAMA_HARI } from '../lib/format'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import { baca, tulis } from '../lib/simpan'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import BarAlat from '../components/BarAlat.svelte'
  import Tabel from '../components/Tabel.svelte'
  import PanelGeser from '../components/PanelGeser.svelte'
  import JadwalMinggu from '../components/JadwalMinggu.svelte'
  import Tombol from '../components/Tombol.svelte'
  import Medan from '../components/Medan.svelte'
  import Kartu from '../components/Kartu.svelte'
  import Memuat from '../components/Memuat.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import TambahBanyak from '../components/TambahBanyak.svelte'
  import PilihSuara from '../components/PilihSuara.svelte'
  import IsiPengumuman from '../components/IsiPengumuman.svelte'

  const masukan =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const HARI = [1, 2, 3, 4, 5, 6, 7]

  type Tampilan = 'minggu' | 'hari'
  const bacaTampilan = (): Tampilan => (baca<string>('tampilanJadwal', 'minggu') === 'hari' ? 'hari' : 'minggu')

  let profil = $state<Profil[]>([])
  let jenis = $state<JenisBel[]>([])
  let audio = $state<Audio[]>([])
  let defaultId = $state(0)
  let profilId = $state(0)
  let hari = $state(1)
  let tampilan = $state<Tampilan>(bacaTampilan())
  let semua = $state<Jadwal[]>([])
  let memuat = $state(true)
  let galatMuat = $state('')
  let memuatDaftar = $state(false)
  let galatDaftar = $state(false)
  let urutMuat = 0
  let mengekspor = $state(false)
  let bukaBanyak = $state(false)
  let bukaIsi = $state(false)
  const profilAktif = $derived(profil.find((p) => p.id === profilId))

  const jam5 = (j: string) => j.slice(0, 5)
  const hariIni = $derived(semua.filter((j) => j.hari === hari).sort((a, b) => jam5(a.jam).localeCompare(jam5(b.jam))))
  const jumlah = (h: number) => semua.filter((j) => j.hari === h).length
  const jenisPer = (id: number) => jenis.find((j) => j.id === id)
  const namaSuara = (jb: JenisBel | undefined) =>
    (jb?.audioIds ?? []).map((id) => audio.find((a) => a.id === id)?.nama).filter(Boolean).join(', ') || '-'

  const namaPengumuman = (id: number | null) => (id === null ? undefined : audio.find((a) => a.id === id)?.nama)

  const belMinggu = $derived(
    semua.map((j) => {
      const jb = jenisPer(j.jenisBelId)
      return { kunci: j.id, hari: j.hari, jam: j.jam, label: j.label, jenisNama: jb?.nama ?? 'Bel', warna: jb?.warna ?? '#94a3b8', pengumuman: namaPengumuman(j.pengumumanId) }
    }),
  )

  function setTampilan(t: Tampilan) {
    tampilan = t
    tulis('tampilanJadwal', t)
  }

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

  async function selesaiGabung() {
    // Gabung bisa membuat jenis bel baru; muat ulang bersama jadwal.
    try {
      jenis = await api.get<JenisBel[]>('/jenis-bel')
    } catch (e) {
      galat(e)
    }
    await muatJadwal()
  }

  async function muatAwal() {
    galatMuat = ''
    memuat = true
    try {
      const stP = api.get<Status>('/status').catch(() => null)
      const audioP = api.get<Audio[]>('/audio').catch(() => [] as Audio[])
      const [p, j, s, st, a] = await Promise.all([
        api.get<Profil[]>('/profil'),
        api.get<JenisBel[]>('/jenis-bel'),
        api.get<Pengaturan>('/pengaturan'),
        stP,
        audioP,
      ])
      profil = p
      jenis = j
      audio = a
      defaultId = s.profilDefaultId
      const dariUrl = Number(new URLSearchParams(location.search).get('profil'))
      profilId = p.some((x) => x.id === dariUrl)
        ? dariUrl
        : p.some((x) => x.id === s.profilDefaultId)
          ? s.profilDefaultId
          : (p[0]?.id ?? 0)
      if (st && st.hari >= 1 && st.hari <= 7) hari = st.hari
      await muatJadwal(true)
    } catch (e) {
      galatMuat = e instanceof Error ? e.message : String(e)
    } finally {
      memuat = false
    }
  }
  onMount(muatAwal)

  // Panel tambah/ubah
  let bukaForm = $state(false)
  let ubahId = $state<number | null>(null)
  let fHari = $state(1)
  let fJam = $state('')
  let fJenis = $state(0)
  let fLabel = $state('')
  let fPengumuman = $state<number | null>(null)
  let pengumumanManual = $state(false)
  let otomatis = $state(false)
  let urutCocok = 0
  let galatForm = $state('')
  let simpan = $state(false)

  function bukaTambah() {
    ubahId = null
    fHari = hari
    fJam = ''
    fJenis = jenis[0]?.id ?? 0
    fLabel = ''
    fPengumuman = null
    pengumumanManual = false
    otomatis = false
    galatForm = ''
    bukaForm = true
  }

  function bukaUbah(j: Jadwal) {
    ubahId = j.id
    fHari = j.hari
    fJam = jam5(j.jam)
    fJenis = j.jenisBelId
    fLabel = j.label
    fPengumuman = j.pengumumanId ?? null
    pengumumanManual = j.pengumumanId != null
    otomatis = false
    galatForm = ''
    bukaForm = true
  }

  // Isi pengumuman otomatis dari keterangan selama belum dipilih manual.
  $effect(() => {
    if (!bukaForm || pengumumanManual) return
    const ket = fLabel.trim()
    untrack(() => urutCocok++)
    const no = untrack(() => urutCocok)
    if (!ket) {
      untrack(() => {
        fPengumuman = null
        otomatis = false
      })
      return
    }
    const t = setTimeout(async () => {
      try {
        const r = await api.get<{ suara: SuaraRingkas | null }>(`/pengumuman/cocok?keterangan=${encodeURIComponent(ket)}`)
        if (no !== urutCocok || pengumumanManual) return
        fPengumuman = r?.suara?.id ?? null
        otomatis = !!r?.suara
      } catch {
        // Diabaikan: pengisian otomatis hanyalah bantuan.
      }
    }, 300)
    return () => clearTimeout(t)
  })

  function pilihKartu(kunci: string | number) {
    const j = semua.find((x) => x.id === kunci)
    if (j) bukaUbah(j)
  }

  async function kirimForm(e: SubmitEvent) {
    e.preventDefault()
    galatForm = ''
    if (!fJam) return (galatForm = 'Jam wajib diisi.')
    if (!fJenis) return (galatForm = 'Jenis bel wajib dipilih.')
    simpan = true
    try {
      const data = { profilId, hari: fHari, jam: fJam.slice(0, 5), jenisBelId: fJenis, label: fLabel.trim(), ...(fPengumuman !== null ? { pengumumanId: fPengumuman } : {}) }
      if (ubahId === null) await api.post('/jadwal', data)
      else await api.put(`/jadwal/${ubahId}`, data)
      bukaForm = false
      hari = fHari
      beriTahu('Bel disimpan')
      await muatJadwal()
    } catch (err) {
      galatForm = err instanceof ApiError ? err.message : String(err)
    } finally {
      simpan = false
    }
  }

  async function hapus(j: Jadwal) {
    const ya = await konfirmasi('Hapus bel?', `Bel ${jam5(j.jam)} pada hari ${NAMA_HARI[j.hari]} akan dihapus.`)
    if (!ya) return
    try {
      await api.hapus(`/jadwal/${j.id}`)
      bukaForm = false
      beriTahu('Bel dihapus')
      await muatJadwal()
    } catch (e) {
      galat(e)
    }
  }

  // Panel salin hari
  let bukaSalin = $state(false)
  let dariHari = $state(2)
  let keHari = $state(1)
  let salin = $state(false)
  let galatSalin = $state('')

  function bukaSalinan() {
    keHari = hari
    dariHari = HARI.find((h) => h !== hari) ?? 1
    galatSalin = ''
    bukaSalin = true
  }

  async function kirimSalin(e: SubmitEvent) {
    e.preventDefault()
    galatSalin = ''
    if (dariHari === keHari) return (galatSalin = 'Hari asal dan tujuan harus berbeda.')
    salin = true
    try {
      await api.post('/jadwal/salin', { profilId, dariHari, keHari })
      bukaSalin = false
      hari = keHari
      beriTahu('Jadwal disalin')
      await muatJadwal()
    } catch (err) {
      galatSalin = err instanceof Error ? err.message : String(err)
    } finally {
      salin = false
    }
  }

  async function ekspor() {
    mengekspor = true
    try {
      await unduh(`/profil/${profilId}/ekspor`)
      beriTahu('Jadwal diekspor')
    } catch (e) {
      galat(e)
    } finally {
      mengekspor = false
    }
  }

  const tombolTampilan = (aktif: boolean) =>
    `min-h-11 px-4 py-2 font-medium transition ${aktif ? 'bg-blue-700 text-white' : 'bg-white text-slate-700 hover:bg-slate-50'}`
</script>

<JudulHalaman judul="Jadwal" keterangan="Atur jam bel untuk setiap hari." />

{#if memuat}
  <Memuat />
{:else if galatMuat}
  <GalatMuat pesan={galatMuat} oncoba={muatAwal} />
{:else}
  <BarAlat>
    {#snippet kiri()}
      <div class="flex items-center gap-2">
        <label for="jadwal-profil" class="text-sm font-medium text-slate-700">Profil</label>
        <select id="jadwal-profil" class="{masukan} w-auto min-w-0" bind:value={profilId} onchange={() => muatJadwal(true)}>
          {#each profil as p (p.id)}
            <option value={p.id}>{p.nama}{p.id === defaultId ? ' (default)' : ''}</option>
          {/each}
        </select>
      </div>
      <div class="inline-flex overflow-hidden rounded-xl ring-1 ring-slate-300" role="group" aria-label="Tampilan">
        <button type="button" class={tombolTampilan(tampilan === 'minggu')} aria-pressed={tampilan === 'minggu'} onclick={() => setTampilan('minggu')}>Minggu</button>
        <button type="button" class={tombolTampilan(tampilan === 'hari')} aria-pressed={tampilan === 'hari'} onclick={() => setTampilan('hari')}>Hari</button>
      </div>
    {/snippet}
    {#snippet kanan()}
      <Tombol varian="kedua" onclick={bukaSalinan} disabled={!profilId}>Salin hari…</Tombol>
      <Tombol varian="kedua" onclick={ekspor} sibuk={mengekspor} disabled={!profilId}>Ekspor</Tombol>
      <Tombol varian="kedua" onclick={() => (bukaBanyak = true)} disabled={!profilId}>Tambah banyak</Tombol>
      <Tombol varian="kedua" onclick={() => (bukaIsi = true)} disabled={!profilAktif}>Isi pengumuman otomatis</Tombol>
      <Tombol onclick={bukaTambah} disabled={!profilId}>+ Tambah bel</Tombol>
    {/snippet}
  </BarAlat>

  {#if memuatDaftar}
    <Memuat />
  {:else if galatDaftar}
    <GalatMuat pesan="Daftar bel gagal dimuat." oncoba={() => muatJadwal(true)} />
  {:else if tampilan === 'minggu'}
    <JadwalMinggu bel={belMinggu} onpilih={pilihKartu} />
  {:else}
    <div class="grid grid-cols-1 gap-4 md:grid-cols-[14rem_minmax(0,1fr)]">
      <Kartu>
        <div role="tablist" aria-label="Hari" aria-orientation="vertical" class="flex gap-2 overflow-x-auto md:flex-col md:overflow-visible">
          {#each HARI as h (h)}
            <button
              type="button"
              role="tab"
              id="tab-hari-{h}"
              aria-selected={hari === h}
              aria-controls="panel-hari"
              class="flex min-h-11 shrink-0 items-center justify-between gap-3 rounded-xl px-4 py-2 text-left font-medium transition {hari === h
                ? 'bg-blue-700 text-white'
                : 'bg-white text-slate-700 ring-1 ring-slate-300 hover:bg-slate-50'}"
              onclick={() => (hari = h)}
            >
              <span>{NAMA_HARI[h]}</span>
              <span class="whitespace-nowrap text-sm font-normal">{jumlah(h)} bel</span>
            </button>
          {/each}
        </div>
      </Kartu>

      <div id="panel-hari" role="tabpanel" aria-labelledby="tab-hari-{hari}" class="min-w-0">
        <Tabel label="Bel hari {NAMA_HARI[hari]}" jumlah={hariIni.length} kosong="Belum ada bel pada hari {NAMA_HARI[hari]}">
          {#snippet kepala()}
            <th scope="col">Jam</th>
            <th scope="col">Keterangan</th>
            <th scope="col">Jenis bel</th>
            <th scope="col">Suara</th>
            <th scope="col">Pengumuman</th>
            <th scope="col" class="text-right">Aksi</th>
          {/snippet}
          {#each hariIni as j (j.id)}
            {@const jb = jenisPer(j.jenisBelId)}
            <tr>
              <td class="text-lg font-bold tabular-nums">{jam5(j.jam)}</td>
              <td>{j.label || jb?.nama || 'Bel'}</td>
              <td>
                {#if jb}
                  <span class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium text-slate-800 ring-1 ring-slate-300">
                    <span class="size-2.5 rounded-full" style="background-color: {jb.warna}" aria-hidden="true"></span>
                    {jb.nama}
                  </span>
                {/if}
              </td>
              <td>{namaSuara(jb)}</td>
              <td>{namaPengumuman(j.pengumumanId) ?? '—'}</td>
              <td class="text-right whitespace-nowrap">
                <button type="button" class="min-h-11 rounded-lg px-3 text-slate-700 hover:bg-slate-100" aria-label="Ubah {jam5(j.jam)}" onclick={() => bukaUbah(j)}>Ubah</button>
                <button type="button" class="min-h-11 rounded-lg px-3 text-red-700 hover:bg-red-50" aria-label="Hapus {jam5(j.jam)}" onclick={() => hapus(j)}>Hapus</button>
              </td>
            </tr>
          {/each}
        </Tabel>
      </div>
    </div>
  {/if}
{/if}

<PanelGeser bind:buka={bukaForm} judul={ubahId === null ? 'Tambah bel' : 'Ubah bel'}>
  <form id="form-bel" class="flex flex-col gap-4" onsubmit={kirimForm}>
    <Medan label="Hari" untuk="bel-hari">
      <select id="bel-hari" class={masukan} bind:value={fHari}>
        {#each HARI as h (h)}
          <option value={h}>{NAMA_HARI[h]}</option>
        {/each}
      </select>
    </Medan>
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
    <div class="flex flex-col gap-1">
      <PilihSuara {audio} bind:nilai={fPengumuman} id="f-pengumuman" label="Pengumuman" onpilih={() => ((pengumumanManual = true), (otomatis = false))} />
      {#if otomatis && !pengumumanManual}
        <p class="text-sm text-slate-500">Diisi otomatis dari keterangan</p>
      {/if}
    </div>
    {#if galatForm}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatForm}</p>
    {/if}
  </form>
  {#snippet aksi()}
    {#if ubahId !== null}
      <Tombol
        varian="bahaya-teks"
        class="mr-auto"
        onclick={() => {
          const j = semua.find((x) => x.id === ubahId)
          if (j) hapus(j)
        }}>Hapus bel</Tombol
      >
    {/if}
    <Tombol varian="kedua" onclick={() => (bukaForm = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-bel" sibuk={simpan}>Simpan</Tombol>
  {/snippet}
</PanelGeser>

<PanelGeser bind:buka={bukaSalin} judul="Salin hari">
  <form id="form-salin" class="flex flex-col gap-4" onsubmit={kirimSalin}>
    <Medan label="Salin dari" untuk="salin-dari">
      <select id="salin-dari" class={masukan} bind:value={dariHari}>
        {#each HARI as h (h)}
          <option value={h}>{NAMA_HARI[h]}</option>
        {/each}
      </select>
    </Medan>
    <Medan label="Salin ke" untuk="salin-ke">
      <select id="salin-ke" class={masukan} bind:value={keHari}>
        {#each HARI as h (h)}
          <option value={h}>{NAMA_HARI[h]}</option>
        {/each}
      </select>
    </Medan>
    <p class="rounded-xl bg-amber-50 px-3 py-2 text-sm text-amber-800">
      Semua bel pada hari {NAMA_HARI[keHari]} akan diganti.{#if jumlah(dariHari) === 0}
        Hari {NAMA_HARI[dariHari]} belum punya bel; semua bel hari {NAMA_HARI[keHari]} akan dihapus.{/if}
    </p>
    {#if galatSalin}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatSalin}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => (bukaSalin = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-salin" sibuk={salin}>Salin</Tombol>
  {/snippet}
</PanelGeser>

{#if profilAktif}
  <TambahBanyak bind:buka={bukaBanyak} profil={profilAktif} jenisBel={jenis} onselesai={selesaiGabung} />
  <IsiPengumuman bind:buka={bukaIsi} profil={profilAktif} onselesai={() => muatJadwal()} />
{/if}
