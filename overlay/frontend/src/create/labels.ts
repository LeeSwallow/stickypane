// The words of the create form. A shape's option labels are English
// phrases from its Go code and the keys of their translations ("label." +
// the phrase), as the terminal's "labels" are; a phrase with no
// translation shows as it is.
import type { Shape } from '../services/create'

type Tr = (key: string, values?: Record<string, unknown>) => string

const lookup = (tr: Tr, key: string, fallback: string) => {
  const s = tr(key)
  return s === key ? fallback : s
}

export const optionLabel = (tr: Tr, label: string) => lookup(tr, 'label.' + label, label)
export const choiceLabel = (tr: Tr, value: string) => lookup(tr, 'choice.' + value, value)
export const shapeLabel = (tr: Tr, s: Shape) => lookup(tr, 'create.shape.' + s.name, s.label)
export const shapeBlurb = (tr: Tr, s: Shape) => lookup(tr, 'create.blurb.' + s.name, s.blurb)
