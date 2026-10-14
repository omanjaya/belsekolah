<script lang="ts">
  import Tombol from './Tombol.svelte'

  let { pesan, oncoba }: { pesan: string; oncoba: () => void } = $props()
</script>

<div class="flex flex-col items-start gap-3" role="alert">
  <p class="rounded-xl bg-red-50 px-4 py-3 text-red-700">{pesan}</p>
  <Tombol varian="kedua" onclick={oncoba}>Coba lagi</Tombol>
</div>
