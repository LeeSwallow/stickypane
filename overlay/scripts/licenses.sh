#!/bin/sh
# licenses.sh: check that every Go module and npm package of the overlay app
# has an allowed license. It lists the ones that do not, by name, and exits 1.
#
#   scripts/licenses.sh [--go-dir DIR | --no-go] [--npm-dir DIR | --no-npm] [--write FILE]
#
# With --write, and only when the check passes, it also writes FILE: the
# license text of every Go module linked into the app (`go list -deps .`)
# and every npm package shipped in it (the dependencies, not the
# devDependencies, plus svelte, whose runtime the bundle carries, and its
# dependencies), each as the package ships it. The release images carry it
# as licenses/stickypane-app.txt (docs/overlay/09 §9.1).
#
# Go modules come from `go list -m -json all`; the license is read from the
# module's LICENSE file. npm packages come from `npm ls --all --parseable`;
# the license is the `license` field of each package.json (an SPDX
# expression: "A OR B" passes when one side is allowed, "A AND B" when both).
#
# Allowed: MIT, BSD-2-Clause, BSD-3-Clause, Apache-2.0, ISC, OFL-1.1.
# Also allowed, because build tools in the current tree need them; each is
# listed in ALLOWED_EXTRA below with its reason. Add one only with a reason.
# MPL-2.0 is allowed only for lightningcss (see BUILD_ONLY), a tool of Vite
# that is not part of the app, and golang-lru (see GO_TEST_ONLY), which only
# a test of modernc.org/libc imports.
set -eu

here="$(cd "$(dirname "$0")/.." && pwd)"
go_dir="$here"
npm_dir="$here/frontend"
write=""
while [ $# -gt 0 ]; do
	case "$1" in
	--go-dir) go_dir="$2"; shift 2 ;;
	--npm-dir) npm_dir="$2"; shift 2 ;;
	--no-go) go_dir=""; shift ;;
	--no-npm) npm_dir=""; shift ;;
	--write) write="$2"; shift 2 ;;
	*) echo "usage: licenses.sh [--go-dir DIR | --no-go] [--npm-dir DIR | --no-npm] [--write FILE]" >&2; exit 2 ;;
	esac
done

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

if [ -n "$go_dir" ]; then
	(cd "$go_dir" && go list -m -json all) >"$work/go.json"
	(cd "$go_dir" && go list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}} {{.Version}} {{.Dir}}{{end}}{{end}}' .) |
		sort -u >"$work/go-ship.txt"
else
	: >"$work/go.json"
	: >"$work/go-ship.txt"
fi
if [ -n "$npm_dir" ]; then
	# npm ls exits non-zero for an extraneous or invalid tree; the list is
	# still what we check.
	(cd "$npm_dir" && npm ls --all --parseable 2>/dev/null) >"$work/npm.txt" || true
	(cd "$npm_dir" && npm ls --omit=dev --all --parseable 2>/dev/null) >"$work/npm-ship.txt" || true
else
	: >"$work/npm.txt"
	: >"$work/npm-ship.txt"
fi

python3 -I - "$work/go.json" "$work/npm.txt" "$go_dir" "$work/go-ship.txt" "$work/npm-ship.txt" "$npm_dir" "$write" <<'PY'
import json, os, re, sys

go_json, npm_list = sys.argv[1], sys.argv[2]
go_dir_arg = sys.argv[3]
go_ship, npm_ship, npm_dir_arg, write = sys.argv[4], sys.argv[5], sys.argv[6], sys.argv[7]
# devDependencies whose code the bundle carries: the Svelte runtime.
BUNDLED_DEV = ("svelte",)

ALLOWED = {"MIT", "BSD-2-Clause", "BSD-3-Clause", "Apache-2.0", "ISC", "OFL-1.1"}
ALLOWED_EXTRA = {
    "0BSD",           # tslib, a TypeScript helper library
    "MIT-0",          # @csstools/*, test-time dependencies of vitest
    "CC0-1.0",        # mdn-data, a data set used by css-tree at test time
    "BlueOak-1.0.0",  # lru-cache, a dependency of the test tools
    "Unlicense",      # robust-predicates (mermaid -> d3 -> d3-delaunay): a public-domain dedication
}
OK = ALLOWED | ALLOWED_EXTRA
# Packages that may carry MPL-2.0: file-level copyleft, fine for a build tool
# that is run but neither linked nor shipped.
BUILD_ONLY = {"MPL-2.0": ("lightningcss",)}
# Go modules that are MPL-2.0 and in the module graph only through the tests
# of a dependency: modernc.org/libc.test imports golang-lru, but no package
# of the app or of its dependencies does (`go list -deps` does not list it).
GO_TEST_ONLY = ("github.com/hashicorp/golang-lru/v2",)

