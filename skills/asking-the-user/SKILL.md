---
name: asking-the-user
description: Ask the user a question with choices or a short text answer on their stickypane board, as a form with buttons, and read the answer back. Use when a decision is the user's to make (which option, which target, go or stop), when a question has a handful of answers rather than a conversation, and when the user is watching the board rather than the chat.
---

# Asking the user on the board

A form is a Markdown note with controls. The user answers with a key press
and presses a button; the answer lands in the file and `stickypane wait`
hands it to you.

## How

1. Write the form. Keep it to one question, or a few that belong together.

   ```sh
   cat > .sticky/deploy.md <<'EOF2'
   ---
   type: form
   title: Deploy now?
   ---
   The build is green. Where should it go?

   ## Target
   - ( ) staging
     Try it there first.
   - ( ) production

   ## Note
   >

   [ Deploy ] [ Cancel ]
   EOF2
   stickypane show deploy
   ```

   `- ( )` lines are one choice, `- [ ]` lines any number, a `> ` line is
   text to type, `[ Label ]` are the buttons. The heading above a group
   names the question. A form without buttons gets `[ Submit ]`.

2. Wait for the answer, with a limit so you are never stuck:

   ```sh
   stickypane wait deploy --timeout 10m
   # submitted: Deploy
   # Target: production
   # Note: after lunch
   ```

   Exit code 3 means the time ran out: say so and go on with a safe
   default, or ask in chat. `--json` gives the same as JSON.

3. Act on the answer, then take the form off the screen (`stickypane hide
   deploy`) or leave it as a record.

## Rules

- Name buttons by what they do (`[ Deploy ] [ Cancel ]`), not `[ OK ]`.
- Put the consequence in the option's description line, not in chat.
- To ask again, rewrite the form without the `submitted` keys, or the old
  answer comes straight back.
- Do not ask on the board what you could decide yourself, and do not ask
  twice for the same thing.
