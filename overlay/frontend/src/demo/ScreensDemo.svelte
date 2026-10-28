<script lang="ts">
  // Dev only (?demo=screens&screen=home|empty|board|canvas|torn|zoom|create
  // |settings&shape=board|form|chart…&mode=dark|light):
  // the real Home and Board screens over made-up data, so they can be seen
  // in a browser without the shell. With &bridge=<devbridge url>
  // (overlay/scripts/devbridge.mjs) the board is a real bridge's, and the
  // home's folder is its project: making a board there runs the real init
  // and turns the page into the board's window. Nothing here ships.
  import Board from '../board/Board.svelte'
  import { fakeProject } from './fakeProject'
  import type { Board as BoardData, Note } from '../board/model'
  import Home from '../home/Home.svelte'
  import TornNote from '../torn/TornNote.svelte'
  import type { Layout } from '../services/canvas'
  import type { Groups, HomeApi, Row } from '../services/home'
  import { THEMES } from './themes'
  import { live } from './live'

  const q = new URLSearchParams(location.search)
  const screen = q.get('screen') ?? 'home'
  const mode = q.get('mode') === 'light' ? 'light' : 'dark'
  const lang = q.get('lang') === 'en' ? 'en' : 'ko'
  const style = q.get('style') === 'neat' ? 'neat' : 'doodle'
  const bridge = q.get('bridge')
  const real = bridge ? live(bridge) : null

  const NOW = new Date()
  const ago = (min: number) => new Date(NOW.getTime() - min * 60_000).toISOString()

  // --- home ----------------------------------------------------------------
  const row = (id: number, title: string, min: number, extra: Partial<Row> = {}): Row => ({
    ID: id,
    Path: `/Users/min/Projects/Personal/${title}`,
    Title: title,
    LastAt: ago(min),
    Agents: null,
    Live: null,
    ...extra,
  })
  const day = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  const today = day(NOW)
  const yesterday = day(new Date(NOW.getTime() - 86_400_000))
  const groups: Groups =
    screen === 'empty'
      ? { Active: null, Days: null, Removed: null }
      : {
          Active: [
            row(1, 'stickypane', 2, { Agents: ['claude', 'codex'], Live: [{ PID: 1, Kind: 'tui', Started: ago(90) }] }),
            row(2, 'keyharbor', 12, { Live: [{ PID: 2, Kind: 'tui', Started: ago(40) }] }),
          ],
          Days: [
            { Date: today, Rows: [row(3, 'ts-voca', 180)] },
            { Date: yesterday, Rows: [row(4, 'escape-room', 1500, { Agents: ['sam'] })] },
          ],
          Removed: [{ ID: 9, Path: '/Users/min/old/thing', Title: 'thing' }],
        }
  const fakeHome: HomeApi = {
    ensure: async () => '/Users/min/.stickypane',
    sessions: async () => groups,
    restore: async () => {},
    pickFolder: async () => '',
    inspect: async (dir: string) => ({ exists: false, path: dir }),
    initProject: async () => {},
    openProject: async () => 1,
    status: async () => ({ found: true, bridge: true, version: '0.2.0' }),
    notices: async () => (screen === 'home' ? [{ code: 'stayInTray', detail: '' }] : []),
    prefs: async () => ({ hotkey: 'alt+space', language: lang, mode }),
    setPrefs: async () => {},
    onNotices: () => () => {},
    copy: async () => {},
  }
  // the page as the board's window: what the shell does when a board opens
  const toBoard = () => {
    const next = new URLSearchParams(location.search)
    next.set('screen', 'board')
    location.search = next.toString()
  }
  const homeApi = real ? real.home(fakeHome, toBoard) : fakeHome

  // --- board ---------------------------------------------------------------
  const base = { tab: 'work', icon: '', open: true, rows: 0, color: 'yellow', pinned: false, version: 'v1', summary: '', modTime: ago(30), linked: false }
  const notes: Note[] = [
    {
      ...base, name: 'work/release.md', title: '릴리스 체크 — v0.2', kind: 'checklist', icon: '☑', size: '1/3', pinned: true, summary: '1/3',
      data: { items: [{ text: 'write tests', checked: true, done: ago(50) }, { text: 'update README', checked: false }, { text: 'tag 0.2.0', checked: false }], prose: [], done: 1, total: 3 },
    },
    {
      ...base, name: 'work/auth.md', title: 'Auth sprint', kind: 'board', icon: '▦', size: '2/3', color: 'green', summary: '4 cards', modTime: ago(1),
      data: {
        desc: [],
        columns: [
          { title: 'Todo', cards: [{ text: 'session expiry', who: ['sam'], detail: [], stalled: false }] },
          { title: 'Doing', cards: [{ text: 'login form', who: ['claude'], moved: ago(4), detail: [], stalled: false }, { text: 'refresh token', who: ['codex'], moved: ago(28), detail: [], stalled: true }] },
          { title: 'Done', cards: [{ text: 'schema', who: ['sam'], moved: ago(60), detail: [], stalled: false }] },
        ],
      },
    },
    {
      ...base, name: 'work/notes.md', title: 'Design notes', kind: 'note', icon: '✎', size: '1/2', color: 'blue', summary: '12 lines', linked: true, target: '/Users/min/notes/design.md',
      data: { markdown: '## Grid\n\nNotes take a **fraction** of the board: `1/4` … `1`.\n\n- quarters and thirds share a row\n- a row is always full\n\n> the terminal tiles the same way' },
    },
    {
      ...base, name: 'work/todo.md', title: 'Small things', kind: 'checklist', icon: '☑', size: '1/4', color: 'pink', summary: '0/2',
      data: { items: [{ text: 'icons in neat', checked: false }, { text: 'tray word', checked: false }], prose: [], done: 0, total: 2 },
    },
    {
      ...base, name: 'work/ideas.md', title: 'Ideas', kind: 'note', icon: '✎', size: '1/4', color: 'purple', summary: '3 lines',
      data: { markdown: 'A canvas mode, notes torn off onto the desktop.' },
    },
    { ...base, name: 'work/old.md', title: 'Old plan', kind: 'note', icon: '✎', size: '1/2', open: false, data: { markdown: '' } },
  ] as unknown as Note[]
  const palette = THEMES.find((t) => t.name === (mode === 'dark' ? 'stickypane-dark' : 'stickypane-light'))
  const board: BoardData = {
    title: 'stickypane',
    tabs: [
      { name: 'work', title: 'work', notes },
      { name: 'notes', title: 'notes', notes: [] },
      { name: 'log', title: 'log', notes: [] },
    ],
    settings: { theme: palette?.name ?? '', style, view: 'overlay', language: { choice: lang, resolved: lang }, tab: 'work', time: '', editor: '', stale: '2h' },
    palette,
  }
  // board: the grid; canvas: notes where they were left, one torn off;
  // torn: that note in its own window
  const canvasLayout: Partial<Layout> = {
    mode: 'canvas',
    places: {
      'work/release.md': { x: 32, y: 40, w: 300, h: 230, tilt: -1.5, z: 3 },
      'work/auth.md': { x: 360, y: 24, w: 470, h: 300, tilt: 0.5, z: 2 },
      'work/notes.md': { x: 96, y: 300, w: 420, h: 290, tilt: 1, z: 4 },
      'work/todo.md': { x: 560, y: 352, w: 260, h: 170, tilt: -1, z: 5 },
    },
    torn: { 'work/ideas.md': { screenX: 1200, screenY: 80, w: 320, h: 260, pinned: true } },
  }
  const layout = screen === 'canvas' || screen === 'torn' ? canvasLayout : { mode: 'grid' as const }
  const boardApi = real ? real.project(mode, lang) : fakeProject(board, mode, lang, layout, screen === 'torn' ? (q.get('note') ?? 'work/release.md') : '')
  const quietBus = real ? real.bus : { on: () => () => {} }
  const setup = real?.setup()
</script>

{#if screen === 'torn'}
  <TornNote api={boardApi} bus={quietBus} note={q.get('note') ?? 'work/release.md'} />
{:else if screen === 'board' || screen === 'canvas' || screen === 'zoom' || screen === 'create' || screen === 'settings'}
  <Board
    api={boardApi}
    zoom={screen === 'zoom' ? 'work/auth.md' : undefined}
    create={screen === 'create' ? (q.get('shape') ?? 'board') : undefined}
    settings={screen === 'settings'}
    bus={quietBus}
    {setup}
  />
{:else}
  <Home api={homeApi} {setup} />
{/if}
