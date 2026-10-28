// Which colours the board window draws with (03 §3.4): the bridge's palette
// for the chosen theme, or, when the app's mode (overlay.json `mode`, the
// system's dark setting by default) asks for the other half of the pair,
// that theme's colours from the bridge's themes.list. Until the list is
// in (or from a bridge without it), the chosen theme's colours stay.

import { resolveThemeName, type Mode } from '../lib/theme/mode'
import type { Palette } from '../lib/theme/palette'

/** modeOf reads overlay.json's mode: "" (or anything else) is the system's. */
export const modeOf = (m: string | undefined): Mode => (m === 'dark' || m === 'light' ? m : 'system')

/** paletteFor is the palette to draw with for the bridge's, the chosen theme, the mode and themes.list. */
export function paletteFor(bridge: Palette, theme: string, mode: Mode, systemDark: boolean, themes: Palette[]): Palette {
  const want = resolveThemeName(theme, mode, systemDark)
  if (want === bridge.name) return bridge
  return themes.find((t) => t.name === want) ?? bridge
}
