// The create service: the shapes a note can take (kinds.list) and
// note.create with a shape's create options. The app never writes a
// note's text: it sends the values of the fields the shape asked for, and
// the shape's own Go code (Kind.Compose) writes the file and checks every
// value (RA12). The helpers here only move values between the form and
// the call.

import { Call } from '@wailsio/runtime'
import { project } from '../lib/shell'

export type OptionType = 'text' | 'lines' | 'choice' | 'number' | 'bool'

/** One field of a shape's create form, as kinds.list gives it. */
export interface ShapeOption {
  name: string
  type: OptionType
  /** the English phrase; the key of its translation */
  label: string
  choices?: string[]
  /** a lines option's values are separated by "\n" */
  default: string
}

/** A shape of note, as kinds.list gives it. */
export interface Shape {
  name: string
  icon: string
  label: string
  blurb: string
  example: string
  data: unknown
  options: ShapeOption[]
}

export type OptionValue = string | number | boolean | string[]

export interface CreateRequest {
  tab: string
  kind: string
  title: string
  options: Record<string, OptionValue>
}

export interface CreateApi {
  kinds(): Promise<Shape[]>
  /** note.create; it answers the new note's name */
  create(r: CreateRequest): Promise<{ note: string }>
}

export const createService: CreateApi = {
  async kinds() {
    const list = (await Call.ByName(project('Kinds'))) as Shape[] | null
    return (list ?? []).map((k) => ({ ...k, options: k.options ?? [] }))
  },
  create: (r) => Call.ByName(project('Create'), r) as Promise<{ note: string }>,
}

/** A form's values: a bool option is a boolean, every other a string. */
export type FormValues = Record<string, string | boolean>

/** initialValues fills a shape's form with its defaults. */
export function initialValues(shape: Shape): FormValues {
  const out: FormValues = {}
  for (const o of shape.options) out[o.name] = o.type === 'bool' ? o.default === 'true' : o.default
  return out
}

/**
 * toOptions turns the form into note.create's options: a lines field is
 * its lines, a number field a number when it reads as one (else the text,
 * for the shape to refuse with its own words), a bool a boolean. Nothing
 * is checked here; the shape checks.
 */
export function toOptions(shape: Shape, values: FormValues): Record<string, OptionValue> {
  const out: Record<string, OptionValue> = {}
  for (const o of shape.options) {
    const v = values[o.name]
    if (v === undefined) continue
    if (o.type === 'bool') out[o.name] = v === true || v === 'true'
    else if (o.type === 'lines') out[o.name] = String(v).split('\n')
    else if (o.type === 'number') {
      const s = String(v).trim()
      out[o.name] = s !== '' && Number.isFinite(Number(s)) ? Number(s) : String(v)
    } else out[o.name] = String(v)
  }
  return out
}

/**
 * fieldOf is the option a bridge error is about, or null: note.create's
 * bad_request names it ("columns of a board: …", "option \"view\" is …",
 * "a chart has no option \"x\"").
 */
export function fieldOf(message: string, shape: Shape): string | null {
  for (const o of shape.options) {
    const n = o.name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
    if (new RegExp(`(^|[\\s:])${n} of an? \\w+:`).test(message) || message.includes(`option "${o.name}"`)) return o.name
  }
  return null
}

/** reason is a bridge error without its "bad option:" lead. */
export const reason = (message: string): string => message.replace(/^bad option:\s*/, '')
