import { describe, expect, it, vi } from 'vitest'
import { act } from './actions'
import type { BoardApi } from '../services/board'
import type { Note } from './model'

const note = { name: 'work/a.md', version: 'v1', kind: 'checklist', data: { items: [] } } as unknown as Note

describe('a note action', () => {
  it('that fails is said after the board is read again, so the read does not clear it', async () => {
    const order: string[] = []
    const api = { act: vi.fn(async () => Promise.reject(new Error('stale version'))) } as unknown as BoardApi
    await act(
      api,
      note,
      { kind: 'tick' } as never,
      (why) => order.push('fail: ' + why),
      async () => void order.push('refresh'),
    )
    expect(order).toEqual(['refresh', 'fail: stale version'])
  })
  it('that works is read again and says nothing', async () => {
    const fail = vi.fn()
    const refresh = vi.fn(async () => {})
    await act({ act: vi.fn(async () => ({})) } as unknown as BoardApi, note, { kind: 'tick' } as never, fail, refresh)
    expect(refresh).toHaveBeenCalledOnce()
    expect(fail).not.toHaveBeenCalled()
  })
})
