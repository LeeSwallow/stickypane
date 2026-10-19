// board-mod: the stickypane board inside Claude Code. It asks the board
// only through the stickypane command line, as an agent would, and writes
// nothing: the pane reads the notes, and what the user changes on the
// board comes back as a toast and the status line.

import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

import type { Note, Page } from '../types'
import { asMarkdown, describe, lines, parse, tabs } from './board'

const PANE = 'stickypane'
const notes = atom({ plugin: 'board-mod', key: 'notes' } as const, [])
const page = atom({ plugin: 'board-mod', key: 'page' } as const, null)

// After one of the agent's own tool calls ends, events for this long are
// taken as its doing, since the watch hears a change a moment after it.
const QUIET_MS = 1500

// How many of the agent's tool calls run now, and when the last one ended.
let running = 0
let quietUntil = 0

// Refresh reads the notes again; false when there is no board here.
async function refresh($: EngineInterface): Promise<boolean> {
  try {
    const listed = await $.process.run(['stickypane', 'index', '--json'])
    if (listed.exitCode !== 0) return false
    const value: Note[] = JSON.parse(listed.stdout)
    await update($, notes, () => value)
    return true
  } catch {
    return false
  }
}

// Follow runs the watch for the session's life and tells the user what
// they changed on the board.
async function follow($: EngineInterface): Promise<void> {
  let rest = ''
  for await (const piece of $.process.spawn({ argv: ['stickypane', 'watch', '--json'] })) {
    if (piece.stream !== 'stdout') continue
    const split = lines(rest + piece.text)
    rest = split.rest
    let changed = false
    for (const line of split.whole) {
      const event = parse(line)
      if (!event) continue
      changed = true
      const isAgents = running > 0 || (await $.clock.now()) < quietUntil
      const said = isAgents ? undefined : describe(event, await read($, notes))
      if (said) {
        $.ui.toast(said)
        $.ui.status(`board: ${said}`)
      }
    }
    if (changed) await refresh($)
  }
}

// OpenNote shows a note's file in the pane in place of the list.
async function openNote($: EngineInterface, note: Note): Promise<void> {
  const cat = await $.process.run(['stickypane', 'cat', note.name])
  const text = cat.exitCode === 0 ? cat.stdout : cat.stderr
  await update($, page, () => ({ name: note.name, title: note.title, text }))
}

export const register: Register = on => {
  on('session.start', async ($, e, next) => {
    const started = await next(e)
    await $.command.register({
      name: 'stickypane',
      description: 'Show the stickypane board of this project in a pane',
    })
    if (await refresh($)) void follow($).catch(err => $.ui.log(`board-mod: the watch stopped: ${err}`, { to: "debug" }))
    return started
  })

  on('command.run', { command: 'stickypane' }, async $ => {
    if (!(await refresh($))) {
      return { text: 'No stickypane board here: run `stickypane init`, or install stickypane.' }
    }
    await update($, page, () => null)
    await $.ui.open({ id: PANE, title: 'stickypane' })
    return { text: 'The board is open in a pane.' }
  })

  on('tool.call', async ($, e, next) => {
    running += 1
    try {
      return await next(e)
    } finally {
      running -= 1
      quietUntil = (await $.clock.now()) + QUIET_MS
    }
  }).catch(($, e, next) => next(e))

  // What the user saw in the status line goes once they write again.
  on('prompt.submit', ($, e, next) => {
    $.ui.status(undefined)
    return next(e)
  }).catch(($, e, next) => next(e))

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e) => {
    const { Box, Button, Markdown, Text } = $.ui.resolve(e)
    const shown: Page | null = await read($, page)

    if (shown) {
      return (
        <Box flexDirection="column">
          <Box>
            <Button key="back" label="← notes" hotkey="b" onPress={() => update($, page, () => null)} />
            <Text bold> {shown.title}</Text>
          </Box>
          <Markdown text={asMarkdown(shown.name, shown.text)} />
        </Box>
      )
    }

    const list: Note[] = await read($, notes)
    if (list.length === 0) return <Text dimColor>The board is empty.</Text>

    return (
      <Box flexDirection="column">
        {tabs(list).map(tab => (
          <Box key={`tab:${tab.title}`} flexDirection="column">
            <Text bold>{tab.title}</Text>
            {tab.notes.map(note => (
              <Button key={note.name} plain onPress={() => openNote($, note)}>
                {note.icon} {note.title} <Text dimColor>{note.gist}</Text>
              </Button>
            ))}
          </Box>
        ))}
      </Box>
    )
  })
}
