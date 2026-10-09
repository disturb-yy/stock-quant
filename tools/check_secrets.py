#!/usr/bin/env python3
"""Find likely committed credentials without printing matched values."""

from __future__ import annotations

import argparse
import os
import re
import sys
from pathlib import Path


SECRET_ASSIGNMENT = re.compile(
    r"(?i)\b(?P<key>TUSHARE_TOKEN|MYSQL_DSN|DB_PASSWORD|"
    r"[A-Z0-9_]*(?:PASSWORD|SECRET|API[_-]?KEY|ACCESS[_-]?TOKEN|DSN))"
    r"\b[ \t]*[\"']?[ \t]*[:=][ \t]*[\"']?(?P<value>[A-Z0-9_./+=@:-]{8,})"
)
PLACEHOLDERS = {
    "changeme",
    "change-me",
    "example",
    "placeholder",
    "replace-me",
    "your-token",
    "your_token",
}
SKIP_DIRS = {".git", ".venv", "node_modules", "vendor", "__pycache__"}


def find_secrets(content: str) -> list[tuple[int, str]]:
    """Return line and key pairs for non-placeholder secret assignments."""
    findings = []
    for match in SECRET_ASSIGNMENT.finditer(content):
        value = match.group("value")
        if value.lower() in PLACEHOLDERS or value.startswith("${"):
            continue
        line = content.count("\n", 0, match.start()) + 1
        findings.append((line, match.group("key")))
    return findings


def format_finding(path: str, line: int, key: str) -> str:
    """Format a finding without including the matched value."""
    return f"potential secret assignment: {path}:{line} ({key}); value redacted"


def scan_root(root: Path) -> list[str]:
    if not root.is_dir():
        raise NotADirectoryError(root)
    findings = []

    def fail_on_walk_error(error: OSError) -> None:
        raise error

    for current, dirs, files in os.walk(root, followlinks=False, onerror=fail_on_walk_error):
        dirs[:] = [name for name in dirs if name not in SKIP_DIRS]
        for name in files:
            path = Path(current, name)
            if path.is_symlink():
                continue
            try:
                content = path.read_text(encoding="utf-8")
            except UnicodeDecodeError:
                continue
            relative = path.relative_to(root).as_posix()
            findings.extend(
                format_finding(relative, line, key)
                for line, key in find_secrets(content)
            )
    return findings


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("root", nargs="?", type=Path, default=Path(__file__).resolve().parent.parent)
    args = parser.parse_args()

    try:
        findings = scan_root(args.root.resolve())
    except OSError as error:
        path = error.filename or str(args.root)
        print(f"secret scan blocked: unable to read {path}", file=sys.stderr)
        return 2
    if findings:
        for finding in findings:
            print(finding, file=sys.stderr)
        return 1
    print("secret scan passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
