# Installing stickypane

stickypane is one program with no runtime dependencies. It runs on macOS,
Linux and Windows, in any terminal that shows Unicode and color: Terminal,
iTerm2, Ghostty, WezTerm, Kitty, Alacritty, Windows Terminal, and inside
tmux or Zellij. No special font is needed.

Back to the [README](../README.md).

## Homebrew

```sh
brew install --cask LeeSwallow/tap/stickypane
```

## The desktop app

Each release from v0.3.0 on carries the app beside the command line
program, on the [releases page](https://github.com/LeeSwallow/stickypane/releases):
a DMG for Apple silicon Macs and a tar.gz for Linux on amd64. It carries
its own `stickypane`, so it is the only thing to install.

macOS: open `stickypane-<version>-<arch>.dmg` and drag **Stickypane** to
**Applications**. The app is not notarized, so the first open is
held back: open **System Settings → Privacy & Security** and choose **Open
Anyway**. Then, in a board's settings, **Terminal and agents** installs the
`stickypane` command for your terminals and connects Claude Code and Codex.

Linux (GTK 4 and WebKitGTK 6.0 needed):

```sh
tar -xzf stickypane-app-<version>-linux-amd64.tar.gz
sh stickypane-app-<version>-linux-amd64/install.sh   # --uninstall takes it out
```

Or, for every user of the machine, the `.rpm` or the `.deb` beside it,
which brings GTK 4 and WebKitGTK 6.0 with it:

```sh
sudo dnf install ./stickypane-app-<version>-linux-amd64.rpm   # Fedora
sudo apt install ./stickypane-app-<version>-linux-amd64.deb   # Debian, Ubuntu
```

The app's own design and build notes are in [docs/desktop](desktop/README.md).

## An rpm or a deb

Each release has a package of the command line program for Fedora and its
kin (`.rpm`) and for Debian and Ubuntu (`.deb`), for `amd64` and `arm64`.
Download the one for your machine from the
[release](https://github.com/LeeSwallow/stickypane/releases), then:

```sh
sudo dnf install ./stickypane_<version>_linux_amd64.rpm   # Fedora
sudo apt install ./stickypane_<version>_linux_amd64.deb   # Debian, Ubuntu
```

It puts `stickypane` in `/usr/bin` and the licenses in
`/usr/share/licenses/stickypane`. There is no package repository yet, so
a newer version is installed the same way.

## A release archive

Every [release](https://github.com/LeeSwallow/stickypane/releases) has an
archive for each system: `darwin` (macOS), `linux` and `windows`, each for
`amd64` (Intel and AMD) and `arm64` (Apple silicon, ARM). Each holds the
program, the license, and the licenses of the libraries it is built from.

macOS and Linux:

```sh
VERSION=0.1.1 OS=linux ARCH=amd64      # or OS=darwin, ARCH=arm64
curl -LO "https://github.com/LeeSwallow/stickypane/releases/download/v$VERSION/stickypane_${VERSION}_${OS}_${ARCH}.tar.gz"
tar -xzf "stickypane_${VERSION}_${OS}_${ARCH}.tar.gz" stickypane
sudo install stickypane /usr/local/bin/   # or any folder on your PATH
```

On macOS, a program downloaded by a browser is held back by Gatekeeper;
`xattr -d com.apple.quarantine stickypane` lets it run. Homebrew does this
for you.

Windows, in PowerShell:

```powershell
$v = "0.1.1"
Invoke-WebRequest "https://github.com/LeeSwallow/stickypane/releases/download/v$v/stickypane_${v}_windows_amd64.zip" -OutFile stickypane.zip
Expand-Archive stickypane.zip -DestinationPath "$env:LOCALAPPDATA\stickypane"
# add it to your PATH, once
[Environment]::SetEnvironmentVariable("Path", "$env:Path;$env:LOCALAPPDATA\stickypane", "User")
```

Each release lists the archives' SHA-256 sums in `checksums.txt`; check one
with `shasum -a 256 <archive>` or, in PowerShell, `Get-FileHash <archive>`.

## With Go

```sh
go install github.com/LeeSwallow/stickypane/cmd/stickypane@latest
```

This needs Go 1.26 or later and puts the program in `$(go env GOPATH)/bin`.

## From source

```sh
git clone https://github.com/LeeSwallow/stickypane
cd stickypane
go build -o stickypane ./cmd/stickypane
```

## Check that it works

```sh
stickypane version     # stickypane 0.1.1
stickypane env         # the system, shell, pane tool, locale and editor it found
```

## Update and remove

Update the way you installed it: `brew upgrade --cask stickypane`, a newer
archive or package, or `go install ...@latest` again. To remove it, run
`stickypane setup --undo` if you installed the plugin (with the same
`--scope`, and `--mcp` if you registered the MCP server), then
`brew uninstall --cask stickypane`, `sudo dnf remove stickypane` (or
`apt remove`), or delete the program. Your notes stay
in each project's `.sticky/` folder; they are plain files, yours to keep or
delete.
