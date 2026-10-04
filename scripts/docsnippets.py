#!/usr/bin/env python3
"""Fill the reference-solution snippets in docs/*.html from the tagged code.

A snippet is a pair of HTML comments in a page:

    <!-- snippet: lesson-1.5 internal/venue/domain/venue.go#RegisterVenue,Venue.Activate -->
    ...generated, do not edit...
    <!-- /snippet -->

The part after the tag is a file path, optionally followed by "#" and a
comma-separated list of Go declarations to extract (with their doc comments):
"Name" for a func, type, var or const, and "Recv.Name" for a method.
Without "#", the whole file is used. "#L10-L20" takes a line range.

A resume line says where a lesson starts and where its finished code is:

    <!-- resume: lesson-1.4 lesson-1.5 -->
    ...generated...
    <!-- /resume -->

Links use commit SHAs, so they never move even if a tag is re-pointed.

The code is read with `git show <tag>:<path>`, so the tags of the solution
branch must exist locally:

    git fetch origin solution && sh scripts/solution-tags.sh

    python3 scripts/docsnippets.py          # rewrite the snippets in place
    python3 scripts/docsnippets.py --check  # exit 1 if any page is out of date
"""

import html
import pathlib
import re
import subprocess
import sys

REPO_URL = "https://github.com/williamokano/go-ddd-by-example"
SNIPPET = re.compile(
    r"(?P<open><!-- snippet: (?P<tag>\S+) (?P<path>[^#\s]+)(?:#(?P<sel>\S+))? -->)"
    r".*?(?P<close><!-- /snippet -->)",
    re.S,
)
RESUME = re.compile(
    r"(?P<open><!-- resume: (?P<start>\S+) (?P<end>\S+) -->).*?(?P<close><!-- /resume -->)",
    re.S,
)


def git(*args: str) -> str:
    try:
        return subprocess.run(["git", *args], check=True, capture_output=True, text=True).stdout
    except subprocess.CalledProcessError as e:
        sys.exit(f"git {' '.join(args)} failed: {e.stderr.strip()}\n"
                 "Hint: git fetch origin solution && sh scripts/solution-tags.sh")


def sha(ref: str) -> str:
    return git("rev-parse", "--short=12", f"{ref}^{{commit}}").strip()


def git_show(tag: str, path: str) -> str:
    return git("show", f"{tag}:{path}")


def decl_start(lines: list[str], name: str) -> int:
    if "." in name:
        recv, meth = name.split(".", 1)
        pat = re.compile(rf"^func \(\w+ \*?{re.escape(recv)}\) {re.escape(meth)}\(")
    else:
        pat = re.compile(rf"^(func {re.escape(name)}\(|(type|var|const) {re.escape(name)}\b)")
    for i, line in enumerate(lines):
        if pat.match(line):
            return i
    sys.exit(f"declaration {name!r} not found")


def extract(src: str, sel: str) -> str:
    lines = src.rstrip("\n").split("\n")
    m = re.fullmatch(r"L(\d+)-L(\d+)", sel)
    if m:
        return "\n".join(lines[int(m.group(1)) - 1 : int(m.group(2))])
    blocks = []
    for name in sel.split(","):
        start = decl_start(lines, name)
        first = start
        while first > 0 and lines[first - 1].startswith("//"):
            first -= 1
        end = start
        opener = lines[start].rstrip()
        if opener.endswith("{") or opener.endswith("("):
            closer = "}" if opener.endswith("{") else ")"
            while lines[end] != closer:
                end += 1
        blocks.append("\n".join(lines[first : end + 1]))
    return "\n\n".join(blocks)


def render(m: re.Match) -> str:
    tag, path, sel = m["tag"], m["path"], m["sel"]
    src = git_show(tag, path)
    code = extract(src, sel) if sel else src.rstrip("\n")
    link = f"{REPO_URL}/blob/{sha(tag)}/{path}"
    return (
        f"{m['open']}\n"
        f'<div class="snippet-src"><a href="{link}"><code>{html.escape(path)}</code></a>'
        f" at <code>{html.escape(tag)}</code></div>\n"
        f"<pre><code>{html.escape(code, quote=False)}</code></pre>\n"
        f"{m['close']}"
    )


def render_resume(m: re.Match) -> str:
    start, end = m["start"], m["end"]
    count = int(git("rev-list", "--first-parent", "--count", f"{start}..{end}"))
    tree = f"{REPO_URL}/tree/{sha(end)}"
    diff = f"{REPO_URL}/compare/{sha(start)}...{sha(end)}"
    return (
        f"{m['open']}\n"
        f'<p class="resume">Start this lesson from <code>git checkout {html.escape(start)}</code>.'
        f' Finished: <a href="{tree}"><code>{html.escape(end)}</code></a>'
        f' (<a href="{diff}">this lesson\'s {count} commits</a>).</p>\n'
        f"{m['close']}"
    )


def main() -> int:
    check = "--check" in sys.argv[1:]
    stale = []
    for page in sorted(pathlib.Path("docs").glob("*.html")):
        old = page.read_text()
        new = RESUME.sub(render_resume, SNIPPET.sub(render, old))
        if new != old:
            stale.append(str(page))
            if not check:
                page.write_text(new)
    if check and stale:
        print("out of date (run make docs-snippets):", *stale, sep="\n  ")
        return 1
    if not check:
        print("updated:", *(stale or ["nothing"]), sep="\n  ")
    return 0


if __name__ == "__main__":
    sys.exit(main())
