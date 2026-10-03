# Rules

- Ask on the board only what you cannot decide yourself, and only once.
- One question, or a few that belong together, per form. A question has a
  heading; its options follow it.
- Name buttons by what they do (`[ Deploy ] [ Cancel ]`), never `[ OK ]`.
- Put the consequence of an option in the indented line under it, not in
  chat.
- Always wait with `--timeout`. When it runs out (exit code 3), say so and
  take the safe path or ask in chat; never guess the answer.
- To ask again, write the form without the `submitted` keys. A form that
  still has them answers at once with the old choice.
- The answer is the user's. Read it with `stickypane wait` or `answers`;
  do not read the file and interpret marks yourself.
