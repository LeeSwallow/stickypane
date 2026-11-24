---
case: worktree-B4
description: Use when you work in a worktree and another worktree of the same repository may change the same files, so a merge would clash.
---

# Two worktrees, one file

Shapes: chat.

```sh
stickypane worktree list                   # the other branches
comm -12 <(git diff --name-only main...HEAD | sort) <(git diff --name-only main...table | sort)
stickypane chat add chat --as chart "chart and table both change render/axis.go; one of us should wait"
```

Compare the files your branch changed with each other branch's, from
the same base. When they share a file, say so in the chat, naming the
file and both worktrees, and let the user decide the order. Do not
change the other branch.

Done when: every shared file is named in the chat, and you changed
nothing outside your worktree.
