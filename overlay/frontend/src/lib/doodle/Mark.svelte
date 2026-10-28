<script lang="ts">
  // A highlighter behind the lower half of its parent's text.
  import { look } from './look.svelte'
  import { markPaths } from './rough'
  import Strokes from './Strokes.svelte'

  let { color = 'var(--yellow)', seed = 4 }: { color?: string; seed?: number } = $props()
  let w = $state(0)
  let h = $state(0)
  const segs = $derived(w ? markPaths(w, h, { color, seed, style: look.style }) : [])
</script>

<span class="m" bind:clientWidth={w} bind:clientHeight={h}><Strokes {segs} viewBox="0 0 {w} {h}" /></span>

<style>
  .m {
    position: absolute;
    inset: 0 -3px;
    pointer-events: none;
    z-index: -1;
  }
</style>
