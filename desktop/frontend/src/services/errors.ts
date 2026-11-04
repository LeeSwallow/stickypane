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

/** What the shell sends for an error it knows (shell.ErrorWords): a code the window says in its own words. */
export type ErrorWords = { code: string; values?: Record<string, string> }

/** errorWords is the code of a failed call to the shell, from its cause; null for an error the shell does not know. */
export function errorWords(e: unknown): ErrorWords | null {
  const c = (e as { cause?: unknown } | null)?.cause
  if (c && typeof c === 'object' && typeof (c as ErrorWords).code === 'string') return c as ErrorWords
  return null
}

// The window's own words for an error with a code (i18n sets it): null
// when it has none, and the English of the shell is said instead.
let sayWords: (w: ErrorWords) => string | null = () => null

/** setErrorWords is how the catalogs say an error with a code. */
export function setErrorWords(f: (w: ErrorWords) => string | null): void {
  sayWords = f
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
 * report keeps a failure in the log, under where it happened (in the
 * shell's English), and is what to tell the person: an error the shell
 * knows in the window's language, any other as it came, in the sentence
 * say makes of it when given.
 */
export function report(where: string, e: unknown, say?: (why: string) => string): string {
  const text = errorText(e)
  log(where, text, e)
  const w = errorWords(e)
  const why = (w && sayWords(w)) || text
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
