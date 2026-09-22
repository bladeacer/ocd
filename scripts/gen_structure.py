#!/usr/bin/env python3
"""Generate a markdown-formatted project structure tree for AGENTS.md.

This script walks the project directory and produces a markdown file tree
that can be embedded in AGENTS.md. Run it whenever the project structure
changes to keep AGENTS.md up to date.

Usage:
    python3 gen_structure.py [directory] [--exclude PATTERN]
    python3 gen_structure.py . --exclude .git --exclude .obsidian_cache --exclude node_modules

The output is written to stdout as a markdown code block.
"""

import os
import sys
import argparse


# Files and directories to skip by default
DEFAULT_EXCLUDES = {
    ".git", ".obsidian_cache", ".headroom", ".lccst", "dist",
    "node_modules", "vendor", "__pycache__", ".cache", ".vscode",
    ".idea", ".DS_Store", "*.exe", "ocd", "cover.out",
    "coverage.out", "coverage.svg", ".gitignore",
}


def should_exclude(name, excludes):
    """Check if a file or directory should be excluded."""
    if name in excludes:
        return True
    for pattern in excludes:
        if pattern.startswith("*") and name.endswith(pattern[1:]):
            return True
        if name == pattern.strip():
            return True
    # Skip hidden files/dirs (starting with .) unless they're the project root
    if name.startswith(".") and name not in (".", ".."):
        return True
    return False


def get_structure(directory, excludes):
    """Walk the directory and build a nested structure."""
    # Sort entries: directories first, then files, both alphabetically
    entries = sorted(os.listdir(directory))
    result = []

    for entry in entries:
        if should_exclude(entry, excludes):
            continue
        path = os.path.join(directory, entry)
        if os.path.isdir(path):
            children = get_structure(path, excludes)
            if children:
                result.append(f"📁 {entry}/")
                for child in children:
                    result.append(f"    {child}")
            else:
                result.append(f"📁 {entry}/")
        else:
            result.append(f"📄 {entry}")

    return result


def generate_markdown_tree(directory, excludes):
    """Generate markdown formatted tree."""
    lines = []
    lines.append(f"```")
    lines.append(f"ocd/")

    structure = get_structure(directory, excludes)
    for item in structure:
        # Indent already handled by get_structure
        lines.append(item)

    lines.append(f"```")
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(
        description="Generate a markdown project structure tree."
    )
    parser.add_argument(
        "directory",
        nargs="?",
        default=".",
        help="Root directory to walk (default: current directory)"
    )
    parser.add_argument(
        "--exclude",
        action="append",
        default=[],
        help="Pattern to exclude (can be specified multiple times)"
    )
    args = parser.parse_args()

    excludes = set(DEFAULT_EXCLUDES)
    excludes.update(args.exclude)

    # Resolve to absolute path
    directory = os.path.abspath(args.directory)
    name = os.path.basename(directory)

    # Replace "ocd/" prefix with the actual directory name
    lines = []
    lines.append(f"```")
    lines.append(f"{name}/")

    structure = get_structure(directory, excludes)
    for item in structure:
        lines.append(item)

    lines.append(f"```")
    print("\n".join(lines))


if __name__ == "__main__":
    main()
