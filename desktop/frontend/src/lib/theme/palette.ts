// Palette is the bridge's `palette`: the colours of the chosen colour theme,
// by role (internal/core/theme.Theme). The screen has no colours of its own.
export interface Palette {
  name: string
  dark: boolean
  base: string
  text: string
  muted: string
  accent: string
  select: string
  good: string
  warn: string
  bad: string
  info: string
  /** yellow, pink, blue, green, purple, orange, in that order */
  notes: string[]
}

export const NOTE_NAMES = ['yellow', 'pink', 'blue', 'green', 'purple', 'orange'] as const

const FIELDS = ['name', 'base', 'text', 'muted', 'accent', 'select', 'good', 'warn', 'bad', 'info'] as const

/**
 * normalizePalette reads a palette as the bridge sends it. The Go struct
 * may arrive with exported (Capitalised) or lower-cased keys; both work.
 */
export function normalizePalette(raw: unknown): Palette {
  const r = raw as Record<string, unknown>
  const get = (k: string) => (r[k] ?? r[k[0].toUpperCase() + k.slice(1)]) as unknown
  const out: Record<string, unknown> = {}
  for (const k of FIELDS) {
    const v = get(k)
    if (typeof v !== 'string') throw new Error(`palette: ${k} is missing`)
    out[k] = v
  }
  const notes = get('notes')
  if (!Array.isArray(notes) || notes.length !== 6) throw new Error('palette: notes needs six colours')
  out.dark = Boolean(get('dark'))
  out.notes = [...notes]
  return out as unknown as Palette
}
