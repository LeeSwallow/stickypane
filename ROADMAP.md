# Roadmap

**The goal: show it in the terminal, simply, with no desktop.** Everything
an agent produces for you to look at — a plan, progress, a structure, a
number, a log, a question — should be one command away from being on the
screen next to the agent, over ssh as well as at your desk, with nothing to
install but one binary. Every item below is measured against that.

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

A web or desktop view, a server, a database, accounts, sync between
machines. The board is files in a folder next to the agent; that is the
point.
