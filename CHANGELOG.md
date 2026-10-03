# Changelog

All notable changes are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the release notes
on GitHub are built from pull request labels.

## [Unreleased]

The first version. stickypane shows the files of a project's `.sticky/`
folder as a board in a terminal pane, for a coding agent and its user to
share.

### Added

- Notes as plain files: Markdown notes, and by front matter kanban boards,
  checklists, charts (bar, spark, heat), forms and chats; `.log` files
  followed live; `.sh` and `.ps1` scripts with a Run button; `.http`
  requests (experimental); folders as tabs and books; Mermaid diagrams.
- Times kept by the board, not the agent: when a note was made, an item
  ticked, a card moved, a message said; shown in the user's locale.
- Notes that meet: `[[links]]` followed with `f`, `![[embeds]]` drawn in
  place, charts computed `from:` other notes, and events (`stickypane
  watch`, MCP `wait_event`) that a command can react to.
- An index of every note (`i` on the board, `stickypane index`, MCP
  `index`), and a settings panel (`S`, `stickypane config`) where every
  setting has a default: theme, language, time format, editor, staleness.
- A screen of fixed panes that redraws when something changes, not on a
  tick; keys and mouse; a built-in vi-like editor, or any editor, terminal
  or GUI; ten color themes.
- A command line and an MCP server for agents: `show`, one-thing edits
  (`todo`, `card`, `chart`, `log`, `say`, `set`), `rm`/`mv`/`restore`/
  `archive`/`link`, forms with `answers` and `wait`.
- A Claude Code and Codex plugin in layers: commands that name a skill,
  five skills (the way in, tracking progress, asking the user, talking in
  a chat, connecting notes) and the rules they share. The program carries
  the same skills for MCP clients, as prompts and `guide` topics.
- macOS, Linux and Windows; tmux, WezTerm, Zellij and Windows Terminal
  panes; sh and PowerShell.

### Known limits

A program that keeps a log file open and appends to it stops being followed
after the board edits that file; no counts or visual mode in the editor.
