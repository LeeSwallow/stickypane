// Dev only: the screens page over a real `stickypane bridge`, through
// desktop/scripts/devbridge.mjs (?bridge=/devbridge). The board,
// its notes, making notes and the settings go to the bridge as the shell
// would send them; where notes sit (the canvas) and the app's own settings
// stay in memory, as fakeProject keeps them. Nothing here ships.
import type { Bus } from '../services/board'
import type { Shape } from '../services/create'
import type { HomeApi } from '../services/home'
import type { CommandState, SetupApi } from '../services/install'
import type { ProjectApi } from '../services/project'
import { readThemes } from '../services/settings'
import type { Board } from '../board/model'
import { fakeProject } from './fakeProject'

interface Info {
  project: string
  version: string
  board: boolean
}

export function live(url: string) {
  const post = async <T>(path: string, v: unknown = {}): Promise<T> => {
    const r = await fetch(url + path, { method: 'POST', body: JSON.stringify(v) })
    return (await r.json()) as T
  }
  const info = async () => (await (await fetch(url + '/info')).json()) as Info
  async function rpc<T>(method: string, params?: unknown): Promise<T> {
    const m = await post<{ result?: T; error?: { message: string } }>('/rpc', { method, params })
    if (m.error) throw new Error(m.error.message)
    return m.result as T
  }

  function project(mode: string, lang: string): ProjectApi {
    const empty = { title: '', tabs: [], settings: {} } as unknown as Board
    const rest = fakeProject(empty, mode, lang)
    return {
      ...rest,
      async status() {
        const i = await info()
        return { window: 1, path: i.project, state: i.board ? 'ready' : 'no_board', pinned: false, version: i.version }
      },
      load: () => rpc('board.load'),
      act: (note, version, action) => rpc('note.act', { note, version, action }),
      arrange: (p) => rpc('note.arrange', p),
      order: (tab, order) => rpc('tab.order', { tab, order }),
      open: (note) => rpc('note.open', { note }),
      async makeBoard() {
        await post('/init')
      },
      archive: (note) => rpc('note.archive', { note }),
      trash: (note) => rpc('note.trash', { note }),
      restore: (note) => rpc('note.restore', { note }),
      move: (note, to) => rpc('note.move', { note, to }),
      run: (note) => rpc('note.run', { note }),
      send: (note, request) => rpc('note.send', { note, request }),
      async kinds() {
        const list = await rpc<Shape[] | null>('kinds.list')
        return (list ?? []).map((k) => ({ ...k, options: k.options ?? [] }))
      },
      create: (r) => rpc('note.create', r),
      themes: async () => readThemes(await rpc('themes.list')),
      settings: () => rpc('settings.get'),
      setSettings: (p) => rpc('settings.set', p),
    }
  }

  // the bridge's board.changed as the shell's event for window 1
  const bus: Bus = {
    on(name, fn) {
      if (name !== 'board.changed:1') return () => {}
      const es = new EventSource(url + '/events')
      es.onmessage = (e) => {
        const m = JSON.parse(e.data) as { method: string; params?: unknown }
        if (m.method === 'board.changed') fn({ data: m.params })
      }
      return () => es.close()
    },
  }

  // the home: the folder picked is the bridge's project folder; making a
  // board there runs the real init, then the page turns into its window
  function home(base: HomeApi, open: () => void): HomeApi {
    return {
      ...base,
      pickFolder: async () => (await info()).project,
      inspect: async () => {
        const i = await info()
        return { exists: i.board, path: i.project }
      },
      initProject: async () => {
        await post('/init')
        open()
      },
      openProject: async () => open(),
      status: async () => ({ found: true, bridge: true, version: (await info()).version }),
    }
  }

  // the terminal command over a made-up machine; what connecting would run is the real setup --dry-run
  function setup(): SetupApi {
    let state: CommandState = {
      bundled: '/Applications/Stickypane.app/Contents/Helpers/stickypane',
      version: '',
      links: [
        { target: 'user', path: '~/.local/bin/stickypane', state: 'none' },
        { target: 'system', path: '/usr/local/bin/stickypane', state: 'none' },
      ],
      first: null,
      shadow: null,
      userDir: '~/.local/bin',
      onPath: true,
      rcFile: '~/.zshrc',
      rcLine: 'export PATH="$HOME/.local/bin:$PATH"',
    }
    const link = (target: string, s: 'none' | 'apps') => (state = { ...state, links: state.links.map((l) => (l.target === target ? { ...l, state: s } : l)) })
    return {
      command: async () => ({ ...state, version: (await info()).version }),
      install: async (target) => ({ state: link(target, 'apps') }),
      uninstall: async () => ({ state: link('user', 'none') }),
      connectPlan: (mcp) => post('/plan', { mcp }),
      connect: async (mcp) => {
        const r = await post<{ command: string; output: string }>('/plan', { mcp })
        return { command: r.command.replace('--dry-run', '--yes'), output: '(dev page: setup ran with --dry-run)\n' + r.output }
      },
      initHere: async () => ({ command: 'stickypane init', output: '' }),
    }
  }

  return { project, bus, home, setup }
}
