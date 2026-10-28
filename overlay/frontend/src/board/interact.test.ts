import { describe, expect, it } from 'vitest'
import { dropAt, keyMove, moveAction, predict, withData } from './interact'
import type { Board, KanbanData, Note } from './model'

const kanban = (): KanbanData => ({
  desc: [],
  columns: [
    { title: 'To do', cards: [{ text: 'a', who: [], detail: [], stalled: false }, { text: 'b', who: [], detail: [], stalled: true }] },
    { title: 'Done', cards: [{ text: 'c', who: [], detail: [], stalled: false }] },
  ],
})

describe('card moves', () => {
  it('moveAction is the board shape\'s move, strings only', () => {
    expect(moveAction(0, 1, 1, 0)).toEqual({ name: 'move', args: { col: '0', i: '1', to: '1', at: '0' } })
    expect(moveAction(0, 1, 1).args).not.toHaveProperty('at')
  })
  it('a drop in another column goes before the card under it, or to the end', () => {
    expect(dropAt(0, 0, 1, 0, 1)).toEqual(moveAction(0, 0, 1, 0))
    expect(dropAt(0, 0, 1, null, 1)).toEqual(moveAction(0, 0, 1))
  })
  it('a drop in its own column counts places without the card', () => {
    expect(dropAt(0, 0, 0, null, 2)).toEqual(moveAction(0, 0, 0, 1))
    expect(dropAt(0, 1, 0, 0, 2)).toEqual(moveAction(0, 1, 0, 0))
    expect(dropAt(0, 0, 0, 1, 2)).toBeNull()
    expect(dropAt(0, 0, 0, 0, 2)).toBeNull()
  })
  it('h l j k move as on the terminal, and not past an edge', () => {
    const d = kanban()
    expect(keyMove(d, 0, 0, 'l')).toEqual(moveAction(0, 0, 1))
    expect(keyMove(d, 0, 0, 'h')).toBeNull()
    expect(keyMove(d, 0, 0, 'j')).toEqual(moveAction(0, 0, 0, 1))
    expect(keyMove(d, 0, 0, 'k')).toBeNull()
    expect(keyMove(d, 1, 0, 'L')).toBeNull()
    expect(keyMove(d, 0, 0, 'x')).toBeNull()
  })
})

describe('predict', () => {
  it('ticks a checklist item and counts it', () => {
    const d = { items: [{ text: 'x', checked: false }, { text: 'y', checked: true }], prose: [], done: 1, total: 2 }
    const p = predict('checklist', d, { name: 'toggle', args: { i: '0' } }) as typeof d
    expect(p.items[0].checked).toBe(true)
    expect(p.done).toBe(2)
    expect(d.items[0].checked).toBe(false)
    expect(predict('checklist', d, { name: 'toggle', args: { i: '9' } })).toBeNull()
  })
  it('moves a card to its place and clears its stall', () => {
    const p = predict('board', kanban(), moveAction(0, 1, 1, 0)) as KanbanData
    expect(p.columns[0].cards.map((c) => c.text)).toEqual(['a'])
    expect(p.columns[1].cards.map((c) => c.text)).toEqual(['b', 'c'])
    expect(p.columns[1].cards[0].stalled).toBe(false)
    const same = predict('board', kanban(), moveAction(0, 0, 0, 1)) as KanbanData
    expect(same.columns[0].cards.map((c) => c.text)).toEqual(['b', 'a'])
  })
  it('leaves other shapes and actions to the bridge', () => {
    expect(predict('form', {}, { name: 'submit' })).toBeNull()
    expect(predict('board', kanban(), { name: 'add', args: { col: '0', text: 'z' } })).toBeNull()
  })
})

describe('withData', () => {
  it('replaces one note\'s data, pages too', () => {
    const n = (name: string, pages?: Note[]) => ({ name, data: 1, pages }) as unknown as Note
    const b = { tabs: [{ name: '', title: '', notes: [n('a'), n('book', [n('book/p1')])] }] } as unknown as Board
    const out = withData(b, 'book/p1', 2)
    expect(out.tabs[0].notes[1].pages![0].data).toBe(2)
    expect(out.tabs[0].notes[0].data).toBe(1)
    expect(b.tabs[0].notes[1].pages![0].data).toBe(1)
  })
})
