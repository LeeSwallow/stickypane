// Markdown to HTML for a note (03 §3.8, 10 §10.4): all of GFM, math, code
// in colour and diagrams, with nothing from the note able to run. The
// sanitizer runs before the renderers, so what KaTeX, Shiki and Mermaid
// add is theirs, never the note's. Nothing is fetched: Shiki's themes and
// grammars and Mermaid are bundled and loaded on first use.

import type { Element, ElementContent, Root, RootContent, Text } from 'hast'
import rehypeKatex from 'rehype-katex'
import rehypeSanitize, { defaultSchema, type Options as Schema } from 'rehype-sanitize'
import rehypeStringify from 'rehype-stringify'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import remarkParse from 'remark-parse'
import remarkRehype from 'remark-rehype'
import { unified } from 'unified'
import type { Palette } from '../lib/theme/palette'
import type { Style } from '../lib/doodle/rough'

/** A diagram renderer: Mermaid source in, SVG out (it throws when it cannot). */
export type DiagramRenderer = (source: string) => Promise<string>

export interface RenderStyle {
  /** colour code with Shiki (default true) */
  highlight?: boolean
  /** how ```mermaid blocks are drawn; without one they stay code */
  mermaid?: DiagramRenderer
}

// The default schema (GitHub's), with a class kept only where a renderer
// needs one: the language of a code block and remark-math's markers.
const schema: Schema = {
  ...defaultSchema,
  attributes: {
    ...defaultSchema.attributes,
    code: [['className', /^language-./, 'math-inline', 'math-display']],
  },
} as Schema

/** render turns a note's Markdown into safe HTML. */
export async function render(md: string, style: RenderStyle = {}): Promise<string> {
  const out = await unified()
    .use(remarkParse)
    .use(remarkGfm)
    .use(remarkMath)
    .use(remarkRehype)
    .use(rehypeSanitize, schema)
    .use(rehypeKatex, { strict: 'ignore' })
    .use(wikiLinks)
    .use(codeBlocks, style)
    .use(rehypeStringify, { allowDangerousHtml: true })
    .process(md)
  return String(out)
}

const classes = (e: Element): string[] => {
  const c = e.properties?.className as unknown
  return Array.isArray(c) ? c.map(String) : typeof c === 'string' ? c.split(' ') : []
}

const textOf = (n: ElementContent | RootContent): string =>
  n.type === 'text' ? n.value : 'children' in n ? n.children.map((c) => textOf(c as ElementContent)).join('') : ''

const el = (tagName: string, properties: Element['properties'], children: ElementContent[]): Element => ({
  type: 'element',
  tagName,
  properties,
  children,
})

const linkRe = /!?\[\[([^\]]+)\]\]/g

/** wikiLinks turns [[name]], [[name#part]] and [[name|alias]] in text (not code) into links to the note. */
function wikiLinks() {
  return (tree: Root) => {
    const walk = (parent: Root | Element) => {
      if (parent.type === 'element' && (parent.tagName === 'code' || parent.tagName === 'pre' || parent.tagName === 'a')) return
      const next: (RootContent | ElementContent)[] = []
      for (const child of parent.children) {
        if (child.type === 'text' && child.value.includes('[[')) next.push(...splitLinks(child))
        else {
          if (child.type === 'element') walk(child)
          next.push(child)
        }
      }
      parent.children = next as typeof parent.children
    }
    walk(tree)
  }
}

function splitLinks(t: Text): ElementContent[] {
  const out: ElementContent[] = []
  let last = 0
  for (const m of t.value.matchAll(linkRe)) {
    const at = m.index ?? 0
    if (at > last) out.push({ type: 'text', value: t.value.slice(last, at) })
    const [target, alias] = m[1].split('|')
    const name = target.split('#')[0].trim()
    const label = (alias ?? name).trim()
    out.push(el('a', { className: ['wikilink'], dataNote: name, href: '#' }, [{ type: 'text', value: label }]))
    last = at + m[0].length
  }
  if (last < t.value.length) out.push({ type: 'text', value: t.value.slice(last) })
  return out
}

/** codeBlocks draws ```mermaid as SVG and colours the other code blocks. */
function codeBlocks(style: RenderStyle) {
  return async (tree: Root) => {
    const jobs: Promise<void>[] = []
    const visit = (parent: Root | Element) => {
      parent.children.forEach((child, i) => {
        if (child.type !== 'element') return
        const code = child.tagName === 'pre' ? child.children.find((c): c is Element => c.type === 'element' && c.tagName === 'code') : undefined
        if (!code) return visit(child)
        const lang = classes(code).find((c) => c.startsWith('language-'))?.slice('language-'.length) ?? ''
        const source = textOf(code)
        const put = (n: ElementContent) => (parent.children[i] = n as (typeof parent.children)[number])
        if (lang === 'mermaid' && style.mermaid) jobs.push(diagram(style.mermaid, source, child).then((n) => void put(n)))
        else if (style.highlight !== false && lang && lang !== 'math') jobs.push(highlight(source, lang).then((n) => void (n && put(n))))
      })
    }
    visit(tree)
    await Promise.all(jobs)
  }
}

