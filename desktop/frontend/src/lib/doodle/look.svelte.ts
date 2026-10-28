// The look every doodle part draws with. The app sets it once (style from
// settings, palette from the bridge, dark from the mode); parts read it.
import type { Palette } from '../theme/palette'
import type { Style } from './rough'

export const look = $state<{ style: Style; dark: boolean; palette: Palette | null }>({
  style: 'doodle',
  dark: false,
  palette: null,
})
