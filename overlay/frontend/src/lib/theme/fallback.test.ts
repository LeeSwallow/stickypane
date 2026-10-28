import { expect, test } from 'vitest'
import { THEMES } from '../../demo/themes'
import { STICKYPANE_DARK, STICKYPANE_LIGHT } from './fallback'

// The home window has no bridge to ask; its two palettes are the default
// themes' own, as internal/core/theme has them.
test('the home window draws with the default themes as internal/core/theme has them', () => {
  expect(THEMES.find((t) => t.name === 'stickypane-dark')).toEqual(STICKYPANE_DARK)
  expect(THEMES.find((t) => t.name === 'stickypane-light')).toEqual(STICKYPANE_LIGHT)
})
