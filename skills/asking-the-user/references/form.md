# Form

```markdown
---
type: form
title: Deploy now?
---
The build is green. Where should it go?

## Target
- ( ) staging
  Try it there first.
- ( ) production

## Also
- [ ] run migrations
- [ ] clear the cache

## Note
>

[ Deploy ] [ Cancel ]
```

| Line | Is |
| --- | --- |
| `- ( ) text` | one choice among the options under the same heading |
| `- [ ] text` | any number of choices |
| indented line under an option | its description |
| `> ` | a line the user types into |
| `[ Label ] [ Label ]` | the buttons; a form without any gets `[ Submit ]` |

A blank line ends a question. Everything else is Markdown and is shown as
such.