def expr_ok(expr):
    """True when the SPDX expression is satisfied by the allowed set."""
    expr = expr.strip()
    if not expr:
        return False
    # Tokenize parentheses, AND, OR, ids.
    tokens = re.findall(r"\(|\)|[^\s()]+", expr)
    pos = 0
    def parse_or():
        nonlocal pos
        v = parse_and()
        while pos < len(tokens) and tokens[pos].upper() == "OR":
            pos += 1
            r = parse_and()
            v = v or r
        return v
    def parse_and():
        nonlocal pos
        v = parse_atom()
        while pos < len(tokens) and tokens[pos].upper() == "AND":
            pos += 1
            r = parse_atom()
            v = v and r
        return v
    def parse_atom():
        nonlocal pos
        if pos >= len(tokens):
            return False
        t = tokens[pos]
        if t == "(":
            pos += 1
            v = parse_or()
            if pos < len(tokens) and tokens[pos] == ")":
                pos += 1
            return v
        pos += 1
        return t.rstrip("+") in OK
    try:
        v = parse_or()
    except Exception:
        return False
    return v and pos == len(tokens)

def go_license(text):
    """Heuristic SPDX id of a LICENSE file's text, or None."""
    t = " ".join(text.split())
    low = t.lower()
    if re.search(r"gnu (affero |lesser )?general public license", low):
        return "GPL-family"
    if "apache license" in low and "version 2.0" in low:
        return "Apache-2.0"
    if "permission is hereby granted, free of charge" in low:
        return "MIT"
    if "permission to use, copy, modify, and/or distribute" in low or \
       "permission to use, copy, modify, and distribute" in low:
        return "ISC"
    if "redistribution and use in source and binary forms" in low:
        if "neither the name" in low or "the names of its contributors" in low or "endorse or promote" in low:
            return "BSD-3-Clause"
        return "BSD-2-Clause"
    if "sil open font license" in low and "1.1" in low:
        return "OFL-1.1"
    return None

bad = []

# The GO_TEST_ONLY exception holds only while no package of the app or of its
# dependencies imports those modules: fail if `go list -deps` lists one.
if go_dir_arg:
    import subprocess
    deps = subprocess.run(["go", "list", "-deps", "./..."], cwd=go_dir_arg,
                          capture_output=True, text=True).stdout.split()
    for m in GO_TEST_ONLY:
        if any(d == m or d.startswith(m + "/") for d in deps):
            bad.append(("go", m, "is linked into the build, so the test-only exception is void"))
checked_go = 0
skipped_go = []

# Go modules.
dec = json.JSONDecoder()
raw = open(go_json).read()
i = 0
while i < len(raw):
    while i < len(raw) and raw[i].isspace():
        i += 1
    if i >= len(raw):
        break
    mod, i = dec.raw_decode(raw, i)
    if mod.get("Main"):
        continue
    path, version = mod["Path"], mod.get("Version", "")
    d = mod.get("Dir")
    if not d or not os.path.isdir(d):
        skipped_go.append(path)  # not downloaded: not part of any build
        continue
    text = None
    for n in sorted(os.listdir(d)):
        if re.match(r"(?i)^(licen[cs]e|copying|unlicense)([._-].*)?$", n) and os.path.isfile(os.path.join(d, n)):
            with open(os.path.join(d, n), errors="replace") as f:
                text = f.read()
            break
    if path in GO_TEST_ONLY:
        continue
    checked_go += 1
    if text is None:
        bad.append(("go", f"{path}@{version}", "no license file"))
        continue
    spdx = go_license(text)
    if spdx is None or not expr_ok(spdx):
        bad.append(("go", f"{path}@{version}", spdx or "unrecognised license text"))

