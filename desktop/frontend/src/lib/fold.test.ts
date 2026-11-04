import { describe, expect, test } from 'vitest'
import { fold } from './fold'

const gate = () => {
  let open = () => {}
  const p = new Promise<void>((r) => (open = r))
  return { p, open }
}

describe('fold', () => {
  test('calls during a run are one more run after it', async () => {
    const gates = [gate(), gate()]
    let runs = 0
    const read = fold(async () => {
      await gates[runs++]?.p
    })
    const first = read()
    const second = read()
    read()
    expect(second).toBe(first)
    expect(runs).toBe(1)
    gates[0].open()
    gates[1].open()
    await first
    expect(runs).toBe(2)
  })
  test('a call after a run is a run of its own', async () => {
    let runs = 0
    const read = fold(async () => void runs++)
    await read()
    await read()
    expect(runs).toBe(2)
  })
  test('a pass that throws ends the run, and the next call runs again', async () => {
    let runs = 0
    const read = fold(async () => {
      runs++
      throw new Error('gone')
    })
    await expect(read()).rejects.toThrow('gone')
    await expect(read()).rejects.toThrow('gone')
    expect(runs).toBe(2)
  })
})
