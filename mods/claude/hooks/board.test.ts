import { describe as group, expect, test } from 'claude-code/testing'

import type { Note } from '../types'
import { asMarkdown, describe, lines, parse, tabs } from './board'

const note = (name: string, tab: string, title: string): Note => ({
  tab,
  tab_title: tab,
  name,
  kind: 'checklist',
  icon: '☐',
  title,
  gist: '',
  open: true,
})

group('the watch', () => {
  test('keeps a line cut between two pieces for the next one', () => {
    const first = lines('{"a":1}\n{"b"')
    expect(first.whole).toEqual(['{"a":1}'])
    expect(lines(first.rest + ':2}\n').whole).toEqual(['{"b":2}'])
  })

  test('reads an event and skips what is not one', () => {
    expect(parse('{"note":"todo.md","type":"item.ticked","item":"ship"}')?.item).toBe('ship')
    expect(parse('not json')).toBeUndefined()
    expect(parse('{"type":"item.ticked"}')).toBeUndefined()
  })
})

group('what the user is told', () => {
  const notes = [note('02-작업/02-checklist.md', '작업', '체크리스트')]

  test('names the note by its title', () => {
    const event = { note: '02-작업/02-checklist.md', type: 'item.ticked', item: '리뷰 요청' }
    expect(describe(event, notes)).toBe('☑ 체크리스트: 리뷰 요청')
  })

  test('says where a card went and who said a message', () => {
    expect(describe({ note: 'k.md', type: 'card.moved', item: 'login', to: 'Done' }, [])).toBe('k.md: login → Done')
    expect(describe({ note: 'c.md', type: 'message.added', item: 'hi', from: 'me' }, [])).toBe('c.md: me: hi')
  })

  test('is not told of a note saved or a log line', () => {
    expect(describe({ note: 'a.md', type: 'note.changed' }, notes)).toBeUndefined()
    expect(describe({ note: 'b.log', type: 'log.appended', item: 'x' }, notes)).toBeUndefined()
  })
})

group('the pane', () => {
  test('groups the notes by tab in the board order', () => {
    const grouped = tabs([note('a', '노트', 'A'), note('b', '노트', 'B'), note('c', '작업', 'C')])
    expect(grouped.map(tab => [tab.title, tab.notes.length])).toEqual([['노트', 2], ['작업', 1]])
  })

  test('shows a file that is not Markdown as code', () => {
    expect(asMarkdown('n.md', '# hi')).toBe('# hi')
    expect(asMarkdown('run.sh', 'echo hi')).toBe('```sh\necho hi\n```')
  })
})
