// Dev only: a project service over made-up data, for the screens page. Its
// shapes are a kinds.list answer saved from a real bridge (kinds.json);
// note.create refuses a few values the way the bridge does, so the create
// form's errors can be seen.
import type { Board } from '../board/model'
import type { BoardApi } from '../services/board'
import type { Shape } from '../services/create'
import type { ProjectApi } from '../services/project'
import kinds from './kinds.json'
import { THEMES } from './themes'

export function fakeProject(board: Board, mode: string, lang: string): ProjectApi {
  const status: BoardApi['status'] = async () => ({ window: 1, path: '/Users/min/Projects/Personal/stickypane', state: 'ready', pinned: true, version: '0.4.0' })
  let prefs = { hotkey: 'Option+Space', language: lang, mode }
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
    kinds: async () => kinds as unknown as Shape[],
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
  }
}
