// @vitest-environment jsdom
// The books of a repository's worktrees, as a real bridge says them
// (shapes/fixtures/board.json): the state the board works out, and a book
// shown a page at a time or every page, as the terminal's v chooses.
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import { context, sample } from '../shapes/fixtures/fixtures'
import { keyAction } from './keys'
import { withArrange, type Board } from './model'
import NoteFrame from './NoteFrame.svelte'

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver

beforeEach(() => setLanguage('en'))
afterEach(cleanup)

const frame = (name: string, over = {}) => render(NoteFrame, { props: { note: sample(name) }, context: context(over) })

it('a worktree book wears the state the board worked out, in words', () => {
  for (const [name, said] of [
    ['worktrees/fix-login', 'Running'],
    ['worktrees/chart', 'Waiting for you (1)'],
    ['worktrees/idle', 'Open'],
    ['worktrees/old', 'Closed'],
  ]) {
    const { container, unmount } = frame(name)
    expect(within(container.querySelector('header') as HTMLElement).getByText(said), name).toBeTruthy()
    unmount()
  }
})

it('paging: single draws one page and n / N; without it every page, each under its name', () => {
  const single = frame('worktrees/fix-login')
  expect(single.container.querySelectorAll('[data-page]').length).toBe(0)
  expect(within(single.container).getByRole('combobox', { name: 'Pages' })).toBeTruthy()
  expect(within(single.container).getByText('1 / 2 · 개요')).toBeTruthy()
  single.unmount()

  const all = frame('worktrees/chart')
  const pages = [...all.container.querySelectorAll('[data-page]')]
  expect(pages.map((p) => p.querySelector('.rule')?.textContent)).toEqual(['개요', 'Merge', 'Plan'])
  expect(within(pages[1] as HTMLElement).getByRole('button', { name: 'OK' })).toBeTruthy()
})

it('⋯ shows a book a page at a time, and every page again, through note.arrange', async () => {
  const arrange = vi.fn(async () => {})
  const all = frame('worktrees/idle', { arrange })
  await fireEvent.click(within(all.container).getByRole('button', { name: 'More' }))
  await fireEvent.click(await screen.findByRole('menuitem', { name: /One page at a time/ }))
  expect(arrange).toHaveBeenCalledWith('worktrees/idle', { paging: 'single' })
  all.unmount()

  const single = frame('worktrees/fix-login', { arrange })
  await fireEvent.click(within(single.container).getByRole('button', { name: 'More' }))
  await fireEvent.click(await screen.findByRole('menuitem', { name: /Every page/ }))
  expect(arrange).toHaveBeenCalledWith('worktrees/fix-login', { paging: '' })
})

it(', and . turn the pages, as [ and ] do; v is the terminal\'s key for one page or all', () => {
  const ctx = { zoomed: false, canBack: false, book: true }
  expect(keyAction({ key: ',' }, ctx)).toEqual({ type: 'page', delta: -1 })
  expect(keyAction({ key: '.' }, ctx)).toEqual({ type: 'page', delta: 1 })
  expect(keyAction({ key: 'v' }, ctx)).toEqual({ type: 'paging' })
  expect(keyAction({ key: ',', metaKey: true }, ctx)).toEqual({ type: 'settings' })
})

it('the guess before the board is read again keeps the paging', () => {
  const b = { tabs: [{ name: 't', title: 't', notes: [sample('worktrees/idle')] }] } as unknown as Board
  expect(withArrange(b, 'worktrees/idle', { paging: 'single' }).tabs[0].notes[0].paging).toBe('single')
  expect(withArrange(b, 'worktrees/idle', { paging: '' }).tabs[0].notes[0].paging).toBeUndefined()
})
