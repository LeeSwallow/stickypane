<script lang="ts">
  // A crayon icon: drawn like the app icon (wash + -41° hatching + heavy outline)
  // in the note colours of the theme. Flat and straight in the neat style.
  import { look } from '../../doodle/look.svelte'
  import { CRAYON_OUTLINE, crayonDrawing, type CrayonName } from './crayon'

  let { name, size = 20, label }: { name: CrayonName; size?: number; label?: string } = $props()
  const uid = $props.id()
  const fallback = ['#F2D45C', '#F28FB1', '#7FB3F5', '#8FD694', '#B79CF2', '#F5A962']
  const outline = $derived(look.dark ? '#0b0d10' : (look.palette?.text ?? '#1F2933'))
  const art = $derived(
    crayonDrawing(name, look.style, { notes: look.palette?.notes ?? fallback, outline, dark: look.dark }),
  )
</script>

<svg
  class="crayon"
  width={size}
  height={size}
  viewBox="0 0 20 20"
  role={label ? 'img' : undefined}
  aria-label={label}
  aria-hidden={label ? undefined : 'true'}
>
  {#if look.dark}
    <!-- a light rim, as the app icon has, so a dark outline holds on dark paper -->
    {#each art.outline as d, i (i)}
      <path {d} fill="none" stroke="rgb(255 255 255 / .22)" stroke-width={CRAYON_OUTLINE + 1.6} stroke-linecap="round" stroke-linejoin="round" />
    {/each}
  {/if}
  {#each art.shapes as s, i (i)}
    <clipPath id="{uid}-{i}"><path d={s.d} /></clipPath>
    <path d={s.d} fill={s.wash} />
    {#if s.hatch.length}
      <g clip-path="url(#{uid}-{i})" stroke={s.hatchColor} stroke-width="1.15" stroke-linecap="round" fill="none">
        {#each s.hatch as h, j (j)}<path d={h} />{/each}
      </g>
    {/if}
  {/each}
  <g fill="none" stroke={outline} stroke-width={CRAYON_OUTLINE} stroke-linecap="round" stroke-linejoin="round">
    {#each art.outline as d, i (i)}<path {d} />{/each}
  </g>
</svg>

<style>
  .crayon {
    display: block;
    flex: none;
    overflow: visible;
  }
</style>
