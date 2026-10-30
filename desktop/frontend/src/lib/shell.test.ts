import { expect, test } from 'vitest'
import { canvas, canvasEvent, changedEvent, home, noteOf, project, restartedEvent, statusEvent, windowKind } from './shell'

test('a page knows which window it is', () => {
  expect(windowKind('')).toBe('home')
  expect(windowKind('?window=project')).toBe('project')
  expect(windowKind('?window=other')).toBe('home')
  expect(windowKind('?window=note&note=work%2Fa.md')).toBe('note')
  expect(noteOf('?window=note&note=work%2Fa.md')).toBe('work/a.md')
  expect(noteOf('?window=project')).toBe('')
})

test("a torn-off note's page, exactly as the Go shell builds it, draws that note (#62)", () => {
  // the same string as tornPage in desktop/internal/shell/windows_test.go
  const search = new URL('/?chrome=own&note=04-%EC%B0%A8%ED%8A%B8%2F02-spark.md&window=note', 'http://wails.localhost').search
  expect(windowKind(search)).toBe('note')
  expect(noteOf(search)).toBe('04-차트/02-spark.md')
})

test('service and event names match the Go shell', () => {
  expect(home('Sessions')).toBe('github.com/LeeSwallow/stickypane/desktop/internal/shell.HomeService.Sessions')
  expect(project('Load')).toBe('github.com/LeeSwallow/stickypane/desktop/internal/shell.ProjectService.Load')
  expect(changedEvent(3)).toBe('board.changed:3')
  expect(restartedEvent(3)).toBe('bridge.restarted:3')
  expect(statusEvent(3)).toBe('project.status:3')
  expect(canvas('TearOff')).toBe('github.com/LeeSwallow/stickypane/desktop/internal/shell.CanvasService.TearOff')
  expect(canvasEvent(3)).toBe('canvas.changed:3')
})
