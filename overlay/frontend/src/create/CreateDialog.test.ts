// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage, t } from '../i18n'
import kinds from '../demo/kinds.json'
import type { CreateApi, CreateRequest, Shape } from '../services/create'
import CreateDialog from './CreateDialog.svelte'
import { optionLabel } from './labels'

const shapes = kinds as unknown as Shape[]

function fakeApi(create?: (r: CreateRequest) => Promise<{ note: string }>): CreateApi & { sent: CreateRequest[] } {
  const sent: CreateRequest[] = []
  return {
    sent,
    kinds: vi.fn(async () => shapes),
    create: vi.fn(async (r: CreateRequest) => {
      sent.push(r)
      return create ? create(r) : { note: 'work/x.md' }
    }),
  }
}

const tabs = [
  { name: '', label: 'Board' },
  { name: 'work', label: 'work' },
]

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

const open = (api: CreateApi, start: string, oncreated = vi.fn()) =>
  render(CreateDialog, { props: { open: true, api, tabs, tab: 'work', start, example: false, oncreated } })

describe('the create dialog', () => {
  it('lists the shapes kinds.list gives, in its order', async () => {
    open(fakeApi(), 'note')
    await screen.findByRole('radio', { name: /Note/ })
    const names = [...document.querySelectorAll<HTMLElement>('[data-shape]')].map((b) => b.dataset.shape)
    expect(names).toEqual(shapes.map((s) => s.name))
  })

  it('asks kinds.list for the examples in the language the window draws in', async () => {
    setLanguage('ko')
    const api = fakeApi()
    open(api, 'note')
    await waitFor(() => expect(api.kinds).toHaveBeenCalledWith('ko'))
    cleanup()
    setLanguage('en')
    const again = fakeApi()
    open(again, 'note')
    await waitFor(() => expect(again.kinds).toHaveBeenCalledWith('en'))
  })

  it('draws a field for each option of the shape, of its type', async () => {
    open(fakeApi(), 'api')
    await screen.findByLabelText('URL')
    // choice: segments; text and number: a line
    expect(screen.getByRole('radiogroup', { name: 'Method' })).toBeTruthy()
    expect((screen.getByLabelText('URL') as HTMLInputElement).value).toBe('https://httpbin.org/get')
    expect((screen.getByLabelText('Expected status') as HTMLInputElement).value).toBe('200')
    await fireEvent.click(document.querySelector('[data-shape="board"]')!)
    // lines: a box with its default lines
    expect((screen.getByLabelText('Columns') as HTMLTextAreaElement).value).toBe('To do\nDoing\nDone')
  })

  it('sends the title, the tab and the values, never a body', async () => {
    const api = fakeApi()
    const oncreated = vi.fn()
    open(api, 'board', oncreated)
    const title = await screen.findByLabelText(t('create.titlePrompt'))
    await fireEvent.input(title, { target: { value: 'Sprint 12' } })
    await fireEvent.input(screen.getByLabelText('Columns'), { target: { value: 'Next\nNow' } })
    await fireEvent.click(screen.getByRole('button', { name: t('create.make') }))
    await waitFor(() => expect(oncreated).toHaveBeenCalledWith('work/x.md'))
    expect(api.sent).toEqual([{ tab: 'work', kind: 'board', title: 'Sprint 12', options: { columns: ['Next', 'Now'], cards: [''] } }])
    expect(api.sent[0]).not.toHaveProperty('body')
  })

  it('shows what the shape refused next to its field, and stays open', async () => {
    const api = fakeApi(async () => {
      throw new Error('bad option: status of an api: "soon" is not a number')
    })
    const oncreated = vi.fn()
    open(api, 'api', oncreated)
    await fireEvent.input(await screen.findByLabelText('Expected status'), { target: { value: 'soon' } })
    await fireEvent.click(screen.getByRole('button', { name: t('create.make') }))
    const field = await waitFor(() => {
      const f = document.querySelector('[data-option="status"]')!
      expect(f.textContent).toContain('is not a number')
      return f
    })
    expect(field.querySelector('[aria-invalid="true"]')).toBeTruthy()
    expect(api.sent[0].options.status).toBe('soon')
    expect(oncreated).not.toHaveBeenCalled()
  })

  it('an error about no field is said under the form', async () => {
    open(fakeApi(async () => Promise.reject(new Error('there is no tab "gone"'))), 'note')
    await screen.findByLabelText(t('create.titlePrompt'))
    await fireEvent.click(screen.getByRole('button', { name: t('create.make') }))
    expect((await screen.findByRole('alert')).textContent).toContain('there is no tab "gone"')
  })

  it('every option label of every shape has its Korean words', () => {
    setLanguage('ko')
    for (const s of shapes) for (const o of s.options) expect(t('label.' + o.label), o.label).not.toBe('label.' + o.label)
    expect(optionLabel(t, 'Columns')).toBe('칸')
    expect(optionLabel(t, 'Something new')).toBe('Something new')
  })
})
