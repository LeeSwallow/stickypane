import { describe, expect, it } from 'vitest'
import { THEMES } from '../demo/themes'
import { chosenFamily, families, nameFor } from './themes'

describe('theme cards', () => {
  const fams = families(THEMES)
  it('the ten palettes are six cards: four pairs and two dark themes alone', () => {
    expect(fams.map((f) => f.label)).toEqual(['stickypane', 'Catppuccin', 'Tokyo Night', 'Gruvbox', 'Nord', 'Dracula'])
    expect(fams.filter((f) => f.dark && f.light)).toHaveLength(4)
    expect(fams.find((f) => f.label === 'Nord')).toMatchObject({ halves: [], light: undefined })
    expect(fams.find((f) => f.label === 'Catppuccin')?.halves).toEqual(['mocha', 'latte'])
    expect(fams.flatMap((f) => f.names).sort()).toEqual(THEMES.map((t) => t.name).sort())
  })
  it('a card picked keeps the half the screen shows', () => {
    const c = fams.find((f) => f.label === 'Catppuccin')!
    expect(nameFor(c, true)).toBe('catppuccin-mocha')
    expect(nameFor(c, false)).toBe('catppuccin-latte')
    expect(nameFor(fams.find((f) => f.label === 'Dracula')!, false)).toBe('dracula')
  })
  it('either half of a pair marks its card chosen; auto marks none', () => {
    expect(chosenFamily(fams, 'tokyonight-day')?.label).toBe('Tokyo Night')
    expect(chosenFamily(fams, 'auto')).toBeUndefined()
  })
  it('a theme with no pair the app knows is its own card', () => {
    const odd = { ...THEMES[0], name: 'solarized-dark' }
    expect(families([odd]).map((f) => f.label)).toEqual(['solarized-dark'])
  })
})
