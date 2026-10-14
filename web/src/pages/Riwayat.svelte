<script lang="ts">
  import { onMount } from 'svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Tombol from '../components/Tombol.svelte'
  import BarAlat from '../components/BarAlat.svelte'
  import Tabel from '../components/Tabel.svelte'
  import Memuat from '../components/Memuat.svelte'
  import GalatMuat from '../components/GalatMuat.svelte'
  import { api, type JenisBel, type Riwayat, type Status } from '../lib/api'
  import { tanggalHariIni } from '../lib/format'

  const input = 'min-h-11 w-auto min-w-0 rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const STATUS: Record<Riwayat['status'], { teks: string; kelas: string }> = {
    berhasil: { teks: 'Berhasil', kelas: 'bg-green-100 text-green-800' },
    gagal: { teks: 'Gagal', kelas: 'bg-red-100 text-red-800' },
    terlewat: { teks: 'Terlewat', kelas: 'bg-slate-200 text-slate-700' },
    cadangan: { teks: 'Nada cadangan', kelas: 'bg-amber-100 text-amber-800' },
    dihentikan: { teks: 'Dihentikan', kelas: 'bg-slate-100 text-slate-800 ring-1 ring-slate-300' },
  }
  const SUMBER = { jadwal: 'Jadwal', manual: 'Manual', darurat: 'Darurat' }

  let tanggal = $state('')
  let status = $state('')
  let daftar = $state<Riwayat[]>([])
  let jenis = $state<JenisBel[]>([])
  let memuat = $state(true)
  let galatMuat = $state('')
  let siap = $state(false)
  let tanggalLokal = $state(false)
  let urut = 0
  const namaJenis = (id?: number) => (id ? (jenis.find((j) => j.id === id)?.nama ?? '-') : '-')

  async function muat() {
    const saya = ++urut
    memuat = true
    galatMuat = ''
    try {
      const q = new URLSearchParams({ batas: '200' })
      if (tanggal) q.set('tanggal', tanggal)
      if (status) q.set('status', status)
      const hasil = await api.get<Riwayat[]>('/riwayat?' + q)
      if (saya === urut) daftar = hasil
    } catch (e) {
      if (saya === urut) galatMuat = e instanceof Error ? e.message : String(e)
    } finally {
      if (saya === urut) memuat = false
    }
  }

  onMount(async () => {
    api.get<JenisBel[]>('/jenis-bel').then((j) => (jenis = j), () => {})
    try {
      tanggal = (await api.get<Status>('/status')).tanggal
    } catch {
      tanggal = tanggalHariIni(Date.now(), -new Date().getTimezoneOffset())
      tanggalLokal = true
    }
    siap = true
  })

  $effect(() => {
    if (!siap) return
    void tanggal
    void status
    daftar = []
    muat()
  })
</script>

<JudulHalaman judul="Riwayat" keterangan="Catatan bel yang sudah berbunyi, gagal, atau terlewat." />

<BarAlat>
  {#snippet kiri()}
    <div class="flex items-center gap-2">
      <label for="rw-tanggal" class="text-sm font-medium text-slate-700">Tanggal</label>
      <input id="rw-tanggal" type="date" bind:value={tanggal} class={input} />
    </div>
    <div class="flex items-center gap-2">
      <label for="rw-status" class="text-sm font-medium text-slate-700">Status</label>
      <select id="rw-status" bind:value={status} class={input}>
        <option value="">Semua</option>
        <option value="berhasil">Berhasil</option>
        <option value="gagal">Gagal</option>
        <option value="terlewat">Terlewat</option>
        <option value="cadangan">Nada cadangan</option>
        <option value="dihentikan">Dihentikan</option>
      </select>
    </div>
    {#if tanggalLokal}<p class="text-sm text-amber-700">Tanggal memakai jam HP karena status server gagal dimuat</p>{/if}
  {/snippet}
  {#snippet kanan()}
    <Tombol varian="kedua" onclick={muat} sibuk={memuat} aria-label="Muat ulang">
      <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 12a9 9 0 1 1-3-6.7L21 8" /><path d="M21 3v5h-5" /></svg>
    </Tombol>
  {/snippet}
</BarAlat>

{#if memuat}
  <Memuat />
{:else if galatMuat}
  <GalatMuat pesan={galatMuat} oncoba={muat} />
{:else}
  <div class="hidden md:block">
  <Tabel label="Riwayat bel" jumlah={daftar.length} kosong="Tidak ada riwayat untuk filter ini">
    {#snippet kepala()}
      <th>Waktu</th>
      <th>Keterangan</th>
      <th>Jenis bel</th>
      <th>Sumber</th>
      <th>Status</th>
      <th>Pesan</th>
    {/snippet}
    {#each daftar as r (r.id)}
      <tr>
        <td class="font-mono font-semibold tabular-nums whitespace-nowrap">{r.waktu.substring(11, 19)}</td>
        <td class="break-words">{r.label}</td>
        <td class="break-words">{namaJenis(r.jenisBelId)}</td>
        <td>{SUMBER[r.sumber]}</td>
        <td><span class="rounded-full px-2.5 py-0.5 text-xs font-medium whitespace-nowrap {STATUS[r.status].kelas}">{STATUS[r.status].teks}</span></td>
        <td class="break-words text-slate-500">{r.pesan || '-'}</td>
      </tr>
    {/each}
  </Tabel>
  </div>

  <ul class="flex flex-col gap-2 md:hidden" aria-label="Riwayat bel">
    {#each daftar as r (r.id)}
      <li class="rounded-xl border border-slate-200 bg-white p-3">
        <div class="flex items-start gap-3">
          <span class="font-mono text-base font-semibold tabular-nums">{r.waktu.substring(11, 19)}</span>
          <span class="min-w-0 flex-1 break-words">{r.label}</span>
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2 text-sm">
          <span class="rounded-full bg-blue-50 px-2.5 py-0.5 font-medium text-blue-800">{SUMBER[r.sumber]}</span>
          <span class="rounded-full px-2.5 py-0.5 font-medium {STATUS[r.status].kelas}">{STATUS[r.status].teks}</span>
          {#if r.jenisBelId}<span class="text-slate-600">{namaJenis(r.jenisBelId)}</span>{/if}
        </div>
        {#if r.pesan}<p class="mt-1.5 break-words text-sm text-slate-500">{r.pesan}</p>{/if}
      </li>
    {:else}
      <li class="rounded-xl border border-slate-200 bg-white px-4 py-6 text-center text-slate-600">Tidak ada riwayat untuk filter ini</li>
    {/each}
  </ul>
{/if}
