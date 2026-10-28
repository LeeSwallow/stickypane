// The html note's sandbox (02 §2.7, ADR-0006): what the frame may do, the
// policy every page gets before any of its own text, the window's policy
// the frame inherits, and the one message the window takes from it.
import { describe, expect, it } from 'vitest'
import index from '../../index.html?raw'
import { clean, CSP, HEIGHT, heightFrom, MAX_HEIGHT, REPORTER, SANDBOX, srcdoc } from './html'

const sha256 = async (s: string) =>
  btoa(String.fromCharCode(...new Uint8Array(await crypto.subtle.digest('SHA-256', new TextEncoder().encode(s)))))
const windowPolicy = index.match(/http-equiv="Content-Security-Policy" content="([^"]+)"/)![1]
const directive = (policy: string, name: string) =>
  policy
    .split(';')
    .map((d) => d.trim().split(/\s+/))
    .find((d) => d[0] === name)
    ?.slice(1) ?? null

describe('the frame', () => {
  it('may run scripts and nothing else: no same origin, no navigation of the window, no forms, no popups', () => {
    expect(SANDBOX.split(/\s+/)).toEqual(['allow-scripts'])
    for (const no of ['allow-same-origin', 'allow-top-navigation', 'allow-forms', 'allow-popups', 'allow-modals', 'allow-downloads'])
      expect(SANDBOX).not.toContain(no)
  })

  it("gets a policy that loads nothing: default-src 'none', images only as data:", () => {
    expect(directive(CSP, 'default-src')).toEqual(["'none'"])
    expect(directive(CSP, 'img-src')).toEqual(['data:'])
    expect(directive(CSP, 'style-src')).toEqual(["'unsafe-inline'"])
    expect(directive(CSP, 'script-src')).toEqual(["'unsafe-inline'"])
    expect(directive(CSP, 'connect-src')).toBeNull() // falls back to default-src 'none'
    expect(directive(CSP, 'form-action')).toEqual(["'none'"])
    expect(directive(CSP, 'base-uri')).toEqual(["'none'"])
  })

  it('the policy comes before anything of the page', () => {
    const page = '<meta http-equiv="Content-Security-Policy" content="default-src *"><script>1</script><h1>x</h1>'
    const doc = srcdoc(page)
    const ours = doc.indexOf(`<meta http-equiv="Content-Security-Policy" content="${CSP}">`)
    expect(ours).toBeGreaterThan(0)
    expect(ours).toBeLessThan(doc.indexOf('<body>'))
    expect(doc.indexOf('default-src *')).toBeGreaterThan(doc.indexOf('<body>')) // a page's own policy is in the body, where it does nothing
    expect(doc.slice(0, ours)).toBe('<!doctype html><html><head><meta charset="utf-8">')
  })

  it('a page cannot send the frame elsewhere or re-aim its links', () => {
    const page = '<META HTTP-EQUIV="refresh" content="0;url=https://evil.example"><meta http-equiv=refresh content=1><base href="https://evil.example/"><p>hi</p>'
    expect(clean(page)).toBe('<p>hi</p>')
    expect(srcdoc(page)).not.toMatch(/refresh|<base/i)
  })

  it("takes the window's colour scheme, so its ground stays the note's", () => {
    expect(srcdoc('', { dark: true })).toContain('<meta name="color-scheme" content="dark">')
    expect(srcdoc('')).toContain('<meta name="color-scheme" content="light">')
  })

  it('the ink is a colour or the system text, never more CSS', () => {
    expect(srcdoc('', { ink: '#1f2328' })).toContain('color:#1f2328')
    expect(srcdoc('', { ink: 'red;}body{background:url(https://x)' })).toContain('color:CanvasText')
    expect(srcdoc('')).not.toContain('url(')
  })
})

describe('the window the frame sits in', () => {
  // A frame drawn from srcdoc inherits the window's policy as well: the
  // two apply together, so these hold for the frame too.
  it('runs no inline script but the height reporter, which it names by hash', async () => {
    const scripts = directive(windowPolicy, 'script-src')!
    expect(scripts).not.toContain("'unsafe-inline'")
    expect(scripts).not.toContain("'unsafe-eval'")
    const hash = await sha256(REPORTER)
    expect(scripts).toContain(`'sha256-${hash}'`)
    expect(srcdoc('')).toContain(`<script>${REPORTER}</script>`)
  })

  it('lets no frame navigate anywhere and connects only to itself, so a frame reaches no binding', () => {
    expect(directive(windowPolicy, 'frame-src')).toEqual(["'none'"])
    expect(directive(windowPolicy, 'connect-src')).toEqual(["'self'"])
    expect(directive(windowPolicy, 'object-src')).toEqual(["'none'"])
  })

  it('the reporter only posts one number to the window', () => {
    expect(REPORTER).toContain(`type:"${HEIGHT}"`)
    expect(REPORTER).not.toMatch(/parent\.(document|_wails|wails|webkit|location)|top\.|fetch|XMLHttpRequest|localStorage|cookie/)
  })
})

describe('heightFrom', () => {
  const frame = {}
  const msg = (data: unknown, over: Partial<{ source: unknown; origin: string }> = {}) => ({ source: frame, origin: 'null', data, ...over })

  it('takes a height from the frame, rounded up and capped', () => {
    expect(heightFrom(msg({ type: HEIGHT, height: 120.2 }), frame)).toBe(121)
    expect(heightFrom(msg({ type: HEIGHT, height: 1e9 }), frame)).toBe(MAX_HEIGHT)
  })

  it('takes nothing from anything else', () => {
    expect(heightFrom(msg({ type: HEIGHT, height: 100 }, { source: {} }), frame)).toBeNull() // another frame
    expect(heightFrom(msg({ type: HEIGHT, height: 100 }, { origin: 'http://wails.localhost' }), frame)).toBeNull() // not sandboxed
    expect(heightFrom(msg({ type: HEIGHT, height: 100 }), null)).toBeNull()
    for (const data of [null, 7, 'x', { height: 100 }, { type: HEIGHT }, { type: HEIGHT, height: '100' }, { type: HEIGHT, height: -1 }, { type: HEIGHT, height: NaN }, { type: HEIGHT, height: Infinity }])
      expect(heightFrom(msg(data), frame), JSON.stringify(data)).toBeNull()
  })
})
