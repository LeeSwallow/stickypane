<script lang="ts">
  import type { Seg } from './rough'
  let { segs, viewBox }: { segs: Seg[]; viewBox: string } = $props()
</script>

<svg class="rough" {viewBox} aria-hidden="true">
  {#each segs as s, i (i)}
    <path
      d={s.d}
      stroke-width={s.strokeWidth}
      stroke-linecap="round"
      stroke-linejoin="round"
      opacity={s.opacity}
      style="fill:{s.fill};stroke:{s.stroke}"
    />
  {/each}
</svg>

<style>
  .rough {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    overflow: visible;
    pointer-events: none;
  }
</style>