# npm packages.
checked_npm = 0
seen = set()
for line in open(npm_list):
    d = line.strip()
    if not d or d in seen:
        continue
    seen.add(d)
    pj = os.path.join(d, "package.json")
    if "node_modules" not in d.split(os.sep) or not os.path.isfile(pj):
        continue  # the project root itself
    with open(pj) as f:
        meta = json.load(f)
    name = f"{meta.get('name', os.path.basename(d))}@{meta.get('version', '?')}"
    lic = meta.get("license")
    if lic is None and meta.get("licenses"):
        lic = " OR ".join(l.get("type", "") if isinstance(l, dict) else str(l) for l in meta["licenses"])
    if isinstance(lic, dict):
        lic = lic.get("type", "")
    if not lic:
        # No license field (khroma, a dependency of mermaid): read the
        # package's license file, as for a Go module.
        for n in sorted(os.listdir(d)):
            if re.match(r"(?i)^(licen[cs]e|copying)([._-].*)?$", n) and os.path.isfile(os.path.join(d, n)):
                with open(os.path.join(d, n), errors="replace") as f:
                    lic = go_license(f.read())
                break
    checked_npm += 1
    only = BUILD_ONLY.get(str(lic), ())
    if lic and any(meta.get("name", "").startswith(p) for p in only):
        continue
    if not lic or not expr_ok(str(lic)):
        bad.append(("npm", name, str(lic) if lic else "no license field"))

if skipped_go:
    print(f"licenses: skipped {len(skipped_go)} go module(s) not in the module cache", file=sys.stderr)
for kind, name, why in bad:
    print(f"licenses: {kind} {name}: {why}", file=sys.stderr)
if bad:
    sys.exit(1)
print(f"licenses: ok ({checked_go} go modules, {checked_npm} npm packages)")

def license_file(d):
    """The package's license file, at its root or in dist/, or None."""
    for sub in ("", "dist"):
        dd = os.path.join(d, sub)
        if not os.path.isdir(dd):
            continue
        for n in sorted(os.listdir(dd)):
            if re.match(r"(?i)^(licen[cs]e|copying|unlicense)([._-].*)?$", n) and os.path.isfile(os.path.join(dd, n)):
                return os.path.join(dd, n)
    return None

MIT_TEXT = """Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE."""

if write:
    out = []
    def entry(title, d, fallback):
        f = license_file(d) if d else None
        out.append("\n\n" + "=" * 64 + "\n" + title + "\n" + "=" * 64 + "\n\n")
        if f:
            with open(f, errors="replace") as fh:
                out.append(fh.read())
        else:
            out.append(fallback + "\n")
    for line in open(go_ship):
        parts = line.split()
        if len(parts) < 3:
            continue
        path, version, d = parts[0], parts[1], parts[2]
        entry(f"{path} {version} (Go)", d, "No license file in the module; see its repository.")
    dirs = []
    for line in open(npm_ship):
        d = line.strip()
        if d and "node_modules" in d.split(os.sep) and d not in dirs:
            dirs.append(d)
    if npm_dir_arg:
        nm = os.path.join(npm_dir_arg, "node_modules")
        for name in BUNDLED_DEV:
            d = os.path.join(nm, name)
            pj = os.path.join(d, "package.json")
            if not os.path.isfile(pj):
                continue
            with open(pj) as f:
                deps = list(json.load(f).get("dependencies", {}))
            for dd in [d] + [os.path.join(nm, x) for x in deps]:
                if os.path.isdir(dd) and dd not in dirs:
                    dirs.append(dd)
    for d in sorted(set(dirs)):
        with open(os.path.join(d, "package.json")) as f:
            meta = json.load(f)
        lic = meta.get("license")
        if isinstance(lic, dict):
            lic = lic.get("type", "")
        repo = meta.get("repository")
        if isinstance(repo, dict):
            repo = repo.get("url", "")
        author = meta.get("author") or ""
        if isinstance(author, dict):
            author = author.get("name", "")
        fallback = f"License: {lic or 'unstated'}. The package ships no license file; see {repo or 'its repository'}."
        if lic == "MIT":
            fallback = (f"The package ships no license file; its package.json says MIT, by {author or 'its authors'}"
                        f" ({repo or 'see its repository'}). The MIT License:\n\n"
                        f"Copyright (c) {author or 'the authors'}\n\n" + MIT_TEXT)
        entry(f"{meta.get('name', os.path.basename(d))} {meta.get('version', '?')} (npm)", d, fallback)
    os.makedirs(os.path.dirname(os.path.abspath(write)), exist_ok=True)
    with open(write, "w") as f:
        f.write("stickypane-app is MIT licensed (LICENSE). It carries the Go modules and npm\n")
        f.write("packages below; each license is reproduced as the package ships it.\n")
        f.write("".join(out))
    print(f"licenses: wrote {write}")
PY
