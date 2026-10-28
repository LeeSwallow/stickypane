// The palettes the home window draws with. It has no bridge (a bridge is a
// project's), so it has no palette from one; these two mirror stickypane-dark
// and stickypane-light in internal/core/theme/palettes.go.
import type { Palette } from './palette'

export const STICKYPANE_DARK: Palette = {
  name: 'stickypane-dark',
  dark: true,
  base: '#1F2933',
  text: '#E6E6E6',
  muted: '#8A94A0',
  accent: '#7FB3F5',
  select: '#2F3B4C',
  good: '#8FD694',
  warn: '#F5A962',
  bad: '#F28FB1',
  info: '#B79CF2',
  notes: ['#F2D45C', '#F28FB1', '#7FB3F5', '#8FD694', '#B79CF2', '#F5A962'],
}

export const STICKYPANE_LIGHT: Palette = {
  name: 'stickypane-light',
  dark: false,
  base: '#FFFFFF',
  text: '#1F2933',
  muted: '#6B7684',
  accent: '#2F6FD1',
  select: '#D9E6FA',
  good: '#2E8B57',
  warn: '#C76B1A',
  bad: '#C9407A',
  info: '#7A4FC9',
  notes: ['#B38F00', '#C9407A', '#2F6FD1', '#2E8B57', '#7A4FC9', '#C76B1A'],
}

/** fallbackPalette is the palette for a dark or light screen. */
export const fallbackPalette = (dark: boolean): Palette => (dark ? STICKYPANE_DARK : STICKYPANE_LIGHT)
