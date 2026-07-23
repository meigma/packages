#!/usr/bin/env python3
"""Tests for the trusted repository-dispatch payload boundary."""

import unittest

from validate_publish_event import validate_event


class ValidatePublishEventTest(unittest.TestCase):
    def test_accepts_incus_gh_runner_v1_1_0(self) -> None:
        validate_event(
            {"client_payload": {"project": "incus-gh-runner", "tag": "v1.1.0"}}
        )

    def test_rejects_privileged_controls(self) -> None:
        privileged_fields = (
            "apply_production",
            "confirmation",
            "delete",
            "empty_staging",
            "r2_prefix",
            "skip_staging",
        )
        for field in privileged_fields:
            with self.subTest(field=field), self.assertRaisesRegex(
                ValueError, "only project and tag"
            ):
                validate_event(
                    {
                        "client_payload": {
                            "project": "incus-gh-runner",
                            "tag": "v1.1.0",
                            field: True,
                        }
                    }
                )

    def test_rejects_missing_or_non_string_identity(self) -> None:
        invalid_payloads = (
            {"project": "incus-gh-runner"},
            {"project": "incus-gh-runner", "tag": 110},
            {"project": "", "tag": "v1.1.0"},
        )
        for payload in invalid_payloads:
            with self.subTest(payload=payload), self.assertRaises(ValueError):
                validate_event({"client_payload": payload})


if __name__ == "__main__":
    unittest.main()
