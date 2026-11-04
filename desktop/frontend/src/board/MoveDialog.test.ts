// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { setLanguage } from '../i18n'
import MoveDialog from './MoveDialog.svelte'
import type { Note, Tab } from './model'

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  )
  setLanguage('en')
})
afterEach(cleanup)

const note = { name: 'work/plan.md', tab: 'work', title: 'Plan', kind: 'note' } as Note
const tabs = [{ name: '' }, { name: 'work' }, { name: 'ideas' }] as Tab[]

test('a tab picked is where the note goes', async () => {
  const onmove = vi.fn()
  render(MoveDialog, { note, tabs, width: 420, onmove, onclose: () => {} })
  const move = screen.getByRole('button', { name: 'Move to...' }) as HTMLButtonElement
  expect(move.disabled).toBe(true)
  await fireEvent.click(screen.getByRole('button', { name: 'ideas/' }))
  expect(move.disabled).toBe(false)
  await fireEvent.click(move)
  expect(onmove).toHaveBeenCalledWith('ideas/')
})

test('the top level is offered to a note in a tab, and cancel closes', async () => {
  const onclose = vi.fn()
  render(MoveDialog, { note, tabs, width: 420, onmove: () => {}, onclose })
  expect(screen.getByRole('button', { name: 'Top level' })).toBeTruthy()
  expect(screen.queryByRole('button', { name: 'work/' })).toBeNull()
  await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
  expect(onclose).toHaveBeenCalled()
})
