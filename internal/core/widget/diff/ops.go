package diff

// A diff has no writer: it is view-only, so there is no doc.Op for a screen
// to apply and Kind.Act is nil. The worktree and the base the user chooses
// are kept by the board in sticky.json, not in the note.
