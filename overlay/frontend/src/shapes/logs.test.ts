import { describe, expect, it } from 'vitest'
import { endLine, logOf, section, tail, toneOf } from './logs'

it('a log is the same name with .log, as the runner and the .http sender name it', () => {
  expect(logOf('build.sh')).toBe('build.log')
  expect(logOf('ops/api.http')).toBe('ops/api.log')
  expect(logOf('v1.2/run')).toBe('v1.2/run.log')
})

describe('toneOf', () => {
  it('reads lines as the terminal colours them', () => {
    expect(toneOf('$ sh build.sh   (2026-10-29 03:14:26)')).toBe('command')
    expect(toneOf('✔ response.statusCode == 200')).toBe('good')
    expect(toneOf('✘ response.statusCode == 200 · got 404')).toBe('bad')
    expect(toneOf('[exit 0 · 0s]')).toBe('end-good')
    expect(toneOf('[exit 2 · 1s]')).toBe('end-bad')
    expect(toneOf('[200 OK · 2ms · Hello · 1/1 ✔]')).toBe('end-good')
    expect(toneOf('[404 File not found · 0ms · Missing · 0/1 ✘]')).toBe('end-bad')
    expect(toneOf('[500 Internal Server Error · 3ms · x]')).toBe('end-bad')
    expect(toneOf('[error · Log in]')).toBe('end-bad')
    expect(toneOf('building')).toBe('plain')
  })
})

describe('section', () => {
  const log = ['### Hello', '$ curl', 'hi', '', '[200 OK]', '', '### Missing', '$ curl', '[404 x]']
  it("is one request's part of the log, without the blank lines around it", () => {
    expect(section(log, 'Hello')).toEqual(['$ curl', 'hi', '', '[200 OK]'])
    expect(section(log, 'Missing')).toEqual(['$ curl', '[404 x]'])
    expect(section(log, 'Other')).toBeNull()
  })
  it('endLine is how it ended', () => {
    expect(endLine(section(log, 'Hello')!)).toBe('[200 OK]')
    expect(endLine(['a'])).toBe('')
  })
})

it('tail is the last lines', () => {
  expect(tail(['a', 'b', 'c'], 2)).toEqual(['b', 'c'])
  expect(tail(['a'], 2)).toEqual(['a'])
})
