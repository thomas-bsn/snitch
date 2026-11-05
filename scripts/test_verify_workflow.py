"""Exercise the checked-in workflow, not a second copy of its publication policy.

The small condition interpreter deliberately supports only the conjunctions used
here and rejects unknown syntax. It models the documented implicit success()
and transitive skip propagation; it is not a replacement for a GitHub runner.
https://docs.github.com/actions/reference/workflows-and-actions/expressions#status-check-functions
https://docs.github.com/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idneeds
"""

import contextlib
import io
import itertools
import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import tempfile
import unittest
from unittest.mock import patch

from ruamel.yaml import YAML


ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = YAML(typ="safe").load(ROOT / ".github/workflows/verify.yml")
JOBS = WORKFLOW["jobs"]
STATES = ("success", "skipped", "failure", "cancelled")
DIGEST = "sha256:" + "a" * 64


def dependencies(name):
    needs = JOBS[name].get("needs", [])
    return [needs] if isinstance(needs, str) else needs


def ancestors(name):
    return {parent for direct in dependencies(name) for parent in {direct} | ancestors(direct)}


def job_condition(name, needs, event="push", cancelled=False):
    expression = JOBS[name].get("if", "success()")
    expression = expression.removeprefix("${{").removesuffix("}}").strip()
    successful = all(needs[parent]["result"] == "success" for parent in ancestors(name))
    # Without a status function, even a successful `check` cannot mask a
    # skipped ancestor (the app-only incident, run 41949052720).
    if not re.search(r"\b(success|failure|cancelled|always)\s*\(", expression) and not successful:
        return False
    values = []
    for clause in expression.split("&&"):
        clause = clause.strip()
        if clause in ("always()", "success()", "!cancelled()"):
            values.append({"always()": True, "success()": successful, "!cancelled()": not cancelled}[clause])
            continue
        match = re.fullmatch(r"([\w.-]+)\s*(==|!=)\s*'([^']*)'", clause)
        if not match:
            raise AssertionError(f"Unsupported workflow condition: {clause}")
        path, operator, expected = match.groups()
        if path == "github.event_name":
            actual = event
        else:
            if not path.startswith("needs."):
                raise AssertionError(f"Unsupported context: {path}")
            actual = needs
            for key in path.split(".")[1:]:
                actual = actual.get(key, "") if isinstance(actual, dict) else ""
        values.append((actual == expected) if operator == "==" else (actual != expected))
    return all(values)


def scenario(app=True, worker=False, core=True, publish=True):
    needs = {name: {"result": "success", "outputs": {}} for name in JOBS}
    needs["changes"]["outputs"] = {
        "app": str(app).lower(), "worker": str(worker).lower(),
        "core": str(core).lower(), "publish": str(publish).lower(),
    }
    for name, selected in (("go-test", core), ("e2e", core), ("worker-test", worker),
                           ("image-app", app), ("image-worker", worker)):
        needs[name]["result"] = "success" if selected else "skipped"
        if name.startswith("image-") and selected:
            needs[name]["outputs"]["digest"] = DIGEST
    return needs


def gate(needs):
    # Execute the actual inline Python gate, with the same JSON stdin as bash.
    # shlex keeps its quoted program intact; compilation happens once below.
    with patch("sys.stdin", io.StringIO(json.dumps({k: needs[k] for k in dependencies("check")}))), \
            contextlib.redirect_stdout(io.StringIO()):
        try:
            exec(GATE_CODE, {})
        except SystemExit as result:
            return result.code == 0
    raise AssertionError("check must exit explicitly")


GATE_CODE = compile(shlex.split(JOBS["check"]["steps"][0]["run"])[-1], "verify.yml:check", "exec")


