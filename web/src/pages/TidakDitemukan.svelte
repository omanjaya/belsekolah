<script lang="ts">
  import JudulHalaman from '../components/JudulHalaman.svelte'
  import { tautan } from '../lib/router.svelte'
</script>

<JudulHalaman judul="Halaman tidak ditemukan" />
<a href="/" onclick={tautan} class="font-medium text-blue-700 underline">Kembali ke Beranda</a>
