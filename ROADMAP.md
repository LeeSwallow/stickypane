# Roadmap

**The goal: show it in the terminal, simply.** Everything an agent produces
for you to look at — a plan, progress, a structure, a number, a log, a
question — should be one command away from being on the screen next to the
agent, over ssh as well as at your desk, with nothing to install but one
binary. Every item below is measured against that. The desktop app is a
second screen of the same folder for those who want one; it never replaces
the terminal.

The open questions that decide the order are pinned in
[Discussions](https://github.com/LeeSwallow/stickypane/discussions).

## Now: one command puts anything on the screen

Done when: an agent can show any of these with one line and no reading
first, and the user can tell what it is at a glance.

- [x] `stickypane show <file or folder>` for anything in the project
- [x] Notes, kanban boards, checklists, charts (bar, trend, calendar), Mermaid
      diagrams, forms with buttons, logs followed live, scripts with a Run
      button, folders as books
- [x] One-line changes that never read the file: `todo`, `card`, `chart`,
      `log`, `set`
- [x] The screen tiles itself; the user never arranges before seeing
- [ ] Tables (`| a | b |`) drawn to fit, sortable by column
- [ ] Trees (indented lists) that fold and unfold
- [ ] A status line: what the agent is doing now, one line, always visible
- [ ] Watch a command: `stickypane watch "go test ./..." --every 30s` makes
      a log or a chart from its output, the way sampler does
- [ ] Korean user interface (hints, messages, help); the agent guide stays
      English

## Next: the user answers, the agent reads it

Done when: a decision never has to be typed into the chat, and the agent
never has to ask twice.

- [x] Forms: choices, text, buttons; `stickypane wait` hands the answer back
- [ ] Buttons that send a message to the agent (`ask:`) and show that it
      was received
- [ ] An inbox the agent drains: everything the user pressed since it last
      looked, in order
- [ ] A form that updates in place as the agent narrows the question

## Next: everywhere a terminal is

Done when: the board looks the same in tmux over ssh on a Linux box as in
WezTerm on a Mac, and a 40-column pane is still usable.

- [x] Plain window, tmux, WezTerm, Zellij; mouse where the terminal passes it
- [x] Ten themes that never paint the background, so any terminal's own
      colors stay
- [ ] Resizing panes in four directions, by key and by mouse edge
- [ ] A one-note mode: `stickypane plan` shows one note full screen, for a
      tiny pane or a popup
- [ ] Windows

## Next: the desktop app ships

Done when: someone who has never used a terminal board installs the app from
a DMG, makes a board, connects their agent and answers its first question
without reading a page of docs.

- [x] Every shape drawn in a window, `.html` pages in a sandbox
- [x] Clicks that are the terminal's keys: tick, answer, move a card, run
- [x] A free canvas, notes torn off into windows of their own, a hotkey
- [x] One install: the app carries `stickypane`, installs the terminal
      command and connects the agents
- [ ] The first release: a DMG and a Linux archive on the releases page,
      and a Homebrew cask for the app
- [ ] Signed and notarized on macOS, so Gatekeeper does not ask
- [ ] Checked on a real Linux desktop: tray, hotkey, windows on top

## Later: the agent as a first-class user

Done when: an agent that has never seen stickypane does the right thing
with the plugin alone.

- [x] A Claude Code and Codex plugin: skills, commands, a session hook
- [x] MCP server with the same commands
- [ ] Usage charts fed by the agent's own session logs (Claude Code, Codex),
      read the way ccusage reads them
- [ ] Several agents on one board, each with its own status line
- [ ] A test suite of "an agent was asked X; the board should show Y"

## Not planned

A web view, a server, accounts, sync between machines. The board is files
in a folder next to the agent; that is the point. (The desktop app keeps an
index of your boards in `~/.stickypane`; the files stay the truth.)
