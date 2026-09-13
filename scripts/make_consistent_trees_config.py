#!/usr/bin/env python3

# AI-GENERATED-BEGIN: v=1; task="extract Consistent-Trees columns and types from an ASCII tree header"; reviewer="Phil Mansfield"; tool="Codex"; review="confirmed"

# Phil's note: yes, I know this is simple, but I didn't have a lot of time.

"""Print a Consistent-Trees ASCII file's columns, types, and first data line."""

import argparse
import re
import sys
from pathlib import Path


INT_COLUMNS = {
    "id",
    "desc_id",
    "num_prog",
    "pid",
    "upid",
    "desc_pid",
    "phantom",
    "mmp?",
    "breadth_first_id",
    "depth_first_id",
    "tree_root_id",
    "orig_halo_id",
    "snap_num",
    "next_coprogenitor_depthfirst_id",
    "last_progenitor_depthfirst_id",
}


def parse_header(path: Path) -> tuple[list[str], int]:
    """Return the column names and the one-based first data-line number."""
    columns: list[str] | None = None

    with path.open(encoding="utf-8") as file:
        for line_number, line in enumerate(file, start=1):
            text = line.strip()
            if columns is None:
                if not text.startswith("#"):
                    continue

                tokens = text[1:].split()
                if tokens and re.fullmatch(r"scale\(0\)", tokens[0], re.IGNORECASE):
                    columns = [re.sub(r"\(\d+\)$", "", token) for token in tokens]
                continue

            if text.startswith("#") or not text:
                continue

            if len(text.split()) == len(columns):
                return columns, line_number

    if columns is None:
        raise ValueError("could not find the Consistent-Trees column header")
    raise ValueError("could not find a data line matching the column count")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tree_file", type=Path, help="ASCII Consistent-Trees file")
    args = parser.parse_args()

    try:
        columns, data_line = parse_header(args.tree_file)
    except (OSError, ValueError) as error:
        print(f"{args.tree_file}: {error}", file=sys.stderr)
        return 1

    print(f"HeaderSize={data_line}")
    print("Separator= ")
    print("---")
    for column in columns:
        column_type = "int" if column.lower() in INT_COLUMNS else "float"
        print(f"{column} {column_type}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())

# AI-GENERATED-END
