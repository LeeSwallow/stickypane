import { afterAll, afterEach, describe, expect, test } from 'vitest'
import { errorText, logged, report, setLog } from './errors'

const caught: [string, string][] = []
const was = setLog((where, why) => caught.push([where, why]))
afterEach(() => void caught.splice(0))
afterAll(() => void setLog(was))

describe('what a failure says', () => {
  test('without the runtime\'s wrapping', () => {
    expect(errorText(new Error('Error: note gone'))).toBe('note gone')
    expect(errorText('{"message":"stale version"}')).toBe('stale version')
    expect(errorText(42)).toBe('42')
  })
})

describe('every failure is kept', () => {
  test('one told is logged once and said in the words given', () => {
    expect(report('board.pin', new Error('no bridge'))).toBe('no bridge')
    expect(report('settings.save', new Error('Error: read-only'), (why) => `Could not save: ${why}`)).toBe('Could not save: read-only')
    expect(caught).toEqual([['board.pin', 'no bridge'], ['settings.save', 'read-only']])
  })
  test('one not told is logged and settles to its fallback', async () => {
    expect(await Promise.reject(new Error('older bridge')).catch(logged('themes.list', [] as string[]))).toEqual([])
    expect(await Promise.reject('x').catch(logged('prefs'))).toBeUndefined()
    expect(caught).toEqual([['themes.list', 'older bridge'], ['prefs', 'x']])
  })
})
