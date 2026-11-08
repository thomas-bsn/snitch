"""Real acoustic fixtures for issue #343 (see tests/fixtures/acoustic/generate.py).

Transformed rules (gain, correlation metrics) need numpy, present in the worker image.
Set SNITCH_REQUIRE_NUMPY=1 to turn a missing numpy into a failure instead of a skip.
"""

import array
import hashlib
import json
import math
import os
import tempfile
import unittest
import wave
from pathlib import Path

from snitch_worker import acoustic_dedup as dedup

try:
    import numpy  # noqa: F401

    HAS_NUMPY = True
except ImportError:
    HAS_NUMPY = False

FIXTURES = Path(__file__).parent / "fixtures" / "acoustic"
RATE = 16000


def fixture(name):
    return str(FIXTURES / name)


def read(path):
    with wave.open(path, "rb") as handle:
        data = array.array("h")
        data.frombytes(handle.readframes(handle.getnframes()))
        return data


def write(path, samples, rate=RATE):
    with wave.open(str(path), "wb") as handle:
        handle.setnchannels(1)
        handle.setsampwidth(2)
        handle.setframerate(rate)
        handle.writeframes(array.array("h", samples).tobytes())
    return str(path)


def event(name, path, track, *, job=None, offset=0.0, start=0.0, end=None, rate=RATE):
    if end is None:
        end = offset + len(read(path)) / rate
    return {"id": name, "track_index": track, "job_id": job or track * 10, "path": path, "clip_offset": offset, "start": start, "end": end}


def needs_numpy(test):
    return unittest.skipUnless(HAS_NUMPY or os.environ.get("SNITCH_REQUIRE_NUMPY"), "numpy absent")(test)


