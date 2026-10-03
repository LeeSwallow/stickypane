# Changelog

All notable changes are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the release notes
on GitHub are built from pull request labels.

## [Unreleased]

The first version. stickypane shows the files of a project's `.sticky/`
folder as a board in a terminal pane, for a coding agent and its user to
share.

### Added

- Notes as plain files: Markdown notes with boards, checklists, charts and
  forms by front matter; `.log` files followed live; `.sh` scripts with a
  Run button; folders as books that scroll as one note; Mermaid blocks
  drawn as diagrams.
- A screen of fixed panes that scroll inside; the arrangement (open, size,
  color, pin, order, names, theme) in `sticky.json`.
- Keys and mouse: focus, open, zoom, resize, reorder, move to a folder,
  trash with undo, a built-in vi-like editor, ten color themes.
- A command line and an MCP server for agents: `show`, one-thing edits
  (`todo`, `card`, `chart`, `log`, `set`), `rm`/`mv`/`restore`/`archive`/
  `link`, forms with `answers` and `wait`.
- Setup on first use: `stickypane` in a git repository makes the board and
  puts the agent guide where each agent reads it, or nowhere when the
  plugin teaches it; `stickypane init` does the same on its own.
- A Claude Code and Codex plugin with three skills, five commands and a
  session hook.

### Known limits

macOS and Linux only; a program that keeps a log file open and appends to
it stops being followed after the board edits that file; no counts or
visual mode in the editor.
