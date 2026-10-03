# Vision

stickypane is a board in your terminal that your coding agent writes to
and you read. These are the principles decisions are checked against. When
a feature fails one of them, it is not added.

1. **The folder is the product.** A note is a file the user owns, readable
   without stickypane, in the project's repository. There is no database,
   no server and no state that is not a file.
2. **The agent needs nothing new.** Writing a Markdown file is enough; the
   commands exist so that an agent never has to read a file to change one
   thing in it. The guide the agent reads fits on one screen.
3. **The user acts on the board, and the agent sees it.** Ticking an item,
   moving a card, answering a form changes the same file the agent reads.
   Nothing the user does is told to the agent in words.
4. **Nothing to configure.** No server, no hooks, no terminal plugin, no
   config file to write by hand. The one settings file is written by the
   board as the user arranges it.
5. **Any terminal, any agent.** A plain window, tmux, WezTerm, Zellij;
   Claude Code, Codex, anything that can edit files.
6. **The screen is calm.** Fixed panes that scroll inside, one accent for
   focus, muted everything else. A note is never cut and never hidden.

## What it is not

Not a task manager with its own data model, not a chat client, not a
replacement for the agent's own UI, and not a general dashboard framework.
