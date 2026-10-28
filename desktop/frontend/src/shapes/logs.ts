// The logs other notes write to, as the terminal reads them: a script's
// output and a .http note's responses go to a .log of the same name
// (internal/adapter/runner LogOf, internal/adapter/httpfile LogName), which the board
// loads like any note. These only read lines; nothing here writes one.

/** logOf is the log a script or a .http note writes: the same name with ".log". */
export function logOf(name: string): string {
  const slash = name.lastIndexOf('/')
  const dot = name.lastIndexOf('.')
  return (dot > slash ? name.slice(0, dot) : name) + '.log'
}

/** Tone is how a line of a log reads: a command, a check that held or failed, the last line. */
export type Tone = 'plain' | 'command' | 'good' | 'bad' | 'end-good' | 'end-bad'

/**
 * toneOf is how the terminal colours a line of a log: "$ …" is the
 * command, ✔ and ✘ are checks, and "[…]" is the line that ends a run or a
 * response, bad when it says so (✘, an exit code other than 0, an error
 * or a status of 400 and up).
 */
export function toneOf(line: string): Tone {
  const t = line.trim()
  if (t.startsWith('$ ')) return 'command'
  if (t.startsWith('✔')) return 'good'
  if (t.startsWith('✘')) return 'bad'
  if (/^\[.*\]$/.test(t)) return endFailed(t) ? 'end-bad' : 'end-good'
  return 'plain'
}

function endFailed(t: string): boolean {
  if (t.includes('✘')) return true
  const exit = t.match(/^\[exit (-?\d+)/)
  if (exit) return exit[1] !== '0'
  if (/^\[error\b/.test(t)) return true
  const status = t.match(/^\[(\d{3})\b/)
  return !!status && Number(status[1]) >= 400
}

/**
 * section is one request's part of a .http note's log: the lines under
 * "### title" up to the next "### ", without the blank lines around them;
 * null when the request has not been sent yet.
 */
export function section(lines: string[], title: string): string[] | null {
  const start = lines.findIndex((l) => l.trim() === '### ' + title)
  if (start < 0) return null
  let end = lines.findIndex((l, i) => i > start && l.startsWith('### '))
  if (end < 0) end = lines.length
  const out = lines.slice(start + 1, end)
  while (out.length && !out[0].trim()) out.shift()
  while (out.length && !out[out.length - 1].trim()) out.pop()
  return out
}

/** endLine is the last "[…]" line: how the run or the response ended, or "". */
export function endLine(lines: string[]): string {
  for (let i = lines.length - 1; i >= 0; i--) if (toneOf(lines[i]).startsWith('end')) return lines[i].trim()
  return ''
}

/** tail is the last n lines. */
export const tail = (lines: string[], n: number): string[] => (lines.length > n ? lines.slice(lines.length - n) : lines)
