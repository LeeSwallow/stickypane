import { createRawSnippet, type Component } from 'svelte'
import { render } from 'svelte/server'
import { afterEach, describe, expect, test } from 'vitest'
import { THEMES } from '../theme/palettes'
import { setLanguage } from '../../i18n'
import { look } from '../doodle/look.svelte'
import Badge from './Badge.svelte'
import Button from './Button.svelte'
import { chromeOf, quitShortcut, runWindowMenu, windowMenu } from './chrome'
import Chip from './Chip.svelte'
import Confirm from './Confirm.svelte'
import EmptyState from './EmptyState.svelte'
import IconButton from './IconButton.svelte'
import Icon from './icons/Icon.svelte'
import CrayonIcon from './icons/CrayonIcon.svelte'
import { CRAYON_NAMES, crayonDrawing } from './icons/crayon'
import { ICON_NAMES, ICON_STROKE, ICONS, iconPaths, parse } from './icons/icons'
import Input from './Input.svelte'
import { nextIndex } from './keys'
import Menu from './Menu.svelte'
import NoteHeader from './NoteHeader.svelte'
import { nextFraction } from '../sizes'
import SegmentedControl from './SegmentedControl.svelte'
import Tabs from './Tabs.svelte'
import TextArea from './TextArea.svelte'
import TitleBar from './TitleBar.svelte'
import Toast from './Toast.svelte'
import Toggle from './Toggle.svelte'
import Toolbar from './Toolbar.svelte'
import Tooltip from './Tooltip.svelte'

const text = (s: string) => createRawSnippet(() => ({ render: () => `<span>${s}</span>` }))
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const html = (c: Component<any>, props: Record<string, unknown>) => render(c, { props }).body

const noop = () => {}
const win = { close: noop, minimise: noop, quit: noop }
const menu = [
  { id: 'edit', label: 'Edit', icon: 'edit-external' as const, shortcut: 'E' },
  { kind: 'separator' as const, id: 's' },
  { id: 'trash', label: 'Delete', icon: 'trash' as const, danger: true },
]

// every component, with props that exercise its states
const cases: [string, Component<any>, Record<string, unknown>][] = [
  ['Button', Button, { variant: 'primary', icon: 'plus', loading: true, children: text('Create') }],
  ['Button danger', Button, { variant: 'danger', disabled: true, children: text('Delete') }],
  ['IconButton', IconButton, { icon: 'pin', label: 'Keep on top', pressed: true, crayon: true }],
  ['Badge', Badge, { count: 3 }],
  ['Badge status', Badge, { tone: 'warn', children: text('Stalled 4m') }],
  ['Chip', Chip, { name: 'claude', color: '#8FD694', detail: 'login', time: '4m', stalled: true, onclick: noop }],
  ['Menu', Menu, { items: menu, label: 'Note actions', inline: true }],
  ['Input', Input, { label: 'Title', error: 'Bad', hint: 'Short' }],
  ['TextArea', TextArea, { label: 'Body' }],
  ['Toggle', Toggle, { label: 'Pin', checked: true }],
  ['SegmentedControl', SegmentedControl, { label: 'View', value: 'grid', iconOnly: true, options: [{ value: 'grid', label: 'Grid', icon: 'grid' }, { value: 'canvas', label: 'Canvas', icon: 'canvas' }] }],
  ['Tabs', Tabs, { label: 'Tabs', value: 'a', tabs: [{ id: 'a', label: 'work', count: 2 }, { id: 'b', label: 'notes' }] }],
  ['Toast', Toast, { message: 'Deleted', action: { label: 'Undo', run: noop }, onclose: noop }],
  ['Confirm', Confirm, { inline: true, danger: true, title: 'Delete?', confirmLabel: 'Delete', onconfirm: noop }],
  ['EmptyState', EmptyState, { title: 'Nothing yet', body: 'Make one' }],
  ['NoteHeader', NoteHeader, { title: 'Auth sprint', color: '#8FD694', summary: '3 cards', pinned: true, updated: true, menu, book: { page: 2, total: 5, onprev: noop, onnext: noop } }],
  ['TitleBar mac', TitleBar, { name: 'demo', path: '/a/very/long/path/demo', chrome: 'mac', window: win }],
  ['TitleBar own', TitleBar, { name: 'demo', path: '/a/b/demo', chrome: 'own', window: win }],
  ['Toolbar', Toolbar, { label: 'Actions', children: text('x') }],
  ['Tooltip', Tooltip, { text: 'Tip', children: text('x') }],
]

afterEach(() => {
  look.style = 'doodle'
  look.palette = null
})

