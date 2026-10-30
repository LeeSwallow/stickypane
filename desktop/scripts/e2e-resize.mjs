// e2e-resize.mjs drags notes' edges with real mouse events in WebKit, the
// engine of the app's window on macOS (#66): the jsdom tests have no layout
// and no pointer, so they cannot tell whether an edge can be grabbed. On
// the screens dev page (frontend/src/demo), over its made-up project:
//
//   grid    the right border of a note, pressed where it is drawn, takes
//           the width in fraction steps (a live preview, then note.arrange
//           with the size); the bottom border takes the height (rows); a
//           double-click on it puts the height back (rows 0); a drag still
//           follows the mouse where pointer capture is refused
//   canvas  the right, bottom and corner of a leaning note, pressed on its
//           drawn border, resize it live and keep the place; the handles
//           show on hover, are at least 8px across and have resize cursors
//   errors  a save the project refuses is told in a toast
//
//   node desktop/scripts/e2e-resize.mjs [webkit|chromium]
//
// It starts the frontend's dev server on 127.0.0.1 and stops it; the
// bindings must exist (wails3 build, or wails3 generate bindings -ts -i).
// Playwright is not a dependency of the app: it is taken from NODE_PATH or
// a global install (npm i -g playwright, then npx playwright install webkit).
import { spawn, execSync } from 'node:child_process'
import { createServer } from 'node:http'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const frontend = join(here, '..', 'frontend')
const engine = process.argv[2] ?? 'webkit'

function playwright() {
  const require = createRequire(import.meta.url)
  const tries = [...(process.env.NODE_PATH ?? '').split(':').filter(Boolean)]
  try {
    tries.push(join(execSync('npm root -g').toString().trim()))
  } catch {}
  for (const dir of ['', ...tries]) {
    try {
      return require(dir ? join(dir, 'playwright') : 'playwright')
    } catch {}
  }
  console.error('e2e-resize: playwright not found (npm i -g playwright, then npx playwright install webkit)')
  process.exit(2)
}

const free = createServer()
const port = await new Promise((r) => free.listen(0, '127.0.0.1', () => r(free.address().port)))
await new Promise((r) => free.close(r))
const vite = spawn('npx', ['vite', '--port', String(port), '--strictPort'], { cwd: frontend, stdio: ['ignore', 'pipe', 'pipe'] })
// the dev server echoes the page's console; keep it for a server that does not start
let said = ''
vite.stderr.on('data', (d) => (said += d))
let stopped = false
const stop = () => {
  if (stopped) return
  stopped = true
  vite.kill()
}
process.on('exit', stop)
await new Promise((resolve, reject) => {
  vite.stdout.on('data', (d) => d.toString().includes('Local') && resolve())
  vite.on('exit', () => reject(new Error(`the dev server did not start\n${said}`)))
  setTimeout(() => reject(new Error('the dev server did not start in 30s')), 30_000)
})

const pw = playwright()
const browser = await pw[engine].launch()
const failures = []
let passed = 0
const check = (ok, what) => (ok ? passed++ : failures.push(what))
const base = `http://127.0.0.1:${port}/?demo=screens&lang=en`

async function open(screen, { fail = '', noCapture = false } = {}) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 860 } })
  page.on('pageerror', (e) => failures.push(`${screen}: page error: ${e.message}`))
  // a webview that will not capture the pointer (it throws, as WebKit does
  // for a pointer it does not count as active)
  if (noCapture)
    await page.addInitScript(() => {
      Element.prototype.setPointerCapture = function () {
        throw new DOMException('no active pointer', 'NotFoundError')
      }
    })
  await page.goto(`${base}&screen=${screen}${fail ? `&fail=${fail}` : ''}`)
  await page.waitForSelector(screen === 'canvas' ? '[data-place]' : '[data-frame]')
  await page.waitForTimeout(400)
  return page
}

const calls = (page) => page.evaluate(() => (window.__demoCalls ?? []).splice(0))

/** drag presses at (x, y), moves by (dx, dy) in steps, reads mid() while held, and lets go. */
async function drag(page, x, y, dx, dy, mid) {
  await page.mouse.move(x, y)
  await page.mouse.down()
  for (let i = 1; i <= 10; i++) await page.mouse.move(x + (dx * i) / 10, y + (dy * i) / 10)
  const during = mid ? await mid() : null
  await page.mouse.up()
  await page.waitForTimeout(300)
  return during
}

