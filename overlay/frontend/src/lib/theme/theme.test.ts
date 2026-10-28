import { describe, expect, test } from 'vitest'
import { THEMES } from './fixtures'
import { agentColor, cardColors, contrast, deriveTokens, mix, noteColors, seedOf } from './tokens'
import { resolveDark, resolveThemeName } from './mode'
import { normalizePalette } from './palette'

const by = (n: string) => THEMES.find((t) => t.name === n)!

describe('palette derivations match 03 §3.4.1', () => {
  test('mix is a linear blend in sRGB bytes', () => {
    expect(mix('#000000', '#ffffff', 0.5)).toBe('#808080')
    expect(mix('#102030', '#102030', 0.7)).toBe('#102030')
  })
  test('dark theme', () => {
    const p = by('stickypane-dark')
    const t = deriveTokens(p, true)
    expect(t['--paper']).toBe(mix(p.base, '#000000', 0.06))
    expect(t['--grid']).toBe(mix(p.base, p.text, 0.07))
    expect(t['--grid-major']).toBe(mix(p.base, p.text, 0.13))
    expect(t['--dot']).toBe(mix(p.base, p.text, 0.12))
    expect(t['--base']).toBe(p.base)
    expect(t['--ink']).toBe(p.text)
    expect(t['--pencil']).toBe(p.muted)
    expect(t['--yellow']).toBe(p.notes[0])
    expect(t['--orange']).toBe(p.notes[5])
    expect(cardColors(p, 0, true).wash).toBe(mix(p.notes[0], '#ffffff', 0.35))
    expect(cardColors(p, 0, true).outline).toBe(mix(p.base, '#000000', 0.55))
    expect(cardColors(p, 0, true).hatch).toBe(p.notes[0])
  })
  test('light theme', () => {
    const p = by('catppuccin-latte')
    const t = deriveTokens(p, false)
    expect(t['--paper']).toBe(mix(p.base, '#ffffff', 0.5))
    expect(t['--grid']).toBe(mix(p.base, p.text, 0.09))
    expect(t['--grid-major']).toBe(mix(p.base, p.text, 0.17))
    expect(t['--dot']).toBe(mix(p.base, p.text, 0.16))
    expect(cardColors(p, 2, false).wash).toBe(mix(p.notes[2], '#ffffff', 0.78))
    expect(cardColors(p, 2, false).outline).toBe(p.text)
  })
  test('noteColors lists the six', () => {
    expect(noteColors(by('nord'))).toHaveLength(6)
  })
  test('normalizePalette accepts Go-cased and lower-cased fields', () => {
    const p = by('nord')
    const go = {
      Name: p.name, Dark: p.dark, Base: p.base, Text: p.text, Muted: p.muted, Accent: p.accent,
      Select: p.select, Good: p.good, Warn: p.warn, Bad: p.bad, Info: p.info, Notes: p.notes,
    }
    expect(normalizePalette(go)).toEqual(p)
    expect(normalizePalette(p)).toEqual(p)
  })
})

describe('mode pairs the colour theme', () => {
  const cases: [string, 'dark' | 'light', string][] = [
    ['stickypane-dark', 'light', 'stickypane-light'],
    ['stickypane-light', 'dark', 'stickypane-dark'],
    ['catppuccin-mocha', 'light', 'catppuccin-latte'],
    ['catppuccin-latte', 'dark', 'catppuccin-mocha'],
    ['tokyonight-night', 'light', 'tokyonight-day'],
    ['tokyonight-day', 'dark', 'tokyonight-night'],
    ['gruvbox-dark', 'light', 'gruvbox-light'],
    ['gruvbox-light', 'dark', 'gruvbox-dark'],
    ['nord', 'light', 'stickypane-light'],
    ['dracula', 'light', 'stickypane-light'],
    ['stickypane-dark', 'dark', 'stickypane-dark'],
    ['catppuccin-latte', 'light', 'catppuccin-latte'],
    ['nord', 'dark', 'nord'],
  ]
  test.each(cases)('%s in %s -> %s', (theme, mode, want) => {
    expect(resolveThemeName(theme, mode, false)).toBe(want)
  })
  test('system follows the OS', () => {
    expect(resolveThemeName('catppuccin-mocha', 'system', false)).toBe('catppuccin-latte')
    expect(resolveThemeName('catppuccin-mocha', 'system', true)).toBe('catppuccin-mocha')
    expect(resolveDark('system', true)).toBe(true)
    expect(resolveDark('light', true)).toBe(false)
  })
  test('auto theme leaves the choice to the mode', () => {
    expect(resolveThemeName('auto', 'dark', false)).toBe('stickypane-dark')
    expect(resolveThemeName('auto', 'light', true)).toBe('stickypane-light')
  })
})

describe('agentColor matches the Go hash for 10 names', () => {
  // computed by running internal/bridge's color() (fnv-1a 32 mod 6) in Go
  const golden: [string, number][] = [
    ['claude', 3], ['codex', 0], ['sam', 2], ['gpt-5', 4], ['reviewer', 4],
    ['한글', 1], ['api.md', 4], ['README.md', 4], ['todo', 1], ['build-bot', 3],
  ]
  const p = by('stickypane-dark')
  test.each(golden)('%s -> note %i', (name, i) => {
    expect(agentColor(name, p.notes)).toBe(p.notes[i])
  })
})

test('seedOf is stable and positive', () => {
  expect(seedOf('abc')).toBe(seedOf('abc'))
  expect(seedOf('abc')).not.toBe(seedOf('abd'))
  expect(seedOf('')).toBeGreaterThan(0)
})

describe('contrast is at least 4.5:1 for every theme x mode x card colour', () => {
  for (const p of THEMES) {
    for (const mode of ['dark', 'light'] as const) {
      const q = by(resolveThemeName(p.name, mode, false))
      test(`${p.name} in ${mode} (${q.name})`, () => {
        const t = deriveTokens(q, q.dark)
        expect(contrast(t['--ink'], t['--paper'])).toBeGreaterThanOrEqual(4.5)
        for (let i = 0; i < 6; i++) {
          const c = cardColors(q, i, q.dark)
          const worst = Math.min(contrast(c.ink, c.wash), contrast(c.ink, c.hatchSeen))
          expect(worst, `${q.name} note ${i}`).toBeGreaterThanOrEqual(4.5)
        }
      })
    }
  }
})
