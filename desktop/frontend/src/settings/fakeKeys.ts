// Tests and demos only: a KeysApi over a made-up keys.json. It refuses
// what the board would: a key another action has, a word that is no key.
import type { Binding, Keys, KeysApi } from '../services/keys'

const ACTIONS: { name: string; keys: string[]; en: string; ko: string; group: string }[] = [
  { name: 'focus-left', keys: ['h'], en: 'note to the left', ko: '왼쪽 노트', group: 'move' },
  { name: 'focus-down', keys: ['j'], en: 'note below', ko: '아래 노트', group: 'move' },
  { name: 'focus-first', keys: ['g g'], en: 'first note', ko: '첫 노트', group: 'move' },
  { name: 'jot', keys: ['N', 'n'], en: 'jot a note', ko: '노트 쓰기', group: 'notes' },
  { name: 'note-archive', keys: ['x'], en: 'move to archive', ko: '보관함으로', group: 'notes' },
  { name: 'note-delete', keys: ['d d'], en: 'delete (to .trash)', ko: '삭제 (.trash로)', group: 'notes' },
  { name: 'note-wider', keys: ['>', '+'], en: 'wider', ko: '넓게', group: 'size' },
  { name: 'scroll-half-down', keys: ['ctrl+d'], en: 'half a page down', ko: '반 페이지 아래로', group: 'scroll' },
  { name: 'page-prev', keys: [','], en: 'previous page', ko: '이전 페이지', group: 'scroll' },
  { name: 'tab-next', keys: ['g t'], en: 'next tab', ko: '다음 탭', group: 'tabs' },
  { name: 'theme-next', keys: ['T'], en: 'next theme', ko: '다음 테마', group: 'board' },
  { name: 'quit', keys: ['q'], en: 'quit', ko: '끝내기', group: 'board' },
]

export function fakeKeys(file: Record<string, string[]> = {}, over: Partial<KeysApi> = {}) {
  const state = { file: { ...file }, calls: [] as string[] }
  const list = (lang: string): Keys => ({
    bindings: ACTIONS.map((a): Binding => {
      const changed = a.name in state.file
      return { name: a.name, keys: changed ? state.file[a.name] : a.keys, words: lang === 'ko' ? a.ko : a.en, changed, defaults: a.keys, group: a.group }
    }),
  })
  const api: KeysApi = {
    keys: async (lang) => {
      state.calls.push('keys ' + lang)
      return list(lang)
    },
    setKeys: async (lang, action, keys) => {
      state.calls.push(`set ${action} ${JSON.stringify(keys)}`)
      for (const k of keys) {
        if (/^[a-z]{2,}$/.test(k) && !['space', 'enter', 'tab', 'esc', 'up', 'down', 'pgup', 'pgdown'].includes(k)) {
          throw new Error(`${action}: "${k}": "${k}" is not a key the terminal names`)
        }
        const other = list(lang).bindings.find((b) => b.name !== action && b.keys.includes(k))
        if (other) throw new Error(`${action}: ${k} is ${other.name}'s key too; keys.json is left as it was`)
      }
      state.file[action] = keys
      return list(lang)
    },
    resetKeys: async (lang, action) => {
      state.calls.push('reset ' + action)
      if (action) delete state.file[action]
      else state.file = {}
      return list(lang)
    },
    ...over,
  }
  return { api, state }
}