async function diagram(draw: DiagramRenderer, source: string, pre: Element): Promise<ElementContent> {
  try {
    const svg = await draw(source)
    // Mermaid made it under securityLevel 'strict' (its own DOMPurify pass).
    return el('div', { className: ['mermaid'] }, [{ type: 'raw', value: svg } as unknown as ElementContent])
  } catch (e) {
    const why = e instanceof Error ? e.message : String(e)
    return el('div', { className: ['mermaid-error'] }, [pre, el('p', { className: ['why'] }, [{ type: 'text', value: why }])])
  }
}

// Shiki, loaded once on first use: two bundled themes (light and dark, both
// in the HTML as CSS variables, so a mode change needs no new render) and
// the JavaScript regex engine (no WebAssembly). Grammars load as needed.
type Highlighter = import('shiki/core').HighlighterCore
let shiki: Promise<Highlighter> | null = null

function highlighter(): Promise<Highlighter> {
  shiki ??= (async () => {
    const [{ createHighlighterCore }, { createJavaScriptRegexEngine }, light, dark] = await Promise.all([
      import('shiki/core'),
      import('shiki/engine/javascript'),
      import('shiki/themes/github-light.mjs'),
      import('shiki/themes/github-dark.mjs'),
    ])
    return createHighlighterCore({ themes: [light.default, dark.default], langs: [], engine: createJavaScriptRegexEngine() })
  })()
  return shiki
}

async function loadLanguage(h: Highlighter, lang: string): Promise<string | null> {
  if (h.getLoadedLanguages().includes(lang)) return lang
  const { bundledLanguages, bundledLanguagesAlias } = await import('shiki/langs')
  const id = lang in bundledLanguages ? lang : (bundledLanguagesAlias as Record<string, unknown>)[lang] ? lang : null
  if (!id) return null
  const load = (bundledLanguages as Record<string, () => Promise<{ default: unknown }>>)[id] ??
    (bundledLanguagesAlias as Record<string, () => Promise<{ default: unknown }>>)[id]
  await h.loadLanguage((await load()).default as Parameters<Highlighter['loadLanguage']>[0])
  return id
}

async function highlight(source: string, lang: string): Promise<ElementContent | null> {
  try {
    const h = await highlighter()
    const id = await loadLanguage(h, lang.toLowerCase())
    if (!id) return null // an unknown language stays plain code
    const root = h.codeToHast(source.replace(/\n$/, ''), {
      lang: id,
      themes: { light: 'github-light', dark: 'github-dark' },
      defaultColor: false,
    })
    return (root.children.find((c) => c.type === 'element') as ElementContent | undefined) ?? null
  } catch {
    return null
  }
}

/** mermaidConfig is how Mermaid draws here: strict, in the board's colours, by hand in doodle. */
export function mermaidConfig(p: Palette, style: Style) {
  return {
    startOnLoad: false,
    securityLevel: 'strict' as const,
    theme: 'base' as const,
    look: style === 'doodle' ? ('handDrawn' as const) : ('classic' as const),
    fontFamily: style === 'doodle' ? 'Gaegu, Pretendard Variable, sans-serif' : 'Pretendard Variable, sans-serif',
    themeVariables: {
      darkMode: p.dark,
      background: p.base,
      primaryColor: p.notes[0],
      primaryTextColor: p.base,
      primaryBorderColor: p.muted,
      secondaryColor: p.notes[2],
      tertiaryColor: p.notes[3],
      lineColor: p.muted,
      textColor: p.text,
      noteBkgColor: p.notes[0],
      noteTextColor: p.base,
    },
  }
}

let diagrams = 0
// Mermaid keeps one configuration for the page, so drawings take turns:
// each sets its configuration and draws before the next begins.
let turn: Promise<unknown> = Promise.resolve()

/** The part of Mermaid used here (a test passes its own). */
export interface MermaidLike {
  initialize(config: ReturnType<typeof mermaidConfig>): void
  render(id: string, source: string): Promise<{ svg: string }>
}

const bundled = () => import('mermaid') as unknown as Promise<{ default: MermaidLike }>

/** mermaidRenderer draws with the bundled Mermaid, loaded on first use, one drawing at a time. */
export function mermaidRenderer(p: Palette, style: Style, load: () => Promise<{ default: MermaidLike }> = bundled): DiagramRenderer {
  return (source) => {
    const run = turn.then(async () => {
      const { default: mermaid } = await load()
      mermaid.initialize(mermaidConfig(p, style))
      const { svg } = await mermaid.render(`sp-diagram-${++diagrams}`, source)
      return svg
    })
    turn = run.catch(() => {})
    return run
  }
}

/**
 * highlightCode is source coloured as Shiki colours a code block of a
 * note (both themes as CSS variables), as HTML; null for a language it
 * does not know, which stays plain.
 */
export async function highlightCode(source: string, lang: string): Promise<string | null> {
  try {
    const h = await highlighter()
    const id = await loadLanguage(h, lang.toLowerCase())
    if (!id) return null
    return h.codeToHtml(source.replace(/\n$/, ''), { lang: id, themes: { light: 'github-light', dark: 'github-dark' }, defaultColor: false })
  } catch {
    return null
  }
}
