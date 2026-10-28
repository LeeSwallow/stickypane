// Mode: system | dark | light. The mode switches the colour theme to its
// pair (03 §3.4); the chosen theme name itself is never rewritten.
export type Mode = 'system' | 'dark' | 'light'

const PAIRS: [dark: string, light: string][] = [
  ['stickypane-dark', 'stickypane-light'],
  ['catppuccin-mocha', 'catppuccin-latte'],
  ['tokyonight-night', 'tokyonight-day'],
  ['gruvbox-dark', 'gruvbox-light'],
]
// no pair: light mode uses stickypane-light
const DARK_ONLY = ['nord', 'dracula']

/** resolveDark says whether the screen is dark for a mode. */
export const resolveDark = (mode: Mode, systemDark: boolean): boolean =>
  mode === 'system' ? systemDark : mode === 'dark'

/** resolveThemeName is the theme to draw with for the chosen one and a mode. */
export function resolveThemeName(theme: string, mode: Mode, systemDark: boolean): string {
  const dark = resolveDark(mode, systemDark)
  if (theme === 'auto' || theme === '') return dark ? 'stickypane-dark' : 'stickypane-light'
  for (const [d, l] of PAIRS) {
    if (theme === d || theme === l) return dark ? d : l
  }
  if (DARK_ONLY.includes(theme)) return dark ? theme : 'stickypane-light'
  return theme
}

/** watchSystemDark calls fn with the OS's current and later dark setting. */
export function watchSystemDark(fn: (dark: boolean) => void): () => void {
  const q = window.matchMedia('(prefers-color-scheme: dark)')
  const on = () => fn(q.matches)
  on()
  q.addEventListener('change', on)
  return () => q.removeEventListener('change', on)
}
