// The notes service: what a project window does to a note as a whole —
// archive, delete and bring back, move, run a script, send a request.
// Each is one bridge call through the shell (RA13).

import { Call } from '@wailsio/runtime'
import { project } from '../lib/shell'

export interface RunResult {
  code: number
  took: number
}

export interface NotesApi {
  archive(note: string): Promise<unknown>
  trash(note: string): Promise<unknown>
  /** restore brings a deleted note back from the trash */
  restore(note: string): Promise<unknown>
  /** move: to is "." (top level), "folder/" (a book) or a new name; it answers the new name */
  move(note: string, to: string): Promise<{ note: string }>
  run(note: string): Promise<RunResult>
  send(note: string, request: string | number): Promise<unknown>
}

export const notesService: NotesApi = {
  archive: (note) => Call.ByName(project('Archive'), note),
  trash: (note) => Call.ByName(project('Trash'), note),
  restore: (note) => Call.ByName(project('Restore'), note),
  move: (note, to) => Call.ByName(project('Move'), { note, to }) as Promise<{ note: string }>,
  run: (note) => Call.ByName(project('Run'), note) as Promise<RunResult>,
  send: (note, request) => Call.ByName(project('Send'), { note, request }),
}

export { errorText } from './errors'
