import { expect, test, vi } from 'vitest'
import { capture } from './pointer'

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
