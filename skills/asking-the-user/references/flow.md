# Flow

1. Write the form into the notes folder (shape in `form.md`), then show it:

   ```sh
   stickypane show deploy
   ```

2. Wait for a button, with a limit:

   ```sh
   stickypane wait deploy --timeout 10m
   ```

   Output, one line per question:

   ```
   submitted: Deploy
   at: 2026-10-02T14:03:05+09:00
   Target: production
   Also: run migrations; clear the cache
   Note: after lunch
   ```

   `--json` gives `{"submitted":true,"button":"Deploy","answers":[...]}`.
   Exit code 3: the time ran out; stdout is empty. Exit code 1: the note is
   not a form or does not exist.

3. Report the answer in one line and act on it in the caller. Then
   `stickypane hide deploy`, or leave the form open as the record of the
   decision.

Reading without waiting: `stickypane answers deploy` prints the same lines
now, with `submitted: no` until a button was pressed.
