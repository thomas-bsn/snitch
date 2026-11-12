"""Separation step of an end-to-end benchmark chain (#402), on synthetic turns, transcripts and audio only.

No model is loaded: the separator, the transcription service and the fingerprint provider are fakes. A fake source
is a WAV whose samples all hold its number, which is how the fake transcriber and fingerprints tell sources apart.
"""

import contextlib
import copy
import io
import os
import shutil
import tempfile
import unittest
import wave
from types import SimpleNamespace
from unittest.mock import patch

from snitch_worker import benchmark_separation, log, runner
from snitch_worker.voice import VoiceProviderUnavailable


def write_source(path, value, seconds=4.0, rate=16000):
    with wave.open(path, "wb") as audio:
        audio.setnchannels(1)
        audio.setsampwidth(2)
        audio.setframerate(rate)
        audio.writeframes(int(value).to_bytes(2, "little", signed=True) * int(seconds * rate))


def source_number(path):
    with wave.open(path, "rb") as audio:
        return int.from_bytes(audio.readframes(1), "little", signed=True)


class FakeSeparator:
    """Separates every window into ``len(sources)`` sources; ``cut`` moves the start the model really separated."""

    def __init__(self, sources=3, cut=0.0, error=None):
        self.calls, self.sources, self.cut, self.error = [], sources, cut, error

    def __call__(self, environ, repo, revision, audio, output_dir, windows):
        self.calls.append((repo, revision, windows))
        if self.error:
            raise VoiceProviderUnavailable(self.error)
        out = []
        for number, (start, end) in enumerate(windows):
            sources = []
            for index in range(self.sources):
                path = os.path.join(output_dir, f"window-{number}-source-{index}.wav")
                write_source(path, index + 1, end - start - self.cut)
                sources.append({"index": index, "path": path, "samples": 1})
            out.append({"sample_rate": 16000, "start": start + self.cut, "end": end, "gain": 1.0, "sources": sources})
        return out


class FakeTranscriber:
    """Source n says the words of ``said[n]``, timed from the start of the source."""

    def __init__(self, said):
        self.said = said

    def transcribe(self, wav_path, vocabulary=None):
        words = [{"start": s, "end": e, "word": w} for s, e, w in self.said.get(source_number(wav_path), [])]
        segments = [{"start": words[0]["start"], "end": words[-1]["end"], "text": " ".join(w["word"] for w in words), "words": words}] if words else []
        return {"provider": "fixture", "model": "asr-x", "version": "v", "segments": segments}


class FakeFingerprints:
    """Local speakers get a fixed vector by label; a source gets the vector of its number."""

    def __init__(self, speakers, sources):
        self.speakers, self.sources = speakers, sources

    def ensure_ready(self, config):
        pass

    def fingerprint(self, config, diarization, audio, stream):
        labels = [s["label"] for s in diarization["speakers"]]
        vectors = {"source": self.sources.get(source_number(audio))} if labels == ["source"] else self.speakers
        return {"fingerprints": [{"speaker_label": label, "scope": "speaker", "embedding": vectors[label]}
                                 for label in labels if vectors.get(label) is not None]}


def turns(**spans):
    return [{"label": label, "turns": [{"index": i, "start": s, "end": e} for i, (s, e) in enumerate(pairs)]}
            for label, pairs in spans.items()]


def segment(*words):
    ws = [{"start": s, "end": e, "word": w} for s, e, w in words]
    return {"start": ws[0]["start"], "end": ws[-1]["end"], "text": " ".join(w["word"] for w in ws), "words": ws}


S0, S1, S2 = [1.0, 0.0, 0.0], [0.0, 1.0, 0.0], [0.0, 0.0, 1.0]


class SeparationStepTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.mkdtemp(prefix="snitch-separation-test-")
        self.addCleanup(shutil.rmtree, self.temporary)
        self.wav = os.path.join(self.temporary, "track.wav")
        write_source(self.wav, 0, 60.0)
        # The sources are already 16 kHz WAV: preparing one is a copy.
        patcher = patch.object(runner, "prepare_audio", lambda src, dst, _index: shutil.copyfile(src, dst))
        patcher.start()
        self.addCleanup(patcher.stop)

    def run_step(self, speakers, segments, separator, said=None, sources=None, margin=0.05, silent=("S1",), spans=None):
        voice = {"config": {"provider": "test"}, "diarization": {"job_id": 5, "speakers": speakers}}
        # Unless told otherwise, Snitch's rule gave every word of the transcript to S0.
        if spans is None:
            spans = [{"speaker": "S0", "start": w["start"], "end": w["end"]} for s in segments for w in s["words"]]
        separation = {"model": "fixture/sep", "revision": "r" * 40, "ambiguity_margin": margin, "segments": segments,
                      "silent_speakers": list(silent), "word_spans": spans}
        providers = {"test": FakeFingerprints({"S0": S0, "S1": S1, "S2": S2}, sources or {})}
        return benchmark_separation.run(voice, separation, self.wav, 9, FakeTranscriber(said or {}), {"language": "fr"}, providers, {}, separate=separator)

    def test_a_track_without_overlap_never_calls_the_model(self):
        separator = FakeSeparator()
        result = self.run_step(turns(S0=[(0.0, 4.0)], S1=[(4.0, 8.0)]), [segment((1.0, 1.5, "bonjour")), segment((5.0, 5.5, "salut"))], separator, silent=[])
        self.assertEqual(separator.calls, [])
        self.assertEqual(result, {"audio_stream_index": 9, "silent_speakers": [], "zones": [], "sources": [], "added": []})

    def test_an_overlap_on_a_track_where_every_local_speaker_has_words_is_not_separated(self):
        separator = FakeSeparator()
        result = self.run_step(turns(S0=[(0.0, 5.0)], S1=[(3.0, 8.0)]), [segment((1.0, 1.5, "oui")), segment((6.5, 7.0, "non"))], separator, silent=[])
        self.assertEqual(separator.calls, [])
        self.assertEqual((result["silent_speakers"], result["zones"]), ([], []))

    def test_the_speakers_without_words_are_those_snitch_gives(self):
        # S1 holds "oui" by its turns, but Snitch's production rule gave that isolated word to S0 (#403): the Worker
        # has no rule of its own and targets the overlap of S1.
        separator = FakeSeparator(sources=1)
        segments = [segment((0.5, 1.0, "bonjour"), (3.5, 3.8, "oui"), (4.5, 4.8, "alors"))]
        result = self.run_step(turns(S0=[(0.0, 3.4), (3.9, 5.0)], S1=[(3.0, 8.0)]), segments, separator, silent=["S1"])
        self.assertEqual(result["silent_speakers"], ["S1"])
        self.assertEqual([z["silent_speakers"] for z in result["zones"]], [["S1"], ["S1"]])
        self.assertEqual(len(separator.calls), 1)

    def test_only_a_source_given_to_a_speaker_without_words_in_the_zone_adds_words(self):
        # S0 speaks 0-5 s and S1 3-8 s; S1 never gets a word, and in the overlap 3-5 s only S0 has words.
        segments = [segment((0.5, 1.0, "bonjour"), (3.5, 4.0, "alors"))]
        before = copy.deepcopy(segments)
        separator = FakeSeparator(sources=3)
        said = {1: [(1.6, 2.0, "attends"), (3.5, 3.8, "après")], 2: [(1.5, 2.0, "alors")], 3: [(1.0, 1.2, "euh")]}
        # Source 1 is clearly S1, source 2 clearly S0 (who already had words), source 3 sits between S0 and S1.
        sources = {1: [0.1, 0.99, 0.0], 2: [0.99, 0.1, 0.0], 3: [0.70, 0.71, 0.0]}
        result = self.run_step(turns(S0=[(0.0, 5.0)], S1=[(3.0, 8.0)]), segments, separator, said, sources)

        self.assertEqual([call[:2] for call in separator.calls], [("fixture/sep", "r" * 40)])
        self.assertEqual(separator.calls[0][2], [(2.0, 6.0)], "the zone with a margin of context, in one call")
        self.assertEqual(result["silent_speakers"], ["S1"])
        self.assertEqual(result["zones"], [{"start": 3.0, "end": 5.0, "silent_speakers": ["S1"]}])
        self.assertEqual([(s["index"], s["speaker"]) for s in result["sources"]], [(0, "S1"), (1, "S0"), (2, None)])
        # Only the words of S1's source that fall in the zone are added; the window's margin brings none.
        self.assertEqual(result["added"], [{"zone": 0, "source": 0, "speaker": "S1", "start": 3.6, "end": 4.0, "word": "attends"}])
        self.assertEqual(segments, before, "the original words are never removed nor replaced")

    def test_a_speaker_with_words_elsewhere_but_none_in_the_zone_gets_the_words_of_its_source(self):
        # S2, without any word, makes 3-5 s a zone; S1 talks there too but its only word is at 6.5 s, outside the zone.
        segments = [segment((0.5, 1.0, "bonjour"), (3.5, 4.0, "alors")), segment((6.5, 7.0, "voilà"))]
        spans = [{"speaker": "S0", "start": 0.5, "end": 1.0}, {"speaker": "S0", "start": 3.5, "end": 4.0}, {"speaker": "S1", "start": 6.5, "end": 7.0}]
        said = {1: [(1.6, 2.0, "attends")], 2: [(1.5, 2.0, "alors")]}
        sources = {1: [0.1, 0.99, 0.0], 2: [0.99, 0.1, 0.0]}
        result = self.run_step(turns(S0=[(0.0, 5.0)], S1=[(3.0, 8.0)], S2=[(3.5, 4.5)]), segments, FakeSeparator(sources=2), said, sources,
                               silent=["S2"], spans=spans)
        self.assertEqual(result["zones"], [{"start": 3.0, "end": 5.0, "silent_speakers": ["S2"]}])
        self.assertEqual([(s["speaker"], [w["word"] for w in s["words"]]) for s in result["sources"]], [("S1", ["attends"]), ("S0", ["alors"])])
        # S1 has no word in the zone: its source's words are added; S0 already has "alors" there: nothing of its source is.
        self.assertEqual(result["added"], [{"zone": 0, "source": 0, "speaker": "S1", "start": 3.6, "end": 4.0, "word": "attends"}])

    def test_a_source_without_a_clear_speaker_adds_nothing(self):
        separator = FakeSeparator(sources=2)
        said = {1: [(1.5, 2.0, "attends")], 2: [(1.5, 2.0, "quoi")]}
        sources = {1: [0.70, 0.71, 0.0], 2: [0.71, 0.70, 0.0]}
        result = self.run_step(turns(S0=[(0.0, 5.0)], S1=[(3.0, 8.0)]), [segment((3.5, 4.0, "alors"))], separator, said, sources)
        self.assertEqual([s["speaker"] for s in result["sources"]], [None, None])
        self.assertEqual(result["added"], [])

    def test_added_words_are_timed_in_the_clip_for_a_zone_late_in_the_track_cut_by_the_model(self):
        # The zone is at 41-43 s; the model separated from 40.25 s instead of 40 s.
        separator = FakeSeparator(sources=2, cut=0.25)
        said = {1: [(1.25, 1.75, "attends")]}
        sources = {1: [0.0, 1.0, 0.0], 2: [1.0, 0.0, 0.0]}
        result = self.run_step(turns(S0=[(38.0, 43.0)], S1=[(41.0, 47.0)]), [segment((41.5, 42.0, "alors"))], separator, said, sources)
        [word] = result["added"]
        self.assertAlmostEqual(word["start"], 41.5, places=6)
        self.assertAlmostEqual(word["end"], 42.0, places=6)

    def test_a_lack_of_gpu_memory_fails_the_step_with_its_cause(self):
        separator = FakeSeparator(error="gpu_out_of_memory")
        with self.assertRaises(VoiceProviderUnavailable) as caught:
            self.run_step(turns(S0=[(0.0, 5.0)], S1=[(3.0, 8.0)]), [segment((3.5, 4.0, "alors"))], separator)
        self.assertEqual(caught.exception.reason, "gpu_out_of_memory")
        self.assertEqual(len(separator.calls), 1)


