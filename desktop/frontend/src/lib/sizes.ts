// A note's width is a fraction of the board, as on the terminal
// (internal/core/widget/size.go): these are the steps, narrowest first. The
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

/** nextFraction is the step one narrower (-1) or wider (+1), stopping at the ends; anything else is the whole width. */
export function nextFraction(f: string, dir: -1 | 1): Fraction {
  const i = isFraction(f) ? FRACTIONS.indexOf(f) : FRACTIONS.length - 1
  return FRACTIONS[Math.max(0, Math.min(FRACTIONS.length - 1, i + dir))]
}

// A board can be set to a fixed number of columns (the `columns` setting,
// 1 to 4): then a note without a size of its own takes one, and a note
// with one spans its share of them. The rule is internal/core/widget's
// ColumnSpan; both are tested against internal/core/widget/testdata/columns.json.

/** MAX_COLUMNS is the most columns a board can be set to. */
export const MAX_COLUMNS = 4

/** columnSpan is how many of n columns a size takes: round(fraction × n), half up, between 1 and n; a card or anything else is one. */
export function columnSpan(size: string, n: number): number {
  if (n <= 1) return 1
  const s = size.trim().toLowerCase()
  const f: string = s === 'half' ? '1/2' : s === 'page' ? '1' : s
  if (!isFraction(f)) return 1
  const [num, den] = f.includes('/') ? f.split('/').map(Number) : [1, 1]
  return Math.max(1, Math.min(n, Math.floor((2 * num * n + den) / (2 * den))))
}

/** columnSize is the fraction k of n columns is written as. */
export function columnSize(k: number, n: number): Fraction {
  n = Math.max(1, n)
  k = Math.max(1, Math.min(n, k))
  if (k === n) return '1'
  let a = k
  let b = n
  while (b) [a, b] = [b, a % b]
  return `${k / a}/${n / a}` as Fraction
}

/** columnSteps are the fractions a note can take on n columns, narrowest first. */
export const columnSteps = (n: number): Fraction[] => Array.from({ length: Math.max(1, n) }, (_, i) => columnSize(i + 1, n))
