#!/usr/bin/env python3
"""Verify README release metadata before publishing a release.

Usage:
  python3 scripts/verify_release_readme.py --version v0.17.2
  python3 scripts/verify_release_readme.py --version-from-readme
  python3 scripts/verify_release_readme.py --self-test
"""

from __future__ import annotations

import argparse
import re
import sys
import unittest
from pathlib import Path

README_PATH = Path(__file__).resolve().parents[1] / "README.md"
LATEST_RELEASE_RE = re.compile(r"^- Latest release: `([^`]+)`$", re.MULTILINE)
RELEASE_PLAN_RE = re.compile(r"^## Release Plan\n(?P<body>.*?)(?:\n## |\Z)", re.MULTILINE | re.DOTALL)
REQUIRED_PLAN_LINES = [
    "1. update README status/release notes first",
    "2. run `python3 scripts/verify_release_readme.py --version vX.Y.Z` locally",
    "3. commit and push the README/code changes",
    "4. create the tag and GitHub release only after the verification passes",
]


def read_readme() -> str:
    return README_PATH.read_text(encoding="utf-8")


def extract_latest_release(readme: str) -> str:
    match = LATEST_RELEASE_RE.search(readme)
    if not match:
        raise ValueError("README is missing the '- Latest release: `vX.Y.Z`' line")
    return match.group(1)


def extract_release_plan(readme: str) -> str:
    match = RELEASE_PLAN_RE.search(readme)
    if not match:
        raise ValueError("README is missing the '## Release Plan' section")
    return match.group("body")


def validate_readme(readme: str, expected_version: str) -> None:
    actual_version = extract_latest_release(readme)
    if actual_version != expected_version:
        raise ValueError(
            f"README Latest release line is '{actual_version}', expected '{expected_version}'"
        )

    if readme.count("## License") != 1:
        raise ValueError("README should contain exactly one License section")

    for required in [
        "cmdatlas profiles add NAME COMMAND [COMMAND ...]",
        "cmdatlas profiles remove NAME COMMAND [COMMAND ...]",
    ]:
        if required not in readme:
            raise ValueError(f"README missing required command doc: {required}")

    release_plan = extract_release_plan(readme)
    for line in REQUIRED_PLAN_LINES:
        if line not in release_plan:
            raise ValueError(f"README Release Plan is missing required step: {line}")


class GuardrailTests(unittest.TestCase):
    def test_extract_latest_release(self) -> None:
        self.assertEqual(extract_latest_release("## Current Status\n\n- Latest release: `v1.2.3`\n"), "v1.2.3")

    def test_validate_happy_path(self) -> None:
        readme = """# cmdatlas\n\n## Current Status\n\n- Latest release: `v1.2.3`\n\ncmdatlas profiles add NAME COMMAND [COMMAND ...]\ncmdatlas profiles remove NAME COMMAND [COMMAND ...]\n\n## Release Plan\n\n1. update README status/release notes first\n2. run `python3 scripts/verify_release_readme.py --version vX.Y.Z` locally\n3. commit and push the README/code changes\n4. create the tag and GitHub release only after the verification passes\n\n## License\n\nMIT\n"""
        validate_readme(readme, "v1.2.3")

    def test_validate_reproduces_stale_readme_case(self) -> None:
        readme = """# cmdatlas\n\n## Current Status\n\n- Latest release: `v0.16.0`\n\ncmdatlas profiles add NAME COMMAND [COMMAND ...]\ncmdatlas profiles remove NAME COMMAND [COMMAND ...]\n\n## Release Plan\n\n1. update README status/release notes first\n2. run `python3 scripts/verify_release_readme.py --version vX.Y.Z` locally\n3. commit and push the README/code changes\n4. create the tag and GitHub release only after the verification passes\n\n## License\n\nMIT\n"""
        with self.assertRaisesRegex(ValueError, "expected 'v0.17.0'"):
            validate_readme(readme, "v0.17.0")

    def test_validate_requires_release_plan_steps(self) -> None:
        readme = """# cmdatlas\n\n## Current Status\n\n- Latest release: `v1.2.3`\n\ncmdatlas profiles add NAME COMMAND [COMMAND ...]\ncmdatlas profiles remove NAME COMMAND [COMMAND ...]\n\n## Release Plan\n\nship it\n\n## License\n\nMIT\n"""
        with self.assertRaisesRegex(ValueError, "Release Plan"):
            validate_readme(readme, "v1.2.3")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", help="expected release version, e.g. v0.17.2")
    parser.add_argument(
        "--version-from-readme",
        action="store_true",
        help="validate the README using its current Latest release value",
    )
    parser.add_argument("--self-test", action="store_true", help="run built-in regression tests")
    args = parser.parse_args()

    if args.self_test:
        suite = unittest.defaultTestLoader.loadTestsFromTestCase(GuardrailTests)
        result = unittest.TextTestRunner(verbosity=2).run(suite)
        return 0 if result.wasSuccessful() else 1

    if bool(args.version) == bool(args.version_from_readme):
        parser.error("pass exactly one of --version or --version-from-readme")

    readme = read_readme()
    expected_version = args.version or extract_latest_release(readme)

    try:
        validate_readme(readme, expected_version)
    except ValueError as exc:
        print(f"README release guard failed: {exc}", file=sys.stderr)
        return 1

    print(f"README release guard passed for {expected_version}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
