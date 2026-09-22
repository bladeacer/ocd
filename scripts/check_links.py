#!/usr/bin/env python3
"""Check all markdown files for broken links."""

import os
import re
import sys
import urllib.request
import urllib.error
from pathlib import Path


def find_md_files(root: Path) -> list[Path]:
    """Find all .md files in the project, excluding hidden directories."""
    return sorted(
        f for f in root.rglob("*.md")
        if not any(part.startswith(".") for part in f.parts)
    )


def parse_links(content: str) -> list[tuple[str, str, int]]:
    """Parse markdown links [text](url) and return (url, text, line_number)."""
    links = []
    lines = content.splitlines()
    pattern = re.compile(r"\[([^\]]*)\]\(([^)]+)\)")
    for i, line in enumerate(lines, start=1):
        for match in pattern.finditer(line):
            text, url = match.group(1), match.group(2)
            links.append((url, text, i))
    return links


def is_external(url: str) -> bool:
    """Check if a URL is external (HTTP/HTTPS)."""
    return url.startswith("http://") or url.startswith("https://")


def check_internal_link(base_file: Path, url: str) -> str | None:
    """Check if an internal link resolves. Returns error message or None."""
    # Strip anchor fragments for file existence check
    target = url.split("#")[0]
    if not target:
        return None

    # Resolve relative to the directory containing the source file
    target_path = (base_file.parent / target).resolve()

    # Also check from project root
    root_path = (base_file.parent.parent / target).resolve()

    if target_path.is_file():
        return None
    if root_path.is_file():
        return None

    return f"  File not found: {target}"


def check_external_link(url: str) -> str | None:
    """Check if an external link returns a non-error status. Returns error message or None."""
    # Strip anchor for the request
    clean_url = url.split("#")[0]
    try:
        req = urllib.request.Request(clean_url, headers={"User-Agent": "ocd-check-links/1.0"})
        with urllib.request.urlopen(req, timeout=10) as response:
            status = response.status
            if status >= 400:
                return f"  HTTP {status}: {clean_url}"
    except urllib.error.HTTPError as e:
        return f"  HTTP {e.code}: {clean_url}"
    except urllib.error.URLError as e:
        return f"  URL error: {clean_url} ({e.reason})"
    except Exception as e:
        return f"  Error: {clean_url} ({e})"
    return None


def main():
    project_root = Path(__file__).resolve().parent.parent
    scripts_dir = Path(__file__).resolve().parent

    # Use project root as base, but also check scripts dir for any .md files there
    md_files = find_md_files(project_root)

    broken_links = []
    total_links = 0
    external_checked = 0

    print(f"Scanning {len(md_files)} markdown files...\n")

    for md_file in md_files:
        try:
            content = md_file.read_text(encoding="utf-8")
        except Exception as e:
            print(f"ERROR reading {md_file}: {e}")
            continue

        links = parse_links(content)
        for url, text, line_num in links:
            total_links += 1

            if is_external(url):
                external_checked += 1
                error = check_external_link(url)
                if error:
                    broken_links.append((md_file, line_num, url, text, error))
            else:
                error = check_internal_link(md_file, url)
                if error:
                    broken_links.append((md_file, line_num, url, text, error))

    # Print results
    if broken_links:
        print(f"Found {len(broken_links)} broken link(s) out of {total_links} total links "
              f"({external_checked} external checked):\n")
        for md_file, line_num, url, text, error in broken_links:
            rel_path = md_file.relative_to(project_root)
            print(f"{rel_path}:{line_num}  [{text}]({url})")
            print(error)
            print()
        sys.exit(1)
    else:
        print(f"All {total_links} links are valid ({external_checked} external checked).")
        sys.exit(0)


if __name__ == "__main__":
    main()
