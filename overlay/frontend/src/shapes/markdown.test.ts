import { describe, expect, test } from 'vitest'
import { THEMES } from '../lib/theme/palettes'
import { mermaidConfig, mermaidRenderer, render } from './markdown'

const plain = { highlight: false }

describe('markdown', () => {
  test('markdown renders GFM table, task list, footnote', async () => {
    const md = '| a | b |\n|---|---|\n| 1 | 2 |\n\n- [x] done\n- [ ] todo\n\nSee this[^1].\n\n[^1]: the note\n\n~~gone~~'
    const html = await render(md, plain)
    expect(html).toContain('<table>')
    expect(html).toContain('<td>2</td>')
    expect(html).toMatch(/<input[^>]*type="checkbox"[^>]*checked/)
    expect(html).toMatch(/<input[^>]*type="checkbox"[^>]*disabled/)
    expect(html).toContain('data-footnote-ref')
    expect(html).toContain('the note')
    expect(html).toContain('<del>gone</del>')
  })

  test('markdown strips script and on* attributes', async () => {
    const md = 'hi <script>alert(1)</script>\n\n<img src="x" onerror="alert(2)">\n\n<a href="javascript:alert(3)" onclick="x()">x</a>\n\n<div class="evil" style="color:red">d</div>\n\n<iframe src="https://example.com"></iframe>'
    const html = await render(md, plain)
    expect(html).not.toMatch(/<script/i)
    expect(html).not.toMatch(/onerror|onclick/i)
    expect(html).not.toMatch(/javascript:/i)
    expect(html).not.toMatch(/<iframe/i)
    expect(html).not.toContain('class="evil"') // class only on code and math
    expect(html).not.toContain('style=')
  })

  test('a class survives only on code and math', async () => {
    const html = await render('```js\nlet a = 1\n```', plain)
    expect(html).toContain('class="language-js"')
  })

  test('mermaid block becomes svg', async () => {
    const seen: string[] = []
    const mermaid = async (src: string) => {
      seen.push(src)
      return '<svg xmlns="http://www.w3.org/2000/svg" class="m"><g><text>A</text></g></svg>'
    }
    const html = await render('before\n\n```mermaid\nflowchart LR\n  A --> B\n```\n\nafter', { ...plain, mermaid })
    expect(seen).toEqual(['flowchart LR\n  A --> B\n'])
    expect(html).toMatch(/<div class="mermaid"><svg/)
    expect(html).not.toContain('language-mermaid')
  })

  test('a diagram that cannot be drawn shows its source and why', async () => {
    const mermaid = async () => {
      throw new Error('Parse error on line 1')
    }
    const html = await render('```mermaid\nnot a diagram\n```', { ...plain, mermaid })
    expect(html).toContain('not a diagram')
    expect(html).toContain('Parse error on line 1')
  })

  test('mermaid runs strict, in the board colours', () => {
    const p = THEMES[0]
    const c = mermaidConfig(p, 'doodle')
    expect(c.securityLevel).toBe('strict')
    expect(c.startOnLoad).toBe(false)
    expect(c.theme).toBe('base')
    expect(c.themeVariables.primaryColor).toBe(p.notes[0])
    expect(c.themeVariables.lineColor).toBe(p.muted)
    expect(c.themeVariables.textColor).toBe(p.text)
    expect(c.look).toBe('handDrawn')
    expect(mermaidConfig(p, 'neat').look).toBe('classic')
  })

  test('Mermaid draws one diagram at a time: its configuration is shared', async () => {
    const log: string[] = []
    let open = 0
    const fake = {
      initialize: (c: { look: string }) => log.push('init ' + c.look),
      render: async (id: string, src: string) => {
        open++
        expect(open).toBe(1)
        await new Promise((r) => setTimeout(r, 5))
        open--
        log.push('render ' + src)
        return { svg: `<svg id="${id}"/>` }
      },
    }
    const load = async () => ({ default: fake })
    const doodle = mermaidRenderer(THEMES[0], 'doodle', load)
    const neat = mermaidRenderer(THEMES[0], 'neat', load)
    const out = await Promise.all([doodle('a'), neat('b'), doodle('c')])
    expect(out.every((s) => s.startsWith('<svg'))).toBe(true)
    expect(log).toEqual(['init handDrawn', 'render a', 'init classic', 'render b', 'init handDrawn', 'render c'])
  })

  test('a failed diagram does not stop the next', async () => {
    let n = 0
    const fake = { initialize: () => {}, render: async () => (++n === 1 ? Promise.reject(new Error('bad')) : { svg: '<svg/>' }) }
    const draw = mermaidRenderer(THEMES[0], 'neat', async () => ({ default: fake }))
    await expect(draw('x')).rejects.toThrow('bad')
    await expect(draw('y')).resolves.toBe('<svg/>')
  })

  test('math renders', async () => {
    const html = await render('Inline $a^2 + b^2$ and\n\n$$\n\\frac{1}{2}\n$$', plain)
    expect(html).toContain('class="katex"')
    expect(html).toContain('katex-display')
    expect(html).toContain('<math')
  })

  test('code is highlighted with Shiki in both modes', async () => {
    const html = await render('```ts\nconst a: number = 1\n```', { highlight: true })
    expect(html).toContain('class="shiki')
    expect(html).toContain('--shiki-dark')
    expect(html).toContain('const')
  }, 20_000)

  test('an unknown language is shown as plain text', async () => {
    const html = await render('```nolang\n<b>x</b>\n```', { highlight: true })
    expect(html).toContain('&#x3C;b>x&#x3C;/b>')
  }, 20_000)

  test('[[links]] become links to the note, not in code', async () => {
    const html = await render('see [[plan]] and [[work/todo#item|the list]]\n\n`[[code]]`', plain)
    expect(html).toContain('<a class="wikilink" data-note="plan" href="#">plan</a>')
    expect(html).toContain('<a class="wikilink" data-note="work/todo" href="#">the list</a>')
    expect(html).toContain('<code>[[code]]</code>')
  })
})
