// i18n: Korean and English (03 §3.9). English is the base; a key missing
// from Korean shows in English. Notes are never translated.
import { derived, get, writable } from 'svelte/store'
import en from './en.json'
import ko from './ko.json'
import { logged, setErrorWords } from '../services/errors'

export type Lang = 'ko' | 'en'
export type Choice = 'auto' | 'ko' | 'en'
export type Catalogs = { en: Record<string, string>; ko: Record<string, string> }
export type Values = Record<string, unknown>

const catalogs: Catalogs = { en, ko }

const langStore = writable<Lang>('en')

/** language is the language being drawn; subscribe to re-render. */
export const language = { subscribe: langStore.subscribe }

/** setLanguage switches the language at once, with no restart. */
export function setLanguage(code: Lang): void {
  const lang = code === 'ko' ? 'ko' : 'en'
  langStore.set(lang)
  // the page says its language: screen readers, hyphens, and the Korean
  // type rules in app.css (no breaks inside a word, the handwriting set closer)
  if (typeof document !== 'undefined') document.documentElement.lang = lang
}

/**
 * translate looks key up in lang, then in English, then is the key. A
 * count {n} picks the plural form a language has for it: key.one, and so
 * on (Intl.PluralRules), where the catalog has one — "1 card", "2 cards";
 * Korean has one form for every count (카드 1장, 카드 2장).
 */
export function translate(cats: Catalogs, lang: Lang, key: string, values?: Values): string {
  const n = values?.n
  const form = typeof n === 'number' ? `${key}.${plural(lang).select(n)}` : undefined
  const s = (form && cats[lang][form]) || cats[lang][key] || (form && cats.en[form]) || cats.en[key] || key
  return values ? s.replace(/\{(\w+)\}/g, (m, k: string) => (k in values ? String(values[k]) : m)) : s
}

const plurals: Partial<Record<Lang, Intl.PluralRules>> = {}
const plural = (lang: Lang) => (plurals[lang] ??= new Intl.PluralRules(localeOf(lang)))

/** PLURAL_FORMS are the endings a plural form of a key may have (Intl.PluralRules' categories). */
export const PLURAL_FORMS = ['zero', 'one', 'two', 'few', 'many'] as const

/** t is the text for key in the current language. */
export function t(key: string, values?: Values): string {
  return translate(catalogs, get(langStore), key, values)
}

/** tr is t as a store, for components: {$tr('home.empty')}. */
export const tr = derived(langStore, (lang) => (key: string, values?: Values) =>
  translate(catalogs, lang, key, values),
)

/**
 * resolveAuto is the language "auto" comes to for a system locale: by
 * default the one the shell read as the terminal reads it (?syslang=, the
 * tray's and the bridge's too), else the browser's.
 */
export function resolveAuto(systemLanguage?: string): Lang {
  const l = systemLanguage ?? shellLanguage() ?? (typeof navigator !== 'undefined' ? navigator.language : 'en')
  return l.toLowerCase().startsWith('ko') ? 'ko' : 'en'
}

function shellLanguage(): string | undefined {
  if (typeof location === 'undefined') return undefined
  return new URLSearchParams(location.search).get('syslang') || undefined
}

/** The bridge's settings.language. */
export type BridgeLanguage = { choice: Choice; resolved: Lang }

/** applyChoice draws in what the bridge resolved the choice to. */
export function applyChoice(l: BridgeLanguage): void {
  setLanguage(l.resolved)
}

/**
 * boardLanguage is what a board's window speaks (08 §8.2.1): the board's
 * own language when it has one (ko or en in its sticky.json); otherwise
 * the app's, which the home window sets for this computer; and when the
 * app has none either, what the bridge made of the system's.
 */
export function boardLanguage(board: { choice?: string; resolved?: string } | null | undefined, app: string | undefined): Lang {
  const c = board?.choice
  if (c === 'ko' || c === 'en') return c
  if (app === 'ko' || app === 'en') return app
  return board?.resolved === 'ko' ? 'ko' : 'en'
}

