import { expect, test } from 'vitest'
import { changedEvent, home, project, restartedEvent, statusEvent, windowKind } from './shell'

test('a page knows which window it is', () => {
  expect(windowKind('')).toBe('home')
  expect(windowKind('?window=project')).toBe('project')
  expect(windowKind('?window=other')).toBe('home')
})

test('service and event names match the Go shell', () => {
  expect(home('Sessions')).toBe('github.com/LeeSwallow/stickypane/overlay/internal/shell.HomeService.Sessions')
  expect(project('Load')).toBe('github.com/LeeSwallow/stickypane/overlay/internal/shell.ProjectService.Load')
  expect(changedEvent(3)).toBe('board.changed:3')
  expect(restartedEvent(3)).toBe('bridge.restarted:3')
  expect(statusEvent(3)).toBe('project.status:3')
})
