import { expect, test } from 'vitest'
import { locate, type Note, type Tab } from './model'

const n = (name: string, pages?: Note[]) => ({ name, pages }) as Note
const tabs = [
  { name: '', notes: [n('plan.md')] },
  { name: 'work', notes: [n('todo.md'), n('book', [n('book/1.md'), n('book/2.md')])] },
] as Tab[]

test('a note is found in its tab, and a page in its book', () => {
  expect(locate(tabs, 'plan.md')).toEqual({ tab: '', book: null, page: 0 })
  expect(locate(tabs, 'todo.md')).toEqual({ tab: 'work', book: null, page: 0 })
  const page = locate(tabs, 'book/2.md')
  expect([page?.tab, page?.book?.name, page?.page]).toEqual(['work', 'book', 1])
  expect(locate(tabs, 'gone.md')).toBeNull()
})
