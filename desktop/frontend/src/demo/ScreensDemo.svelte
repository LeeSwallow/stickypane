<script lang="ts">
  // Dev only (?demo=screens&screen=home|home-list|empty|home-empty|board
  // |board-first|board-empty|canvas|torn|zoom|create|settings-board
  // |settings-app|settings-cli|settings-about|settings-none
  // &shape=board|form|chart…&mode=dark|light&lang=ko|en):
  // the real Home and Board screens over made-up data, so they can be seen
  // in a browser without the shell. With &bridge=<devbridge url>
  // (desktop/scripts/devbridge.mjs) the board is a real bridge's, and the
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
  import SettingsWindow from '../settings/SettingsWindow.svelte'
  import type { AppApi, AppPrefs } from '../services/settings'
  import type { CommandState, SetupApi } from '../services/install'

  const q = new URLSearchParams(location.search)
  const asked = q.get('screen') ?? 'home'
  // the names the redesign's captures use (#63)
  const screen = asked === 'home-list' ? 'home' : asked === 'home-empty' ? 'empty' : asked
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
    peeks: async () => ({
      '/Users/min/Projects/Personal/stickypane': { notes: 6, stalled: 1, agents: ['claude', 'codex'], colors: THEMES[mode === 'dark' ? 2 : 3].notes, theme: 'catppuccin-mocha' },
      '/Users/min/Projects/Personal/keyharbor': { notes: 3, colors: THEMES[mode === 'dark' ? 4 : 5].notes },
      '/Users/min/Projects/Personal/escape-room': { notes: 9, agents: ['sam'], colors: THEMES[mode === 'dark' ? 8 : 7].notes },
    }),
    openSettings: async (section: string) => {
      const next = new URLSearchParams(location.search)
      next.set('screen', 'settings-none')
      next.set('section', section)
      location.search = next.toString()
    },
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
  const welcome = {
    ...base, tab: '', name: 'welcome.md', title: lang === 'ko' ? 'stickypane에 오신 걸 환영해요' : 'Welcome to stickypane', kind: 'note', icon: '✎', size: '2/3', pinned: true, modTime: ago(0),
    data: {
      markdown: lang === 'ko'
        ? '에이전트는 `.sticky/`에 파일을 써서 여기에 노트를 붙여요. 체크하고,\n답하고, 카드를 옮기는 건 여러분이 하고, 에이전트는 그 결과를 읽어요.\n\n- **앱에서:** **+ 새 노트**로 노트를 만들고, **×** 버튼으로 접어 둬요.\n  설정(**⌘,**)의 **터미널과 에이전트**에서 Claude Code와 Codex를 연결해요.\n- **터미널에서:** **tab**으로 노트를 오가고, **enter**로 열고 키워요.\n  **N**은 빠른 메모, **a**는 다른 모양의 노트, **o**는 접기예요.\n  모든 키는 **?** 키로 봐요.\n\n다 읽었으면 이 노트를 접어 두세요.'
        : 'Your agent sticks notes here by writing files in `.sticky/`. You tick,\nanswer and move cards; the agent reads what you did.\n\n- **In the app:** **+ New note** makes one, **×** folds a note away, and\n  **Terminal and agents** in Settings (**⌘,**) connects Claude Code and Codex.\n- **In the terminal:** **tab** moves between notes, **enter** opens one and\n  zooms in, **N** jots a note, **a** adds another shape, **o** folds a note\n  away, **?** lists every key.\n\nFold this note away when you are done with it.',
    },
  } as unknown as Note
  const palette = THEMES.find((t) => t.name === (mode === 'dark' ? 'stickypane-dark' : 'stickypane-light'))
  const tabs =
    screen === 'board-first'
      ? [{ name: '', title: 'my-app', notes: [welcome] }]
      : screen === 'board-empty'
        ? [{ name: '', title: 'my-app', notes: [] }]
        : [
            { name: 'work', title: 'work', notes },
            { name: 'notes', title: 'notes', notes: [] },
            { name: 'log', title: 'log', notes: [] },
          ]
  const board: BoardData = {
    title: screen === 'board-first' || screen === 'board-empty' ? 'my-app' : 'stickypane',
    tabs,
    settings: { theme: palette?.name ?? '', style, view: 'overlay', language: { choice: lang, resolved: lang }, tab: tabs[0].name, time: '', editor: '', stale: '2h' },
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
  const layout = screen === 'canvas' || screen === 'torn' ? canvasLayout : screen === 'board-first' || screen === 'board-empty' ? { mode: 'canvas' as const } : { mode: 'grid' as const }
  const own = screen === 'board-first' || screen === 'board-empty'
  const boardApi0 = real ? real.project(mode, lang) : fakeProject(board, mode, lang, layout, screen === 'torn' ? (q.get('note') ?? 'work/release.md') : '')
  const boardApi = own && !real ? { ...boardApi0, status: async () => ({ ...(await boardApi0.status()), path: '/Users/min/Projects/my-app', pinned: false }) } : boardApi0
  const quietBus = real ? real.bus : { on: () => () => {} }
  const setup = real?.setup()

  // --- settings --------------------------------------------------------------
  let appPrefs: AppPrefs = { hotkey: 'Option+Space', language: lang, mode, pinNew: false }
  const fakeApp: AppApi = {
    prefs: async () => appPrefs,
    setPrefs: async (p) => void (appPrefs = p),
    onPrefs: () => () => {},
    async checkHotkey(accel) {
      if (/^(Cmd\+Space|Cmd\+Q|Cmd\+W|Ctrl\+Space)$/.test(accel)) return { ok: false, accel, code: 'reserved' }
      if (!/(Cmd|Ctrl|Option)\+/.test(accel)) return { ok: false, accel, code: 'modifier' }
      if (accel === 'Ctrl+Option+K') return { ok: false, accel, code: 'taken' }
      return { ok: true, accel }
    },
    autostart: async () => ({ supported: true, enabled: false }),
    setAutostart: async (on) => ({ supported: true, enabled: on }),
    about: async () => ({ app: '0.1.0', stickypane: { found: true, bridge: true, path: '/Applications/stickypane.app/Contents/Helpers/stickypane', version: '0.4.0' } }),
    onBoard: () => () => {},
    openURL: async (u) => void window.open(u),
  }
  const cmdState: CommandState = {
    bundled: '/Applications/stickypane.app/Contents/Helpers/stickypane',
    version: '0.4.0',
    links: [
      { target: 'user', path: '/Users/min/.local/bin/stickypane', state: 'apps' },
      { target: 'system', path: '/usr/local/bin/stickypane', state: 'none' },
    ],
    first: { path: '/opt/homebrew/bin/stickypane', real: '/opt/homebrew/bin/stickypane', version: '0.2.1', apps: false },
    shadow: { kind: 'brew', command: 'brew uninstall stickypane' },
    userDir: '/Users/min/.local/bin',
    onPath: true,
    rcFile: '/Users/min/.zshrc',
    rcLine: 'export PATH="$HOME/.local/bin:$PATH"',
  }
  const fakeSetup: SetupApi = {
    command: async () => (screen === 'empty' ? { ...cmdState, links: cmdState.links.map((l) => ({ ...l, state: 'none' as const })), first: null, shadow: null } : cmdState),
    install: async () => ({ state: cmdState }),
    uninstall: async () => ({ state: cmdState }),
    connectPlan: async (mcp) => ({ command: 'stickypane setup --dry-run', output: 'stickypane setup runs:\n  claude plugin marketplace add LeeSwallow/stickypane\n  claude plugin install board@stickypane --scope user' + (mcp ? '\n  claude mcp add stickypane -- stickypane mcp' : '') }),
    connect: async () => ({ command: 'stickypane setup --yes', output: 'done' }),
    initHere: async () => ({ command: 'stickypane init', output: '' }),
  }
  const settingsSection = screen.startsWith('settings-') ? screen.slice(9) : ''
  const noBoard = settingsSection === 'none'
  const settingsBoard = noBoard ? { ...boardApi, status: async () => ({ window: 2, path: '', state: 'closed', pinned: false }) } : boardApi
</script>

{#if settingsSection}
  <SettingsWindow
    board={settingsBoard}
    app={fakeApp}
    setup={setup ?? fakeSetup}
    section={(noBoard ? (q.get('section') ?? 'board') : settingsSection) as 'board' | 'app' | 'cli' | 'about'}
    chrome={(q.get('chrome') as 'mac' | 'own' | null) ?? 'mac'}
  />
{:else if screen === 'torn'}
  <TornNote api={boardApi} bus={quietBus} note={q.get('note') ?? 'work/release.md'} />
{:else if screen === 'board' || screen === 'board-first' || screen === 'board-empty' || screen === 'canvas' || screen === 'zoom' || screen === 'create'}
  <Board
    api={boardApi}
    zoom={screen === 'zoom' ? 'work/auth.md' : undefined}
    create={screen === 'create' ? (q.get('shape') ?? 'board') : undefined}
    bus={quietBus}
  />
{:else}
  <Home api={homeApi} setup={setup ?? fakeSetup} onPrefs={() => () => {}} />
{/if}
