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
PUBLISH_DISPATCH_PAYLOAD_FIELDS = {"project", "tag"}
DISPATCH_CONFIRMATION = "${{ github.event_name == 'repository_dispatch' && format('publish {0} {1} to production', github.event.client_payload.project, github.event.client_payload.tag) || inputs.production_confirmation }}"


def validate_publish_dispatch_contract(text: str) -> list[str]:
    """Return violations in the trusted consumer dispatch boundary."""
    violations: list[str] = []
    required_fragments = (
        (
            "publish workflow must accept only the publish-package dispatch type",
            "  repository_dispatch:\n    types: [publish-package]",
            1,
        ),
        (
            "consumer dispatch must map only its project into every privileged job",
            "PROJECT: ${{ github.event_name == 'repository_dispatch' && github.event.client_payload.project || inputs.project }}",
            3,
        ),
        (
            "consumer dispatch must map only its tag into every privileged job",
            "TAG: ${{ github.event_name == 'repository_dispatch' && github.event.client_payload.tag || inputs.tag }}",
            3,
        ),
        (
            "consumer dispatch must always enter staging",
            "github.event_name == 'repository_dispatch' || inputs.apply_staging",
            2,
        ),
        (
            "consumer dispatch must always continue to production",
            "github.event_name == 'repository_dispatch' || inputs.apply_production",
            2,
        ),
        (
            "consumer dispatch must not expose staging deletion",
            "EMPTY_STAGING: ${{ github.event_name == 'workflow_dispatch' && inputs.empty_staging }}",
            2,
        ),
        (
            "consumer dispatch confirmation must be derived from project and tag",
            f"PRODUCTION_CONFIRMATION: {DISPATCH_CONFIRMATION}",
            2,
        ),
        (
            "consumer dispatch payload shape must be validated before publication",
            'run: python3 scripts/validate_publish_event.py "$GITHUB_EVENT_PATH"',
            1,
        ),
        (
            "manual production confirmation must be derived from project and tag",
            'expected_confirmation="publish $PROJECT $TAG to production"',
            1,
        ),
        (
            "staging must use only its protected R2 prefix variable",
            "R2_PREFIX: ${{ vars.R2_PREFIX }}",
            1,
        ),
    )
    for message, fragment, count in required_fragments:
        if text.count(fragment) != count:
            violations.append(message)

    payload_fields = set(
        re.findall(r"github\.event\.client_payload\.([A-Za-z0-9_-]+)", text)
    )
    if payload_fields != PUBLISH_DISPATCH_PAYLOAD_FIELDS:
        violations.append(
            "repository dispatch payload must be confined to project and tag"
        )

    return violations


def validate_publish_script_contract(text: str) -> list[str]:
    """Return violations in the protected publisher's local fail-closed guards."""
    violations: list[str] = []
    required_fragments = (
        (
            "protected publisher must independently validate project and tag",
            "go run ./cmd/meigma-packages validate-request",
        ),
        (
            "package version must be derived from exactly one validated leading v",
            '"$TAG" != "v$package_version"',
        ),
        (
            "production confirmation must be derived from validated project and tag",
            'production_confirmation="publish $validated_project $validated_tag to production"',
        ),
        (
            "staging publication must remain isolated to _staging/",
            '"${R2_PREFIX:-}" != \'_staging/\'',
        ),
        (
            "production publication must require an empty R2 prefix",
            '[[ -n "${R2_PREFIX:-}" ]]',
        ),
        (
            "production publication must use the protected root mode",
            "apply_arguments+=(--production-root)",
        ),
        (
            "APT clean installs must assert the validated package version",
            'dpkg-query --show --showformat="\\${Version}" "$PACKAGE_NAME"',
        ),
        (
            "RPM clean installs must assert the validated package version",
            'rpm --query --queryformat "%{VERSION}" "$PACKAGE_NAME"',
        ),
    )
    for message, fragment in required_fragments:
        if fragment not in text:
            violations.append(message)
    if "incus-gh-runner" in text or "v1.0.0" in text:
        violations.append("protected publisher must not pin a project or release")

    return violations


def validate_workflow(path: Path) -> list[str]:
    """Return policy violations found in one workflow."""
    text = path.read_text(encoding="utf-8")
    lines = text.splitlines()
    violations: list[str] = []

    if path.name == "publish.yml":
        violations.extend(validate_publish_dispatch_contract(text))

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
            if '[[ "$PRODUCTION_CONFIRMATION" == "$expected_confirmation" ]]' not in text:
                violations.append("production publication requires the derived exact confirmation phrase")
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
    violations.extend(
        f"scripts/phase5-publish.sh: {violation}"
        for violation in validate_publish_script_contract(
            Path("scripts/phase5-publish.sh").read_text(encoding="utf-8")
        )
    )
    if violations:
        print("\n".join(violations), file=sys.stderr)
        return 1

    print(f"Workflow and image policy passed for {len(workflows)} workflow files.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
