<script lang="ts">
  import { MediaQuery } from 'svelte/reactivity'
  import type { BarisGabung, HasilPeriksaGabung, IsiGabung, StatusGabung } from '../lib/api'
  import { NAMA_HARI } from '../lib/format'
  import { kunciSlot } from '../lib/tambahBanyak'
  import KartuStatistik from './KartuStatistik.svelte'
  import Tabel from './Tabel.svelte'

  let { hasil, timpa = $bindable() }: { hasil: HasilPeriksaGabung; timpa: Set<string> } = $props()

  const uid = $props.id()
  const kelasPilih =
    'min-h-11 rounded-xl border border-slate-300 bg-white px-3 py-2 text-base focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-600/20'
  const kelasLencana: Record<StatusGabung, string> = {
    baru: 'bg-emerald-50 text-emerald-700',
    kembar: 'bg-slate-100 text-slate-600',
    bentrok: 'bg-amber-50 text-amber-800',
  }
  const teksStatus: Record<StatusGabung, string> = { baru: 'Baru', kembar: 'Kembar', bentrok: 'Bentrok' }

  // Hanya satu tata letak yang dirender (tabel di laptop, kartu di HP) agar
  // kontrol tindakan tidak ganda di pohon aksesibilitas.
  const laptop = new MediaQuery('min-width: 768px')

  let tampilkanKembar = $state(false)

  const kunci = (b: BarisGabung) => kunciSlot(b.hari, b.jam)
  const bentrok = $derived(hasil.baris.filter((b) => b.status === 'bentrok'))
  const semuaTimpa = $derived(bentrok.length > 0 && bentrok.every((b) => timpa.has(kunci(b))))

  const tidakAdaTimpa = $derived(bentrok.every((b) => !timpa.has(kunci(b))))

  const tampil = $derived(hasil.baris.filter((b) => tampilkanKembar || b.status !== 'kembar'))
  const perHari = $derived(
    [...new Set(tampil.map((b) => b.hari))]
      .sort((a, b) => a - b)
      .map((h) => ({ hari: h, baris: tampil.filter((b) => b.hari === h) })),
  )

  const isi = (i: IsiGabung | null) => (i ? `${i.label || '—'} · ${i.jenisBel}` : '—')

  function aturSemua(ya: boolean) {
    const s = new Set(timpa)
    for (const b of bentrok) {
      if (ya) s.add(kunci(b))
      else s.delete(kunci(b))
    }
    timpa = s
  }
  function aturSatu(b: BarisGabung, ya: boolean) {
    const s = new Set(timpa)
    if (ya) s.add(kunci(b))
    else s.delete(kunci(b))
    timpa = s
  }
</script>

<div class="flex flex-col gap-4">
  <div class="grid grid-cols-3 gap-3">
    <KartuStatistik label="Baru" nilai={String(hasil.ringkasan.baru)} nada="baik" />
    <KartuStatistik label="Kembar" nilai={String(hasil.ringkasan.kembar)} />
    <KartuStatistik label="Bentrok" nilai={String(hasil.ringkasan.bentrok)} nada={hasil.ringkasan.bentrok ? 'peringatan' : 'biasa'} />
  </div>

  {#if bentrok.length > 0}
    <div class="flex flex-col gap-1">
      <p class="text-sm font-medium text-slate-700" id="{uid}-bentrok">Bel yang bentrok</p>
      <div role="radiogroup" aria-labelledby="{uid}-bentrok" class="flex flex-col gap-1">
        <label class="flex min-h-11 items-center gap-2">
          <input type="radio" class="size-5 accent-blue-700" name="{uid}-aksi" checked={bentrok.length > 0 && tidakAdaTimpa} onchange={() => aturSemua(false)} />
          Pertahankan yang lama
        </label>
        <label class="flex min-h-11 items-center gap-2">
          <input type="radio" class="size-5 accent-blue-700" name="{uid}-aksi" checked={semuaTimpa} onchange={() => aturSemua(true)} />
          Timpa dengan yang baru
        </label>
      </div>
    </div>
  {/if}

  <label class="flex min-h-11 items-center gap-2">
    <input type="checkbox" class="size-5 accent-blue-700" bind:checked={tampilkanKembar} />
    Tampilkan kembar
  </label>

  {#if perHari.length === 0}
    <p class="rounded-xl bg-slate-50 px-3 py-4 text-center text-slate-600">Tidak ada bel untuk ditampilkan.</p>
  {/if}

  {#each perHari as g (g.hari)}
    <section class="flex flex-col gap-2">
      <h3 class="font-semibold text-slate-900">{NAMA_HARI[g.hari]}</h3>

      {#if laptop.current}
        <div class="hidden md:block">
          <Tabel label="Pratinjau {NAMA_HARI[g.hari]}" jumlah={g.baris.length}>
            {#snippet kepala()}
              <th scope="col">Jam</th>
              <th scope="col">Status</th>
              <th scope="col">Lama</th>
              <th scope="col">Baru</th>
              <th scope="col">Tindakan</th>
            {/snippet}
            {#each g.baris as b (kunci(b))}
              <tr>
                <td class="font-bold tabular-nums">{b.jam}</td>
                <td>{@render lencana(b.status)}</td>
                <td>{isi(b.lama)}</td>
                <td>{isi(b.baru)}</td>
                <td>
                  {#if b.status === 'bentrok'}
                    {@render pilih(b)}
                  {:else}
                    —
                  {/if}
                </td>
              </tr>
            {/each}
          </Tabel>
        </div>
      {:else}
        <ul class="flex flex-col gap-2 md:hidden">
          {#each g.baris as b (kunci(b))}
            <li class="rounded-xl border border-slate-200 bg-white p-3">
              <div class="flex items-center justify-between gap-2">
                <span class="text-lg font-bold tabular-nums">{b.jam}</span>
                {@render lencana(b.status)}
              </div>
              <dl class="mt-2 grid grid-cols-[4.5rem_1fr] items-center gap-x-2 gap-y-1 text-sm">
                <dt class="text-slate-600">Lama</dt>
                <dd>{isi(b.lama)}</dd>
                <dt class="text-slate-600">Baru</dt>
                <dd>{isi(b.baru)}</dd>
                <dt class="text-slate-600">Tindakan</dt>
                <dd>
                  {#if b.status === 'bentrok'}
                    {@render pilih(b)}
                  {:else}
                    —
                  {/if}
                </dd>
              </dl>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {/each}
</div>

{#snippet lencana(s: StatusGabung)}
  <span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium {kelasLencana[s]}">{teksStatus[s]}</span>
{/snippet}

{#snippet pilih(b: BarisGabung)}
  <select
    class={kelasPilih}
    aria-label="Tindakan {NAMA_HARI[b.hari]} {b.jam}"
    value={timpa.has(kunci(b)) ? 'timpa' : 'pertahankan'}
    onchange={(e) => aturSatu(b, e.currentTarget.value === 'timpa')}
  >
    <option value="pertahankan">Pertahankan</option>
    <option value="timpa">Timpa</option>
  </select>
{/snippet}