// --- grid --------------------------------------------------------------------
// a point on the drawn border of a grid note: its right side, or its bottom
const gridBorder = (page, name, side) =>
  page.evaluate(
    ([name, side]) => {
      const r = document.querySelector(`[data-frame="${name}"]`).getBoundingClientRect()
      return side === 'right' ? { x: r.right - 1, y: r.top + r.height / 2 } : { x: r.left + r.width / 2, y: r.bottom - 1 }
    },
    [name, side],
  )
const gridNote = (page, name) =>
  page.evaluate((name) => {
    const slot = document.querySelector(`[data-slot="${name}"]`)
    const frame = slot.querySelector('.frame')
    return {
      span: slot.style.gridColumn,
      width: frame.getBoundingClientRect().width,
      body: frame.querySelector('.body').getBoundingClientRect().height,
      landing: document.querySelector('.landing')?.textContent ?? null,
    }
  }, name)

async function gridWidth(page, label) {
  const name = 'work/release.md'
  const before = await gridNote(page, name)
  const at = await gridBorder(page, name, 'right')
  await calls(page)
  const mid = await drag(page, at.x, at.y, 300, 0, () => gridNote(page, name))
  const after = await gridNote(page, name)
  const sent = (await calls(page)).filter((c) => c.method === 'arrange')
  check(mid.width > before.width + 200, `${label}: the width does not follow the right border (${before.width} -> ${mid.width} while dragging)`)
  check(!!mid.landing, `${label}: no fraction is shown while dragging the width`)
  check(after.span !== before.span, `${label}: the width did not land on another fraction (${before.span} -> ${after.span})`)
  check(sent.some((c) => c.params.note === name && c.params.size && c.params.size !== '1/3'), `${label}: note.arrange was not sent with the new size (${JSON.stringify(sent)})`)
}

{
  const page = await open('board')
  await gridWidth(page, 'grid width')

  const name = 'work/todo.md'
  const before = await gridNote(page, name)
  let at = await gridBorder(page, name, 'bottom')
  const mid = await drag(page, at.x, at.y, 0, 120, () => gridNote(page, name))
  const sent = (await calls(page)).filter((c) => c.method === 'arrange')
  check(mid.body > before.body + 80, `grid height: the height does not follow the bottom border (${before.body} -> ${mid.body} while dragging)`)
  check(sent.some((c) => c.params.note === name && c.params.rows > 0), `grid height: note.arrange was not sent with rows (${JSON.stringify(sent)})`)
  check((await gridNote(page, name)).body > before.body + 80, 'grid height: the height did not stay after letting go')

  // the taller note now reaches below the window: bring its border in
  await page.evaluate((name) => document.querySelector(`[data-frame="${name}"] .edge`).scrollIntoView({ block: 'center' }), name)
  at = await gridBorder(page, name, 'bottom')
  await page.mouse.dblclick(at.x, at.y)
  await page.waitForTimeout(300)
  const reset = (await calls(page)).filter((c) => c.method === 'arrange')
  check(reset.some((c) => c.params.note === name && c.params.rows === 0), `grid height: a double-click on the bottom border did not put the height back (${JSON.stringify(reset)})`)

  const cursor = await page.evaluate(() => [...document.querySelectorAll('[data-frame="work/release.md"] .edge')].map((e) => getComputedStyle(e).cursor))
  check(cursor.includes('ew-resize') && cursor.includes('ns-resize'), `grid: the edges have no resize cursors (${cursor})`)
  await page.close()
}
{
  const page = await open('board', { noCapture: true })
  await gridWidth(page, 'grid width without pointer capture')
  await page.close()
}
{
  const page = await open('board', { fail: 'arrange' })
  const at = await gridBorder(page, 'work/release.md', 'right')
  await drag(page, at.x, at.y, 300, 0)
  await page.waitForTimeout(300)
  const toast = await page.locator('[role="alert"]').textContent().catch(() => null)
  check(!!toast && toast.includes('refused'), `grid: a refused note.arrange is not told (${toast})`)
  await page.close()
}

