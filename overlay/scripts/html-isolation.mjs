// html-isolation.mjs is the html note's isolation check (docs/overlay/06
// §6.3, anti-goal X3), in headless Chromium: the shapes dev page with the
// probe note (frontend/src/demo/probe.ts), which tries to reach a server
// here by every load a page can make, and with its own scripts fetch, the
// window around it, top.location, the shell's bindings and storage. It
// passes when the server saw no request, the probe's scripts did not run,
// the window's title, address and storage are as they were.
//
//   node overlay/scripts/html-isolation.mjs
//
// It starts the frontend's dev server and a counting server on 127.0.0.1,
// and stops both. Playwright is not a dependency of the app: it is taken
// from NODE_PATH or a global install (npm i -g playwright). The app's
// webview is WebKit (macOS) or WebKitGTK (Linux); Chromium checks the same
// sandbox and policies, and the person checks the real window once
// (docs/overlay/09 §9.6).
import { spawn, execSync } from 'node:child_process'
import { createServer } from 'node:http'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const frontend = join(here, '..', 'frontend')

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
  console.error('html-isolation: playwright not found (npm i -g playwright, then npx playwright install chromium)')
  process.exit(2)
}

const listen = (srv) => new Promise((r) => srv.listen(0, '127.0.0.1', () => r(srv.address().port)))

const hits = []
const leak = createServer((req, res) => {
  hits.push(req.url)
  res.end('leaked')
})
const leakPort = await listen(leak)
const free = createServer()
const vitePort = await listen(free)
await new Promise((r) => free.close(r))

const vite = spawn('npx', ['vite', '--port', String(vitePort), '--strictPort'], { cwd: frontend, stdio: ['ignore', 'pipe', 'inherit'] })
let stopped = false
const stop = () => {
  if (stopped) return
  stopped = true
  vite.kill()
  leak.close()
}
process.on('exit', stop)

await new Promise((resolve, reject) => {
  vite.stdout.on('data', (d) => d.toString().includes('Local') && resolve())
  vite.on('exit', () => reject(new Error('the dev server did not start')))
  setTimeout(() => reject(new Error('the dev server did not start in 30s')), 30_000)
})

const { chromium } = playwright()
const browser = await chromium.launch()
const failures = []
try {
  const page = await browser.newPage()
  const url = `http://127.0.0.1:${vitePort}/?demo=shapes&probe=${encodeURIComponent(`http://127.0.0.1:${leakPort}`)}`
  await page.goto(url)
  await page.waitForSelector('iframe[title="Sandbox probe"]', { timeout: 15_000 })
  await page.waitForTimeout(3000)

  if (hits.length) failures.push(`the server saw requests: ${hits.join(' ')}`)
  const frame = page.frames().find((f) => f !== page.mainFrame() && f.parentFrame() === page.mainFrame() && f.url() === 'about:srcdoc')
  const said = frame ? await frame.locator('#s').textContent().catch(() => null) : null
  if (said == null) failures.push('the probe frame was not found')
  else if (!said.includes('did not run')) failures.push(`the probe's script ran: ${said}`)
  if ((await page.title()) !== 'stickypane') failures.push(`the window's title changed: ${await page.title()}`)
  if (page.url() !== url) failures.push(`the window went elsewhere: ${page.url()}`)
  if (await page.evaluate(() => localStorage.getItem('stickypane.leak'))) failures.push('the probe wrote to the window storage')
  const sandbox = await page.getAttribute('iframe[title="Sandbox probe"]', 'sandbox')
  if (sandbox !== 'allow-scripts') failures.push(`the frame's sandbox is ${sandbox}`)
} finally {
  await browser.close()
  stop()
}

if (failures.length) {
  for (const f of failures) console.error('FAIL', f)
  process.exit(1)
}
console.log('html isolation: ok (no request reached the server, the page ran no script of its own, the window is untouched)')
