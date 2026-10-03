#!/usr/bin/env python3
"""screenshots.py [board-dir]: photograph a stickypane board as PNG files.

It runs the board in a tmux session of a fixed size, presses keys to reach
each view, captures the screen with its colors (tmux capture-pane -e),
turns it into an HTML page and has headless Chrome save that as a PNG in
demo/screenshots/. Build a board to photograph with demo/showcase.sh.
Needs tmux, stickypane on the PATH, and Google Chrome or Chromium.
"""
import html, os, re, shutil, subprocess, sys, tempfile, time

COLS, ROWS = 150, 44
HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "screenshots")
SESSION = "stickypane-shots"

# Each shot: a file name, and the keys pressed from the first tab to get there.
SHOTS = [
    ("1-board", []),
    ("2-release", ["2"]),
    ("3-notes", ["3"]),
    ("4-index", ["1", "i"]),
    ("5-settings", ["Escape", "S"]),
    ("6-api", ["Escape", "4", "j"]),
]

BASE16 = ["#45475a", "#f38ba8", "#a6e3a1", "#f9e2af", "#89b4fa", "#f5c2e7", "#94e2d5", "#bac2de",
          "#585b70", "#f38ba8", "#a6e3a1", "#f9e2af", "#89b4fa", "#f5c2e7", "#94e2d5", "#a6adc8"]
FG, BG = "#cdd6f4", "#1e1e2e"  # catppuccin-mocha, the demo's theme


def color256(n):
    if n < 16:
        return BASE16[n]
    if n < 232:
        n -= 16
        steps = [0, 95, 135, 175, 215, 255]
        return "#%02x%02x%02x" % (steps[n // 36], steps[n // 6 % 6], steps[n % 6])
    v = 8 + (n - 232) * 10
    return "#%02x%02x%02x" % (v, v, v)


def to_html(ansi):
    out, style = [], {}

    def span(text):
        if not text:
            return
        fg, bg = style.get("fg"), style.get("bg")
        if style.get("rev"):
            fg, bg = bg or BG, fg or FG
        css = []
        if fg: css.append("color:" + fg)
        if bg: css.append("background:" + bg)
        if style.get("bold"): css.append("font-weight:700")
        if style.get("faint"): css.append("opacity:.6")
        if style.get("italic"): css.append("font-style:italic")
        if style.get("under"): css.append("text-decoration:underline")
        t = html.escape(text)
        out.append(f'<span style="{";".join(css)}">{t}</span>' if css else t)

    pos = 0
    for m in re.finditer(r"\x1b\[([0-9;:]*)m", ansi):
        span(ansi[pos:m.start()])
        pos = m.end()
        codes = [c for c in re.split("[;:]", m.group(1))] or ["0"]
        i = 0
        while i < len(codes):
            c = int(codes[i] or 0)
            if c == 0: style = {}
            elif c == 1: style["bold"] = True
            elif c == 2: style["faint"] = True
            elif c == 3: style["italic"] = True
            elif c == 4: style["under"] = True
            elif c == 7: style["rev"] = True
            elif c == 22: style.pop("bold", None); style.pop("faint", None)
            elif c == 23: style.pop("italic", None)
            elif c == 24: style.pop("under", None)
            elif c == 27: style.pop("rev", None)
            elif 30 <= c <= 37: style["fg"] = BASE16[c - 30]
            elif 90 <= c <= 97: style["fg"] = BASE16[c - 82]
            elif 40 <= c <= 47: style["bg"] = BASE16[c - 40]
            elif 100 <= c <= 107: style["bg"] = BASE16[c - 92]
            elif c == 39: style.pop("fg", None)
            elif c == 49: style.pop("bg", None)
            elif c in (38, 48):
                key = "fg" if c == 38 else "bg"
                if i + 1 < len(codes) and codes[i + 1] == "2":
                    r, g, b = (int(x or 0) for x in codes[i + 2:i + 5])
                    style[key] = "#%02x%02x%02x" % (r, g, b)
                    i += 4
                elif i + 1 < len(codes) and codes[i + 1] == "5":
                    style[key] = color256(int(codes[i + 2]))
                    i += 2
            i += 1
    span(ansi[pos:])
    body = "".join(out)
    return f"""<!doctype html><meta charset="utf-8"><style>
body{{margin:0;background:{BG}}}
pre{{margin:0;padding:18px 20px;color:{FG};background:{BG};
font:15px/1.22 "JetBrainsMono Nerd Font Mono","JetBrains Mono","SF Mono",Menlo,monospace;
font-variant-ligatures:none;white-space:pre}}</style><pre>{body}</pre>"""


def chrome():
    for c in ["/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
              shutil.which("google-chrome"), shutil.which("chromium"), shutil.which("chromium-browser")]:
        if c and os.path.exists(c):
            return c
    sys.exit("needs Google Chrome or Chromium")


def tmux(*args):
    return subprocess.run(["tmux", *args], capture_output=True, text=True, check=True).stdout


def main():
    board = sys.argv[1] if len(sys.argv) > 1 else sys.exit("usage: screenshots.py <board-dir>")
    os.makedirs(OUT, exist_ok=True)
    subprocess.run(["tmux", "kill-session", "-t", SESSION], capture_output=True)
    tmux("new-session", "-d", "-s", SESSION, "-x", str(COLS), "-y", str(ROWS),
         f"cd {board!r} && TERM=xterm-256color COLORTERM=truecolor stickypane")
    time.sleep(2.5)
    browser, tmp = chrome(), tempfile.mkdtemp()
    try:
        for name, keys in SHOTS:
            for k in keys:
                tmux("send-keys", "-t", SESSION, k)
                time.sleep(0.6)
            time.sleep(0.8)
            page = os.path.join(tmp, name + ".html")
            with open(page, "w") as f:
                f.write(to_html(tmux("capture-pane", "-e", "-p", "-t", SESSION).rstrip("\n")))
            png = os.path.join(OUT, name + ".png")
            subprocess.run([browser, "--headless=new", "--disable-gpu", "--hide-scrollbars",
                            "--force-device-scale-factor=2", f"--window-size={COLS * 9 + 40},{ROWS * 18 + 40}",
                            f"--screenshot={png}", "file://" + page], capture_output=True, check=True)
            print("wrote", os.path.relpath(png))
    finally:
        subprocess.run(["tmux", "kill-session", "-t", SESSION], capture_output=True)
        shutil.rmtree(tmp, ignore_errors=True)


if __name__ == "__main__":
    main()
