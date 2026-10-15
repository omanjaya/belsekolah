<script lang="ts">
  import type { Snippet } from 'svelte'
  let { kiri, kanan }: { kiri?: Snippet; kanan?: Snippet } = $props()
</script>

<div class="mb-4 flex flex-wrap items-center gap-2">
  {#if kiri}<div class="flex flex-wrap items-center gap-2">{@render kiri()}</div>{/if}
  {#if kanan}<div class="flex flex-wrap items-center gap-2 md:ml-auto">{@render kanan()}</div>{/if}
</div>
