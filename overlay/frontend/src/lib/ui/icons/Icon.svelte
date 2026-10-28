<script lang="ts">
  // One icon of the set, in the current style, in currentColor. Decorative
  // unless it has a label (a control names itself; see IconButton).
  import { look } from '../../doodle/look.svelte'
  import { ICON_SIZE, ICON_STROKE, iconPaths, type IconName } from './icons'

  let { name, size = ICON_SIZE, label }: { name: IconName; size?: number; label?: string } = $props()
  const p = $derived(iconPaths(name, look.style))
</script>

<svg
  class="icon"
  width={size}
  height={size}
  viewBox="0 0 {ICON_SIZE} {ICON_SIZE}"
  fill="none"
  stroke="currentColor"
  stroke-width={ICON_STROKE}
  stroke-linecap="round"
  stroke-linejoin="round"
  role={label ? 'img' : undefined}
  aria-label={label}
  aria-hidden={label ? undefined : 'true'}
>
  {#each p.d as d, i (i)}<path {d} />{/each}
  {#each p.fill as d, i (i)}<path {d} fill="currentColor" stroke="none" />{/each}
</svg>

<style>
  .icon {
    display: block;
    flex: none;
  }
</style>