class RunnerSeparationTests(unittest.TestCase):
    def test_the_worker_reports_a_lack_of_gpu_memory_with_its_stable_cause(self):
        log.configure("INFO", stream=io.StringIO())
        task = SimpleNamespace(id=7, clip_id=3, step="benchmark.separation.0", input_version="bs-d1-t2", cycle=0, attempts=1, lease_owner="wk_synthetic")
        task_input = SimpleNamespace(media_url="/media", pipeline=None, transcription_model="asr-x", vocabulary=[], voice=SimpleNamespace(to_dict=dict), separation=SimpleNamespace(to_dict=dict))
        sent = []
        with contextlib.ExitStack() as stack:
            for active in (
                patch.object(runner, "pulse", return_value=object()),
                patch.object(runner, "claim_job", return_value=SimpleNamespace(job=task, lease_token="lt_synthetic-lease-token")),
                patch.object(runner, "heartbeat_job", return_value=object()),
                patch.object(runner, "get_job_input", return_value=task_input),
                patch.object(runner, "download_media"),
                patch.object(runner, "prepare_audio"),
                patch.object(runner.benchmark_separation, "run", side_effect=VoiceProviderUnavailable("gpu_out_of_memory")),
                patch.object(runner, "fail_job", side_effect=lambda **kwargs: sent.append(kwargs["body"]) or object()),
            ):
                stack.enter_context(active)
            self.assertTrue(runner.run_once(SimpleNamespace(_base_url="https://snitch.example"), transcriber=FakeTranscriber({})))
        self.assertEqual([body.to_dict()["error"] for body in sent], ["gpu_out_of_memory"])

    def completed(self, step, modules):
        """The result a Worker sends for one benchmark task whose work is faked, with ``modules`` imported."""
        log.configure("INFO", stream=io.StringIO())
        task = SimpleNamespace(id=7, clip_id=3, step=step, input_version="bs-d1-t2", cycle=0, attempts=1, lease_owner="wk_synthetic")
        task_input = SimpleNamespace(media_url="/media", pipeline=None, transcription_model="asr-x", vocabulary=[], voice=SimpleNamespace(to_dict=dict), separation=SimpleNamespace(to_dict=dict))
        sent = []
        with contextlib.ExitStack() as stack:
            for active in (
                patch.dict("sys.modules", modules),
                patch.object(runner, "pulse", return_value=object()),
                patch.object(runner, "claim_job", return_value=SimpleNamespace(job=task, lease_token="lt_synthetic-lease-token")),
                patch.object(runner, "heartbeat_job", return_value=object()),
                patch.object(runner, "get_job_input", return_value=task_input),
                patch.object(runner, "download_media"),
                patch.object(runner, "prepare_audio"),
                patch.object(runner.benchmark_separation, "run", return_value={"audio_stream_index": 0, "added": []}),
                patch.object(runner, "complete_job", side_effect=lambda **kwargs: sent.append(kwargs["body"]) or object()),
            ):
                stack.enter_context(active)
            self.assertTrue(runner.run_once(SimpleNamespace(_base_url="https://snitch.example"), transcriber=FakeTranscriber({})))
        [body] = sent
        return body.to_dict()["result"]

    def test_a_benchmark_task_reports_its_inference_time_and_its_gpu_peak_under_cuda(self):
        cuda = SimpleNamespace(is_available=lambda: True, reset_peak_memory_stats=lambda: None, max_memory_allocated=lambda: 123456)
        result = self.completed("benchmark.separation.0", {"torch": SimpleNamespace(cuda=cuda)})
        self.assertGreaterEqual(result["inference_seconds"], 0.0)
        self.assertEqual(result["gpu_peak_bytes"], 123456)

    def test_without_cuda_the_gpu_peak_is_absent_not_zero(self):
        cuda = SimpleNamespace(is_available=lambda: False)
        result = self.completed("benchmark.separation.0", {"torch": SimpleNamespace(cuda=cuda)})
        self.assertIn("inference_seconds", result)
        self.assertNotIn("gpu_peak_bytes", result)


if __name__ == "__main__":
    unittest.main()
