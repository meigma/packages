#!/usr/bin/env python3
"""Validate the exact trusted repository-dispatch payload contract."""

import json
from pathlib import Path
import sys
from typing import Any


EXPECTED_FIELDS = {"project", "tag"}


def validate_event(event: Any) -> None:
    """Require an object whose client payload contains only project and tag strings."""
    if not isinstance(event, dict):
        raise ValueError("repository dispatch event must be an object")
    payload = event.get("client_payload")
    if not isinstance(payload, dict) or set(payload) != EXPECTED_FIELDS:
        raise ValueError("repository dispatch payload must contain only project and tag")
    for field in sorted(EXPECTED_FIELDS):
        if not isinstance(payload[field], str) or payload[field] == "":
            raise ValueError(f"repository dispatch {field} must be a non-empty string")


def main() -> int:
    """Load and validate the GitHub event file named on the command line."""
    if len(sys.argv) != 2:
        print("usage: validate_publish_event.py EVENT_PATH", file=sys.stderr)
        return 2
    try:
        event = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
        validate_event(event)
    except (OSError, json.JSONDecodeError, ValueError) as error:
        print(error, file=sys.stderr)
        return 1

    print("Repository dispatch payload is confined to project and tag.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
