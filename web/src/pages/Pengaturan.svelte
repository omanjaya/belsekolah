<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kartu from '../components/Kartu.svelte'
  import Tombol from '../components/Tombol.svelte'
  import Medan from '../components/Medan.svelte'
  import Memuat from '../components/Memuat.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import { api, type Audio, type Pengaturan, type Profil, type Sistem } from '../lib/api'
  import { bacaBoot, tungguMulaiUlang } from '../lib/mulaiUlang'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import { formatJam, offsetDariISO } from '../lib/format'

  const input = 'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'

  let memuatSuara = $state(true)
  let memuatSistem = $state(true)
  let galatSuara = $state(false)
  let galatSistem = $state(false)
  let profilTermuat = 0
  let volumeTermuat = 70
  let batal = false
  let tungguHabis = $state(false)
  let volume = $state(70)
  let profilId = $state(0)
  let profil = $state<Profil[]>([])
  let audio = $state<Audio[]>([])
  let sistem = $state<Sistem | null>(null)
  let simpanSibuk = $state(false)
  let tesSibuk = $state(false)

  let lama = $state('')
  let baru = $state('')
  let ulangi = $state('')
  let pwSibuk = $state(false)
  let pwGalat = $state('')
  let galatBaru = $state('')
  const galatUlang = $derived(ulangi && ulangi !== baru ? 'Password tidak sama' : '')

  let berkas = $state<File | null>(null)
  let restoreSibuk = $state(false)
  let memulaiUlang = $state(false)
  let mulaiMode = $state<'restore' | 'zona'>('restore')

  const zonaPilihan = [
    { kode: 'WIB', nama: 'WIB — Waktu Indonesia Barat (Jawa, Sumatra, Kalimantan Barat & Tengah)' },
    { kode: 'WITA', nama: 'WITA — Waktu Indonesia Tengah (Bali, NTB, NTT, Sulawesi, Kalimantan Selatan, Timur & Utara)' },
    { kode: 'WIT', nama: 'WIT — Waktu Indonesia Timur (Maluku, Papua)' },
  ]
  let zona = $state('WIB')
  let zonaTermuat = $state('WIB')
  let zonaSibuk = $state(false)

  async function muatSuara() {
    memuatSuara = true
    galatSuara = false
    try {
      const [p, pr, a] = await Promise.all([
        api.get<Pengaturan>('/pengaturan'),
        api.get<Profil[]>('/profil'),
        api.get<Audio[]>('/audio'),
      ])
      volume = p.volume
      profilId = p.profilDefaultId
      profilTermuat = p.profilDefaultId
      volumeTermuat = p.volume
      zona = p.zonaWaktu
      zonaTermuat = p.zonaWaktu
      profil = pr
      audio = a
    } catch {
      galatSuara = true
    } finally {
      memuatSuara = false
    }
  }

  async function muatSistem() {
    memuatSistem = true
    galatSistem = false
    try {
      sistem = await api.get<Sistem>('/sistem')
    } catch {
      galatSistem = true
    } finally {
      memuatSistem = false
    }
  }

  onMount(() => {
    void muatSuara()
    void muatSistem()
  })
  onDestroy(() => { batal = true })

  async function simpan() {
    simpanSibuk = true
    try {
      // Ambil ulang tepat sebelum menyimpan; hanya field yang diubah pengguna yang ditimpa.
      const kini = await api.get<Pengaturan>('/pengaturan')
      const badan = { ...kini }
      if (volume !== volumeTermuat) badan.volume = volume
      if (profilId !== profilTermuat) badan.profilDefaultId = profilId
      await api.put('/pengaturan', badan)
      volumeTermuat = badan.volume
      profilTermuat = badan.profilDefaultId
      beriTahu('Pengaturan disimpan')
    } catch (e) {
      galat(e)
    } finally {
      simpanSibuk = false
    }
  }

  async function tes() {
    const dingDong = audio.find((a) => a.nama === 'Ding Dong') ?? audio.find((a) => a.sumber === 'bawaan')
    if (!dingDong) { galat(new Error('Suara bawaan tidak ditemukan')); return }
    tesSibuk = true
    try {
      // Simpan dulu agar volume yang dipakai sama dengan yang terlihat di slider.
      const v = volume
      await api.put('/pengaturan', { volume: v, profilDefaultId: profilTermuat, zonaWaktu: zonaTermuat })
      await api.post(`/audio/${dingDong.id}/tes`)
      beriTahu(`Volume ${v}% disimpan, suara diputar di speaker`)
    } catch (e) {
      galat(e)
    } finally {
      tesSibuk = false
    }
  }

  async function gantiPassword(e: SubmitEvent) {
    e.preventDefault()
    pwGalat = ''
    galatBaru = ''
    if (baru.length < 8) { galatBaru = 'Password baru minimal 8 karakter'; return }
    if (baru !== ulangi) { if (!galatUlang) pwGalat = 'Password tidak sama'; return }
    pwSibuk = true
    try {
      await api.post('/password', { lama, baru })
      beriTahu('Password berhasil diganti')
      lama = baru = ulangi = ''
    } catch (err) {
      pwGalat = (err as Error).message
    } finally {
      pwSibuk = false
    }
  }

  async function salin(teks: string) {
    try {
      await navigator.clipboard.writeText(teks)
    } catch {
      const t = document.createElement('textarea')
      t.value = teks
      t.style.position = 'fixed'
      t.style.opacity = '0'
      document.body.appendChild(t)
      t.select()
      const ok = document.execCommand('copy')
      t.remove()
      if (!ok) { galat(new Error('Tidak bisa menyalin. Salin alamat secara manual.')); return }
    }
    beriTahu('Alamat disalin')
  }

  async function tungguHidup(bootLama: string | null) {
    tungguHabis = false
    const ok = await tungguMulaiUlang(bootLama, { batal: () => batal })
    if (!ok && !batal) tungguHabis = true
  }

  async function pulihkan() {
    if (!berkas) { galat(new Error('Pilih file backup (.zip) terlebih dahulu')); return }
    const ya = await konfirmasi(
      'Pulihkan backup?',
      'Semua data saat ini (jadwal, suara, pengaturan) akan diganti isi backup. Aplikasi akan dimulai ulang.',
      'Pulihkan',
    )
    if (!ya) return
    restoreSibuk = true
    try {
      const bootLama = await bacaBoot()
      const fd = new FormData()
      fd.append('file', berkas)
      await api.post('/restore', fd)
      mulaiMode = 'restore'
      memulaiUlang = true
      void tungguHidup(bootLama)
    } catch (e) {
      galat(e)
    } finally {
      restoreSibuk = false
    }
  }

  async function simpanZona() {
    const ya = await konfirmasi(
      'Ganti zona waktu?',
      'Aplikasi akan dimulai ulang (±2 detik). Bel yang jatuh tepat saat itu bisa terlewat.',
      'Ganti zona',
    )
    if (!ya) return
    zonaSibuk = true
    try {
      const bootLama = await bacaBoot()
      const kini = await api.get<Pengaturan>('/pengaturan')
      const hasil = await api.put<Pengaturan>('/pengaturan', { ...kini, zonaWaktu: zona })
      if (hasil?.mulaiUlang) {
        mulaiMode = 'zona'
        memulaiUlang = true
        void tungguHidup(bootLama)
      } else {
        zonaTermuat = zona
        beriTahu('Zona waktu disimpan')
      }
    } catch (e) {
      galat(e)
    } finally {
      zonaSibuk = false
    }
  }

  const alatBaris = $derived<[string, boolean][]>(
    sistem?.alat
      ? [
          ['Pemutar audio (mpv)', sistem.alat.mpv],
          ['Pembaca audio (ffprobe)', sistem.alat.ffprobe],
          ['Suara TTS (edge-tts)', sistem.alat.edgeTts],
        ]
      : [],
  )

  function formatUkuranMb(mb: number): string {
    return mb >= 1024 ? `${(mb / 1024).toLocaleString('id-ID', { maximumFractionDigits: 1 })} GB` : `${Math.round(mb).toLocaleString('id-ID')} MB`
  }

  const ntp = $derived(
    sistem?.ntp === true
      ? { teks: 'Tersinkron', kelas: 'bg-green-100 text-green-800' }
      : sistem?.ntp === false
        ? { teks: 'Belum tersinkron', kelas: 'bg-amber-100 text-amber-800' }
        : { teks: 'Tidak diketahui', kelas: 'bg-slate-200 text-slate-700' },
  )
