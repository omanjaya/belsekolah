<script lang="ts">
  import { onMount } from 'svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kartu from '../components/Kartu.svelte'
  import Tombol from '../components/Tombol.svelte'
  import Medan from '../components/Medan.svelte'
  import Memuat from '../components/Memuat.svelte'
  import { api, type Audio, type Pengaturan, type Profil, type Sistem } from '../lib/api'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import { formatJam, offsetDariISO } from '../lib/format'

  const input = 'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'

  let memuat = $state(true)
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
  let keluarSibuk = $state(false)

  onMount(async () => {
    try {
      const [p, pr, a, s] = await Promise.all([
        api.get<Pengaturan>('/pengaturan'),
        api.get<Profil[]>('/profil'),
        api.get<Audio[]>('/audio'),
        api.get<Sistem>('/sistem'),
      ])
      volume = p.volume
      profilId = p.profilDefaultId
      profil = pr
      audio = a
      sistem = s
    } catch (e) {
      galat(e)
    } finally {
      memuat = false
    }
  })

  async function simpan() {
    simpanSibuk = true
    try {
      await api.put('/pengaturan', { volume, profilDefaultId: profilId })
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
      await api.put('/pengaturan', { volume, profilDefaultId: profilId })
      await api.post(`/audio/${dingDong.id}/tes`)
      beriTahu('Suara diputar di speaker')
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
    if (baru !== ulangi) { pwGalat = 'Password tidak sama'; return }
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

  async function tungguHidup() {
    const batas = Date.now() + 120_000
    await new Promise((r) => setTimeout(r, 2000))
    while (Date.now() < batas) {
      try {
        await api.get('/sesi')
        location.reload()
        return
      } catch {
        /* server masih mati */
      }
      await new Promise((r) => setTimeout(r, 2000))
    }
    memulaiUlang = false
    galat(new Error('Aplikasi belum menyala kembali. Muat ulang halaman ini beberapa saat lagi.'))
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
      const fd = new FormData()
      fd.append('file', berkas)
      await api.post('/restore', fd)
      memulaiUlang = true
      void tungguHidup()
    } catch (e) {
      galat(e)
    } finally {
      restoreSibuk = false
    }
  }

  async function keluar() {
    keluarSibuk = true
    try {
      await api.post('/logout')
      location.reload()
    } catch (e) {
      galat(e)
      keluarSibuk = false
    }
  }

  const ntp = $derived(
    sistem?.ntp === true
      ? { teks: 'Tersinkron', kelas: 'bg-green-100 text-green-800' }
      : sistem?.ntp === false
        ? { teks: 'Belum tersinkron', kelas: 'bg-amber-100 text-amber-800' }
        : { teks: 'Tidak diketahui', kelas: 'bg-slate-200 text-slate-700' },
  )
</script>

<JudulHalaman judul="Pengaturan" keterangan="Suara, keamanan, dan perawatan aplikasi." />

{#if memuat}
  <Memuat />
{:else}
  <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Suara</h2>
      <div class="flex flex-col gap-4">
        <Medan label="Volume: {volume}%" untuk="pg-volume">
          <input id="pg-volume" type="range" min="0" max="100" step="1" bind:value={volume} class="h-11 w-full accent-blue-700" />
        </Medan>
        <Medan label="Profil default" untuk="pg-profil">
          <select id="pg-profil" bind:value={profilId} class={input}>
            {#each profil as p (p.id)}<option value={p.id}>{p.nama}</option>{/each}
          </select>
        </Medan>
        <div class="flex flex-wrap gap-2">
          <Tombol onclick={simpan} sibuk={simpanSibuk}>Simpan pengaturan</Tombol>
          <Tombol varian="kedua" onclick={tes} sibuk={tesSibuk}>Tes volume</Tombol>
        </div>
      </div>
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
        {#if pwGalat}<p class="text-sm text-red-600" role="alert">{pwGalat}</p>{/if}
        <div><Tombol type="submit" sibuk={pwSibuk}>Ganti password</Tombol></div>
      </form>
    </Kartu>

    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Info sistem</h2>
      {#if sistem}
        <dl class="grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-2">
          <dt class="text-slate-600">Versi</dt>
          <dd class="break-words font-medium">{sistem.versi}</dd>
          <dt class="text-slate-600">Zona waktu</dt>
          <dd class="break-words font-medium">{sistem.zonaWaktu}</dd>
          <dt class="text-slate-600">Jam server</dt>
          <dd class="font-medium tabular-nums">{formatJam(Date.parse(sistem.sekarang), offsetDariISO(sistem.sekarang))}</dd>
          <dt class="text-slate-600">Sinkron jam</dt>
          <dd><span class="rounded-full px-2.5 py-0.5 text-sm font-medium {ntp.kelas}">{ntp.teks}</span></dd>
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

  <div class="mt-6">
    <Tombol varian="kedua" onclick={keluar} sibuk={keluarSibuk}>Keluar</Tombol>
  </div>
{/if}

{#if memulaiUlang}
  <div class="fixed inset-0 z-50 flex flex-col items-center justify-center gap-4 bg-white/95 px-6 text-center" role="alertdialog" aria-live="assertive" aria-label="Memulai ulang aplikasi">
    <span class="size-10 animate-spin rounded-full border-4 border-blue-700 border-t-transparent" aria-hidden="true"></span>
    <p class="text-xl font-semibold text-slate-900">Memulai ulang aplikasi…</p>
    <p class="text-slate-600">Mohon tunggu, halaman akan dimuat ulang otomatis.</p>
  </div>
{/if}