describe('components render in both styles and both modes', () => {
  for (const style of ['doodle', 'neat'] as const) {
    for (const dark of [true, false]) {
      test.each(cases)(`%s (${style}, ${dark ? 'dark' : 'light'})`, (_, c, props) => {
        look.style = style
        look.dark = dark
        look.palette = THEMES.find((t) => t.dark === dark)!
        const out = html(c, props)
        expect(out.length).toBeGreaterThan(0)
        // colours come from tokens: no component writes a hex colour into its markup
        // (a crayon icon's colours are the palette's notes, derived in crayonDrawing)
        const own = out.replace(/<svg class="crayon[\s\S]*?<\/svg>/g, '')
        expect(own).not.toMatch(/(fill|stroke|color)="#[0-9a-f]{3,6}"/i)
      })
    }
  }
})

describe('accessible names', () => {
  test('every icon button says what it does', () => {
    setLanguage('en')
    for (const name of ICON_NAMES) {
      const out = html(IconButton, { icon: name, label: `do ${name}` })
      expect(out).toContain(`aria-label="do ${name}"`)
      expect(out).toMatch(/<svg[^>]*aria-hidden="true"/)
    }
  })
  test('a toggle icon button says whether it is on', () => {
    expect(html(IconButton, { icon: 'pin', label: 'Pin', pressed: true })).toContain('aria-pressed="true"')
    expect(html(IconButton, { icon: 'pin', label: 'Pin', pressed: false })).toContain('aria-pressed="false"')
  })
  test('every IconButton in the system and the demo is given a label', () => {
    const files = import.meta.glob(['./*.svelte', '../../demo/*.svelte'], { query: '?raw', import: 'default', eager: true }) as Record<string, string>
    expect(Object.keys(files).length).toBeGreaterThan(10)
    for (const [f, src] of Object.entries(files)) {
      for (const m of src.matchAll(/<IconButton\b[\s\S]*?\/>/g)) expect(m[0], `${f}: ${m[0]}`).toMatch(/\blabel=/)
    }
  })
  test('the title bar names its controls, in Korean too', () => {
    setLanguage('ko')
    const own = html(TitleBar, { name: 'demo', path: '/x/demo', chrome: 'own', window: win })
    expect(own).toContain('aria-label="창 닫기"')
    expect(own).toContain('aria-label="최소화"')
    expect(own).toContain('aria-label="더 보기"')
    const mac = html(TitleBar, { name: 'demo', path: '/x/demo', chrome: 'mac', window: win })
    expect(mac).not.toContain('aria-label="창 닫기"') // the traffic lights do it
    setLanguage('en')
  })
  test('the path is a tooltip, never a line of the bar', () => {
    const out = html(TitleBar, { name: 'demo', path: '/very/long/path/demo', chrome: 'mac', window: win })
    expect(out).toMatch(/role="tooltip"[^>]*>[\s\S]*\/very\/long\/path\/demo/)
    expect(out.match(/\/very\/long\/path\/demo/g)).toHaveLength(1)
  })
  test('fields tie their error to the input', () => {
    const out = html(Input, { label: 'Title', error: 'Too long' })
    expect(out).toContain('aria-invalid="true"')
    expect(out).toMatch(/aria-describedby="[^"]+-err"/)
  })
  test('roles of the composite controls', () => {
    expect(html(Tabs, cases[11][2])).toContain('role="tablist"')
    expect(html(SegmentedControl, cases[10][2])).toContain('role="radiogroup"')
    expect(html(Toggle, { label: 'x' })).toContain('role="switch"')
    expect(html(Menu, cases[6][2])).toContain('role="menu"')
    expect(html(Confirm, cases[13][2])).toContain('role="alertdialog"')
  })
})

describe('icons', () => {
  const wanted = [
    'pin', 'pin-off', 'tear-off', 'back', 'close', 'more', 'plus', 'size-card', 'size-half', 'size-page', 'zoom',
    'grid', 'canvas', 'run', 'send', 'edit-external', 'search', 'settings', 'sun', 'moon', 'auto', 'tab',
    'book-prev', 'book-next', 'archive', 'trash', 'restore', 'link', 'minimise',
  ]
  test('the set has every icon the brief names', () => {
    for (const n of wanted) expect(ICON_NAMES).toContain(n)
  })
  test('one family: 20px, stroke 1.75, round caps', () => {
    const out = html(Icon, { name: 'pin' })
    expect(out).toContain('viewBox="0 0 20 20"')
    expect(out).toContain(`stroke-width="${ICON_STROKE}"`)
    expect(out).toContain('stroke-linecap="round"')
    expect(ICON_STROKE).toBe(1.75)
  })
  test('every point is inside the 20px box', () => {
    for (const n of ICON_NAMES) {
      for (const d of [...ICONS[n].d, ...('fill' in ICONS[n] ? (ICONS[n] as { fill: string[] }).fill : [])]) {
        for (const c of parse(d)) {
          const xy = c.op === 'A' ? c.args.slice(5) : c.args
          for (const v of xy) {
            expect(v, `${n}: ${d}`).toBeGreaterThanOrEqual(0)
            expect(v, `${n}: ${d}`).toBeLessThanOrEqual(20)
          }
        }
      }
    }
  })
  test('neat draws them as written; doodle bends the lines, the same way every time', () => {
    for (const n of ICON_NAMES) {
      const neat = iconPaths(n, 'neat')
      expect(neat.d.join('')).not.toMatch(/Q/)
      const a = iconPaths(n, 'doodle')
      expect(a).toEqual(iconPaths(n, 'doodle'))
    }
    expect(iconPaths('close', 'doodle').d.join('')).toMatch(/Q/)
  })
  test('crayon icons: wash, hatching and outline in doodle; flat in neat', () => {
    const notes = THEMES[0].notes
    for (const n of CRAYON_NAMES) {
      const d = crayonDrawing(n, 'doodle', { notes, outline: '#000', dark: true })
      expect(d.shapes.length, n).toBeGreaterThan(0)
      expect(d.shapes.every((s) => s.hatch.length > 0)).toBe(true)
      expect(d).toEqual(crayonDrawing(n, 'doodle', { notes, outline: '#000', dark: true }))
      const f = crayonDrawing(n, 'neat', { notes, outline: '#000', dark: true })
      expect(f.shapes.every((s) => s.hatch.length === 0)).toBe(true)
      expect(f.outline.join('')).not.toMatch(/Q/)
    }
    for (const n of ['pin', 'tear-off', 'plus', 'run', 'trash', 'archive', 'settings', 'search', 'sun', 'moon']) {
      expect(CRAYON_NAMES).toContain(n)
    }
  })
  test('a crayon icon renders its hatching clipped to its shape', () => {
    look.palette = THEMES[0]
    const out = html(CrayonIcon, { name: 'pin' })
    expect(out).toContain('<clipPath')
    expect(out).toMatch(/clip-path="url\(#/)
  })
})

describe('window chrome and quit', () => {
  test('the page reads its chrome', () => {
    expect(chromeOf('?window=project&chrome=mac')).toBe('mac')
    expect(chromeOf('?chrome=own')).toBe('own')
    expect(chromeOf('?demo=system')).toBe('none')
  })
  test('the ⋯ menu ends with Close window and Quit stickypane', () => {
    setLanguage('ko')
    const t = (k: string) => ({ 'ui.closeWindow': '창 닫기', 'app.quit': 'stickypane 종료' })[k] ?? k
    const items = windowMenu('mac', t)
    const labels = items.filter((m) => m.kind !== 'separator').map((m) => ('label' in m ? m.label : ''))
    expect(labels).toEqual(['창 닫기', 'stickypane 종료'])
    expect(items.at(-1)).toMatchObject({ id: 'app.quit', shortcut: '⌘Q' })
    expect(quitShortcut('own')).toBeUndefined()
    setLanguage('en')
  })
  test('its items close the window and quit the app', () => {
    const calls: string[] = []
    const a = { close: () => calls.push('close'), minimise: () => calls.push('min'), quit: () => calls.push('quit') }
    expect(runWindowMenu('window.close', a)).toBe(true)
    expect(runWindowMenu('app.quit', a)).toBe(true)
    expect(runWindowMenu('edit', a)).toBe(false)
    expect(calls).toEqual(['close', 'quit'])
  })
  test('the tray hint has words in both languages', async () => {
    const en = (await import('../../i18n/en.json')).default as Record<string, string>
    const ko = (await import('../../i18n/ko.json')).default as Record<string, string>
    expect(ko['notice.stayInTray']).toBe('트레이에 남아 있어요')
    expect(ko['app.quit']).toBe('stickypane 종료')
    expect(en['app.quit']).toBe('Quit stickypane')
  })
})

describe('keys', () => {
  test('arrows wrap and skip the disabled; Home and End jump', () => {
    expect(nextIndex('ArrowRight', 2, 3)).toBe(0)
    expect(nextIndex('ArrowLeft', 0, 3)).toBe(2)
    expect(nextIndex('ArrowDown', 0, 3, 'vertical', (i) => i === 1)).toBe(2)
    expect(nextIndex('Home', 2, 3)).toBe(0)
    expect(nextIndex('End', 0, 3)).toBe(2)
    expect(nextIndex('ArrowDown', 0, 3)).toBeNull()
    expect(nextIndex('a', 0, 3)).toBeNull()
  })
})

describe('the note header steps the size through the fractions', () => {
  const head = (size: string) => html(NoteHeader, { title: 'Plan', color: '#8FD694', size })
  const button = (out: string, label: string) => out.match(new RegExp(`<button[^>]*aria-label="${label}"[^>]*>`))?.[0] ?? ''
  test('it shows the fraction the note is at, between narrower and wider', () => {
    const out = head('1/2')
    expect(out).toMatch(/aria-label="Width 1\/2"[^>]*>1\/2</)
    expect(button(out, 'Narrower')).not.toContain('disabled')
    expect(button(out, 'Wider')).not.toContain('disabled')
    expect(out).not.toMatch(/size-(card|half|page)/)
  })
  test('the ends cannot be stepped past', () => {
    expect(button(head('1/4'), 'Narrower')).toContain('disabled')
    expect(button(head('1/4'), 'Wider')).not.toContain('disabled')
    expect(button(head('1'), 'Wider')).toContain('disabled')
  })
  test('it asks for the next fraction', () => {
    expect(nextFraction('1/2', 1)).toBe('2/3')
    expect(nextFraction('1/2', -1)).toBe('1/3')
    expect(nextFraction('1', 1)).toBe('1')
    expect(nextFraction('1/4', -1)).toBe('1/4')
  })
})
