# `vhs demo/live.tape` makes demo/live.gif: an agent's shell on the left
# writes to the board on the right. Needs stickypane, tmux and vhs.

Output demo/live.gif
Set FontSize 14
Set Width 1600
Set Height 860
Set Theme "Catppuccin Mocha"
Set Padding 8
Set TypingSpeed 45ms

Hide
Type "sh demo/live-setup.sh $(mktemp -d)/acme" Enter
Sleep 7s
Type "clear" Enter
Show

Sleep 1.5s
Type "stickypane todo plan check endpoint" Enter
Sleep 1.5s
Type "stickypane card work move login --to Done" Enter
Sleep 1.5s
Type "stickypane log build --time 'go test ./... ok (42 tests)'" Enter
Sleep 2s
Type "stickypane say chat --as claude 'tests pass. ship it?'" Enter
Sleep 2.5s
Type "stickypane chart tokens set input 61200" Enter
Sleep 1s
Type "stickypane chart tokens set output 18400" Enter
Sleep 3s
Hide
Type "tmux -L live kill-server" Enter
