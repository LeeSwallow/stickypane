<script lang="ts">
  // A little board in one palette's own colours, whatever the window's
  // theme: graph paper, a note with a hand-drawn border and a title, two
  // crayon cards (03 §3.12). The theme cards, the style previews and the
  // appearance previews draw with it. Decorative: the control round it
  // says what it is.
  import { boxPaths, crayonPaths, linePaths, type Seg, type Style } from '../lib/doodle/rough'
  import type { Palette } from '../lib/theme/palette'
  import { cardColorsFor, deriveTokens, noteTitleColor, seedOf } from '../lib/theme/tokens'

  let {
    palette: p,
    style = 'doodle',
    width = 128,
    height = 80,
    seed = 'swatch',
  }: { palette: Palette; style?: Style; width?: number; height?: number; seed?: string } = $props()

  // small: a note and one card, no lines of text (the theme picker's halves)
  const small = $derived(width < 72)

  const uid = $props.id()
  const s = $derived(seedOf(seed))
  const tok = $derived(deriveTokens(p, p.dark))
  const at = (segs: Seg[], x: number, y: number) => segs.map((g) => ({ ...g, x, y }))
  const note = $derived.by(() => {
    const w = Math.round(width * (small ? 0.5 : 0.56))
    const h = height - (small ? 14 : 20)
    const color = p.notes[2]
    return {
      w,
      h,
      border: at(boxPaths(w, h, { stroke: color, strokeWidth: 1.5, roughness: style === 'neat' ? 0 : 0.9, style, seed: s }), small ? 5 : 8, small ? 7 : 10),
      title: at(linePaths(small ? 5 : 8, small ? 8 : 12, w * 0.62, small ? 8 : 12, { stroke: noteTitleColor(color, p, p.dark), strokeWidth: small ? 2.2 : style === 'neat' ? 3 : 3.4, roughness: style === 'neat' ? 0 : 0.6, style, seed: s + 3 }), small ? 5 : 8, small ? 7 : 10),
      lines: (small ? [] : [20, 27, 34]).map((y, i) =>
        at(linePaths(8, y, 8 + (w - 22) * [0.9, 0.7, 0.8][i], y, { stroke: tok['--ink-3'] ?? p.muted, strokeWidth: 1.4, roughness: 0, style: 'neat', seed: s + 7 + i }), 8, 10),
      ),
    }
  })
  const cards = $derived(
    (small ? [0] : [0, 3]).map((ni, i) => {
      const w = Math.round(width * (small ? 0.34 : 0.3))
      const h = small ? Math.round(height * 0.42) : Math.round((height - 26) / 2)
      const c = cardColorsFor(p.notes[ni], p, p.dark)
      return at(crayonPaths(w, h, { ...c, style, seed: s + 11 + i }), Math.round(width * (small ? 0.6 : 0.64)), (small ? 7 : 10) + i * (h + 6))
    }),
  )
  const grid = $derived({ minor: tok['--grid'], major: tok['--grid-major'], paper: tok['--paper'] })
</script>

<svg class="swatch" viewBox="0 0 {width} {height}" {width} {height} aria-hidden="true">
  <defs>
    <pattern id="g-{uid}" width="9" height="9" patternUnits="userSpaceOnUse">
      <path d="M9 0V9M0 9H9" stroke={grid.minor} stroke-width="0.6" fill="none" />
    </pattern>
  </defs>
  <rect width={width} height={height} fill={grid.paper} />
  <rect width={width} height={height} fill="url(#g-{uid})" />
  {#each [...note.border, ...note.title, ...note.lines.flat(), ...cards.flat()] as g, i (i)}
    <path
      d={g.d}
      transform="translate({g.x} {g.y})"
      stroke-width={g.strokeWidth}
      stroke-linecap="round"
      stroke-linejoin="round"
      style="fill:{g.fill};stroke:{g.stroke}"
    />
  {/each}
</svg>

<style>
  .swatch {
    display: block;
    width: 100%;
    height: auto;
  }
</style>
