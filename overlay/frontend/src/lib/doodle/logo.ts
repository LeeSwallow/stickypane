import { crayonLikeRect, type Seg, type Style } from './rough'

/**
 * logoPaths draws the icon: three memo sheets (blue, pink, yellow, bottom to
 * top) with crayon hatching and a thick outline. It is drawn rather than the
 * PNG so that it follows the theme and the style.
 */
export function logoPaths(notes: string[], style: Style, dark: boolean, text: string): Seg[] {
  if (notes.length < 3) return []
  const sheets: [number, number, string][] = [
    [2, 9, notes[2]],
    [7, 5, notes[1]],
    [12, 1, notes[0]],
  ]
  return sheets.flatMap(([x, y, c], i) =>
    crayonLikeRect(x, y, 18, 18, { color: c, outline: dark ? '#0b0d10' : text, style, seed: 60 + i }),
  )
}

/**
 * logoSheet is one sheet of the icon (0 blue, 1 pink, 2 yellow) alone, at
 * 0,0 in an 18 × 18 box. boil changes its seed, so the outline is drawn
 * again a little differently each time: the launch intro's wriggle.
 */
export function logoSheet(i: number, notes: string[], style: Style, dark: boolean, text: string, boil = 0): Seg[] {
  const color = [notes[2], notes[1], notes[0]][i]
  if (!color) return []
  return crayonLikeRect(1, 1, 16, 16, { color, outline: dark ? '#0b0d10' : text, style, seed: 60 + i + boil * 7 })
}
