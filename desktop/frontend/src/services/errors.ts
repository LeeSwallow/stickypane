// What a failed call to the shell says, for the person (no runtime here,
// so pure code and tests can use it), and where every failure goes: a
// failure the window tells is said once, in the person's words, and kept
// in the log; one it does not tell (a check that will be asked again, an
// older bridge without a call) is kept in the log only, never dropped.

/** errorText is what a failed call says, without the wrapping the runtime adds. */
export function errorText(e: unknown): string {
  let s = e instanceof Error ? e.message : String(e)
  s = s.replace(/^Error:\s*/, '')
  try {
    const j = JSON.parse(s)
    if (j && typeof j.message === 'string') return j.message
  } catch {
    // not JSON: the text as it is
  }
  return s
}

/** A Log is where failures are written: the console, or a test's own. */
export type Log = (where: string, why: string, e: unknown) => void

let log: Log = (where, why, e) => console.warn(`stickypane: ${where}: ${why}`, e)

/** setLog puts another log in place (a test); it returns the one it replaced. */
export function setLog(l: Log): Log {
  const was = log
  log = l
  return was
}

/**
 * report keeps a failure in the log, under where it happened, and is
 * what to tell the person: errorText, or the sentence key makes of it
 * (its {error} filled in) when a translate function is given.
 */
export function report(where: string, e: unknown, say?: (why: string) => string): string {
  const why = errorText(e)
  log(where, why, e)
  return say ? say(why) : why
}

/**
 * logged is a catch for a call whose failure the window does not tell:
 * it is written to the log and the promise settles to fallback.
 */
export function logged<T = undefined>(where: string, fallback?: T): (e: unknown) => T {
  return (e) => {
    report(where, e)
    return fallback as T
  }
}
