// What the mod makes of the board's command line: the lines of
// `stickypane watch --json` and the notes of `stickypane index --json`.
// Nothing here touches the engine, so the tests can call it directly.

import type { Note } from '../types'

// One line of `stickypane watch --json`.
export type BoardEvent = {
  note: string
  kind?: string
  type: string
  item?: string
  from?: string
  to?: string
}

// Lines splits what the watch wrote so far into whole lines and the rest,
// which waits for the next piece.
export function lines(buffer: string): { whole: string[]; rest: string } {
  const parts = buffer.split('\n')
  const rest = parts.pop() ?? ''
  return { whole: parts.filter(line => line.trim() !== ''), rest }
}

// Parse reads one line of the watch, or nothing when it is not an event.
export function parse(line: string): BoardEvent | undefined {
  try {
    const value: unknown = JSON.parse(line)
    if (typeof value !== 'object' || value === null) return undefined
    const event = value as Partial<BoardEvent>
    if (typeof event.note !== 'string' || typeof event.type !== 'string') return undefined
    return event as BoardEvent
  } catch {
    return undefined
  }
}

// Describe is the line a person reads for what happened on the board, or
// nothing for a change not worth a toast (a note saved, a log line).
export function describe(event: BoardEvent, notes: readonly Note[]): string | undefined {
  const title = notes.find(note => note.name === event.note)?.title ?? event.note
  const item = event.item ?? ''
  switch (event.type) {
    case 'item.ticked':
      return `☑ ${title}: ${item}`
    case 'item.unticked':
      return `☐ ${title}: ${item}`
    case 'item.added':
    case 'card.added':
      return `+ ${title}: ${item}`
    case 'card.moved':
      return `${title}: ${item} → ${event.to ?? ''}`
    case 'form.submitted':
      return `? ${title}: answered`
    case 'message.added':
      return `${title}: ${event.from ? `${event.from}: ` : ''}${item}`
    default:
      return undefined
  }
}

// Tabs groups the notes by the tab they are in, in the board's order.
export function tabs(notes: readonly Note[]): { title: string; notes: Note[] }[] {
  const out: { title: string; notes: Note[] }[] = []
  for (const note of notes) {
    const last = out[out.length - 1]
    const title = note.tab_title || note.tab
    if (last && last.title === title) last.notes.push(note)
    else out.push({ title, notes: [note] })
  }
  return out
}

// AsMarkdown is a note's file as the pane draws it: Markdown as it is, any
// other file (a script, a log, a request) as a code block.
export function asMarkdown(name: string, text: string): string {
  if (name.endsWith('.md')) return text
  const fence = text.includes('```') ? '~~~~' : '```'
  const language = name.split('.').pop() ?? ''
  return `${fence}${language}\n${text}\n${fence}`
}
