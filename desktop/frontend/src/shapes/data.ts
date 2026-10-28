// What board.load says of the shapes this folder draws beyond note,
// checklist and board (board/model.ts has those): each shape's Kind.Data
// as JSON (internal/core/widget/*/model.go). The app reads these and never the
// note's file; what it changes goes back as note.act, note.run or
// note.send.

/** A form's line (internal/core/widget/form Line). */
export interface FormLine {
  kind: 'text' | 'option' | 'detail' | 'field' | 'buttons'
  text: string
  /** the question an option or a field answers */
  group?: string
  /** a ( ) option: one of its question */
  radio?: boolean
  checked?: boolean
  labels?: string[]
}
export interface FormData {
  lines: FormLine[]
  /** the button pressed, "" while not submitted */
  submitted: string
  /** when, RFC 3339 */
  at: string
}

export interface ChartPoint {
  label: string
  /** the number as written */
  text: string
  /** null when it is not a finite number */
  value: number | null
}
export interface ChartData {
  view: 'bar' | 'spark' | 'heat'
  prose: string[]
  points: ChartPoint[]
}

export interface LogData {
  lines: string[]
}

export interface ChatMessage {
  day: string
  who: string
  clock: string
  text: string
}
export interface ChatData {
  messages: ChatMessage[]
}

export interface ScriptData {
  source: string
  lang: 'sh' | 'ps1'
}

/** One request of a .http note, as the terminal lists it. */
export interface Requested {
  /** what note.send takes */
  title: string
  method: string
  where: string
  asserts: number
  hooks: boolean
  /** the request as written under its request line */
  block: string[]
}
export interface ApiData {
  source: string
  env?: string
  requests?: Requested[]
}

export interface HtmlData {
  html: string
}

/** dataOf is a note's data as T, or the empty value when it has none. */
export const dataOf = <T>(data: unknown, empty: T): T => (data && typeof data === 'object' ? (data as T) : empty)
