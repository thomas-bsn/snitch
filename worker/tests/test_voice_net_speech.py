"""Net speech and fingerprint measures (#389, ADR 0064), on synthetic diarizations and audio only."""

import math
import os
import struct
import tempfile
import unittest
import wave

from snitch_worker import voice, voice_pyannote

from test_voice_real import CONFIG, model_dirs, provider


class RecordingAudio:
    """An embedder that records every window and returns a vector telling which window it saw."""

    def __init__(self, vectors=None):
        self.calls = []
        self.vectors = vectors or {}

    def open(self, _path):
        def embed(start, end):
            self.calls.append((round(start, 6), round(end, 6)))
            return self.vectors.get((round(start, 6), round(end, 6)), [1.0, start, end])

        return voice_pyannote.DecodedAudio(embed, 16000, 1, 16000 * 3600)


def fingerprint(speakers, audio=None, audio_path="/absent.wav"):
    audio = audio or RecordingAudio()
    diarization = {"job_id": 3, "speakers": [
        {"label": label, "turns": [{"index": i, "start": s, "end": e} for i, (s, e) in enumerate(turns)]}
        for label, turns in speakers
    ]}
    with model_dirs() as env:
        result = provider(env, load_embedder=lambda path, device: audio.open).fingerprint(CONFIG, diarization, audio_path, 0)
    return result, audio


def items(result, label, scope="speaker"):
    return [f for f in result["fingerprints"] if f["speaker_label"] == label and f["scope"] == scope]


class NetSpeechTests(unittest.TestCase):
    def test_an_overlap_never_enters_the_vector_of_a_local_speaker(self):
        # A speaks 0-3 s, B 2-4 s: 2-3 s is an overlap and is embedded for nobody.
        _, audio = fingerprint([("A", [(0.0, 3.0)]), ("B", [(2.0, 4.0)])])
        self.assertEqual(sorted(audio.calls), [(0.0, 2.0), (3.0, 4.0)])

    def test_a_net_interval_shorter_than_half_a_second_is_not_embedded(self):
        # B's net speech is 3.0-3.3 s (0.3 s) plus 5-7 s: only the latter is embedded for B.
        _, audio = fingerprint([("A", [(0.0, 3.0)]), ("B", [(2.0, 3.3), (5.0, 7.0)])])
        self.assertEqual(sorted(audio.calls), [(0.0, 2.0), (5.0, 7.0)])

    def test_an_entirely_overlapped_speaker_keeps_a_fingerprint_of_its_whole_turns_marked_as_overlapped(self):
        # B only ever speaks while A speaks.
        result, audio = fingerprint([("A", [(0.0, 6.0)]), ("B", [(1.0, 2.0), (3.0, 4.5)])])
        self.assertIn((1.0, 2.0), audio.calls)
        self.assertIn((3.0, 4.5), audio.calls)
        [speaker] = items(result, "B")
        self.assertTrue(speaker["measures"]["overlapped"])
        self.assertEqual(speaker["measures"]["net_speech_seconds"], 0.0)
        self.assertEqual(speaker["measures"]["overlap_ratio"], 1.0)
        self.assertTrue(all(t["measures"]["overlapped"] for t in items(result, "B", "turn")))
        # A keeps its net speech, outside B's turns, and is not marked.
        [first] = items(result, "A")
        self.assertFalse(first["measures"]["overlapped"])
        self.assertEqual(first["measures"]["net_speech_seconds"], 3.5)
        self.assertEqual(first["measures"]["overlap_ratio"], 0.4167)