// --- canvas ------------------------------------------------------------------
// a point on the drawn border of a placed note, which leans by --tilt
// around its centre: its right side (e), its bottom (s) or the corner (se)
const canvasBorder = (page, name, edge) =>
  page.evaluate(
    ([name, edge]) => {
      const item = document.querySelector(`[data-place="${name}"]`)
      const r = item.getBoundingClientRect()
      const cx = r.left + r.width / 2
      const cy = r.top + r.height / 2
      const w = item.offsetWidth
      const h = item.offsetHeight
      const t = (parseFloat(item.style.getPropertyValue('--tilt')) * Math.PI) / 180
      const lx = edge === 's' ? 0 : w / 2 - 1
      const ly = edge === 'e' ? 0 : h / 2 - 1
      return { x: cx + lx * Math.cos(t) - ly * Math.sin(t), y: cy + lx * Math.sin(t) + ly * Math.cos(t) }
    },
    [name, edge],
  )
const placeOf = (page, name) =>
  page.evaluate((name) => {
    const s = document.querySelector(`[data-place="${name}"]`)
    return { w: parseFloat(s.style.width), h: parseFloat(s.style.height) }
  }, name)

{
  const page = await open('canvas')
  // release.md leans by -1.5°
  const name = 'work/release.md'
  for (const [edge, dx, dy] of [
    ['e', 64, 0],
    ['s', 0, 48],
    ['se', 40, 40],
  ]) {
    const before = await placeOf(page, name)
    const at = await canvasBorder(page, name, edge)
    await calls(page)
    const mid = await drag(page, at.x, at.y, dx, dy, () => placeOf(page, name))
    const sent = (await calls(page)).filter((c) => c.method === 'place' && c.params[name])
    const want = { w: before.w + dx, h: before.h + dy }
    const near = (p) => Math.abs(p.w - want.w) <= 8 && Math.abs(p.h - want.h) <= 8
    check(near(mid), `canvas ${edge}: the note does not follow its border (${JSON.stringify(before)} -> ${JSON.stringify(mid)} while dragging)`)
    check(sent.length && near(sent.at(-1).params[name]), `canvas ${edge}: the size was not kept (${JSON.stringify(sent)})`)
  }

  const handles = await page.evaluate(() => {
    const item = document.querySelector('[data-place="work/todo.md"]')
    return ['e', 's', 'se'].map((edge) => {
      const h = item.querySelector(`.handle.${edge}`)
      const r = h.getBoundingClientRect()
      return { edge, cursor: getComputedStyle(h).cursor, across: Math.min(r.width, r.height) }
    })
  })
  for (const h of handles) {
    check(h.across >= 8, `canvas ${h.edge}: the handle is ${h.across}px across`)
    check(h.cursor === { e: 'ew-resize', s: 'ns-resize', se: 'nwse-resize' }[h.edge], `canvas ${h.edge}: the cursor is ${h.cursor}`)
  }
  // shown on hover: each handle draws a mark that can be seen
  const at = await canvasBorder(page, 'work/todo.md', 'e')
  await page.mouse.move(at.x - 40, at.y)
  await page.waitForTimeout(250)
  const seen = await page.evaluate(() =>
    ['e', 's', 'se'].map((edge) => {
      const s = getComputedStyle(document.querySelector(`[data-place="work/todo.md"] .handle.${edge}`), '::after')
      return s.content !== 'none' && Number(s.opacity) > 0
    }),
  )
  check(seen.every(Boolean), `canvas: not every handle shows on hover (e, s, se: ${seen})`)
  await page.close()
}
{
  const page = await open('canvas', { noCapture: true })
  const name = 'work/todo.md'
  const before = await placeOf(page, name)
  const at = await canvasBorder(page, name, 'se')
  const mid = await drag(page, at.x, at.y, 48, 48, () => placeOf(page, name))
  check(mid.w >= before.w + 40 && mid.h >= before.h + 40, `canvas without pointer capture: the corner does not follow (${JSON.stringify(before)} -> ${JSON.stringify(mid)})`)
  await page.close()
}
{
  const page = await open('canvas', { fail: 'place' })
  const at = await canvasBorder(page, 'work/todo.md', 'e')
  await drag(page, at.x, at.y, 48, 0)
  await page.waitForTimeout(300)
  const toast = await page.locator('[role="alert"]').textContent().catch(() => null)
  check(!!toast && toast.includes('refused'), `canvas: a refused place is not told (${toast})`)
  await page.close()
}

await browser.close()
stop()
if (failures.length) {
  for (const f of failures) console.error('FAIL', f)
  console.error(`e2e resize (${engine}): ${failures.length} failed, ${passed} passed`)
  process.exit(1)
}
console.log(`e2e resize (${engine}): ok, ${passed} checks (grid edges, canvas handles, errors told)`)
