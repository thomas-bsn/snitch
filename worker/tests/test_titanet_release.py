"""Release inputs of the TitaNet runtime (#355), checked with the shared publisher tool (scripts/engine_release.py).

Offline. The release is deliberately NOT publishable yet: one package of NeMo's closure has no wheel (see ADR 0055).
These tests keep that state honest: they fail the day the lock changes so that a manifest can be built, so that the
manifest gets committed and checked like the ECAPA one, instead of drifting.
"""
import importlib.util
import json
import tomllib
import unittest
from pathlib import Path

from snitch_worker import engine_runtime

ROOT = Path(__file__).resolve().parents[2]
RELEASE = ROOT / "worker/engine-releases/nemo-titanet-3.0.0-r1"
_spec = importlib.util.spec_from_file_location("engine_release", ROOT / "scripts/engine_release.py")
tool = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(tool)

SDIST_ONLY = "antlr4-python3-runtime"


def pylock():
    return tomllib.loads((RELEASE / "pylock.toml").read_text())


class TitaNetReleaseInputsTests(unittest.TestCase):
    def test_the_release_constants_describe_the_titanet_adapter(self):
        spec = json.loads((RELEASE / "release.json").read_text())
        self.assertEqual((spec["engine"], spec["kind"], spec["architecture"]), ("nemo-titanet", "fingerprint", "titanet"))
        self.assertEqual(spec["capabilities"], ["fingerprint"])
        self.assertEqual((spec["input_contract"], spec["output_contract"]), ("audio-mono-16khz", "speaker-embedding-unit-v1"))
        self.assertTrue(spec["cpu_fallback"])
        self.assertEqual(spec["disk_factor"], 3, "the unpacked NeMo closure is about 1.8 times its wheels: factor 2 would fail the install budget")
        self.assertEqual((RELEASE / "requirements.in").read_text().split(),
                         ["nemo-toolkit[asr]==3.0.0", "torch==2.8.0", "torchaudio==2.8.0", "soundfile==0.14.0"], "direct pins are exact")

    def test_every_package_is_pinned_with_hashes_and_hosted_on_pypi(self):
        packages = pylock()["packages"]
        self.assertGreater(len(packages), 100)
        names = {package["name"] for package in packages}
        self.assertTrue({"nemo-toolkit", "torch", "torchaudio", "soundfile", "pyyaml", "omegaconf"} <= names)
        for package in packages:
            self.assertTrue(package.get("version"), package["name"])
            files = package.get("wheels", []) + ([package["sdist"]] if "sdist" in package else [])
            self.assertTrue(files, package["name"])
            for entry in files:
                self.assertTrue(entry["url"].startswith("https://files.pythonhosted.org/"), package["name"])
                self.assertRegex(entry["hashes"]["sha256"], r"^[0-9a-f]{64}$")
                self.assertGreater(entry["size"], 0)

    def test_the_only_package_without_a_wheel_is_the_known_one_and_the_build_refuses_it_by_name(self):
        packages = pylock()["packages"]
        without_wheel = [package["name"] for package in packages if not package.get("wheels")]
        self.assertEqual(without_wheel, [SDIST_ONLY])
        with self.assertRaises(SystemExit) as caught:
            tool.build_manifest(RELEASE)
        self.assertIn(SDIST_ONLY, str(caught.exception))
        self.assertFalse((RELEASE / "manifest.json").exists(), "no publishable manifest while a package would be built from source")

    def test_without_that_package_the_rest_builds_a_valid_reproducible_manifest(self):
        """What the release becomes once the rule for the one source-only package is decided: all else is ready."""
        import tempfile

        lock = pylock()
        lock["packages"] = [package for package in lock["packages"] if package["name"] != SDIST_ONLY]
        with tempfile.TemporaryDirectory() as directory:
            folder = Path(directory)
            (folder / "release.json").write_text((RELEASE / "release.json").read_text())
            lines = []
            for package in lock["packages"]:
                lines.append('[[packages]]\nname = "%s"\nversion = "%s"' % (package["name"], package["version"]))
                wheels = ", ".join('{ url = "%s", size = %d, hashes = { sha256 = "%s" } }' % (w["url"], w["size"], w["hashes"]["sha256"]) for w in package["wheels"])
                lines.append("wheels = [%s]" % wheels)
            (folder / "pylock.toml").write_text("\n".join(lines) + "\n")
            manifest = tool.build_manifest(folder)
            self.assertEqual(tool.canonical(manifest), tool.canonical(tool.build_manifest(folder)))
        engine_runtime.validate(manifest, "nemo-titanet", "fingerprint")
        self.assertEqual(manifest["version"], "3.0.0-cu128-r1")
        self.assertEqual(len(manifest["wheels"]), len(lock["packages"]))
        wheel_names = {w["name"].split("-")[0].lower().replace("_", "-") for w in manifest["wheels"]}
        self.assertEqual(wheel_names, {package["name"].lower().replace("_", "-") for package in lock["packages"]}, "one wheel per pinned package")
        self.assertLessEqual(sum(w["size"] for w in manifest["wheels"]) * 3, manifest["disk_bytes"])


