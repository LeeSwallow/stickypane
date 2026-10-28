// A note's width is a fraction of the board, as on the terminal
// (internal/widget/size.go): these are the steps, narrowest first. The
// board's layout (board/layout.ts) places them; the note header steps them.

export const FRACTIONS = ['1/4', '1/3', '1/2', '2/3', '3/4', '1'] as const
export type Fraction = (typeof FRACTIONS)[number]

export const isFraction = (s: string): s is Fraction => (FRACTIONS as readonly string[]).includes(s)

/** shareOf is a fraction as a number (1/3 is 0.333…); anything else is the whole width. */
export function shareOf(f: string): number {
  if (!isFraction(f)) return 1
  const [n, d] = f.includes('/') ? f.split('/').map(Number) : [1, 1]
  return n / d
}

const GLYPHS: Record<Fraction, string> = { '1/4': '¼', '1/3': '⅓', '1/2': '½', '2/3': '⅔', '3/4': '¾', '1': '1' }

/** fractionGlyph is how the header writes a fraction: one vulgar-fraction character (½), or 1. */
export const fractionGlyph = (f: string): string => (isFraction(f) ? GLYPHS[f] : '1')

/** nextFraction is the step one narrower (-1) or wider (+1), stopping at the ends; anything else is the whole width. */
export function nextFraction(f: string, dir: -1 | 1): Fraction {
  const i = isFraction(f) ? FRACTIONS.indexOf(f) : FRACTIONS.length - 1
  return FRACTIONS[Math.max(0, Math.min(FRACTIONS.length - 1, i + dir))]
}
