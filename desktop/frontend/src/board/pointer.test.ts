import { expect, test, vi } from 'vitest'
import { capture, follow } from './pointer'

const press = (el: unknown) => ({ currentTarget: el, pointerId: 7 }) as unknown as PointerEvent

test('capture keeps the pointer on the element it went down on', () => {
  const el = { setPointerCapture: vi.fn() }
  capture(press(el))
  expect(el.setPointerCapture).toHaveBeenCalledWith(7)
})

test('a webview that refuses the capture does not stop the drag (#66)', () => {
  const el = {
    setPointerCapture: () => {
      throw new Error('NotFoundError: no active pointer')
    },
  }
  expect(() => capture(press(el))).not.toThrow()
  expect(() => capture(press(null))).not.toThrow()
  expect(() => capture(press({}))).not.toThrow()
})

const pointer = (type: string, pointerId: number) => Object.assign(new Event(type), { pointerId })

test('follow hears its own pointer until it is let go, then nothing', () => {
  const on = new EventTarget()
  const moves: number[] = []
  const ends: boolean[] = []
  follow(press({}), (m) => moves.push(m.pointerId), (keep) => ends.push(keep), on)
  on.dispatchEvent(pointer('pointermove', 7))
  on.dispatchEvent(pointer('pointermove', 8)) // another finger
  on.dispatchEvent(pointer('pointerup', 8))
  on.dispatchEvent(pointer('pointerup', 7))
  on.dispatchEvent(pointer('pointermove', 7))
  on.dispatchEvent(pointer('pointerup', 7))
  expect(moves).toEqual([7])
  expect(ends).toEqual([true])
})

test('a pointer the system takes away ends the drag without keeping it', () => {
  const on = new EventTarget()
  const ends: boolean[] = []
  follow(press({}), () => {}, (keep) => ends.push(keep), on)
  on.dispatchEvent(pointer('pointercancel', 7))
  on.dispatchEvent(pointer('pointerup', 7))
  expect(ends).toEqual([false])
})
