// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, expect, it, vi } from 'vitest'
import AddLine from './AddLine.svelte'

afterEach(cleanup)

it('Enter adds the line, trimmed, and empties it; an empty line adds nothing', async () => {
  const onadd = vi.fn(async () => {})
  render(AddLine, { props: { label: 'New item', onadd } })
  const input = screen.getByLabelText('New item') as HTMLInputElement
  await fireEvent.keyDown(input, { key: 'Enter' })
  expect(onadd).not.toHaveBeenCalled()
  await fireEvent.input(input, { target: { value: '  write docs ' } })
  await fireEvent.keyDown(input, { key: 'Enter' })
  expect(onadd).toHaveBeenCalledWith('write docs')
  await vi.waitFor(() => expect(input.value).toBe(''))
})

it('Escape lets go without adding', async () => {
  const onadd = vi.fn(async () => {})
  render(AddLine, { props: { label: 'New card', onadd } })
  const input = screen.getByLabelText('New card') as HTMLInputElement
  await fireEvent.input(input, { target: { value: 'half' } })
  await fireEvent.keyDown(input, { key: 'Escape' })
  expect(input.value).toBe('')
  expect(onadd).not.toHaveBeenCalled()
})