class MeasureTests(unittest.TestCase):
    def test_measures_carry_their_version_and_the_net_speech_of_each_fingerprint(self):
        result, _ = fingerprint([("A", [(0.0, 3.0), (5.0, 6.0)]), ("B", [(2.0, 4.0)])])
        [speaker] = items(result, "A")
        self.assertEqual(speaker["measures"]["version"], 1)
        self.assertEqual(speaker["measures"]["net_speech_seconds"], 3.0)  # 0-2 s and 5-6 s
        self.assertEqual(speaker["measures"]["overlap_ratio"], 0.25)  # 1 s of 4 s
        self.assertEqual([t["measures"]["net_speech_seconds"] for t in items(result, "A", "turn")], [2.0, 1.0])
        # The speaker's stored duration is its net speech.
        self.assertEqual(speaker["end"] - speaker["start"], 3.0)

    def test_coherence_is_the_agreement_between_turn_vectors_and_absent_with_a_single_turn(self):
        audio = RecordingAudio({(0.0, 1.0): [1.0, 0.0, 0.0], (2.0, 3.0): [1.0, 1.0, 0.0], (5.0, 6.0): [0.0, 0.0, 1.0]})
        result, _ = fingerprint([("A", [(0.0, 1.0), (2.0, 3.0)]), ("B", [(5.0, 6.0)])], audio)
        self.assertEqual(items(result, "A")[0]["measures"]["coherence"], 0.7071)
        self.assertIsNone(items(result, "B")[0]["measures"]["coherence"])
        self.assertTrue(all(t["measures"]["coherence"] is None for t in items(result, "A", "turn")))

    def test_clipping_and_mean_level_are_measured_on_the_prepared_wav(self):
        with tempfile.TemporaryDirectory() as root:
            path = os.path.join(root, "prepared.wav")
            write_wav(path, [32767] * 16000 + [3277] * 16000)  # 1 s at full scale, then 1 s at -20 dBFS
            result, _ = fingerprint([("A", [(0.0, 1.0), (1.0, 2.0)])], audio_path=path)
        [speaker] = items(result, "A")
        first, second = items(result, "A", "turn")
        self.assertEqual(speaker["measures"]["clipped_ratio"], 0.5)
        self.assertEqual((first["measures"]["clipped_ratio"], first["measures"]["mean_level_dbfs"]), (1.0, 0.0))
        self.assertEqual((second["measures"]["clipped_ratio"], second["measures"]["mean_level_dbfs"]), (0.0, -20.0))

    def test_without_a_readable_wav_the_audio_measures_are_absent_and_the_fingerprint_is_kept(self):
        result, _ = fingerprint([("A", [(0.0, 2.0)])], audio_path="/no/such/prepared.wav")
        [speaker] = items(result, "A")
        self.assertIsNone(speaker["measures"]["clipped_ratio"])
        self.assertIsNone(speaker["measures"]["mean_level_dbfs"])

    def test_a_digital_silence_has_a_finite_floor_level(self):
        with tempfile.TemporaryDirectory() as root:
            path = os.path.join(root, "prepared.wav")
            write_wav(path, [0] * 16000)
            result, _ = fingerprint([("A", [(0.0, 1.0)])], audio_path=path)
        self.assertEqual(items(result, "A")[0]["measures"]["mean_level_dbfs"], -120.0)


class FakeProviderMeasureTests(unittest.TestCase):
    CONFIG = {"provider": "fake", "model": "fixture-diarization", "model_version": "fixture-v1", "fake": {"speakers": 2, "dimensions": 4}}

    def test_the_fake_provider_reports_the_same_geometric_measures_without_audio(self):
        diarization = {"job_id": 5, "speakers": [
            {"label": "SPEAKER_00", "turns": [{"index": 0, "start": 0.0, "end": 3.0}, {"index": 1, "start": 5.0, "end": 6.0}]},
            {"label": "SPEAKER_01", "turns": [{"index": 0, "start": 2.0, "end": 2.8}]},
        ]}
        result = voice.process_voice_step("voice.fingerprint.2", {"config": self.CONFIG, "diarization": diarization}, None)
        by_key = {(f["speaker_label"], f["scope"], f.get("turn_index")): f["measures"] for f in result["fingerprints"]}
        first = by_key[("SPEAKER_00", "speaker", None)]
        self.assertEqual((first["version"], first["net_speech_seconds"], first["overlap_ratio"], first["overlapped"]), (1, 3.0, 0.2, False))
        self.assertIsNone(first["clipped_ratio"])
        self.assertIsNotNone(first["coherence"])
        second = by_key[("SPEAKER_01", "speaker", None)]
        self.assertEqual((second["net_speech_seconds"], second["overlapped"], second["coherence"]), (0.0, True, None))


def write_wav(path, samples, rate=16000):
    with wave.open(path, "wb") as out:
        out.setnchannels(1)
        out.setsampwidth(2)
        out.setframerate(rate)
        out.writeframes(struct.pack(f"<{len(samples)}h", *samples))


if __name__ == "__main__":
    unittest.main()
