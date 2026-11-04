import { describe, expect, test } from 'vitest'
import { baseName, homeOf, shortPath, tildePath } from './path'

describe('a folder as it is written', () => {
  test('its name is its last part', () => {
    expect([baseName('/Users/me/proj'), baseName('/Users/me/proj/'), baseName('proj'), baseName(''), baseName('/')]).toEqual(['proj', 'proj', 'proj', '', '/'])
  })
  test('a long, deep path keeps its end', () => {
    expect(shortPath('/Users/someone/Projects/Personal/stickypane')).toBe('…/Personal/stickypane')
    expect(shortPath('/a/b')).toBe('/a/b')
    expect(shortPath('/Users/someone/Projects/Personal/stickypane', 48)).toBe('/Users/someone/Projects/Personal/stickypane')
    expect(shortPath('/a-very-long-folder-name-that-goes-on-and-on')).toBe('/a-very-long-folder-name-that-goes-on-and-on')
  })
  test('the home is ~, and is found from the app folder in it', () => {
    expect(tildePath('/Users/me/proj', '/Users/me')).toBe('~/proj')
    expect(tildePath('/Users/meo/proj', '/Users/me')).toBe('/Users/meo/proj')
    expect(tildePath('/Users/me/proj', '')).toBe('/Users/me/proj')
    expect(homeOf('/Users/me/.stickypane')).toBe('/Users/me')
    expect(homeOf('/elsewhere')).toBe('')
  })
})
