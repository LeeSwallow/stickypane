// The theme picker's families (03 §3.12): the ten palettes as the cards
// show them — a pair of a dark and a light half, or a dark theme alone —
// and which name a card keeps when it is picked. Pure.

import { DARK_ONLY, PAIRS } from '../lib/theme/mode'
import type { Palette } from '../lib/theme/palette'

export interface Family {
  /** the card's id: the family's first name ("stickypane-dark") */
  id: string
  /** what the card is called ("Catppuccin") */
  label: string
  /** the halves' own names ("mocha", "latte") */
  halves: string[]
  dark?: Palette
  light?: Palette
  /** every theme name the card stands for */
  names: string[]
}

const LABELS: Record<string, string> = {
  stickypane: 'stickypane',
  catppuccin: 'Catppuccin',
  tokyonight: 'Tokyo Night',
  gruvbox: 'Gruvbox',
  nord: 'Nord',
  dracula: 'Dracula',
}

const head = (name: string) => name.split('-')[0]
const tail = (name: string) => name.split('-').slice(1).join('-') || name

/** familyLabel names a theme's family for a card. */
export const familyLabel = (name: string): string => LABELS[head(name)] ?? name

/**
 * families groups the bridge's themes.list into cards, in its order: a
 * pair once, a theme with no pair alone. A theme the app has no pair for
 * is a card of its own.
 */
export function families(list: Palette[]): Family[] {
  const by = new Map(list.map((p) => [p.name, p]))
  const seen = new Set<string>()
  const out: Family[] = []
  for (const p of list) {
    if (seen.has(p.name)) continue
    const pair = PAIRS.find(([d, l]) => d === p.name || l === p.name)
    if (pair) {
      const [d, l] = pair
      seen.add(d).add(l)
      out.push({ id: d, label: familyLabel(d), halves: [tail(d), tail(l)], dark: by.get(d), light: by.get(l), names: [d, l] })
    } else {
      seen.add(p.name)
      out.push({
        id: p.name,
        label: familyLabel(p.name),
        halves: [],
        dark: p.dark || DARK_ONLY.includes(p.name) ? p : undefined,
        light: p.dark || DARK_ONLY.includes(p.name) ? undefined : p,
        names: [p.name],
      })
    }
  }
  return out
}

/** chosenFamily is the card a theme name belongs to ("auto" and "" none). */
export const chosenFamily = (fams: Family[], theme: string): Family | undefined => fams.find((f) => f.names.includes(theme))

/** nameFor is the theme a card keeps when picked: its half for the screen's light or dark, else its one theme. */
export function nameFor(f: Family, dark: boolean): string {
  if (f.dark && f.light) return dark ? f.dark.name : f.light.name
  return (f.dark ?? f.light)!.name
}
