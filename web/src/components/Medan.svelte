<script lang="ts">
  // Input di dalam Medan sebaiknya memakai aria-describedby="{untuk}-galat" dan aria-invalid saat ada galat.
  import type { Snippet } from 'svelte'
  let { label, untuk, galat = '', bantuan = '', children }: {
    label: string
    untuk: string
    galat?: string
    bantuan?: string
    children: Snippet
  } = $props()
</script>

<div class="flex flex-col gap-1.5">
  <label for={untuk} class="text-sm font-medium text-slate-800">{label}</label>
  {@render children()}
  {#if galat}
    <p id="{untuk}-galat" class="text-sm text-red-700" role="alert">{galat}</p>
  {:else if bantuan}
    <p class="text-sm text-slate-500">{bantuan}</p>
  {/if}
</div>
