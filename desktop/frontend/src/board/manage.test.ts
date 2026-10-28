import { beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import type { NotesApi } from '../services/notes'
import * as manage from './manage'
import type { Note } from './model'

const note = (over: Partial<Note> = {}) => ({ name: 'work/plan.md', tab: 'work', title: 'Plan', kind: 'note', ...over }) as Note

function rig(over: Partial<NotesApi> = {}) {
  const calls: string[] = []
  const said: manage.Said[] = []
  const failed: string[] = []
  let refreshed = 0
  const api: NotesApi = {
    archive: vi.fn(async (n) => void calls.push(`archive ${n}`)),
    trash: vi.fn(async (n) => void calls.push(`trash ${n}`)),
    restore: vi.fn(async (n) => void calls.push(`restore ${n}`)),
    move: vi.fn(async (n, to) => (calls.push(`move ${n} ${to}`), { note: 'notes/plan.md' })),
    run: vi.fn(async (n) => (calls.push(`run ${n}`), { code: 0, took: 12 })),
    send: vi.fn(async (n, r) => void calls.push(`send ${n} ${r}`)),
    ...over,
  }
  const deps: manage.ManageDeps = { api, refresh: async () => void refreshed++, say: (s) => said.push(s), fail: (m) => failed.push(m) }
  return { deps, calls, said, failed, refreshed: () => refreshed }
}

beforeEach(() => setLanguage('en'))

describe('the ⋯ menu of a note', () => {
  it('offers run to a script, send to a request, and only delete to a note it cannot read', () => {
    expect(manage.manageItems(note({ kind: 'script' }))).toEqual(['run', 'move', 'archive', 'trash'])
    expect(manage.manageItems(note({ kind: 'api' }))).toEqual(['send', 'move', 'archive', 'trash'])
    expect(manage.manageItems(note())).toEqual(['move', 'archive', 'trash'])
    expect(manage.manageItems(note({ error: 'bad' }))).toEqual(['trash'])
  })

  it('a delete leaves a line whose 되돌리기 restores the note', async () => {
    const r = rig()
    expect(await manage.trash(r.deps, note())).toBe(true)
    expect(r.calls).toEqual(['trash work/plan.md'])
    expect(r.said[0].action).toBeTruthy()
    r.said[0].action!.run()
    await vi.waitFor(() => expect(r.calls).toEqual(['trash work/plan.md', 'restore work/plan.md']))
    expect(r.refreshed()).toBeGreaterThanOrEqual(1)
  })

  it('archive, move, run and send each make one call and say what happened', async () => {
    const r = rig()
    await manage.archive(r.deps, note())
    await manage.move(r.deps, note(), ' notes/ ')
    await manage.run(r.deps, note({ kind: 'script' }))
    await manage.send(r.deps, note({ kind: 'api' }))
    expect(r.calls).toEqual(['archive work/plan.md', 'move work/plan.md notes/', 'run work/plan.md', 'send work/plan.md 1'])
    expect(r.said.map((s) => s.message)).toEqual(['Archived Plan', 'Moved to notes/plan.md', 'Ran in 12 ms', 'Sent'])
    expect(r.refreshed()).toBe(4)
  })

  it('a .http note\'s own Send sends the request it names', async () => {
    const r = rig()
    await manage.send(r.deps, note({ kind: 'api' }), { at: 2, title: 'Log in' })
    expect(r.calls).toEqual(['send work/plan.md 2'])
  })

  it('a script that fails says its exit code', async () => {
    const r = rig({ run: async () => ({ code: 2, took: 5 }) })
    await manage.run(r.deps, note({ kind: 'script' }))
    expect(r.said[0].message).toContain('2')
  })

  it('a refused call is told, not thrown, and the board is read again', async () => {
    const r = rig({ trash: async () => Promise.reject(new Error('Error: there is no note "x"')) })
    expect(await manage.trash(r.deps, note())).toBe(false)
    expect(r.failed).toEqual(['Plan: there is no note "x"'])
    expect(r.said).toEqual([])
    expect(r.refreshed()).toBe(1)
  })

  it('a note moves to the top level or another tab', () => {
    const tabs = [{ name: '' }, { name: 'work' }, { name: 'notes' }]
    expect(manage.moveTargets(tabs, 'work')).toEqual(['.', 'notes/'])
    expect(manage.moveTargets(tabs, '')).toEqual(['work/', 'notes/'])
  })
})
