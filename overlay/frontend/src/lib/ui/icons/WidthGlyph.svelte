<script lang="ts">
  // A note's width as a picture: the frame of the size-* line icons, filled
  // to the share of the board the note takes (1/4 … 1). Decorative: the
  // control around it says the width in words.
  import { look } from '../../doodle/look.svelte'
  import { ICON_SIZE, ICON_STROKE, iconPaths } from './icons'

  let { share, size = 16 }: { share: number; size?: number } = $props()
  // the frame, bent like the rest of the family in the doodle style
  const frame = $derived(iconPaths('size-card', look.style).d)
  // the inside of the frame runs from 5.25 to 14.75 (as size-page fills it)
  const right = $derived(5.25 + 9.5 * Math.max(0, Math.min(1, share)))
</script>

<svg
  class="glyph"
  width={size}
  height={size}
  viewBox="0 0 {ICON_SIZE} {ICON_SIZE}"
  fill="none"
  stroke="currentColor"
  stroke-width={ICON_STROKE}
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
>
  {#each frame as d, i (i)}<path {d} />{/each}
  <path d="M5.25 6.75H{right}V13.25H5.25Z" fill="currentColor" stroke="none" />
</svg>

<style>
  .glyph {
    display: block;
    flex: none;
  }
</style>
