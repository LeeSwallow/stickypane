// Dev only: a project service over made-up data, for the screens page. Its
// shapes are kinds.list answers saved from a real bridge (kinds.json, and
// kinds-ko.json for {"language": "ko"});
// note.create refuses a few values the way the bridge does, so the create
// form's errors can be seen.
import type { Board } from '../board/model'
import type { BoardApi } from '../services/board'
import { layoutOf, type Layout } from '../services/canvas'
import type { Shape } from '../services/create'
import type { ProjectApi } from '../services/project'
import kinds from './kinds.json'
import kindsKo from './kinds-ko.json'
import { THEMES } from './themes'

// The layout is kept in memory: a note torn off here shows on the shelf;
// note is the note a torn-off window's status names.
export function fakeProject(board: Board, mode: string, lang: string, start: Partial<Layout> = {}, note = ''): ProjectApi {
  const status: BoardApi['status'] = async () => ({ window: 1, path: '/Users/min/Projects/Personal/stickypane', state: 'ready', pinned: true, version: '0.4.0', note })
  let prefs = { hotkey: 'Option+Space', language: lang, mode }
  let layout = layoutOf(start)
  return {
    status,
    load: async () => board,
    act: async () => {},
    arrange: async () => {},
    order: async () => {},
    open: async () => {},
    pin: async () => {},
    makeBoard: async () => {},
    prefs: async () => prefs,
    archive: async () => {},
    trash: async () => {},
    restore: async () => {},
    move: async (note) => ({ note }),
    run: async () => ({ code: 0, took: 120 }),
    send: async () => ({}),
    kinds: async (l) => (l === 'ko' ? kindsKo : kinds) as unknown as Shape[],
    async create(r) {
      const v = r.options.view
      if (r.kind === 'chart' && typeof v === 'string' && !['bar', 'spark', 'heat'].includes(v))
        throw new Error(`bad option: view of a chart: "${v}" is not one of bar, spark, heat`)
      const s = r.options.status
      if (r.kind === 'api' && typeof s === 'string') throw new Error(`bad option: status of a api: "${s}" is not a number`)
      return { note: `${r.tab ? r.tab + '/' : ''}${r.title || 'note'}.md` }
    },
    themes: async () => THEMES,
    settings: async () => board.settings,
    setSettings: async () => ({}),
    appPrefs: async () => prefs,
    setAppPrefs: async (p) => void (prefs = p),
    layout: async () => layout,
    setMode: async (m) => void (layout = { ...layout, mode: m }),
    setSnap: async (on) => void (layout = { ...layout, snap: on }),
    place: async (places) => void (layout = { ...layout, places: { ...layout.places, ...places } }),
    tearOff: async (n) => void (layout = { ...layout, torn: { ...layout.torn, [n]: { screenX: 0, screenY: 0, w: 360, h: 420, pinned: true } } }),
    async putBack(n) {
      const torn = { ...layout.torn }
      delete torn[n ?? note]
      layout = { ...layout, torn }
    },
    onLayout: () => () => {},
  }
}