/**
 * Where a language choice is kept. A project window writes the board's
 * sticky.json through the bridge (shared with the terminal); the home
 * window writes ~/.stickypane/overlay.json through the shell.
 */
export type LanguageHost =
  | {
      kind: 'project'
      settingsSet: (key: 'language', value: Choice) => Promise<unknown>
      // settings.get: its language.resolved is what the bridge made of the choice
      settingsGet: () => Promise<{ language?: Partial<BridgeLanguage> } | null | undefined>
      /** the app's language (overlay.json): what a board with none follows */
      app?: string
    }
  | { kind: 'home'; save: (choice: Choice) => Promise<void>; systemLanguage?: string }

/** chooseLanguage stores the choice and re-renders at once. */
export async function chooseLanguage(choice: Choice, host: LanguageHost): Promise<void> {
  if (host.kind === 'project') {
    // settings.set answers {}; the resolved value comes from settings.get,
    // because the bridge resolves "auto" in the terminal's order
    // (STICKYPANE_LANG, LC_ALL, ..., system). navigator.language is only the
    // fallback when the bridge gives nothing.
    await host.settingsSet('language', choice)
    if (choice === 'auto' && (host.app === 'ko' || host.app === 'en')) {
      // a board with no language of its own follows the app's
      setLanguage(host.app)
      return
    }
    const got = await host.settingsGet().catch(logged('language.get', null))
    const r = got?.language?.resolved
    setLanguage(r === 'ko' || r === 'en' ? r : choice === 'auto' ? resolveAuto() : choice)
    return
  }
  await host.save(choice)
  setLanguage(choice === 'auto' ? resolveAuto(host.systemLanguage) : choice)
}

function localeOf(lang: Lang): string {
  return lang === 'ko' ? 'ko-KR' : 'en-US'
}

const ymd = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate())

/** dayLabel names a YYYY-MM-DD day: Today, Yesterday, or "Oct 25" / "10월 25일". */
export function dayLabel(date: string, today: Date, lang: Lang): string {
  const [y, m, d] = date.split('-').map(Number)
  const day = new Date(y, m - 1, d)
  if (!Number.isFinite(day.getTime())) return date
  const diff = Math.round((ymd(today).getTime() - day.getTime()) / 86_400_000)
  if (diff === 0) return translate(catalogs, lang, 'sessions.today')
  if (diff === 1) return translate(catalogs, lang, 'sessions.yesterday')
  return new Intl.DateTimeFormat(localeOf(lang), { month: 'short', day: 'numeric' })
    .format(day)
    .replace(/^(\d+)월\s*(\d+)일$/, '$1월 $2일')
}

/** MINUTE_MS is relativeTime's smallest step: a window that says elapsed times reads the clock this often. */
export const MINUTE_MS = 60_000

/** relativeTime says how long before now d was: "2분 전" / "2m ago". */
export function relativeTime(d: Date, now: Date, lang: Lang): string {
  const sec = Math.round((now.getTime() - d.getTime()) / 1000)
  if (sec < 45) return translate(catalogs, lang, 'time.now')
  const units: [Intl.RelativeTimeFormatUnit, number, string][] = [
    ['minute', 60, 'm'],
    ['hour', 3600, 'h'],
    ['day', 86400, 'd'],
  ]
  let pick = units[0]
  for (const u of units) if (sec >= u[1]) pick = u
  const n = Math.max(1, Math.floor(sec / pick[1]))
  if (lang === 'en') {
    // en-US "narrow" gives "2m ago" / "3h ago" / "1d ago" where ICU has it
    const s = new Intl.RelativeTimeFormat('en-US', { numeric: 'always', style: 'narrow' }).format(-n, pick[0])
    return /^\d+[mhd] ago$/.test(s) ? s : `${n}${pick[2]} ago`
  }
  return new Intl.RelativeTimeFormat('ko-KR', { numeric: 'always', style: 'long' }).format(-n, pick[0])
}

// an error the shell knows is said in the window's language (error.<code>)
setErrorWords((w) => {
  const key = 'error.' + w.code
  const s = translate(catalogs, get(langStore), key, w.values)
  return s === key ? null : s
})
