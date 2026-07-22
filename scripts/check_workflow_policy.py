#!/usr/bin/env python3
"""Enforce unprivileged, staging, and production workflow trust boundaries."""

from pathlib import Path
import re
import sys


WORKFLOW_ROOT = Path(".github/workflows")
FULL_SHA = re.compile(r"^[0-9a-f]{40}$")
USES = re.compile(r"^\s*(?:-\s+)?uses:\s*([^@\s]+)@([^\s#]+)")
IMAGE_DIGEST = re.compile(r"@sha256:[0-9a-f]{64}$")
IMAGE_ASSIGNMENT = re.compile(r"^[a-z0-9_]+_image=['\"]?([^'\"\s]+)")
PRIVILEGED_WORKFLOWS = {"publish.yml"}


def validate_workflow(path: Path) -> list[str]:
    """Return policy violations found in one workflow."""
    text = path.read_text(encoding="utf-8")
    lines = text.splitlines()
    violations: list[str] = []

    if not re.search(r"(?m)^permissions: \{\}$", text):
        violations.append("top-level permissions must be empty")
    if re.search(r"(?m)^\s+(?:pull_request_target|workflow_run):", text):
        violations.append("privileged pull-request triggers are forbidden")
    has_secrets = "${{ secrets." in text
    has_environment = re.search(r"(?m)^\s+environment:\s*", text) is not None
    if (has_secrets or has_environment) and path.name not in PRIVILEGED_WORKFLOWS:
        violations.append("secrets and deployment environments require an approved privileged workflow")
    if has_secrets or has_environment:
        staging_marker = "\n  staging:\n"
        staging_offset = text.find(staging_marker)
        if staging_offset == -1:
            violations.append("privileged configuration requires a dedicated staging job")
        else:
            unprivileged_jobs = text[:staging_offset]
            if "${{ secrets." in unprivileged_jobs or re.search(
                r"(?m)^\s+environment:\s*", unprivileged_jobs
            ):
                violations.append("secrets and environments must be confined to the staging job")
        if not re.search(r"(?m)^\s+name:\s*staging\s*$", text):
            violations.append("privileged publish workflows must use the staging environment")
        if not re.search(r"(?m)^\s+needs:\s*validate\s*$", text):
            violations.append("the staging job must depend on read-only validation")
        if re.search(r"(?m)^\s+(?:pull_request|pull_request_target|workflow_run):", text):
            violations.append("privileged workflows must not use pull-request-derived triggers")
        production_marker = "\n  production:\n"
        production_offset = text.find(production_marker)
        if production_offset != -1:
            if production_offset < staging_offset:
                violations.append("the production job must follow the staging job")
            if not re.search(r"(?m)^\s+name:\s*production\s*$", text):
                violations.append("the production job must use the production environment")
            production_block = text[production_offset:]
            if not re.search(
                r"(?m)^\s+group:\s*meigma-packages-r2-production\s*$",
                production_block,
            ):
                violations.append("production publication requires serialized concurrency")
            if "publish incus-gh-runner v1.0.0 to production" not in text:
                violations.append("production publication requires an exact confirmation phrase")
            if "R2_PREFIX: ''" not in production_block:
                violations.append("production publication requires an explicit empty R2 prefix")
            if not re.search(
                r"(?ms)^\s+needs:\s*\n\s+-\s+validate\s*\n\s+-\s+staging\s*$",
                production_block,
            ):
                violations.append("production publication must depend on validation and staging")
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
