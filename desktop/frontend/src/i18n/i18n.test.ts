import { describe, expect, it, vi } from 'vitest'
import { get } from 'svelte/store'
import ko from './ko.json'
import en from './en.json'
// the terminal's words: a create option's label is translated the same in both
import core from '../../../../internal/core/i18n/translations/ko.json'
import {
  t, setLanguage, language, translate, dayLabel, relativeTime, resolveAuto,
  chooseLanguage, applyChoice, boardLanguage, PLURAL_FORMS,
} from './index'
import { report } from '../services/errors'
import { failedKey, said } from './said'

const now = () => get(language)

describe('catalogs', () => {
  // a plural form (key.one) is a language's own: Korean has none
  const isForm = (k: string) => PLURAL_FORMS.some((f) => k.endsWith('.' + f)) && k.slice(0, k.lastIndexOf('.')) in en
  it('every ko key exists in en, and the reverse, but for plural forms', () => {
    expect(Object.keys(ko).sort()).toEqual(Object.keys(en).filter((k) => !isForm(k)).sort())
  })
  it('placeholders match between ko and en, and between a plural form and its key', () => {
    const ph = (v: string) => (v.match(/\{\w+\}/g) ?? []).sort()
    const cat = en as Record<string, string>
    for (const k of Object.keys(en)) {
      if (isForm(k)) expect(ph(cat[k]), k).toEqual(ph(cat[k.slice(0, k.lastIndexOf('.'))]))
      else expect(ph((ko as Record<string, string>)[k]), k).toEqual(ph(cat[k]))
    }
  })
  it('a plural form counts {n}', () => {
    for (const k of Object.keys(en).filter(isForm)) expect((en as Record<string, string>)[k], k).toContain('{n}')
  })
  it('uses the terminal terms: 삭제 and 되돌리기', () => {
    expect(ko['note.trash']).toBe('삭제')
    expect(ko['home.undo']).toBe('되돌리기')
    expect(ko['note.restore']).toBe('되돌리기')
  })
  it('a create option label reads as the terminal reads it', () => {
    const labels = (core as { labels: Record<string, string> }).labels
    const keys = Object.keys(ko).filter((k) => k.startsWith('label.'))
    expect(keys.length).toBeGreaterThan(10)
    for (const k of keys) {
      expect(labels[k.slice(6)], k).toBe(ko[k as keyof typeof ko])
      expect(en[k as keyof typeof en]).toBe(k.slice(6))
    }
  })
  it('no value is empty', () => {
    for (const v of [...Object.values(ko), ...Object.values(en)]) expect(v.length).toBeGreaterThan(0)
  })
})

describe('t', () => {
  it('translates and fills values', () => {
    setLanguage('en')
    expect(t('home.empty')).toBe(en['home.empty'])
    setLanguage('ko')
    expect(t('home.empty')).toBe(ko['home.empty'])
    expect(t('home.removed', { n: 2 })).toContain('2')
  })
  it('a count picks its plural form: 1 card, 2 cards, 카드 1장', () => {
    expect(translate({ en, ko }, 'en', 'kanban.count', { n: 1 })).toBe('1 card')
    expect(translate({ en, ko }, 'en', 'kanban.count', { n: 2 })).toBe('2 cards')
    expect(translate({ en, ko }, 'en', 'kanban.count', { n: 0 })).toBe('0 cards')
    expect(translate({ en, ko }, 'ko', 'kanban.count', { n: 1 })).toBe('카드 1장')
    expect(translate({ en, ko }, 'en', 'home.notes', { n: 1 })).toBe('1 note')
    expect(translate({ en, ko }, 'ko', 'home.notes', { n: 3 })).toBe('노트 3장')
    // a language without the key takes English's form
    const cats = { en: { c: '{n} things', 'c.one': '{n} thing' }, ko: {} as Record<string, string> }
    expect(translate(cats, 'ko', 'c', { n: 1 })).toBe('1 things')
    expect(translate(cats, 'en', 'c', { n: 1 })).toBe('1 thing')
  })
  it('missing key falls back to English, then to the key', () => {
    const cats = { en: { a: 'A {x}' }, ko: {} as Record<string, string> }
    expect(translate(cats, 'ko', 'a', { x: 1 })).toBe('A 1')
    expect(translate(cats, 'ko', 'zzz')).toBe('zzz')
  })
  it('setLanguage re-renders: subscribers hear it', () => {
    const seen: string[] = []
    setLanguage('ko')
    const off = language.subscribe((l) => seen.push(l))
    setLanguage('en')
    setLanguage('ko')
    off()
    expect(seen).toEqual(['ko', 'en', 'ko'])
  })
})

describe('Intl labels', () => {
  const today = new Date(2026, 9, 28, 12)
  it('day labels say 오늘/어제 and Today/Yesterday', () => {
    expect(dayLabel('2026-10-28', today, 'ko')).toBe('오늘')
    expect(dayLabel('2026-10-27', today, 'ko')).toBe('어제')
    expect(dayLabel('2026-10-28', today, 'en')).toBe('Today')
    expect(dayLabel('2026-10-27', today, 'en')).toBe('Yesterday')
    expect(dayLabel('2026-10-25', today, 'ko')).toBe('10월 25일')
    expect(dayLabel('2026-10-25', today, 'en')).toBe('Oct 25')
  })
  it('relative time 2분 전 / 2m ago', () => {
    const n = new Date(2026, 9, 28, 12, 0, 0)
    const d = new Date(n.getTime() - 2 * 60_000)
    expect(relativeTime(d, n, 'ko')).toBe('2분 전')
    expect(relativeTime(d, n, 'en')).toBe('2m ago')
    const h = new Date(n.getTime() - 3 * 3_600_000)
    expect(relativeTime(h, n, 'en')).toBe('3h ago')
    expect(relativeTime(h, n, 'ko')).toBe('3시간 전')
    expect(relativeTime(new Date(n.getTime() - 10_000), n, 'en')).toBe('now')
  })
})

