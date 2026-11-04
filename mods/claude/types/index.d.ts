// A note as `stickypane index --json` lists it.
export type Note = {
  tab: string
  tab_title: string
  name: string
  kind: string
  icon: string
  title: string
  summary?: string // what the note counts: 2/5, 7 cards
  gist?: string // a line: what to look at first
  open: boolean
  modified: string
  links?: string[]
  backlinks?: string[]
}

// The note the pane shows in full, or none for the list.
export type Page = { name: string; title: string; text: string }

declare module 'claude-code' {
  interface PluginState {
    'board-mod': { notes: Note[]; page: Page | null }
  }
}
