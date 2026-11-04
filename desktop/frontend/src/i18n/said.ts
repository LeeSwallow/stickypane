// What a window says when a call fails: what could not be done, in the
// window's language, then the reason — an error the shell knows in the
// window's language too, any other in the words it came in (a bridge's,
// ssh's). The log keeps the failure under where it happened.
import { t } from './index'
import { report } from '../services/errors'

// where a call failed (the part after the window's name) → the sentence
const SAID: Record<string, string> = {
  refresh: 'failed.read',
  readBoard: 'failed.read',
  readLayout: 'failed.read',
  tear: 'failed.tear',
  tearOff: 'failed.tear',
  navigate: 'failed.tear',
  putBack: 'failed.putBack',
  putBackAll: 'failed.putBack',
  arrange: 'failed.layout',
  place: 'failed.layout',
  order: 'failed.layout',
  created: 'failed.layout',
  setMode: 'failed.layout',
  setSnap: 'failed.layout',
  resetLayout: 'failed.layout',
  resetSizes: 'failed.layout',
  'resetSizes.undo': 'failed.layout',
  configure: 'failed.settings',
  edit: 'failed.edit',
  pin: 'failed.pin',
  makeBoard: 'failed.make',
  load: 'failed.hosts',
  readAliases: 'failed.aliases',
}

/** failedKey is the sentence for a failure at where ("board.pin"): what could not be done. */
export function failedKey(where: string): string {
  return SAID[where.slice(where.indexOf('.') + 1)] ?? 'failed.other'
}

/** said logs a failure at where and is what the window says of it. */
export function said(where: string, e: unknown): string {
  return report(where, e, (error) => t(failedKey(where), { error }))
}
