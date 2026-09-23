#!/usr/bin/env python3
"""Merge Go statement coverage profiles produced with mode=set."""

import sys
from pathlib import Path


def merge(paths: list[Path]) -> list[str]:
    blocks: dict[str, tuple[int, int]] = {}
    for path in paths:
        lines = path.read_text().splitlines()
        if not lines or lines[0] != "mode: set":
            raise ValueError(f"{path}: expected mode: set coverage profile")
        for line in lines[1:]:
            location, statements, count = line.split()
            statements, count = int(statements), int(count)
            previous = blocks.get(location)
            if previous is not None and previous[0] != statements:
                raise ValueError(f"{path}: statement count differs for {location}")
            blocks[location] = (statements, max(previous[1] if previous else 0, count))
    return ["mode: set", *(f"{loc} {stmts} {count}" for loc, (stmts, count) in sorted(blocks.items()))]


if __name__ == "__main__":
    if len(sys.argv) < 4:
        raise SystemExit("usage: merge-go-coverage.py OUTPUT INPUT INPUT...")
    output = Path(sys.argv[1])
    output.write_text("\n".join(merge([Path(value) for value in sys.argv[2:]])) + "\n")
