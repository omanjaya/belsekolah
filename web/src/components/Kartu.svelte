<script lang="ts">
  import type { Snippet } from 'svelte'
  let { class: kelas = '', children }: { class?: string; children: Snippet } = $props()
</script>

<section class="rounded-2xl bg-white p-4 shadow-sm ring-1 ring-slate-200 sm:p-5 {kelas}">
  {@render children()}
</section>
