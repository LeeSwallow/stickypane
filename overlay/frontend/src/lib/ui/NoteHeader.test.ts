// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'
import { setLanguage } from '../../i18n'
import NoteHeader from './NoteHeader.svelte'

beforeEach(() => setLanguage('en'))
afterEach(cleanup)

const title = () => screen.getByRole('heading', { level: 3 })

for (const key of ['F2', 'Enter']) {
  test(`${key} on the focused title renames the note, and goes no further`, async () => {
    const onrename = vi.fn()
    render(NoteHeader, { title: 'Plan', color: '#8FD694', onrename })
    const h = title()
    expect(h.tabIndex).toBe(0)
    const seen = vi.fn()
    window.addEventListener('keydown', seen) // the board's keys listen here
    h.focus()
    await fireEvent.keyDown(h, { key })
    window.removeEventListener('keydown', seen)
    expect(seen).not.toHaveBeenCalled()
    const input = screen.getByRole('textbox', { name: 'Double-click to rename' }) as HTMLInputElement
    expect(input.value).toBe('Plan')
    await fireEvent.input(input, { target: { value: 'Roadmap' } })
    await fireEvent.keyDown(input, { key: 'Enter' })
    expect(onrename).toHaveBeenCalledWith('Roadmap')
  })
}

test('a title that cannot be renamed takes no focus and no keys', async () => {
  render(NoteHeader, { title: 'Plan', color: '#8FD694', editable: false, onrename: vi.fn() })
  expect(title().hasAttribute('tabindex')).toBe(false)
  await fireEvent.keyDown(title(), { key: 'F2' })
  expect(screen.queryByRole('textbox')).toBeNull()
})
