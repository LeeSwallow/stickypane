<script lang="ts">
  import { look } from './look.svelte'
  import { ellipsePaths } from './rough'
  import Strokes from './Strokes.svelte'

  let { selected = false, seed = 1, size = 18 }: { selected?: boolean; seed?: number; size?: number } = $props()
  const c = $derived(size / 2)
  const segs = $derived([
    ...ellipsePaths(c, c, size - 3, size - 3, {
      stroke: selected ? 'var(--accent)' : 'var(--pencil)',
      strokeWidth: 1.4,
      seed,
      style: look.style,
    }),
    ...(selected
      ? ellipsePaths(c, c, size * 0.45, size * 0.45, { stroke: 'none', fill: 'var(--accent)', seed: seed + 1, style: look.style, roughness: 1 })
      : []),
  ])
</script>

<span class="radio" style:width="{size}px" style:height="{size}px"><Strokes {segs} viewBox="0 0 {size} {size}" /></span>

<style>
  .radio {
    position: relative;
    display: inline-block;
    flex: none;
  }
</style>
