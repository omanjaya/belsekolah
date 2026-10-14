<script lang="ts">
  import { onMount } from 'svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Tombol from '../components/Tombol.svelte'
  import Medan from '../components/Medan.svelte'
  import Memuat from '../components/Memuat.svelte'
  import Kosong from '../components/Kosong.svelte'
  import { api, type Riwayat, type Status } from '../lib/api'
  import { galat } from '../lib/notif.svelte'

  const input = 'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
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
  let memuat = $state(true)
  let siap = $state(false)
  let urut = 0

  async function muat() {
    const saya = ++urut
    memuat = true
    try {
      const q = new URLSearchParams({ batas: '200' })
      if (tanggal) q.set('tanggal', tanggal)
      if (status) q.set('status', status)
      const hasil = await api.get<Riwayat[]>('/riwayat?' + q)
      if (saya === urut) daftar = hasil
    } catch (e) {
      if (saya === urut) galat(e)
    } finally {
      if (saya === urut) memuat = false
    }
  }

  onMount(async () => {
    try {
      tanggal = (await api.get<Status>('/status')).tanggal
    } catch (e) {
      galat(e)
    }
    siap = true
  })

  $effect(() => {
    if (!siap) return
    void tanggal
    void status
    muat()
  })
</script>

<JudulHalaman judul="Riwayat" keterangan="Catatan bel yang sudah berbunyi, gagal, atau terlewat.">
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={muat} sibuk={memuat} aria-label="Muat ulang">
      <svg class="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 12a9 9 0 1 1-3-6.7L21 8" /><path d="M21 3v5h-5" /></svg>
    </Tombol>
  {/snippet}
</JudulHalaman>

<div class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
  <Medan label="Tanggal" untuk="rw-tanggal">
    <input id="rw-tanggal" type="date" bind:value={tanggal} class={input} />
  </Medan>
  <Medan label="Status" untuk="rw-status">
    <select id="rw-status" bind:value={status} class={input}>
      <option value="">Semua</option>
      <option value="berhasil">Berhasil</option>
      <option value="gagal">Gagal</option>
      <option value="terlewat">Terlewat</option>
      <option value="cadangan">Nada cadangan</option>
      <option value="dihentikan">Dihentikan</option>
    </select>
  </Medan>
</div>

{#if memuat && daftar.length === 0}
  <Memuat />
{:else if daftar.length === 0}
  <Kosong pesan="Tidak ada riwayat untuk filter ini" />
{:else}
  <ul class="flex flex-col gap-2">
    {#each daftar as r (r.id)}
      <li class="rounded-2xl bg-white p-3 shadow-sm ring-1 ring-slate-200">
        <div class="flex items-start gap-3">
          <span class="font-mono text-base font-semibold tabular-nums text-slate-900">{r.waktu.substring(11, 19)}</span>
          <span class="min-w-0 flex-1 break-words text-slate-800">{r.label}</span>
        </div>
        <div class="mt-2 flex flex-wrap items-center gap-2 text-sm">
          <span class="rounded-full bg-blue-50 px-2.5 py-0.5 font-medium text-blue-800">{SUMBER[r.sumber]}</span>
          <span class="rounded-full px-2.5 py-0.5 font-medium {STATUS[r.status].kelas}">{STATUS[r.status].teks}</span>
        </div>
        {#if r.pesan}<p class="mt-1.5 break-words text-sm text-slate-500">{r.pesan}</p>{/if}
      </li>
    {/each}
  </ul>
{/if}
