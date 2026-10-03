Closes #

## Why

<!-- What this changes and why. A sentence or two is enough. -->

## How I checked

<!-- What you ran or looked at: the tests you added, the board in which terminal, a pasted screen. If an AI agent wrote part of it, say what you read and checked yourself. -->

## Checklist

- [ ] The title is a Conventional Commit phrased for the release notes ("feat: follow linked logs", not "fix bug")
- [ ] One kind label: feature, enhancement, bug, breaking change, docs, plugin or chore
- [ ] `go vet ./... && go test -race ./...` pass, and `gofmt -l .` prints nothing
- [ ] For a change in behaviour: the README, and the agent guide or a skill when agents need to know, say what it does now
- [ ] No small fix without an issue unless it has the no-issue label
