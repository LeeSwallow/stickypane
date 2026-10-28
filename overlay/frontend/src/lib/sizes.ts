// A note's width is a fraction of the board, as on the terminal
// (internal/widget/size.go): these are the steps, narrowest first. The
// board's layout (board/layout.ts) places them; the note header steps them.

export const FRACTIONS = ['1/4', '1/3', '1/2', '2/3', '3/4', '1'] as const
export type Fraction = (typeof FRACTIONS)[number]

export const isFraction = (s: string): s is Fraction => (FRACTIONS as readonly string[]).includes(s)

/** nextFraction is the step one narrower (-1) or wider (+1), stopping at the ends; anything else is the whole width. */
export function nextFraction(f: string, dir: -1 | 1): Fraction {
  const i = isFraction(f) ? FRACTIONS.indexOf(f) : FRACTIONS.length - 1
  return FRACTIONS[Math.max(0, Math.min(FRACTIONS.length - 1, i + dir))]
}
