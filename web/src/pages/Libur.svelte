<script lang="ts">
  import { onMount } from 'svelte'
  import { Pencil, Trash2 } from '@lucide/svelte'
  import { api, type Libur, type Status } from '../lib/api'
  import { formatRentang } from '../lib/format'
  import { beriTahu, galat } from '../lib/notif.svelte'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import Kartu from '../components/Kartu.svelte'
  import Kosong from '../components/Kosong.svelte'
  import Memuat from '../components/Memuat.svelte'
  import Modal from '../components/Modal.svelte'
  import Medan from '../components/Medan.svelte'
  import Tombol from '../components/Tombol.svelte'

  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const kelasIkon = 'inline-grid size-11 place-items-center rounded-lg text-slate-700 hover:bg-slate-100'

  let memuat = $state(true)
  let daftar = $state<Libur[]>([])
  let hariIni = $state('')
  let galatMuat = $state('')

  async function muat() {
    galatMuat = ''
    try {
      const [l, st] = await Promise.all([api.get<Libur[]>('/libur'), api.get<Status>('/status')])
      daftar = l
      hariIni = st.tanggal
    } catch (e) {
      galatMuat = e instanceof Error ? e.message : String(e)
    } finally {
      memuat = false
    }
  }
  onMount(muat)

  const akanDatang = $derived(daftar.filter((l) => l.selesai >= hariIni).sort((a, b) => a.mulai.localeCompare(b.mulai)))
  const sudahLewat = $derived(daftar.filter((l) => l.selesai < hariIni).sort((a, b) => b.mulai.localeCompare(a.mulai)))

  let buka = $state(false)
  let editId = $state(0)
  let satuHari = $state(true)
  let fMulai = $state('')
  let fSelesai = $state('')
  let fKet = $state('')
  let galatForm = $state('')
  let sibuk = $state(false)

  function bukaForm(l?: Libur) {
    editId = l?.id ?? 0
    fMulai = l?.mulai ?? hariIni
    fSelesai = l?.selesai ?? hariIni
    satuHari = l ? l.mulai === l.selesai : true
    fKet = l?.keterangan ?? ''
    galatForm = ''
    buka = true
  }

  async function simpan(e: SubmitEvent) {
    e.preventDefault()
    galatForm = ''
    sibuk = true
    const data = { mulai: fMulai, selesai: satuHari ? fMulai : fSelesai, keterangan: fKet }
    try {
      if (editId) await api.put(`/libur/${editId}`, data)
      else await api.post('/libur', data)
      buka = false
      beriTahu('Libur disimpan')
      await muat()
    } catch (err) {
      galatForm = err instanceof Error ? err.message : String(err)
    } finally {
      sibuk = false
    }
  }

  async function hapus(l: Libur) {
    if (!(await konfirmasi('Hapus libur?', `${l.keterangan || 'Libur'} (${formatRentang(l.mulai, l.selesai)}) akan dihapus.`))) return
    try {
      await api.hapus(`/libur/${l.id}`)
      beriTahu('Libur dihapus')
      await muat()
    } catch (err) {
      galat(err)
    }
  }
</script>

{#snippet baris(l: Libur)}
  <li class="flex items-center gap-2 py-2">
    <div class="min-w-0 flex-1">
      <p class="font-medium">{formatRentang(l.mulai, l.selesai)}</p>
      {#if l.keterangan}<p class="break-words text-sm text-slate-600">{l.keterangan}</p>{/if}
    </div>
    <button type="button" class={kelasIkon} aria-label="Ubah {l.keterangan || formatRentang(l.mulai, l.selesai)}" onclick={() => bukaForm(l)}>
      <Pencil class="size-5" aria-hidden="true" />
    </button>
    <button type="button" class="{kelasIkon} text-red-700" aria-label="Hapus {l.keterangan || formatRentang(l.mulai, l.selesai)}" onclick={() => hapus(l)}>
      <Trash2 class="size-5" aria-hidden="true" />
    </button>
  </li>
{/snippet}

<JudulHalaman judul="Libur" keterangan="Pada tanggal libur tidak ada bel terjadwal.">
  {#snippet aksi()}
    <Tombol onclick={() => bukaForm()}>Tambah libur</Tombol>
  {/snippet}
</JudulHalaman>

{#if memuat}
  <Memuat />
{:else if galatMuat}
  <Kartu>
    <p class="mb-3 text-red-700" role="alert">{galatMuat}</p>
    <Tombol onclick={muat}>Coba lagi</Tombol>
  </Kartu>
{:else}
  <div class="flex flex-col gap-5">
    <Kartu>
      <h2 class="mb-3 text-lg font-semibold">Akan datang</h2>
      {#if akanDatang.length === 0}
        <Kosong pesan="Belum ada libur yang akan datang" />
      {:else}
        <ul class="divide-y divide-slate-200">
          {#each akanDatang as l (l.id)}{@render baris(l)}{/each}
        </ul>
      {/if}
    </Kartu>

    {#if sudahLewat.length > 0}
      <Kartu>
        <details>
          <summary class="flex min-h-11 cursor-pointer items-center text-lg font-semibold">Sudah lewat ({sudahLewat.length})</summary>
          <ul class="divide-y divide-slate-200">
            {#each sudahLewat as l (l.id)}{@render baris(l)}{/each}
          </ul>
        </details>
      </Kartu>
    {/if}
  </div>
{/if}

<Modal bind:buka judul={editId ? 'Ubah libur' : 'Tambah libur'}>
  <form id="form-libur" class="flex flex-col gap-3" onsubmit={simpan}>
    <label class="flex min-h-11 items-center gap-3 text-base">
      <input type="checkbox" class="size-5 accent-blue-700" bind:checked={satuHari} onchange={() => { if (!satuHari && fSelesai < fMulai) fSelesai = fMulai }} />
      Hanya satu hari
    </label>
    <Medan label="Tanggal mulai" untuk="lib-mulai">
      <input id="lib-mulai" type="date" class={kelasInput} bind:value={fMulai} required />
    </Medan>
    {#if !satuHari}
      <Medan label="Tanggal selesai" untuk="lib-selesai">
        <input id="lib-selesai" type="date" class={kelasInput} bind:value={fSelesai} required min={fMulai} />
      </Medan>
    {/if}
    <Medan label="Keterangan" untuk="lib-ket">
      <input id="lib-ket" class={kelasInput} bind:value={fKet} maxlength="100" />
    </Medan>
    {#if galatForm}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galatForm}</p>
    {/if}
  </form>
  {#snippet aksi()}
    <Tombol varian="hantu" onclick={() => (buka = false)}>Batal</Tombol>
    <Tombol type="submit" form="form-libur" {sibuk}>Simpan</Tombol>
  {/snippet}
</Modal>
