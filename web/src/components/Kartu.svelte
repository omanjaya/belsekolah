<script lang="ts">
  import type { Snippet } from 'svelte'
  let { class: kelas = '', children }: { class?: string; children: Snippet } = $props()
</script>

<section class="rounded-xl border border-slate-200 bg-white p-4 sm:p-5 {kelas}">
  {@render children()}
</section>
