"""Release tooling of approved engines (scripts/engine_release.py, ADR 0055): reproducible build, signature."""
import base64
import importlib.util
import json
import os
import stat
import tempfile
import unittest
from pathlib import Path

from snitch_worker import engine_runtime

ROOT = Path(__file__).resolve().parents[2]
RELEASE = ROOT / "worker/engine-releases/speechbrain-ecapa-1.0.3-r1"
_spec = importlib.util.spec_from_file_location("engine_release", ROOT / "scripts/engine_release.py")
tool = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(tool)

# RFC 8032, section 7.1, tests 1 and 2.
VECTORS = [
    ("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60",
     "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a", "",
     "e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b"),
    ("4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb",
     "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c", "72",
     "92a009a9f0d4cab8720e820b5f642540a2b27b5416503f8fb3762223ebdb69da085ac1e43e15996e458f3613d0f11d8c387b2eaeb4302aeeb00d291612bb0c00"),
]


class Ed25519Test(unittest.TestCase):
    def test_rfc_8032_vectors(self):
        for seed, public, message, signature in VECTORS:
            self.assertEqual(tool.public_key(bytes.fromhex(seed)).hex(), public)
            self.assertEqual(tool.sign_bytes(bytes.fromhex(seed), bytes.fromhex(message)).hex(), signature)
            self.assertTrue(tool.verify_bytes(bytes.fromhex(public), bytes.fromhex(message), bytes.fromhex(signature)))

    def test_a_modified_message_or_signature_is_rejected(self):
        seed = bytes(range(32))
        public, signature = tool.public_key(seed), tool.sign_bytes(seed, b"payload")
        self.assertTrue(tool.verify_bytes(public, b"payload", signature))
        self.assertFalse(tool.verify_bytes(public, b"payloaD", signature))
        self.assertFalse(tool.verify_bytes(public, b"payload", signature[:-1] + bytes([signature[-1] ^ 1])))
        self.assertFalse(tool.verify_bytes(tool.public_key(bytes(32)), b"payload", signature))


class ReleaseTest(unittest.TestCase):
    def test_ecapa_disk_budget_covers_wheels_and_measured_unpacked_size(self):
        manifest = tool.build_manifest(RELEASE)
        # ZIP-directory measurement recorded during the release audit: 6.43 GiB.
        # Round up to keep the check conservative without downloading the wheels.
        unpacked_bytes = 644 * (1 << 30) // 100
        self.assertGreaterEqual(manifest["disk_bytes"], sum(w["size"] for w in manifest["wheels"]) + unpacked_bytes)

    def test_the_committed_manifest_is_what_the_inputs_build(self):
        built = tool.canonical(tool.build_manifest(RELEASE)) + b"\n"
        self.assertEqual((RELEASE / "manifest.json").read_bytes(), built)
        self.assertEqual(built, tool.canonical(tool.build_manifest(RELEASE)) + b"\n")

    def test_the_manifest_pins_every_dependency_and_passes_the_worker_checks(self):
        manifest = json.loads((RELEASE / "manifest.json").read_text())
        engine_runtime.validate(manifest, "speechbrain-ecapa", "fingerprint")
        names = {w["name"].split("-")[0].lower().replace("_", "-") for w in manifest["wheels"]}
        for required in ("torch", "torchaudio", "speechbrain", "numpy", "pyyaml", "soundfile", "hyperpyyaml", "sentencepiece"):
            self.assertIn(required, names)
        self.assertEqual(manifest["capabilities"], ["fingerprint"])
        self.assertEqual(manifest["output_contract"], "fingerprint-192-unit-v1")
        self.assertTrue(all(w["url"].startswith("https://files.pythonhosted.org/") and len(w["sha256"]) == 64 for w in manifest["wheels"]))

    def test_every_wheel_of_the_lock_is_in_the_manifest(self):
        import tomllib
        lock = tomllib.loads((RELEASE / "pylock.toml").read_text())
        manifest = json.loads((RELEASE / "manifest.json").read_text())
        self.assertEqual(len(lock["packages"]), len(manifest["wheels"]))

    def test_a_source_or_foreign_url_is_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            folder = Path(tmp)
            (folder / "release.json").write_text((RELEASE / "release.json").read_text())
            wheel = 'wheels = [{ url = "%s", size = 10, hashes = { sha256 = "%s" } }]'
            good = "https://files.pythonhosted.org/packages/aa/bb/x-1-py3-none-any.whl"
            for lock, expected in (
                ('[[packages]]\nname = "x"\nversion = "1"\n' + wheel % ("https://example.com/x-1-py3-none-any.whl", "a" * 64), "pythonhosted"),
                ('[[packages]]\nname = "x"\nversion = "1"\nsdist = { url = "https://files.pythonhosted.org/x.tar.gz" }\n' + wheel % (good, "a" * 64), "wheel"),
                ('[[packages]]\nname = "x"\nversion = "1"\n' + wheel % (good, "nothex"), "sha256"),
            ):
                (folder / "pylock.toml").write_text(lock)
                with self.assertRaises(SystemExit) as caught:
                    tool.build_manifest(folder)
                self.assertIn(expected, str(caught.exception))


class SigningTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)

    def key(self, content, mode=0o600):
        path = Path(self.tmp.name) / "key"
        path.write_bytes(content)
        os.chmod(path, mode)
        return path

    def test_key_formats_and_permissions(self):
        seed = bytes(range(32))
        pem = b"-----BEGIN PRIVATE KEY-----\n" + base64.b64encode(bytes.fromhex("302e020100300506032b657004220420") + seed) + b"\n-----END PRIVATE KEY-----\n"
        for content in (seed, seed.hex().encode(), base64.b64encode(seed), pem):
            self.assertEqual(tool.read_seed(self.key(content)), seed)
        with self.assertRaises(SystemExit):
            tool.read_seed(self.key(seed, 0o644))
        with self.assertRaises(SystemExit):
            tool.read_seed(self.key(b"not a key"))

    def test_envelope_carries_the_exact_signed_bytes_and_never_the_key(self):
        seed = bytes(range(32))
        manifest = (RELEASE / "manifest.json").read_bytes().rstrip(b"\n")
        envelope = tool.make_envelope(manifest, seed, "publisher-1")
        self.assertEqual(set(envelope), {"key_id", "payload", "signature"})
        self.assertEqual(base64.b64decode(envelope["payload"]), manifest)
        self.assertNotIn(seed.hex(), json.dumps(envelope))
        self.assertNotIn(base64.b64encode(seed).decode(), json.dumps(envelope))
        public = base64.b64encode(tool.public_key(seed)).decode()
        self.assertEqual(tool.verify_envelope(envelope, public)["engine"], "speechbrain-ecapa")
        tampered = {**envelope, "payload": base64.b64encode(manifest.replace(b"1.0.3", b"9.9.9")).decode()}
        with self.assertRaises(SystemExit):
            tool.verify_envelope(tampered, public)

    def test_cli_round_trip_leaves_no_key_in_the_outputs(self):
        seed = bytes(range(32))
        key = self.key(seed.hex().encode())
        out = Path(self.tmp.name) / "envelope.json"
        self.assertEqual(tool.main(["sign", "--manifest", str(RELEASE / "manifest.json"), "--key-file", str(key), "--key-id", "k1", "--out", str(out)]), 0)
        self.assertNotIn(seed.hex(), out.read_text())
        self.assertEqual(tool.main(["verify", "--envelope", str(out), "--public-key", base64.b64encode(tool.public_key(seed)).decode()]), 0)
        self.assertTrue(stat.S_ISREG(os.stat(out).st_mode))


if __name__ == "__main__":
    unittest.main()