</script>

<JudulHalaman judul="Pengaturan" keterangan="Suara, zona waktu, keamanan, dan perawatan aplikasi." />

<div class="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Volume</h2>
      {#if memuatSuara}
        <Memuat />
      {:else if galatSuara}
        <GalatMuat pesan="Pengaturan gagal dimuat" oncoba={muatSuara} />
      {:else}
        <div class="flex flex-col gap-4">
          <Medan label="Volume: {volume}%" untuk="pg-volume">
            <input id="pg-volume" type="range" min="0" max="100" step="1" bind:value={volume} class="h-11 w-full accent-blue-700" />
          </Medan>
          <div class="flex flex-wrap gap-2">
            <Tombol onclick={simpan} sibuk={simpanSibuk}>Simpan pengaturan</Tombol>
            <Tombol varian="kedua" onclick={tes} sibuk={tesSibuk}>Tes volume</Tombol>
          </div>
        </div>
      {/if}
    </Kartu>

    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Zona waktu</h2>
      {#if memuatSuara}
        <Memuat />
      {:else if galatSuara}
        <GalatMuat pesan="Pengaturan gagal dimuat" oncoba={muatSuara} />
      {:else}
        <fieldset class="flex flex-col gap-2">
          <legend class="sr-only">Zona waktu</legend>
          {#each zonaPilihan as z (z.kode)}
            <label class="flex min-h-11 cursor-pointer items-start gap-3 rounded-xl px-2 py-2 hover:bg-slate-50">
              <input type="radio" name="zona-waktu" value={z.kode} bind:group={zona} class="mt-1 size-5 accent-blue-700" />
              <span>{z.nama}</span>
            </label>
          {/each}
        </fieldset>
        <div class="mt-3"><Tombol onclick={simpanZona} sibuk={zonaSibuk} disabled={zona === zonaTermuat}>Simpan zona</Tombol></div>
      {/if}
    </Kartu>

    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Profil default</h2>
      {#if memuatSuara}
        <Memuat />
      {:else if galatSuara}
        <GalatMuat pesan="Pengaturan gagal dimuat" oncoba={muatSuara} />
      {:else}
        <div class="flex flex-col gap-4">
          <Medan label="Profil default" untuk="pg-profil">
            <select id="pg-profil" bind:value={profilId} class={input}>
              {#each profil as p (p.id)}<option value={p.id}>{p.nama}</option>{/each}
            </select>
          </Medan>
          <div><Tombol onclick={simpan} sibuk={simpanSibuk}>Simpan profil</Tombol></div>
        </div>
      {/if}
    </Kartu>

    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Ganti password</h2>
      <form novalidate onsubmit={gantiPassword} class="flex flex-col gap-4">
        <Medan label="Password lama" untuk="pw-lama">
          <input id="pw-lama" type="password" autocomplete="current-password" bind:value={lama} class={input} />
        </Medan>
        <Medan label="Password baru" untuk="pw-baru" galat={galatBaru} bantuan="Minimal 8 karakter.">
          <input id="pw-baru" type="password" autocomplete="new-password" aria-invalid={galatBaru ? true : undefined} aria-describedby={galatBaru ? 'pw-baru-galat' : undefined} bind:value={baru} class={input} />
        </Medan>
        <Medan label="Ulangi password baru" untuk="pw-ulangi" galat={galatUlang}>
          <input id="pw-ulangi" type="password" autocomplete="new-password" aria-invalid={galatUlang ? true : undefined} aria-describedby={galatUlang ? 'pw-ulangi-galat' : undefined} bind:value={ulangi} class={input} />
        </Medan>
        {#if pwGalat}<p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{pwGalat}</p>{/if}
        <div><Tombol type="submit" sibuk={pwSibuk}>Ganti password</Tombol></div>
      </form>
    </Kartu>

    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Info sistem</h2>
      {#if memuatSistem}
        <Memuat />
      {:else if galatSistem}
        <GalatMuat pesan="Info sistem gagal dimuat" oncoba={muatSistem} />
      {:else if sistem}
        <dl class="grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-2">
          <dt class="text-slate-600">Versi</dt>
          <dd class="break-words font-medium">{sistem.versi}</dd>
          <dt class="text-slate-600">Zona waktu</dt>
          <dd class="break-words font-medium">{sistem.zonaWaktu} ({sistem.zonaIana})</dd>
          <dt class="text-slate-600">Jam server</dt>
          <dd class="font-medium tabular-nums">{formatJam(Date.parse(sistem.sekarang), offsetDariISO(sistem.sekarang))}</dd>
          <dt class="text-slate-600">Sinkron jam</dt>
          <dd><span class="rounded-full px-2.5 py-0.5 text-sm font-medium {ntp.kelas}">{ntp.teks}</span></dd>
          {#if sistem.alat}
            {#each alatBaris as [nama, ada] (nama)}
              <dt class="text-slate-600">{nama}</dt>
              <dd><span class="rounded-full px-2.5 py-0.5 text-sm font-medium {ada ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">{ada ? 'Terpasang' : 'Belum terpasang'}</span></dd>
            {/each}
          {/if}
          {#if sistem.ruangKosongMb != null}
            <dt class="text-slate-600">Ruang disk kosong</dt>
            <dd class="font-medium tabular-nums">{formatUkuranMb(sistem.ruangKosongMb)}</dd>
          {/if}
          {#if sistem.backupTerakhir}
            <dt class="text-slate-600">Backup harian terakhir</dt>
            <dd class="font-medium">{sistem.backupTerakhir}</dd>
          {/if}
        </dl>
        <h3 class="mb-2 mt-4 font-medium text-slate-800">Alamat akses</h3>
        {#if sistem.ip.length === 0}
          <p class="text-slate-500">Alamat jaringan tidak ditemukan.</p>
        {:else}
          <ul class="flex flex-col gap-2">
            {#each sistem.ip as ip (ip)}
              <li class="flex items-center justify-between gap-2">
                <span class="min-w-0 select-all break-all font-mono">http://{ip}</span>
                <Tombol varian="kedua" onclick={() => salin(`http://${ip}`)} aria-label="Salin http://{ip}">Salin</Tombol>
              </li>
            {/each}
          </ul>
        {/if}
      {/if}
    </Kartu>

    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Backup &amp; Restore</h2>
      <div class="flex flex-col gap-4">
        <div>
          <a href="/api/backup" download class="inline-flex min-h-11 items-center justify-center rounded-xl bg-white px-4 py-2 font-medium text-slate-800 ring-1 ring-slate-300 hover:bg-slate-50">Unduh backup</a>
        </div>
        <Medan label="File backup (.zip)" untuk="pg-berkas" bantuan="Memulihkan akan mengganti semua data dan memulai ulang aplikasi.">
          <input id="pg-berkas" type="file" accept=".zip,application/zip"
            onchange={(e) => (berkas = e.currentTarget.files?.[0] ?? null)}
            class="min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base file:mr-3 file:min-h-9 file:rounded-lg file:border-0 file:bg-slate-100 file:px-3 file:py-1 file:font-medium" />
        </Medan>
        <div><Tombol varian="bahaya" onclick={pulihkan} sibuk={restoreSibuk}>Pulihkan dari backup</Tombol></div>
      </div>
    </Kartu>
  </div>

{#if memulaiUlang}
  <div class="fixed inset-0 z-50 flex flex-col items-center justify-center gap-4 bg-white/95 px-6 text-center" role="alertdialog" aria-live="assertive" aria-label={mulaiMode === 'zona' ? 'Menerapkan zona waktu' : 'Memulai ulang aplikasi'}>
    <span class="size-10 {tungguHabis ? 'hidden' : 'animate-spin'} rounded-full border-4 border-blue-700 border-t-transparent" aria-hidden="true"></span>
    {#if tungguHabis}
      <p class="text-xl font-semibold text-slate-900">{mulaiMode === 'zona' ? 'Aplikasi belum kembali. Muat ulang halaman sebentar lagi.' : 'Aplikasi belum menyala kembali. Periksa mini PC, lalu muat ulang halaman ini.'}</p>
      <Tombol onclick={() => location.reload()}>Muat ulang</Tombol>
    {:else}
      <p class="text-xl font-semibold text-slate-900">{mulaiMode === 'zona' ? 'Menerapkan zona waktu…' : 'Memulai ulang aplikasi…'}</p>
      <p class="text-slate-600">Mohon tunggu, halaman akan dimuat ulang otomatis.</p>
    {/if}
  </div>
{/if}
