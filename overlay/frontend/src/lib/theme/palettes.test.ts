import { expect, test } from 'vitest'
import golden from './palettes.golden.json'
import { THEMES } from './palettes'

// palettes.golden.json is made from internal/theme by overlay/scripts/palettes
// (its Go test fails when it is out of date). Task A11 replaces the mirror
// with a bridge `themes.list`.
test('the palette mirror equals internal/theme', () => {
  expect(THEMES).toEqual(golden)
})