class SeveralWheelsOfOnePinTests(unittest.TestCase):
    """The shared tool picks the wheel pip would pick on CPython 3.12 / Linux x86_64 / glibc 2.28, instead of failing."""

    def test_the_most_specific_installable_wheel_wins(self):
        rank = tool._wheel_rank
        specific = "c-1-cp312-cp312-manylinux2014_x86_64.manylinux_2_17_x86_64.manylinux_2_28_x86_64.whl"
        abi3 = "c-1-cp37-abi3-manylinux1_x86_64.manylinux_2_28_x86_64.manylinux_2_5_x86_64.whl"
        pure = "c-1-py3-none-any.whl"
        self.assertGreater(rank(specific), rank(abi3))
        self.assertGreater(rank(abi3), rank(pure))
        self.assertGreater(rank("s-1-py2.py3-none-manylinux_2_28_x86_64.whl"), rank("s-1-py2.py3-none-any.whl"))
        for refused in ("c-1-cp311-cp311-manylinux_2_28_x86_64.whl", "c-1-cp312-cp312-manylinux_2_34_x86_64.whl",
                        "c-1-cp312-cp312-musllinux_1_2_x86_64.whl", "c-1-cp312-cp312-manylinux_2_28_aarch64.whl",
                        "c-1-cp312-cp312-win_amd64.whl", "c-1-cp313-abi3-manylinux_2_17_x86_64.whl",
                        "c-1-cp312-cp312t-manylinux_2_28_x86_64.whl", "c-1-cp312-cp312-macosx_11_0_arm64.whl"):
            self.assertIsNone(rank(refused), refused)

    def test_a_pin_whose_files_cannot_be_ranked_apart_is_still_refused(self):
        import tempfile

        entry = '{ url = "https://files.pythonhosted.org/packages/aa/bb/%s", size = 10, hashes = { sha256 = "%s" } }'
        with tempfile.TemporaryDirectory() as directory:
            folder = Path(directory)
            (folder / "release.json").write_text((RELEASE / "release.json").read_text())
            files = [entry % ("x-1-cp312-cp312-manylinux_2_17_x86_64.whl", "a" * 64), entry % ("x-1-cp312-cp312-manylinux_2_17_x86_64.manylinux2014_x86_64.whl", "b" * 64)]
            (folder / "pylock.toml").write_text('[[packages]]\nname = "x"\nversion = "1"\nwheels = [%s]\n' % ", ".join(files))
            with self.assertRaises(SystemExit):
                tool.build_manifest(folder)


class PublisherEnvelopeFixtureTests(unittest.TestCase):
    def test_the_committed_envelope_verifies_with_the_shared_tool(self):
        """The same file is accepted by the Go registry (voice_engine_release_tool_test.go): one format, two checkers.
        It was signed with a throwaway key whose private half was not kept."""
        directory = ROOT / "worker/tests/fixtures/engine-release"
        manifest = tool.verify_envelope(json.loads((directory / "envelope.json").read_text()), (directory / "public-key.txt").read_text().strip())
        self.assertEqual((manifest["engine"], manifest["version"], len(manifest["wheels"])), ("nemo-titanet", "9.9.9", 2))
        engine_runtime.validate(manifest, "nemo-titanet", "fingerprint")
        envelope = json.loads((directory / "envelope.json").read_text())
        self.assertEqual(set(envelope), {"key_id", "payload", "signature"})


if __name__ == "__main__":
    unittest.main()
