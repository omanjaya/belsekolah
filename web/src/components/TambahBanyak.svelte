<script lang="ts">
  import { api, type FileJadwal, type HasilGabung, type HasilPeriksaGabung, type JenisBel, type Profil } from '../lib/api'
  import { konfirmasi } from '../lib/konfirmasi.svelte'
  import { beriTahu } from '../lib/notif.svelte'
  import { BARU_JENIS, kePemetaan } from '../lib/tambahBanyak'
  import PemetaanJenis from './PemetaanJenis.svelte'
  import PanelGeser from './PanelGeser.svelte'
  import PratinjauGabung from './PratinjauGabung.svelte'
  import SumberTambahBanyak from './SumberTambahBanyak.svelte'
  import Tombol from './Tombol.svelte'

  let { buka = $bindable(false), profil, jenisBel, onselesai }: {
    buka?: boolean
    profil: Profil
    jenisBel: JenisBel[]
    onselesai: () => void
  } = $props()

  type Langkah = 'sumber' | 'pemetaan' | 'pratinjau'

  let langkah = $state<Langkah>('sumber')
  let data = $state<FileJadwal | null>(null)
  let galatSumber = $state('')
  let pilihan = $state<Record<string, string>>({})
  let timpa = $state<Set<string>>(new Set())
  let hasil = $state<HasilPeriksaGabung | null>(null)
  let memeriksa = $state(false)
  let menggabung = $state(false)
  let galat = $state('')
  let urutan = 0

  // Atur ulang semuanya setiap panel dibuka.
  $effect(() => {
    if (!buka) return
    urutan++
    langkah = 'sumber'
    data = null
    galatSumber = ''
    pilihan = {}
    timpa = new Set()
    hasil = null
    memeriksa = false
    menggabung = false
    galat = ''
  })

  const pesan = (e: unknown) => (e instanceof Error ? e.message : String(e))
  const jumlahPerubahan = $derived(hasil ? hasil.ringkasan.baru + timpa.size : 0)

  async function periksa(dariPemetaan: boolean) {
    if (!data) return
    const saya = ++urutan
    galat = ''
    memeriksa = true
    // Dari langkah Sumber, pemetaan lama (sumber sebelumnya) tidak berlaku.
    const dasar = dariPemetaan ? pilihan : {}
    try {
      const r = await api.post<HasilPeriksaGabung>(`/profil/${profil.id}/gabung/periksa`, {
        data,
        pemetaan: kePemetaan(dasar),
      })
      if (saya !== urutan) return
      hasil = r
      const takDikenal = r.jenisBelTakDikenal ?? []
      const belumDipetakan = takDikenal.filter((n) => !(n in dasar))
      if (!dariPemetaan && belumDipetakan.length > 0) {
        const p: Record<string, string> = { ...dasar }
        for (const n of takDikenal) if (!(n in p)) p[n] = BARU_JENIS
        pilihan = p
        langkah = 'pemetaan'
      } else {
        if (!dariPemetaan) pilihan = {}
        timpa = new Set()
        langkah = 'pratinjau'
      }
    } catch (e) {
      if (saya !== urutan) return
      galat = pesan(e)
    } finally {
      if (saya === urutan) memeriksa = false
    }
  }

  function kembali() {
    // Buang respons periksa yang masih berjalan.
    urutan++
    memeriksa = false
    galat = ''
    if (langkah === 'pratinjau') langkah = hasil && hasil.jenisBelTakDikenal?.length ? 'pemetaan' : 'sumber'
    else langkah = 'sumber'
  }

  async function gabungkan() {
    if (!data || !hasil) return
    if (timpa.size > 0) {
      const ya = await konfirmasi('Timpa bel lama?', `${timpa.size} bel lama akan diganti dengan yang baru.`, 'Timpa')
      if (!ya) return
    }
    galat = ''
    menggabung = true
    try {
      const r = await api.post<HasilGabung>(`/profil/${profil.id}/gabung`, {
        data,
        pemetaan: kePemetaan(pilihan),
        timpa: [...timpa].map((k) => ({ hari: Number(k.split(' ')[0]), jam: k.split(' ')[1] })),
      })
      buka = false
      beriTahu(`${r.ditambah} ditambah, ${r.ditimpa} ditimpa, ${r.dilewati} dilewati`)
      onselesai()
    } catch (e) {
      galat = pesan(e)
    } finally {
      menggabung = false
    }
  }
</script>

<PanelGeser bind:buka lebar="lg" judul="Tambah banyak ke profil {profil.nama}">
  <div class="flex flex-col gap-4">
    <!-- Tetap ter-mount (hanya disembunyikan) agar isian tidak hilang saat Kembali. -->
    <div hidden={langkah !== 'sumber'}>
      <SumberTambahBanyak profilId={profil.id} {jenisBel} bind:data bind:galat={galatSumber} />
    </div>
    {#if langkah === 'pemetaan' && hasil}
      <PemetaanJenis nama={hasil.jenisBelTakDikenal} {jenisBel} bind:pilihan awalanId="tb-peta" />
    {:else if langkah === 'pratinjau' && hasil}
      <PratinjauGabung {hasil} bind:timpa />
    {/if}

    {#if memeriksa}
      <p class="text-sm text-slate-600" role="status">Memeriksa…</p>
    {/if}
    {#if galat}
      <p class="rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{galat}</p>
    {/if}
  </div>

  {#snippet aksi()}
    {#if langkah === 'sumber'}
      <Tombol varian="kedua" onclick={() => (buka = false)}>Batal</Tombol>
      <Tombol onclick={() => periksa(false)} sibuk={memeriksa} disabled={!data || !!galatSumber}>Periksa</Tombol>
    {:else if langkah === 'pemetaan'}
      <Tombol varian="kedua" onclick={kembali} disabled={memeriksa}>Kembali</Tombol>
      <Tombol onclick={() => periksa(true)} sibuk={memeriksa}>Lanjut</Tombol>
    {:else}
      <Tombol varian="kedua" onclick={kembali} disabled={memeriksa}>Kembali</Tombol>
      <Tombol onclick={gabungkan} sibuk={menggabung} disabled={jumlahPerubahan === 0}>Gabungkan ({jumlahPerubahan} perubahan)</Tombol>
    {/if}
  {/snippet}
</PanelGeser>
