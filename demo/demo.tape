# A VHS tape (https://github.com/charmbracelet/vhs): `vhs demo/demo.tape`
# makes demo/demo.gif. It needs stickypane on the PATH and runs in a fresh
# temporary project.

Output demo/demo.gif
Set FontSize 14
Set Width 1100
Set Height 640
Set Theme "Catppuccin Mocha"
Set Padding 12

Hide
Type "cd $(mktemp -d) && git init -q && stickypane init --no-agent-docs >/dev/null" Enter
Type "stickypane todo plan add 'read the request' >/dev/null; stickypane todo plan add 'write the tests' >/dev/null; stickypane todo plan add 'ship it' >/dev/null" Enter
Type "stickypane card work add 'login API' --to Doing >/dev/null; stickypane card work add 'payments' >/dev/null" Enter
Type "printf -- '---\ntype: form\ntitle: Deploy now?\n---\n## Target\n- ( ) staging\n- ( ) production\n\n[ Deploy ] [ Cancel ]\n' > .sticky/deploy.md" Enter
Type "clear" Enter
Show

Type "stickypane show deploy && stickypane todo plan check tests" Enter
Sleep 1s
Type "stickypane" Enter
Sleep 2s
Type "tab" Sleep 500ms
Type "j" Sleep 300ms
Type " " Sleep 1s
Type "tab" Sleep 500ms
Type "j" Sleep 300ms
Type "enter" Sleep 800ms
Type "j" Sleep 300ms
Type "j" Sleep 300ms
Type "enter" Sleep 1.5s
Type "q"
Sleep 500ms
Type "stickypane answers deploy" Enter
Sleep 2s
