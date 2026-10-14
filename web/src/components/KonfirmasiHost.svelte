<script lang="ts">
  import Modal from './Modal.svelte'
  import Tombol from './Tombol.svelte'
  import { permintaanKonfirmasi as p } from '../lib/konfirmasi.svelte'

  function jawab(ya: boolean) {
    const f = p.jawab
    p.jawab = null
    p.buka = false
    f?.(ya)
  }

  // Ditutup lewat Escape/tombol X = batal.
  $effect(() => {
    if (!p.buka && p.jawab) jawab(false)
  })
</script>

<Modal bind:buka={p.buka} judul={p.judul}>
  <p class="text-slate-700">{p.pesan}</p>
  {#snippet aksi()}
    <Tombol varian="kedua" onclick={() => jawab(false)}>Batal</Tombol>
    <Tombol varian="bahaya" onclick={() => jawab(true)}>{p.label}</Tombol>
  {/snippet}
</Modal>