describe('choice', () => {
  it('auto resolves from the system language', () => {
    expect(resolveAuto('ko-KR')).toBe('ko')
    expect(resolveAuto('en-US')).toBe('en')
    expect(resolveAuto('fr')).toBe('en')
  })
  it('applyChoice uses the bridge-resolved value', () => {
    applyChoice({ choice: 'auto', resolved: 'ko' })
    expect(now()).toBe('ko')
    applyChoice({ choice: 'en', resolved: 'en' })
    expect(now()).toBe('en')
  })
  it('changing language in a project window calls settings.set language', async () => {
    setLanguage('en')
    const set = vi.fn().mockResolvedValue({})
    const get = vi.fn().mockResolvedValue({ language: { choice: 'ko', resolved: 'ko' } })
    await chooseLanguage('ko', { kind: 'project', settingsSet: set, settingsGet: get })
    expect(set).toHaveBeenCalledWith('language', 'ko')
    expect(now()).toBe('ko')
    // auto: the bridge's resolved value wins over the browser's
    get.mockResolvedValue({ language: { choice: 'auto', resolved: 'en' } })
    await chooseLanguage('auto', { kind: 'project', settingsSet: set, settingsGet: get })
    expect(now()).toBe('en')
  })
  it('a board with no language of its own follows the app: choosing auto there draws in the app language', async () => {
    setLanguage('en')
    const set = vi.fn().mockResolvedValue({})
    const get = vi.fn().mockResolvedValue({ language: { choice: 'auto', resolved: 'en' } })
    await chooseLanguage('auto', { kind: 'project', settingsSet: set, settingsGet: get, app: 'ko' })
    expect(set).toHaveBeenCalledWith('language', 'auto')
    expect(now()).toBe('ko')
  })
  it('boardLanguage: the board own, else the app, else what the bridge resolved', () => {
    expect(boardLanguage({ choice: 'en', resolved: 'en' }, 'ko')).toBe('en')
    expect(boardLanguage({ choice: 'ko', resolved: 'ko' }, 'en')).toBe('ko')
    expect(boardLanguage({ choice: 'auto', resolved: 'en' }, 'ko')).toBe('ko')
    expect(boardLanguage({ choice: 'auto', resolved: 'ko' }, '')).toBe('ko')
    expect(boardLanguage({ choice: 'auto', resolved: 'en' }, undefined)).toBe('en')
    expect(boardLanguage(undefined, 'ko')).toBe('ko')
    expect(boardLanguage(null, '')).toBe('en')
  })
  it('invalid day strings come back as they are', () => {
    expect(dayLabel('nope', new Date(2026, 9, 28), 'en')).toBe('nope')
  })
  it('in the home window it saves to overlay.json instead', async () => {
    const save = vi.fn().mockResolvedValue(undefined)
    await chooseLanguage('en', { kind: 'home', save })
    expect(save).toHaveBeenCalledWith('en')
    expect(now()).toBe('en')
    await chooseLanguage('auto', { kind: 'home', save, systemLanguage: 'ko-KR' })
    expect(save).toHaveBeenCalledWith('auto')
    expect(now()).toBe('ko')
  })
})

describe('errors in the window\'s language', () => {
  const coded = (code: string, values?: Record<string, string>) =>
    Object.assign(new Error('the shell\'s English'), { cause: { code, values } })
  it('an error the shell knows is said in the window\'s language', () => {
    setLanguage('ko')
    expect(report('test', coded('host_taken', { name: 'box' }))).toBe("'box' 이름은 다른 호스트가 쓰고 있어요.")
    setLanguage('en')
    expect(report('test', coded('host_taken', { name: 'box' }))).toBe('Another host is already named box.')
  })
  it('an unknown code, or none, keeps the words it came in', () => {
    setLanguage('ko')
    expect(report('test', coded('nobody_knows'))).toBe("the shell's English")
    expect(report('test', new Error('exit status 1'))).toBe('exit status 1')
  })
  it('every code the shell sends has its words', () => {
    // desktop/internal/shell/errors.go and remote/remote.go
    const codes = ['no_board', 'remote', 'unsafe_folder', 'no_host', 'host_taken', 'host_open', 'no_autostart', 'index',
      'no_stickypane', 'old_stickypane', 'no_project', 'host_name', 'no_target', 'target_dash', 'target_word',
      'command_line', 'folder_absolute', 'folder_line']
    for (const c of codes) expect(Object.keys(en), c).toContain('error.' + c)
  })
  it('a failure says what could not be done, then why', () => {
    setLanguage('ko')
    expect(said('board.pin', coded('no_board'))).toBe('창을 맨 위에 두지 못했어요 · 이 창의 보드가 멈췄어요. 창을 닫고 보드를 다시 열어 주세요.')
    setLanguage('en')
    expect(said('torn.refresh', new Error('stale version'))).toBe('Could not read the board · stale version')
    expect(failedKey('board.resetSizes.undo')).toBe('failed.layout')
    expect(failedKey('board.somethingNew')).toBe('failed.other')
  })
})

describe('auto, as the shell reads it', () => {
  it('a window told ?syslang= speaks what the tray and the bridge speak', () => {
    const was = globalThis.location
    Object.defineProperty(globalThis, 'location', { value: { search: '?window=settings&syslang=ko' }, configurable: true })
    try {
      expect(resolveAuto()).toBe('ko')
      expect(resolveAuto('en-US')).toBe('en')
    } finally {
      Object.defineProperty(globalThis, 'location', { value: was, configurable: true })
    }
  })
})
