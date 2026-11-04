// Every word a person sees comes from the catalogs: a template's text, and
// the attributes a person or a screen reader reads (title, aria-label,
// placeholder, alt, label, hint), are $tr(...) or data, never a literal.
// So is a string an expression there comes to ({busy ? 'Saving' : x}).
// A literal that is not a word to translate (a product name, a key name, an
// example address) is listed in ALLOWED with why.
import { parse } from 'svelte/compiler'
import { describe, expect, it } from 'vitest'

// every component of the app, as text; the demo pages are the design
// system's catalogue, not the app
const sources = import.meta.glob(['/src/**/*.svelte', '!/src/demo/**'], { query: '?raw', import: 'default', eager: true }) as Record<string, string>

// Attributes whose text a person reads, on elements and on our components.
const READ = new Set([
  'title', 'aria-label', 'aria-description', 'aria-roledescription', 'aria-valuetext',
  'placeholder', 'alt', 'label', 'hint', 'summary', 'message', 'confirmLabel', 'cancelLabel',
])

// file → literals that stay as written, each with its reason.
const ALLOWED: Record<string, string[]> = {
  // an example of what to type, not a sentence
  'settings/Hosts.svelte': ['me@server', 'stickypane'],
  // the product's name
  'home/Welcome.svelte': ['stickypane'],
  // the product's name, and a type specimen in both scripts
  'settings/SettingsWindow.svelte': ['stickypane', 'Aa 가나'],
  // the shell a script runs in, by its program's name
  'shapes/Script.svelte': ['PowerShell', 'sh'],
}

/** A letter of any script: a literal with none (·, …, ⋯, ×, →) is a mark, not a word. */
const WORD = /\p{L}/u

type Found = { line: number; text: string; where: string }

type Js = { type: string; start: number; value?: unknown; [k: string]: unknown }

/** said is the strings an expression can come to as it is: its literals, through ?:, && and ||, and a template's text. */
function said(e: Js | undefined): Js[] {
  if (!e) return []
  switch (e.type) {
    case 'Literal':
      return typeof e.value === 'string' ? [e] : []
    case 'ConditionalExpression':
      return [...said(e.consequent as Js), ...said(e.alternate as Js)]
    case 'LogicalExpression':
      return [...said(e.left as Js), ...said(e.right as Js)]
    case 'TemplateLiteral':
      return [
        ...(e.quasis as Js[]).map((q) => ({ ...q, value: (q.value as { cooked: string }).cooked })),
        ...(e.expressions as Js[]).flatMap(said),
      ]
  }
  return []
}

/** literals finds the text a person would read written straight into a component. */
export function literals(source: string): Found[] {
  const ast = parse(source, { modern: true }) as unknown as { fragment: Node }
  const found: Found[] = []
  const lineOf = (i: number) => source.slice(0, i).split('\n').length
  type Node = { type: string; start: number; data?: string; name?: string; value?: unknown; [k: string]: unknown }
  const add = (start: number, raw: unknown, where: string) => {
    const text = String(raw ?? '').trim()
    if (WORD.test(text)) found.push({ line: lineOf(start), text, where })
  }
  const expression = (tag: Node, where: string) => {
    for (const l of said(tag.expression as Js)) add(l.start, l.value, where)
  }
  const walk = (n: unknown): void => {
    if (!n || typeof n !== 'object') return
    if (Array.isArray(n)) return n.forEach(walk)
    const node = n as Node
    switch (node.type) {
      case 'Script': case 'StyleSheet': case 'Comment': case 'StyleDirective':
        return
      case 'Text':
        return add(node.start, node.data, 'text')
      case 'ExpressionTag':
        return expression(node, 'text')
      case 'Attribute': {
        if (!node.name || !READ.has(node.name)) return
        const vs = (Array.isArray(node.value) ? node.value : [node.value]) as Node[]
        for (const v of vs) {
          if (v?.type === 'Text') add(v.start, v.data, node.name)
          else if (v?.type === 'ExpressionTag') expression(v, node.name)
        }
        return
      }
    }
    for (const [k, v] of Object.entries(node)) {
      if (k !== 'parent' && k !== 'metadata' && v && typeof v === 'object') walk(v)
    }
  }
  walk(ast.fragment)
  return found
}

describe('words a person reads', () => {
  const files = Object.keys(sources).map((p) => p.replace(/^\/src\//, ''))

  it('finds the components', () => {
    expect(files.length).toBeGreaterThan(40)
  })

  it('are translated: no literal text in a component', () => {
    const bad: string[] = []
    for (const f of files) {
      const allowed = ALLOWED[f] ?? []
      for (const l of literals(sources['/src/' + f])) {
        if (!allowed.includes(l.text)) bad.push(`${f}:${l.line} ${l.where}: ${JSON.stringify(l.text)}`)
      }
    }
    expect(bad).toEqual([])
  })

  it('the scanner sees text, labels and titles, and not code', () => {
    const src = `<script>let a = 'Hello'</script>
<p>Save</p>
<button title="Close" aria-label={$tr('x')}>·</button>
<Input placeholder="Name" />
<div class="Wide">{$tr('y')}</div>
<span title={busy ? 'Saving' : $tr('z')}>{n === 'one' ? $tr('w') : \`\${n} more\`}</span>
<style>.a { content: 'x' }</style>`
    expect(literals(src).map((l) => l.text)).toEqual(['Save', 'Close', 'Name', 'Saving', 'more'])
  })
})
