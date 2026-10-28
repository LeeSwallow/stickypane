// What a failed call to the shell says, for the person (no runtime here,
// so pure code and tests can use it).

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
