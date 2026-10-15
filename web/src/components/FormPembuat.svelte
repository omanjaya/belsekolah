<script lang="ts">
  import { api, type FileJadwal, type ParameterPembuat } from '../lib/api'
  import { nilaiAwal } from '../lib/pembuat'
  import Medan from './Medan.svelte'
  import Tombol from './Tombol.svelte'

  let { hasil = $bindable(), menghitung = $bindable(), galat = $bindable(), hariSekolah = $bindable() }: {
    hasil: FileJadwal | null
    menghitung: boolean
    galat: string
    hariSekolah: number[]
  } = $props()

  const kelasInput =
    'min-h-11 w-full rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
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
    // Naikkan urutan sekarang juga agar respons permintaan lama langsung dibuang.
    const saya = ++urutan
    menghitung = true
    const timer = setTimeout(async () => {
      try {
        const data = await api.post<FileJadwal>('/template/buat', JSON.parse(kirim))
        if (saya !== urutan) return
        hasil = data
        galat = ''
      } catch (e) {
        if (saya !== urutan) return
        galat = e instanceof Error ? e.message : String(e)
      } finally {
        if (saya === urutan) menghitung = false
      }
    }, 300)
    return () => clearTimeout(timer)
  })

  $effect(() => {
    hariSekolah = param.hariSekolah === 6 ? [1, 2, 3, 4, 5, 6] : [1, 2, 3, 4, 5]
  })
</script>

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
