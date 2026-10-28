// Dev pages and tests only: every theme's palette, as the bridge's
// themes.list gives them. The app itself asks the bridge (services/
// settings.ts) and keeps no copy; this fixture is made from internal/theme
// by overlay/scripts/palettes, whose Go test fails when it is out of date.
import type { Palette } from '../lib/theme/palette'
import themes from './themes.json'

export const THEMES: Palette[] = themes
