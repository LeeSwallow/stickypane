import { expect, test } from 'vitest'
import { THEMES } from '../lib/theme/palettes'
import { modeOf, paletteFor } from './theme'

const by = (n: string) => THEMES.find((t) => t.name === n)!

test('the bridge palette is used when the mode keeps the chosen theme', () => {
  const sent = { ...by('catppuccin-mocha'), accent: '#123456' } // whatever the bridge says wins
  expect(paletteFor(sent, 'catppuccin-mocha', 'system', true)).toEqual(sent)
  expect(paletteFor(sent, 'catppuccin-mocha', 'dark', false)).toEqual(sent)
})

test('a light system turns the theme to its pair', () => {
  const sent = by('catppuccin-mocha')
  expect(paletteFor(sent, 'catppuccin-mocha', 'system', false).name).toBe('catppuccin-latte')
  expect(paletteFor(by('stickypane-dark'), 'auto', 'system', false).name).toBe('stickypane-light')
  expect(paletteFor(by('nord'), 'nord', 'light', true).name).toBe('stickypane-light')
})

test('the overlay mode "" is the system', () => {
  expect(modeOf('')).toBe('system')
  expect(modeOf(undefined)).toBe('system')
  expect(modeOf('dark')).toBe('dark')
  expect(modeOf('light')).toBe('light')
  expect(modeOf('weird')).toBe('system')
})