class PromotionTest(unittest.TestCase):
    def test_app_only_run_41949052720_and_worker_only_mirror(self):
        for image in ("app", "worker"):
            with self.subTest(image=image):
                needs = scenario(app=image == "app", worker=image == "worker", core=image == "app")
                self.assertTrue(gate(needs))
                self.assertTrue(job_condition(f"promote-{image}", needs))
                other = "worker" if image == "app" else "app"
                self.assertFalse(job_condition(f"promote-{other}", needs))

    def test_all_image_selections(self):
        for app, worker, core in itertools.product((False, True), repeat=3):
            needs = scenario(app=app, worker=worker, core=core)
            with self.subTest(app=app, worker=worker, core=core):
                self.assertTrue(gate(needs))
                self.assertEqual(job_condition("promote-app", needs), app)
                self.assertEqual(job_condition("promote-worker", needs), worker)

    def test_every_direct_dependency_state_even_if_gate_claims_success(self):
        for image in ("app", "worker"):
            for changes, check, build in itertools.product(STATES, repeat=3):
                needs = scenario(app=True, worker=True)
                for name, state in (("changes", changes), ("check", check), (f"image-{image}", build)):
                    needs[name]["result"] = state
                with self.subTest(image=image, changes=changes, check=check, build=build):
                    self.assertEqual(job_condition(f"promote-{image}", needs),
                                     changes == check == build == "success")

    def test_failed_or_cancelled_validation_blocks_both_promotions(self):
        for name in dependencies("check"):
            for state in ("failure", "cancelled"):
                needs = scenario(app=True, worker=True)
                needs[name]["result"] = state
                self.assertFalse(gate(needs), (name, state))
                needs["check"]["result"] = "failure"
                for image in ("app", "worker"):
                    self.assertFalse(job_condition(f"promote-{image}", needs), (image, name, state))

    def test_missing_digest_and_workflow_cancellation_block_publication(self):
        for image in ("app", "worker"):
            needs = scenario(app=True, worker=True)
            self.assertFalse(job_condition(f"promote-{image}", needs, cancelled=True))
            for outputs in ({}, {"digest": ""}):
                needs[f"image-{image}"]["outputs"] = outputs
                self.assertFalse(job_condition(f"promote-{image}", needs))

    def test_prs_and_disabled_publication_never_promote(self):
        for image in ("app", "worker"):
            # Defense in depth: even an erroneous publish=true on a PR is refused.
            for publish in (False, True):
                needs = scenario(app=True, worker=True, publish=publish)
                self.assertFalse(job_condition(f"promote-{image}", needs, event="pull_request"))
            self.assertFalse(job_condition(f"promote-{image}", scenario(app=True, worker=True, publish=False)))

    def test_gate_truth_table(self):
        names = dependencies("check")
        for states in itertools.product(STATES, repeat=len(names)):
            needs = scenario(app=True, worker=True)
            for name, state in zip(names, states):
                needs[name]["result"] = state
            expected = (needs["changes"]["result"] == needs["lint"]["result"] == "success"
                        and all(state in ("success", "skipped") for state in states))
            self.assertEqual(gate(needs), expected, dict(zip(names, states)))

    def test_other_jobs_have_no_unhandled_skippable_dependencies(self):
        for name in ("go-test", "e2e", "worker-test", "image-app", "image-worker"):
            self.assertEqual(dependencies(name), ["changes"])
            for state in STATES[1:]:
                needs = scenario(app=True, worker=True)
                needs["changes"]["result"] = state
                self.assertFalse(job_condition(name, needs), (name, state))
        self.assertEqual(JOBS["check"]["if"], "always()")
        self.assertEqual(JOBS["metrics"]["if"], "always()")


class PublicationEventTest(unittest.TestCase):
    def test_changes_script_publish_scope_and_image_push_guards(self):
        script = JOBS["changes"]["steps"][1]["run"]
        for event, ref, ref_type, expected in (
            ("pull_request", "refs/pull/1/merge", "branch", "false"),
            ("push", "refs/heads/main", "branch", "true"),
            ("push", "refs/tags/v1.2.3", "tag", "true"),
            ("workflow_dispatch", "refs/heads/main", "branch", "true"),
            ("workflow_dispatch", "refs/heads/topic", "branch", "false"),
            ("push", "refs/heads/topic", "branch", "false"),
        ):
            with self.subTest(event=event, ref=ref), tempfile.TemporaryDirectory() as tmp:
                output = Path(tmp) / "outputs"
                env = dict(os.environ, EVENT=event, REF=ref, REF_TYPE=ref_type, REF_NAME=ref.rsplit("/", 1)[-1],
                           SHA="0" * 40, BEFORE="", ALL="false", GITHUB_OUTPUT=str(output))
                subprocess.run(["bash", "-eu", "-c", script], cwd=ROOT, env=env, check=True, capture_output=True)
                outputs = dict(line.split("=", 1) for line in output.read_text().splitlines())
                self.assertEqual(outputs["publish"], expected)
        for image in ("app", "worker"):
            steps = JOBS[f"image-{image}"]["steps"]
            build = next(step for step in steps if step.get("id") == "build")
            self.assertEqual(build["if"], "needs.changes.outputs.publish == 'true'")
            self.assertIn("push-by-digest=true", build["with"]["outputs"])
            local = next(step for step in steps if step.get("name", "").startswith("Build (pull request"))
            self.assertEqual(local["if"], "needs.changes.outputs.publish != 'true'")
            self.assertFalse(local["with"].get("push", False))
            self.assertNotIn("cache-to", local["with"])


if __name__ == "__main__":
    unittest.main()
