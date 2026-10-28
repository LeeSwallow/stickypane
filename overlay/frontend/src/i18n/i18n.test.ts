import { describe, expect, it, vi } from 'vitest'
import { get } from 'svelte/store'
import ko from './ko.json'
import en from './en.json'
// the terminal's words: a create option's label is translated the same in both
import core from '../../../../internal/i18n/translations/ko.json'
import {
  t, setLanguage, language, translate, dayLabel, relativeTime, resolveAuto,
  chooseLanguage, applyChoice,
} from './index'

const now = () => get(language)

describe('catalogs', () => {
  it('every ko key exists in en, and the reverse', () => {
    expect(Object.keys(ko).sort()).toEqual(Object.keys(en).sort())
  })
  it('placeholders match between ko and en', () => {
    const ph = (v: string) => (v.match(/\{\w+\}/g) ?? []).sort()
    for (const k of Object.keys(en) as (keyof typeof en)[]) expect(ph(ko[k])).toEqual(ph(en[k]))
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
