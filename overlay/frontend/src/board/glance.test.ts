import { expect, test } from 'vitest'
import { glance } from './glance'
import { htmlFor, resolveLink, splitEmbeds } from './model'
import type { Tab } from './model'

const kanban = (name: string, columns: { title: string; cards: object[] }[]) => ({
  name, tab: '', title: name, kind: 'board', icon: '', open: true, size: 'page', rows: 0, color: 'blue',
  pinned: false, version: 'v', summary: '', modTime: '', linked: false, data: { desc: [], columns },
})
const card = (text: string, who: string[], moved?: string, stalled = false) => ({ text, who, moved, detail: [], stalled })

test('glance picks the Doing card per agent', () => {
  const tabs = [
    {
      name: '', title: '', notes: [
        kanban('work.md', [
          { title: 'Todo', cards: [card('write docs', ['claude'], '2026-10-28T09:00:00Z'), card('plan', ['sam'])] },
          { title: 'Doing', cards: [card('build grid', ['claude'], '2026-10-28T10:00:00Z'), card('review', ['codex'], '2026-10-28T08:00:00Z', true)] },
          { title: 'Done', cards: [card('ship', ['claude'], '2026-10-28T11:00:00Z'), card('old', ['dana'], '2026-10-27T11:00:00Z')] },
        ]),
      ],
    },
  ] as unknown as Tab[]
  const now = new Date('2026-10-28T12:00:00Z')
  const got = glance(tabs, now)
  expect(got.map((g) => g.who)).toEqual(['claude', 'sam', 'codex', 'dana'])
  const claude = got[0]
  // the most recently moved card not in the last column (03 §3.6); the
  // newer Done card does not count
  expect(claude.card?.text).toBe('build grid')
  expect(claude.note).toBe('work.md')
  expect(claude.since).toBe(2 * 3600_000)
  expect(claude.stalled).toBe(false)
  expect(got[1].card?.text).toBe('plan') // no time: still the card it holds
  expect(got[1].since).toBeNull()
  expect(got[2]).toMatchObject({ who: 'codex', stalled: true })
  expect(got[3].card).toBeNull() // only done cards: resting
})

test('on a tie the later column (Doing) wins', () => {
  const at = '2026-10-28T10:00:00Z'
  const tabs = [{ name: '', title: '', notes: [kanban('w', [
    { title: 'Todo', cards: [card('a', ['x'], at)] },
    { title: 'Doing', cards: [card('b', ['x'], at)] },
    { title: 'Done', cards: [] },
  ])] }] as unknown as Tab[]
  expect(glance(tabs, new Date(at))[0].card?.text).toBe('b')
})

test('links resolve as the terminal resolves them', () => {
  const files = ['plan.md', 'work/todo.md', 'work/plan.sh', 'a/x.md', 'b/x.md']
  expect(resolveLink('plan', files)).toBe('plan.md')
  expect(resolveLink('todo', files)).toBe('work/todo.md')
  expect(resolveLink('work/todo.md', files)).toBe('work/todo.md')
  expect(resolveLink('x', files)).toBe('') // two could be meant
  expect(resolveLink('nothing', files)).toBe('')
})

test('HTML made for older segments is not drawn against new ones', () => {
  const a = splitEmbeds('one')
  const b = splitEmbeds('one\n![[x]]\ntwo')
  expect(htmlFor({ segs: a, html: ['<p>one</p>'] }, a)).toEqual(['<p>one</p>'])
  expect(htmlFor({ segs: a, html: ['<p>one</p>'] }, b)).toEqual([])
})

test('an embed is a line that is only ![[name]], outside code', () => {
  const md = 'intro\n![[plan]]\ntext ![[inline]] here\n```\n![[code]]\n```\n  ![[todo#x|alias]]  \nend'
  expect(splitEmbeds(md)).toEqual([
    { md: 'intro' },
    { embed: 'plan' },
    { md: 'text ![[inline]] here\n```\n![[code]]\n```' },
    { embed: 'todo' },
    { md: 'end' },
  ])
})
