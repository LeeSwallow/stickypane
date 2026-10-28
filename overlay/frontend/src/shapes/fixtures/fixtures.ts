// The samples the shapes' tests draw: board.load from a real bridge over a
// project with a note of every shape, made by overlay/scripts/shape-fixtures.sh.
// Run the script again when a shape's data changes.
import { boardContext, type BoardCtx } from '../../board/context'
import { allNotes, findNote, resolveLink, type Board, type Note } from '../../board/model'
import board from './board.json'

export const fixture = board as unknown as Board
const notes = allNotes(fixture.tabs)
const files = notes.map((n) => n.name)

/** sample is the fixture's note of that name; it fails when there is none. */
export function sample(name: string): Note {
  const n = findNote(fixture.tabs, name)
  if (!n) throw new Error(`no note ${name} in fixtures/board.json`)
  return n
}

/** The fixture was made at about this time: relative times read the same in every run. */
export const NOW = new Date(Math.max(...notes.map((n) => Date.parse(n.modTime))) + 5 * 60_000)

/** context is a board context over the fixture, with what a test overrides. */
export function context(over: Partial<BoardCtx> = {}): Map<unknown, unknown> {
  const c: BoardCtx = {
    find: (link) => findNote(fixture.tabs, resolveLink(link, files)),
    navigate: () => {},
    act: async () => {},
    arrange: async () => {},
    zoom: () => {},
    back: () => {},
    canBack: () => false,
    flip: () => {},
    pageOf: () => 0,
    changed: () => false,
    seen: () => {},
    filter: () => null,
    now: () => NOW,
    ...over,
  }
  return new Map([[boardContext, c]])
}
