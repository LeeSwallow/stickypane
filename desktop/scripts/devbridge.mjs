// devbridge.mjs lets the dev screens page talk to a real `stickypane
// bridge`, for walking the first run in a browser without the shell. It
// listens on 127.0.0.1 only and answers the page's requests by passing
// them to the bridge it runs in one project folder:
//
//   HOME=$tmp/home STICKYPANE_HOME=$tmp/sp \
//     node desktop/scripts/devbridge.mjs --bin $tmp/stickypane --project $tmp/proj --port 9300
//
// and the dev server with DEVBRIDGE=http://127.0.0.1:9300 (it passes
// /devbridge here), then open
// http://127.0.0.1:9245/?demo=screens&screen=empty&bridge=/devbridge
//
//   GET  /info    the folder, the program's version, whether it has a board
//   POST /init    stickypane init in the folder, then the bridge starts
//   POST /rpc     {method, params} to the bridge; {result} or {error}
//   GET  /events  the bridge's notifications, as server-sent events
//   POST /plan    stickypane setup --scope user [--mcp] --dry-run: what connecting would run
//
// Set HOME and STICKYPANE_HOME to temporary folders: init and the bridge
// write there. Nothing else of the machine is touched; setup only ever
// runs with --dry-run. Dev only: nothing here ships.
import { spawn, execFile } from 'node:child_process'
import { existsSync } from 'node:fs'
import { createServer } from 'node:http'
import { join } from 'node:path'
import { createInterface } from 'node:readline'

const arg = (name, def) => {
  const i = process.argv.indexOf(`--${name}`)
  return i > 0 ? process.argv[i + 1] : def
}
const bin = arg('bin')
const project = arg('project')
const port = Number(arg('port', '9300'))
if (!bin || !project) {
  console.error('usage: node devbridge.mjs --bin <stickypane> --project <folder> [--port 9300]')
  process.exit(2)
}

const run = (args) =>
  new Promise((resolve) =>
    execFile(bin, args, { cwd: project, env: process.env }, (err, stdout, stderr) =>
      resolve({ code: err ? (err.code ?? 1) : 0, output: (stdout + stderr).trim() }),
    ),
  )

let bridge = null
let next = 1
const pending = new Map()
const listeners = new Set()

function startBridge() {
  if (bridge || !existsSync(join(project, '.sticky'))) return
  bridge = spawn(bin, ['bridge'], { cwd: project, env: process.env, stdio: ['pipe', 'pipe', 'inherit'] })
  createInterface({ input: bridge.stdout }).on('line', (line) => {
    let m
    try {
      m = JSON.parse(line)
    } catch {
      return
    }
    if (m.id != null && pending.has(m.id)) {
      pending.get(m.id)(m)
      pending.delete(m.id)
    } else if (m.method) {
      for (const send of listeners) send(m)
    }
  })
  bridge.on('exit', () => (bridge = null))
}

function call(method, params) {
  startBridge()
  if (!bridge) return Promise.resolve({ error: { code: 'no_board', message: 'no .sticky in ' + project } })
  const id = next++
  return new Promise((resolve) => {
    pending.set(id, resolve)
    bridge.stdin.write(JSON.stringify({ id, method, params: params ?? {} }) + '\n')
  })
}

const body = (req) =>
  new Promise((resolve) => {
    let s = ''
    req.on('data', (c) => (s += c))
    req.on('end', () => {
      try {
        resolve(s ? JSON.parse(s) : {})
      } catch {
        resolve({})
      }
    })
  })

const server = createServer(async (req, res) => {
  res.setHeader('Access-Control-Allow-Origin', '*')
  res.setHeader('Access-Control-Allow-Headers', 'content-type')
  if (req.method === 'OPTIONS') return res.end()
  const json = (v) => {
    res.setHeader('content-type', 'application/json')
    res.end(JSON.stringify(v))
  }
  switch (req.url) {
    case '/info': {
      const v = await run(['version'])
      return json({ project, version: v.output.split(/\s+/).pop(), board: existsSync(join(project, '.sticky')) })
    }
    case '/init': {
      const r = await run(['init'])
      startBridge()
      return json(r)
    }
    case '/rpc': {
      const { method, params } = await body(req)
      return json(await call(method, params))
    }
    case '/plan': {
      const { mcp } = await body(req)
      const args = ['setup', '--scope', 'user', ...(mcp ? ['--mcp', '--command', bin] : []), '--dry-run']
      return json({ command: 'stickypane ' + args.join(' '), ...(await run(args)) })
    }
    case '/events': {
      res.writeHead(200, { 'content-type': 'text/event-stream', 'cache-control': 'no-cache' })
      const send = (m) => res.write(`data: ${JSON.stringify(m)}\n\n`)
      listeners.add(send)
      req.on('close', () => listeners.delete(send))
      startBridge()
      return
    }
  }
  res.statusCode = 404
  res.end()
})

server.listen(port, '127.0.0.1', () => console.log(`devbridge on http://127.0.0.1:${port} for ${project}`))
const stop = () => {
  bridge?.stdin.end()
  server.close()
  process.exit(0)
}
process.on('SIGINT', stop)
process.on('SIGTERM', stop)
