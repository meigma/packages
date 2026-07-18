#!/usr/bin/env python3
"""Enforce the secrets-free workflow and pinned-image boundary for Phase 3."""

from pathlib import Path
import re
import sys


WORKFLOW_ROOT = Path(".github/workflows")
FULL_SHA = re.compile(r"^[0-9a-f]{40}$")
USES = re.compile(r"^\s*(?:-\s+)?uses:\s*([^@\s]+)@([^\s#]+)")
IMAGE_DIGEST = re.compile(r"@sha256:[0-9a-f]{64}$")
IMAGE_ASSIGNMENT = re.compile(r"^[a-z0-9_]+_image=['\"]?([^'\"\s]+)")


def validate_workflow(path: Path) -> list[str]:
    """Return policy violations found in one workflow."""
    text = path.read_text(encoding="utf-8")
    lines = text.splitlines()
    violations: list[str] = []

    if not re.search(r"(?m)^permissions: \{\}$", text):
        violations.append("top-level permissions must be empty")
    if re.search(r"(?m)^\s+(?:pull_request_target|workflow_run):", text):
        violations.append("privileged pull-request triggers are forbidden")
    if "${{ secrets." in text:
        violations.append("secrets are forbidden before Phase 4")
    if re.search(r"(?m)^\s+environment:\s*", text):
        violations.append("deployment environments are forbidden before Phase 4")
    if re.search(r"(?m)^\s+[a-z-]+:\s*write\s*$", text):
        violations.append("write permissions are forbidden before Phase 4")
    if "self-hosted" in text:
        violations.append("self-hosted runners are forbidden for unprivileged workflows")

    for index, line in enumerate(lines):
        match = USES.match(line)
        if match is None:
            continue
        action, revision = match.groups()
        if action.startswith("./"):
            continue
        if FULL_SHA.fullmatch(revision) is None:
            violations.append(f"line {index + 1}: {action} must use a full commit SHA")
        if action == "actions/checkout":
            checkout_block = "\n".join(lines[index + 1 : index + 8])
            if not re.search(r"(?m)^\s+persist-credentials:\s*false\s*$", checkout_block):
                violations.append(
                    f"line {index + 1}: checkout must disable credential persistence"
                )

    return violations


def validate_container_images() -> list[str]:
    """Return unpinned external container images used by proof scripts."""
    violations: list[str] = []
    paths = [Path("spikes/phase0/Dockerfile.tools")]
    paths.extend(sorted(Path("scripts").glob("*.sh")))
    paths.extend(sorted(Path("spikes/phase0").glob("*.sh")))

    for path in paths:
        for index, line in enumerate(path.read_text(encoding="utf-8").splitlines()):
            image = ""
            if line.startswith("FROM "):
                image = line.removeprefix("FROM ").split()[0]
            else:
                match = IMAGE_ASSIGNMENT.match(line)
                if match is not None:
                    image = match.group(1)
            if image == "" or image.endswith(":local"):
                continue
            if IMAGE_DIGEST.search(image) is None:
                violations.append(
                    f"{path}: line {index + 1}: external image must use a SHA-256 digest"
                )

    return violations


def main() -> int:
    """Validate every checked-in workflow and print actionable failures."""
    workflows = sorted(WORKFLOW_ROOT.glob("*.yml")) + sorted(
        WORKFLOW_ROOT.glob("*.yaml")
    )
    violations = [
        f"{path}: {violation}"
        for path in workflows
        for violation in validate_workflow(path)
    ]
    violations.extend(validate_container_images())
    if violations:
        print("\n".join(violations), file=sys.stderr)
        return 1

    print(f"Workflow and image policy passed for {len(workflows)} workflow files.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
