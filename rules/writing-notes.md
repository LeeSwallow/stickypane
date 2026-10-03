# Writing notes the user reads on the board

- Write in the language the user writes in.
- A title is a noun phrase the user would say: "Auth refactor", not
  "Progress tracking for the authentication refactoring task".
- A checklist item or a card is one action: verb first, under ten words.
- A log line is one event, past tense, with the number that matters:
  "tests passed, 3 flaky left".
- A note that explains something is Markdown with headings; a Mermaid block
  for structure; a chart for numbers. Not prose where a list will do.
- Do not copy values that live elsewhere (file paths, command outputs,
  issue numbers) into a note by hand; write the command that produces them
  or link the file with `stickypane show`.
