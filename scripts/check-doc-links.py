#!/usr/bin/env python3
"""Check that relative links in Markdown files resolve.

Checks file links ([text](other.md), ![img](pic.png)) and heading anchors
([text](other.md#heading), [text](#heading)). Anchors use GitHub's algorithm,
because GitHub renders these files. External links (http, https, mailto) are
not checked. Links inside fenced code blocks and inline code are ignored.

Usage: scripts/check-doc-links.py [files...]
Without arguments, every tracked *.md file is checked. Exits 1 if a link is
broken, with one line "<file>:<line>: broken link <target> (<reason>)" each.
"""

import re
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent

HEADING_RE = re.compile(r"^(#{1,6})\s+(.*?)\s*#*\s*$")
FENCE_RE = re.compile(r"^\s*(```|~~~)")
INLINE_CODE_RE = re.compile(r"`[^`]*`")
# [text](target) and ![alt](target); the target ends at whitespace or ")".
LINK_RE = re.compile(r"\[[^\]]*\]\(\s*<?([^)\s>]+)>?(?:\s+\"[^\"]*\")?\s*\)")
EXTERNAL = ("http://", "https://", "mailto:")


def github_anchor(heading: str) -> str:
    """Reproduce GitHub's heading-to-anchor conversion."""
    text = re.sub(r"\[([^\]]*)\]\([^)]*\)", r"\1", heading)  # [text](url) -> text
    text = text.replace("`", "")
    text = re.sub(r"\*{1,3}(.+?)\*{1,3}", r"\1", text)  # *bold* / *italic*
    # _emphasis_ only at word boundaries: CGO_ENABLED keeps its underscore.
    text = re.sub(r"(?<!\w)_{1,3}(.+?)_{1,3}(?!\w)", r"\1", text)
    text = text.replace("<", "").replace(">", "").lower()
    text = re.sub(r"[^\w\s-]", "", text)
    return re.sub(r"\s", "-", text.strip())


def outside_fences(path: Path):
    """Yield (line_number, line) for every line outside fenced code blocks."""
    in_fence = False
    for lineno, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
        if FENCE_RE.match(line):
            in_fence = not in_fence
            continue
        if not in_fence:
            yield lineno, line


def collect_anchors(path: Path) -> set[str]:
    """Every anchor GitHub generates for this file; repeated headings get -1, -2, ..."""
    anchors: set[str] = set()
    seen: dict[str, int] = {}
    for _, line in outside_fences(path):
        match = HEADING_RE.match(line)
        if not match:
            continue
        base = github_anchor(match.group(2))
        if not base:
            continue
        count = seen.get(base, 0)
        seen[base] = count + 1
        anchors.add(base if count == 0 else f"{base}-{count}")
    return anchors


def check_files(files: list[Path], root: Path) -> list[str]:
    """Return one error line per broken link in files (paths shown relative to root)."""
    anchor_cache: dict[Path, set[str]] = {}

    def anchors_of(path: Path) -> set[str]:
        if path not in anchor_cache:
            anchor_cache[path] = collect_anchors(path)
        return anchor_cache[path]

    errors = []
    for path in files:
        path = path.resolve()
        if not path.exists():  # deleted in the commit that is being checked
            continue
        rel = path.relative_to(root.resolve())
        for lineno, line in outside_fences(path):
            for target in LINK_RE.findall(INLINE_CODE_RE.sub("", line)):
                if target.startswith(EXTERNAL):
                    continue
                file_part, _, anchor = target.partition("#")
                resolved = (path.parent / file_part).resolve() if file_part else path
                if not resolved.exists():
                    errors.append(f"{rel}:{lineno}: broken link {target} (no such file)")
                elif anchor and resolved.suffix == ".md" and anchor not in anchors_of(resolved):
                    errors.append(f"{rel}:{lineno}: broken link {target} (no such heading)")
    return errors


def select_markdown(names: list[str], root: Path) -> list[Path]:
    # git ls-files still lists files that are deleted but not yet committed.
    return [root / name for name in names if (root / name).exists()]


def tracked_markdown() -> list[Path]:
    out = subprocess.run(
        ["git", "ls-files", "*.md"], cwd=REPO_ROOT, capture_output=True, text=True, check=True
    ).stdout
    return select_markdown(out.splitlines(), REPO_ROOT)


def main(argv: list[str]) -> int:
    files = [Path(a) for a in argv] if argv else tracked_markdown()
    errors = check_files(files, REPO_ROOT)
    for error in errors:
        print(error, file=sys.stderr)
    if errors:
        print(f"{len(errors)} broken link(s) in {len(files)} file(s)", file=sys.stderr)
        return 1
    print(f"links ok in {len(files)} Markdown file(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
