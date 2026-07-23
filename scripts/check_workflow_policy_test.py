#!/usr/bin/env python3
"""Regression tests for protected publication workflow policy."""

from pathlib import Path
import tempfile
import unittest

from check_workflow_policy import validate_publish_script_contract, validate_workflow


class PublishWorkflowPolicyTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.workflow = Path(".github/workflows/publish.yml").read_text(encoding="utf-8")
        cls.publisher = Path("scripts/phase5-publish.sh").read_text(encoding="utf-8")

    def workflow_violations(self, text: str) -> list[str]:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "publish.yml"
            path.write_text(text, encoding="utf-8")
            return validate_workflow(path)

    def test_accepts_generalized_v1_1_0_contract(self) -> None:
        self.assertIn("default: v1.1.0", self.workflow)
        self.assertEqual([], self.workflow_violations(self.workflow))
        self.assertEqual([], validate_publish_script_contract(self.publisher))

    def test_rejects_mismatched_confirmation_contract(self) -> None:
        changed = self.workflow.replace(
            'expected_confirmation="publish $PROJECT $TAG to production"',
            'expected_confirmation="publish incus-gh-runner v1.0.0 to production"',
        )
        self.assertIn(
            "manual production confirmation must be derived from project and tag",
            self.workflow_violations(changed),
        )

    def test_rejects_staging_bypass(self) -> None:
        changed = self.workflow.replace("      - staging\n", "", 1)
        self.assertIn(
            "production publication must depend on validation and staging",
            self.workflow_violations(changed),
        )

    def test_rejects_protected_r2_target_changes(self) -> None:
        changed_workflow = self.workflow.replace("      R2_PREFIX: ''", "      R2_PREFIX: releases/")
        self.assertIn(
            "production publication requires an explicit empty R2 prefix",
            self.workflow_violations(changed_workflow),
        )

        changed_publisher = self.publisher.replace("'_staging/'", "'releases/'", 1)
        self.assertIn(
            "staging publication must remain isolated to _staging/",
            validate_publish_script_contract(changed_publisher),
        )


if __name__ == "__main__":
    unittest.main()