class Base(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.tmp = Path(self.directory.name)
        if os.environ.get("SNITCH_REQUIRE_NUMPY"):
            self.assertTrue(HAS_NUMPY, "SNITCH_REQUIRE_NUMPY is set but numpy is missing")

    def tearDown(self):
        self.directory.cleanup()

    def compare(self, a, b, **config):
        return dedup.compare(a, b, dedup.DedupConfig(**config))


class FixtureIntegrity(unittest.TestCase):
    def test_lossless_originals_match_the_committed_manifest(self):
        manifest = json.loads((FIXTURES / "manifest.json").read_text())
        for name, digest in manifest["files"].items():
            self.assertEqual(digest, hashlib.sha256((FIXTURES / name).read_bytes()).hexdigest(), name)


class ExactPcm(Base):
    def test_identical_pcm_with_different_container_is_a_full_duplicate(self):
        a = event("a", fixture("voice-a.wav"), 2)
        b = event("b", fixture("voice-a-flac-roundtrip.wav"), 7)
        self.assertNotEqual((FIXTURES / "voice-a.wav").read_bytes(), (FIXTURES / "voice-a-flac-roundtrip.wav").read_bytes())
        result = self.compare(a, b)
        self.assertEqual("duplicate", result["decision"])
        self.assertEqual("exact_pcm", result["evidence_kind"])
        self.assertGreater(result["coverage"]["left"], 0.99)
        self.assertGreater(result["coverage"]["right"], 0.99)
        self.assertEqual(0.0, result["lag_seconds"])
        self.assertNotIn("partial_coverage", result["reason_codes"])
        self.assertEqual(1, len(result["matched"]))

    def test_result_does_not_depend_on_argument_order(self):
        a = event("a", fixture("voice-a.wav"), 2)
        b = event("b", fixture("voice-a-flac-roundtrip.wav"), 7)
        self.assertEqual(self.compare(a, b), self.compare(b, a))
        self.assertEqual(2, self.compare(b, a)["left"]["track_index"])

    def test_identical_composite_mix_on_two_tracks_is_a_duplicate(self):
        a = event("a", fixture("mix-a-and-b.wav"), 2)
        b = event("b", fixture("mix-a-and-b.wav"), 3)
        self.assertEqual("duplicate", self.compare(a, b)["decision"])

    def test_same_samples_with_stream_offsets_keep_clip_times(self):
        voice = read(fixture("voice-a.wav"))
        shifted = write(self.tmp / "shifted.wav", array.array("h", bytes(2 * 8000)) + voice)
        a = event("a", fixture("voice-a.wav"), 2, offset=0.0)
        for lead in (-0.5, 0.5):
            with self.subTest(offset=lead):
                # The second stream's first sample sits 0.5 s before (or after) clip zero.
                if lead < 0:
                    b = event("b", shifted, 7, offset=-0.5, start=0.0, end=a["end"])
                else:
                    b = event("b", fixture("voice-a.wav"), 7, offset=0.0)
                result = self.compare(a, b)
                self.assertEqual("duplicate", result["decision"])
                self.assertAlmostEqual(0.0, result["matched"][0]["left_start"] - result["matched"][0]["right_start"], places=4)

    def test_wrong_offset_beyond_the_alignment_window_is_not_a_duplicate(self):
        voice = read(fixture("voice-a.wav"))
        shifted = write(self.tmp / "shifted.wav", array.array("h", bytes(2 * 8000)) + voice)
        a = event("a", fixture("voice-a.wav"), 2)
        # Claims a zero offset although the audio really starts 0.5 s later.
        b = event("b", shifted, 7, offset=0.0, end=a["end"])
        self.assertNotEqual("duplicate", self.compare(a, b)["decision"])

    def test_small_residual_lag_is_reported_not_hidden(self):
        voice = read(fixture("voice-a.wav"))
        shifted = write(self.tmp / "lag.wav", array.array("h", bytes(2 * 320)) + voice)  # 20 ms
        a = event("a", fixture("voice-a.wav"), 2)
        b = event("b", shifted, 7, offset=0.0, end=a["end"])
        result = self.compare(a, b)
        self.assertEqual("duplicate", result["decision"])
        self.assertAlmostEqual(0.02, result["lag_seconds"], places=4)
        self.assertAlmostEqual(0.02, result["matched"][0]["right_start"] - result["matched"][0]["left_start"], places=4)

    def test_partial_overlap_matches_only_the_proven_range(self):
        voice = read(fixture("voice-a.wav"))
        other = read(fixture("voice-b-same-words.wav"))
        cut = int(len(voice) * 0.6)
        hybrid = write(self.tmp / "hybrid.wav", voice[:cut] + other[cut:])
        a = event("a", fixture("voice-a.wav"), 2)
        b = event("b", hybrid, 7)
        result = self.compare(a, b)
        self.assertEqual("duplicate", result["decision"])
        self.assertIn("partial_coverage", result["reason_codes"])
        self.assertLess(result["coverage"]["left"], 0.7)
        self.assertLessEqual(result["matched"][0]["left_end"], cut / RATE + 0.011)
        self.assertGreater(result["matched"][0]["left_end"], cut / RATE - 0.03)

    def test_same_recording_at_a_different_clip_time_is_a_distinct_event(self):
        a = event("a", fixture("voice-a.wav"), 2, offset=0.0)
        b = event("b", fixture("voice-a.wav"), 7, offset=100.0, start=100.0)
        result = self.compare(a, b)
        self.assertEqual("distinct", result["decision"])
        self.assertIn("timeline_disjoint", result["reason_codes"])


class HardNegatives(Base):
    def assertKept(self, a, b, **config):
        result = self.compare(a, b, **config)
        self.assertNotEqual("duplicate", result["decision"], result)
        self.assertEqual([], result["matched"])
        return result

    @needs_numpy
    def test_same_words_by_another_voice_is_distinct(self):
        a = event("a", fixture("voice-a.wav"), 2)
        b = event("b", fixture("voice-b-same-words.wav"), 7)
        self.assertEqual("distinct", self.assertKept(a, b)["decision"])

    @needs_numpy
    def test_same_voice_second_take_is_distinct(self):
        a = event("a", fixture("voice-a.wav"), 2)
        b = event("b", fixture("voice-a-second-take.wav"), 7)
        result = self.assertKept(a, b)
        self.assertLess(result["metrics"]["normalized_correlation"], 0.5)

    def test_two_voices_saying_the_same_words_together_are_kept(self):
        together = event("t", fixture("mix-a-and-b.wav"), 2)
        for name in ("voice-a.wav", "voice-b-same-words.wav"):
            with self.subTest(name):
                other = event("o", fixture(name), 7)
                self.assertNotEqual("duplicate", self.compare(together, other)["decision"])

    def test_common_loud_bed_with_different_voices_never_suppresses_the_speech(self):
        a = event("a", fixture("mix-a-game.wav"), 2)
        b = event("b", fixture("mix-b-game.wav"), 7)
        result = self.compare(a, b)
        voice = read(fixture("voice-a.wav"))
        speech = [i for i, v in enumerate(voice) if abs(v) > 200]
        first, last = speech[0] / RATE, speech[-1] / RATE
        for run in result["matched"]:
            # Any identical stretch is bed-only: it never spans the spoken words.
            self.assertFalse(run["left_start"] <= first + 0.2 and run["left_end"] >= last - 0.2, run)
            self.assertLess(run["left_end"] - run["left_start"], last - first)

    @needs_numpy
    def test_clean_voice_against_voice_plus_bed_is_only_a_candidate(self):
        a = event("a", fixture("voice-a.wav"), 2)
        b = event("b", fixture("mix-a-game.wav"), 7)
        result = self.assertKept(a, b)
        self.assertEqual("uncertain_keep", result["decision"])
        self.assertEqual("mixture", result["evidence_kind"])

    @needs_numpy
    def test_aac_copy_is_evaluated_but_kept(self):
        a = event("a", fixture("voice-a.wav"), 2)
        b = event("b", fixture("voice-a-aac64k.wav"), 7)
        result = self.assertKept(a, b)
        self.assertEqual("uncertain_keep", result["decision"])
        self.assertEqual("codec", result["evidence_kind"])
        self.assertGreater(result["metrics"]["normalized_correlation"], 0.99)
        self.assertIn("codec_or_transform_candidate_not_validated", result["reason_codes"])

    def test_without_numpy_a_transformed_copy_stays_uncertain(self):
        original = dedup._numpy
        dedup._numpy = lambda: None
        try:
            a = event("a", fixture("voice-a.wav"), 2)
            b = event("b", fixture("voice-a-gain-0.5.wav"), 7)
            result = self.compare(a, b)
        finally:
            dedup._numpy = original
        self.assertEqual("uncertain_keep", result["decision"])
        self.assertIn("numpy_unavailable_transforms_not_evaluated", result["reason_codes"])

    def test_silence_short_events_and_unknown_timeline_never_suppress(self):
        silence = write(self.tmp / "silence.wav", [0] * 32000)
        a, b = event("a", silence, 2), event("b", silence, 7)
        self.assertNotEqual("duplicate", self.compare(a, b)["decision"])
        short_a = event("a", fixture("voice-a.wav"), 2, end=0.1)
        short_b = event("b", fixture("voice-a.wav"), 7, end=0.1)
        self.assertNotEqual("duplicate", self.compare(short_a, short_b)["decision"])
        unknown = event("b", fixture("voice-a.wav"), 7)
        unknown["clip_offset"] = None
        result = self.compare(event("a", fixture("voice-a.wav"), 2), unknown)
        self.assertEqual("uncertain_keep", result["decision"])
        self.assertIn("unknown_timeline", result["reason_codes"])

    def test_same_track_sample_rate_and_format_mismatches_are_kept(self):
        a = event("a", fixture("voice-a.wav"), 2)
        self.assertIn("same_track", self.compare(a, event("b", fixture("voice-a.wav"), 2))["reason_codes"])
        slow = write(self.tmp / "r8.wav", read(fixture("voice-a.wav")), rate=8000)
        result = self.compare(a, event("b", slow, 7, end=a["end"]))
        self.assertIn("sample_rate_not_validated", result["reason_codes"])
        stereo = self.tmp / "stereo.wav"
        with wave.open(str(stereo), "wb") as handle:
            handle.setnchannels(2)
            handle.setsampwidth(2)
            handle.setframerate(RATE)
            handle.writeframes(bytes(4 * 16000))
        self.assertIn("unsupported_format", self.compare(a, event("b", str(stereo), 7, end=a["end"]))["reason_codes"])
        corrupt = self.tmp / "corrupt.wav"
        corrupt.write_bytes(b"not a wav")
        self.assertIn("undecodable", self.compare(a, event("b", str(corrupt), 7, end=a["end"]))["reason_codes"])

    def test_two_different_periodic_tones_are_not_duplicates(self):
        tone = lambda hz: [int(8000 * math.sin(2 * math.pi * hz * i / RATE)) for i in range(32000)]
        a = event("a", write(self.tmp / "t1.wav", tone(440)), 2)
        b = event("b", write(self.tmp / "t2.wav", tone(445)), 7)
        self.assertNotEqual("duplicate", self.compare(a, b)["decision"])

    @needs_numpy
    def test_clipped_gain_never_covers_a_clipped_sample(self):
        samples = read(fixture("voice-a.wav"))
        clipped = write(self.tmp / "clipped.wav", [max(-32768, min(32767, v * 4)) for v in samples])
        clip_times = [i / RATE for i, v in enumerate(read(clipped)) if abs(v) >= 32767]
        self.assertTrue(clip_times)
        result = self.compare(event("a", fixture("voice-a.wav"), 2), event("b", clipped, 7))
        for run in result["matched"]:
            self.assertFalse([t for t in clip_times if run["left_start"] <= t < run["left_end"]], run)
        if result["matched"]:
            self.assertLess(result["coverage"]["left"], 1.0)


class GainRule(Base):
    @needs_numpy
    def test_unclipped_gain_copies_are_duplicates_with_the_gain_recorded(self):
        a = event("a", fixture("voice-a.wav"), 2)
        for gain in ("0.25", "0.5", "2.0"):
            with self.subTest(gain=gain):
                result = self.compare(a, event("b", fixture(f"voice-a-gain-{gain}.wav"), 7))
                self.assertEqual("duplicate", result["decision"], result)
                self.assertEqual("gain", result["evidence_kind"])
                self.assertEqual("gain", result["strategy"])
                self.assertAlmostEqual(float(gain), result["gain"], places=2)
                self.assertGreater(result["coverage"]["left"], 0.97)

    @needs_numpy
    def test_inverted_polarity_is_not_validated(self):
        inverted = write(self.tmp / "inverted.wav", [-v for v in read(fixture("voice-a.wav"))])
        result = self.compare(event("a", fixture("voice-a.wav"), 2), event("b", inverted, 7))
        self.assertNotEqual("duplicate", result["decision"])


class Assembly(Base):
    def payload(self, events, **extra):
        return {"schema_version": 1, "events": events, **extra}

    def test_run_output_is_independent_of_event_order_and_lists_enabled_rules(self):
        events = [
            event("a", fixture("voice-a.wav"), 2),
            event("b", fixture("voice-a-flac-roundtrip.wav"), 7),
            event("c", fixture("voice-b-same-words.wav"), 9),
        ]
        first = dedup.run(self.payload(events))
        second = dedup.run(self.payload(list(reversed(events))))
        self.assertEqual(first["comparisons"], second["comparisons"])
        self.assertEqual({"exact_pcm": "1", "gain": "1"}, first["enabled_strategies"])
        self.assertEqual(3, first["stats"]["pairs"])
        self.assertEqual(1, first["stats"]["decisions"]["duplicate"])

    def test_explicit_pairs_are_deduplicated_by_id_before_loading_events(self):
        events = [event("a", fixture("voice-a.wav"), 2), event("b", fixture("voice-a.wav"), 7)]
        out = dedup.run(self.payload(events, pairs=[["a", "b"], ["b", "a"]]))
        self.assertEqual(1, out["stats"]["pairs"])
        self.assertEqual("duplicate", out["comparisons"][0]["decision"])

    def test_pair_budget_keeps_what_it_did_not_compare(self):
        events = [event(n, fixture("voice-a.wav"), t) for n, t in (("a", 2), ("b", 3), ("c", 4))]
        out = dedup.run(self.payload(events, config={"max_pairs": 1}))
        self.assertEqual(2, out["stats"]["skipped_pairs"])
        skipped = [c for c in out["comparisons"] if "pair_budget_exceeded" in c["reason_codes"]]
        self.assertEqual(2, len(skipped))
        self.assertTrue(all(c["decision"] == "uncertain_keep" for c in skipped))

    def test_cancellation_stops_between_pairs(self):
        events = [event(n, fixture("voice-a.wav"), t) for n, t in (("a", 2), ("b", 3))]

        def cancel():
            raise dedup.Cancelled()

        with self.assertRaises(dedup.Cancelled):
            dedup.run(self.payload(events), cancel=cancel)

    def test_contract_rejects_unknown_versions_keys_and_duplicate_ids(self):
        with self.assertRaises(ValueError):
            dedup.run({"schema_version": 2, "events": []})
        with self.assertRaises(ValueError):
            dedup.run(self.payload([], config={"threshold": 0.9}))
        e = event("a", fixture("voice-a.wav"), 2)
        with self.assertRaises(ValueError):
            dedup.run(self.payload([e, dict(e)]))

    def test_codec_and_mixture_rules_are_never_enabled(self):
        for name in dedup.EVALUATED_ONLY:
            self.assertNotIn(name, dedup.ENABLED_STRATEGIES)


class GoContract(Base):
    """The Go merge (audioplans.ParseEvidence) reads this exact sample; both must stay in step."""

    @needs_numpy
    def test_committed_evidence_sample_is_what_the_adapter_produces_today(self):
        import importlib.util

        spec = importlib.util.spec_from_file_location("generate_evidence", FIXTURES / "generate_evidence.py")
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        report = dedup.run({"schema_version": 1, "events": module.events()})
        del report["stats"]

        def project(r):
            return [
                (c["left"]["id"], c["right"]["id"], c["decision"], c["evidence_kind"], c["strategy"], c["strategy_version"], c["reason_codes"], [[round(m[k], 2) for k in ("left_start", "left_end", "right_start", "right_end")] for m in c["matched"]])
                for c in r["comparisons"]
            ]

        committed = json.loads((Path(__file__).parents[2] / "app/internal/audioplans/fixtures/acoustic-evidence.json").read_text())
        self.assertEqual(project(committed), project(report))
        self.assertEqual(committed["enabled_strategies"], report["enabled_strategies"])


if __name__ == "__main__":
    unittest.main()
