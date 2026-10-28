import { expect, test } from 'vitest'
import { connect, type Bus } from './session'

function fakeBus() {
  const handlers = new Map<string, (e: { data?: unknown }) => void>()
  let offs = 0
  const bus: Bus = {
    on(name, fn) {
      handlers.set(name, fn)
      return () => {
        offs++
        handlers.delete(name)
      }
    },
  }
  return { bus, handlers, offs: () => offs }
}

const tick = () => new Promise<void>((r) => setTimeout(r, 0))

test('a first status that fails is asked again, and updates still arrive', async () => {
  const { bus, handlers } = fakeBus()
  let calls = 0
  const seen: string[] = []
  connect({
    status: async () => {
      if (++calls === 1) throw new Error('bridge starting')
      return { window: 7 }
    },
    bus,
    sleep: () => tick(),
    onChanged: (d) => seen.push('changed:' + String(d)),
    onRestarted: () => seen.push('restarted'),
    onStatus: () => seen.push('status'),
  })
  for (let i = 0; i < 5; i++) await tick()
  expect(calls).toBe(2)
  expect([...handlers.keys()].sort()).toEqual(['board.changed:7', 'bridge.restarted:7', 'project.status:7'])
  handlers.get('board.changed:7')!({ data: 'x' })
  handlers.get('bridge.restarted:7')!({})
  expect(seen).toEqual(['changed:x', 'restarted'])
})

test('closing the window before the id is known leaves no subscription', async () => {
  const { bus, handlers } = fakeBus()
  let answer!: (s: { window: number }) => void
  const stop = connect({
    status: () => new Promise((r) => (answer = r)),
    bus,
    sleep: () => tick(),
    onChanged: () => {},
    onRestarted: () => {},
    onStatus: () => {},
  })
  await tick()
  stop()
  answer({ window: 3 })
  for (let i = 0; i < 3; i++) await tick()
  expect(handlers.size).toBe(0)
})

test('closing the window removes its subscriptions', async () => {
  const { bus, handlers, offs } = fakeBus()
  const stop = connect({ status: async () => ({ window: 1 }), bus, sleep: () => tick(), onChanged: () => {}, onRestarted: () => {}, onStatus: () => {} })
  for (let i = 0; i < 3; i++) await tick()
  expect(handlers.size).toBe(3)
  stop()
  expect(handlers.size).toBe(0)
  expect(offs()).toBe(3)
})

test('it stops asking once closed', async () => {
  const { bus } = fakeBus()
  let calls = 0
  const stop = connect({
    status: async () => {
      calls++
      throw new Error('down')
    },
    bus,
    sleep: () => tick(),
    onChanged: () => {},
    onRestarted: () => {},
    onStatus: () => {},
  })
  await tick()
  stop()
  const at = calls
  for (let i = 0; i < 5; i++) await tick()
  expect(calls).toBeLessThanOrEqual(at + 1)
})
