// A read that is asked for again while it runs: the calls during one run
// are folded into one more run after it, never run side by side (the board
// window's and a torn note's refresh).

/**
 * fold makes pass a call that runs it at most once at a time: a call
 * while it runs gets the running promise and asks for one more pass
 * after it, however many calls came. pass catches its own failures; one
 * it throws ends the run and is the promise's.
 */
export function fold(pass: () => Promise<void>): () => Promise<void> {
  let running: Promise<void> | null = null
  let again = false
  return () => {
    if (running) {
      again = true
      return running
    }
    running = (async () => {
      try {
        do {
          again = false
          await pass()
        } while (again)
      } finally {
        running = null
      }
    })()
    return running
  }
}
