// The home's cards in a grid (08 §8.3): two or three to a row as the window
// allows, every card in a row as wide as the others, and a last row that
// is short of cards shares the whole width rather than leaving a hole on
// the right. Pure.

/** UNITS is the grid under the cards: six, so that rows of 1, 2 and 3 divide it. */
export const UNITS = 6

/** columnsFor is how many cards go in a row for a list width px wide. */
export const columnsFor = (width: number): number => (width >= 760 ? 3 : width >= 440 ? 2 : 1)

/** spans is how many of the UNITS each of n cards takes, in a grid of cols to a row. */
export function spans(n: number, cols: number): number[] {
  const full = UNITS / cols
  const rest = n % cols
  return Array.from({ length: n }, (_, i) => (rest && i >= n - rest ? UNITS / rest : full))
}
