// What a chart draws, as numbers: the same three views as the terminal's
// (internal/widget/chart/view.go) — bars scaled to the largest value, a
// trend line with its range and latest value, a calendar of days shaded
// by rank — so the app and the terminal say the same of the same file.
// Pure: Chart.svelte turns these into SVG.

import type { ChartPoint } from './data'

export interface Bar {
  label: string
  text: string
  /** 0…1 of the widest bar */
  ratio: number
}

/** bars is one bar per value, scaled to the largest; a value that is not a number, or not above 0, has no bar. */
export function bars(points: ChartPoint[]): Bar[] {
  const peak = Math.max(0, ...points.map((p) => (p.value !== null && Number.isFinite(p.value) ? p.value : 0)))
  return points.map((p) => ({
    label: p.label,
    text: p.text,
    ratio: peak > 0 && p.value !== null && p.value > 0 ? Math.min(p.value / peak, 1) : 0,
  }))
}

export interface Spark {
  /** points of the line in a w × h box, y down */
  line: [number, number][]
  low: number
  high: number
  first: string
  last: string
  /** the latest value, as written */
  latest: string
}

/** spark is the trend line through the values that are numbers, in a w × h box; null with fewer than two. */
export function spark(points: ChartPoint[], w: number, h: number): Spark | null {
  const ps = points.filter((p): p is ChartPoint & { value: number } => p.value !== null && Number.isFinite(p.value))
  if (ps.length < 2) return null
  const vs = ps.map((p) => p.value)
  const low = Math.min(...vs)
  const high = Math.max(...vs)
  const at = (v: number) => (high === low ? 0.5 : (v - low) / (high - low))
  const line = ps.map((p, i): [number, number] => [(w * i) / (ps.length - 1), h - h * at(p.value)])
  return { line, low, high, first: ps[0].label, last: ps[ps.length - 1].label, latest: ps[ps.length - 1].text }
}

/** A day of a calendar: its date, and its shade 0…3, or "zero" (a dot) or null (no value). */
export interface Day {
  date: string
  level: number | 'zero' | null
  text: string
}

export interface Calendar {
  /** a column per week, Monday first, seven days each */
  weeks: Day[][]
  /** the month (1…12) a week starts, or 0 when it is the same as the week before */
  months: number[]
  total: number
  peak: { text: string; date: string }
}

const DAY = 86_400_000
const parseDay = (s: string): number | null => {
  const m = s.trim().match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!m) return null
  const t = Date.UTC(+m[1], +m[2] - 1, +m[3])
  const d = new Date(t)
  return d.getUTCMonth() === +m[2] - 1 && d.getUTCDate() === +m[3] ? t : null
}
const dayText = (t: number) => new Date(t).toISOString().slice(0, 10)
const monday = (t: number) => t - ((new Date(t).getUTCDay() + 6) % 7) * DAY

/**
 * calendar is the values whose labels are dates as weeks of days, like a
 * contribution graph, the last `fit` weeks. A day's shade is its rank
 * among the days above zero, not its share of the peak, as on the
 * terminal; null when no label is a date.
 */
export function calendar(points: ChartPoint[], fit = 53): Calendar | null {
  const days = points.flatMap((p) => {
    const t = parseDay(p.label)
    return t === null ? [] : [{ t, v: p.value ?? 0, text: p.text }]
  })
  if (!days.length) return null
  const values = new Map(days.map((d) => [d.t, d]))
  const ranked = days.map((d) => d.v).filter((v) => v > 0).sort((a, b) => a - b)
  let peak = days[0]
  let total = 0
  for (const d of days) {
    if (d.v > peak.v) peak = d
    total += d.v
  }
  const end = monday(Math.max(...days.map((d) => d.t)))
  let start = monday(Math.min(...days.map((d) => d.t)))
  let count = Math.round((end - start) / (7 * DAY)) + 1
  if (count > fit) {
    start = end - (fit - 1) * 7 * DAY
    count = fit
  }
  const weeks: Day[][] = []
  const months: number[] = []
  let month = 0
  for (let w = 0; w < count; w++) {
    const first = start + w * 7 * DAY
    const m = new Date(first).getUTCMonth() + 1
    months.push(m === month ? 0 : m)
    month = m
    const week: Day[] = []
    for (let r = 0; r < 7; r++) {
      const t = first + r * DAY
      const d = values.get(t)
      let level: Day['level'] = null
      if (d && d.v <= 0) level = 'zero'
      else if (d) {
        const atOrBelow = ranked.filter((v) => v <= d.v).length
        level = Math.max(0, Math.min(3, Math.floor((atOrBelow * 4 + ranked.length - 1) / ranked.length) - 1))
      }
      week.push({ date: dayText(t), level, text: d?.text ?? '' })
    }
    weeks.push(week)
  }
  return { weeks, months, total: Math.round(total * 1e6) / 1e6, peak: { text: peak.text, date: dayText(peak.t) } }
}

/** grouped writes a sum the way the terminal does: no needless ".0", thousands grouped. */
export function grouped(v: number): string {
  v = Math.round(v * 100) / 100
  const [i, f] = String(Math.abs(v)).split('.')
  const g = i.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  return (v < 0 ? '-' : '') + g + (f ? '.' + f : '')
}
