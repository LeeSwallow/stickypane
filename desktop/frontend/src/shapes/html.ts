// An html note is drawn in a sandboxed iframe (02 §2.7, ADR-0006): the
// page as written, in a frame with an opaque origin (sandbox without
// allow-same-origin), so it reaches neither the window around it, nor the
// shell's bindings, nor any storage; with a policy that loads nothing
// from the network; and with no forms, popups or navigation of the
// window. The one thing the frame tells the window is its height, as one
// number, and the window takes it only from that frame.
//
// The window's own policy (index.html) is inherited by the frame too, and
// it runs no inline script but the height reporter below, which it names
// by hash: a page's own scripts do not run in the app. What a page shows
// is its HTML and CSS.

/** SANDBOX is the frame's sandbox: scripts (the height reporter) and nothing else. */
export const SANDBOX = 'allow-scripts'

/** CSP is the policy written into every page: nothing from the network, images only as data:. */
export const CSP = "default-src 'none'; img-src data:; style-src 'unsafe-inline'; script-src 'unsafe-inline'; form-action 'none'; base-uri 'none'"

/** The message the height reporter posts. */
export const HEIGHT = 'stickypane:height'

/**
 * REPORTER tells the window the page's height when it loads and whenever
 * it changes. index.html allows it by its SHA-256 (html.test.ts checks
 * the two agree): change one, change both.
 */
export const REPORTER = `(function(){var s=function(){parent.postMessage({type:"${HEIGHT}",height:Math.ceil(document.documentElement.getBoundingClientRect().height)},"*")};addEventListener("load",s);new ResizeObserver(s).observe(document.documentElement)})()`

/** The frame's own look: the note's ink on the note's paper, nothing scrolls sideways. */
const base = (ink: string) =>
  `html,body{margin:0;background:transparent;color:${ink};font:13px/1.5 system-ui,sans-serif;overflow-wrap:anywhere}body{padding:2px}img,svg,video,canvas{max-width:100%;height:auto}`

/** A colour the look may name: a hex colour only, or the system's text colour. */
const inkOf = (c: string | undefined) => (c && /^#[0-9a-f]{3,8}$/i.test(c.trim()) ? c.trim() : 'CanvasText')

// tags that would send the frame somewhere or change where its links go;
// the policies stop them as well, these never reach the frame at all
const strip = [
  /<meta\b[^>]*http-equiv\s*=\s*["']?\s*refresh[^>]*>/gi,
  /<base\b[^>]*>/gi,
]

/** clean is the page without the tags that navigate or re-aim it. */
export function clean(html: string): string {
  let out = html
  for (const re of strip) out = out.replace(re, '')
  return out
}

/**
 * srcdoc is the document the frame draws: the policy first, before any of
 * the page, then the frame's look, the reporter, and the page. The frame
 * takes the window's colour scheme (a frame whose scheme differs from the
 * window's is painted opaque), so its ground stays the note's paper; its
 * text is the note's ink.
 */
export function srcdoc(html: string, look: { dark?: boolean; ink?: string } = {}): string {
  return (
    '<!doctype html><html><head><meta charset="utf-8">' +
    `<meta http-equiv="Content-Security-Policy" content="${CSP}">` +
    '<meta name="referrer" content="no-referrer">' +
    `<meta name="color-scheme" content="${look.dark ? 'dark' : 'light'}">` +
    `<style>${base(inkOf(look.ink))}</style>` +
    `<script>${REPORTER}</script>` +
    `</head><body>${clean(html)}</body></html>`
  )
}

/** The most a page grows to before it scrolls inside its frame. */
export const MAX_HEIGHT = 4000

/**
 * heightFrom is the height a message gives, or null when the window must
 * not take it: it did not come from this frame, or from the opaque origin
 * a sandboxed frame has, or it is not one number in range.
 */
export function heightFrom(e: { source: unknown; origin: string; data: unknown }, frame: unknown): number | null {
  if (!frame || e.source !== frame || e.origin !== 'null') return null
  const d = e.data as { type?: unknown; height?: unknown } | null
  if (!d || typeof d !== 'object' || d.type !== HEIGHT || typeof d.height !== 'number') return null
  if (!Number.isFinite(d.height) || d.height < 0) return null
  return Math.min(Math.ceil(d.height), MAX_HEIGHT)
}
