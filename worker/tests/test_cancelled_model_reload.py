"""A cancelled in-process inference leaves its model unusable: it is reloaded before the next job.

``cancellation.interruptible`` raises ``JobCancelled`` at an arbitrary Python instruction of the
inference (ADR 0029). A library lock taken there may never be released, so reusing the cached
model would deadlock every later job. The fakes below hold such a lock: it is only released when
the inference completes, never when it is interrupted. No real model.
"""

import os
import tempfile
import threading
import time
import unittest

from snitch_worker import cancellation, voice_nemo, voice_pyannote

CONFIG = {"provider": "fixture", "model": "synthetic/model", "model_version": "v1", "min_fingerprint_seconds": 1.0, "min_quality": 0.4}


class LockingModel:
    """A model whose inference takes a lock it releases only on completion.

    The first inference of the test (``long_runs``) computes for a minute, in Python as torch does
    between its layers; later ones answer at once. A lock left held makes a later inference fail
    instead of hanging."""

    def __init__(self, answer, long_runs):
        self.lock = threading.Lock()
        self.started = threading.Event()
        self.answer = answer
        self.long_runs = long_runs

    def __call__(self, *_args, **_kwargs):
        if not self.lock.acquire(timeout=1):
            raise RuntimeError("deadlock: the model lock is still held")
        if self.long_runs:
            self.long_runs.pop()
            self.started.set()
            deadline = time.monotonic() + 60
            while time.monotonic() < deadline:
                sum(range(100))
        self.lock.release()
        return self.answer()


class _Segment:
    def __init__(self, start, end):
        self.start, self.end = start, end


class _Annotation:
    def itertracks(self, yield_label=False):
        return iter([(_Segment(0.0, 1.5), None, "speaker_0")])


def _env(root):
    env = {
        "SNITCH_VOICE_DIARIZATION_MODEL_PATH": os.path.join(root, "diarization"),
        "SNITCH_VOICE_EMBEDDING_MODEL_PATH": os.path.join(root, "embedding"),
        "SNITCH_VOICE_EMBEDDING_MODEL_VERSION": "wespeaker-test-1",
    }
    os.makedirs(env["SNITCH_VOICE_DIARIZATION_MODEL_PATH"])
    os.makedirs(env["SNITCH_VOICE_EMBEDDING_MODEL_PATH"])
    with open(os.path.join(env["SNITCH_VOICE_DIARIZATION_MODEL_PATH"], "model.nemo"), "w"):
        pass
    return env


def _cancel_when(started, call):
    """Run ``call`` in a job scope whose lease is refused once ``started`` is set."""
    refused = threading.Event()

    def refuse():
        started.wait(10)
        refused.set()

    threading.Thread(target=refuse, daemon=True).start()
    began = time.monotonic()
    with cancellation.scope(refused):
        try:
            call()
        except cancellation.JobCancelled:
            return time.monotonic() - began
    raise AssertionError("the inference was not cancelled")


class CancelledModelReloadTests(unittest.TestCase):
    def _diarization_reloads(self, provider_class, answer):
        models, long_runs = [], [1]

        def load(_path, _device):
            models.append(LockingModel(answer, long_runs))
            return models[-1]

        with tempfile.TemporaryDirectory() as root:
            provider = provider_class(environ=_env(root), load_pipeline=load, device_probe=lambda: "cpu")
            duration = _cancel_when(_Started(models), lambda: provider.diarize(CONFIG, "/a.wav", 0))
            self.assertLess(duration, 10)
            self.assertTrue(models[0].lock.locked(), "the fake leaves its lock held, as a library might")
            result = provider.diarize(CONFIG, "/a.wav", 0)
        self.assertEqual(len(models), 2, "the interrupted model is reloaded before the next job")
        self.assertEqual(result["speakers"], [{"label": "SPEAKER_00", "turns": [{"start": 0.0, "end": 1.5}]}])

    def test_pyannote_pipeline_is_reloaded_after_a_cancelled_diarization(self):
        self._diarization_reloads(voice_pyannote.PyannoteProvider, _Annotation)

    def test_nemo_model_is_reloaded_after_a_cancelled_diarization(self):
        self._diarization_reloads(voice_nemo.NemoProvider, lambda: ["0.0 1.5 speaker_0"])

    def test_embedder_is_reloaded_after_a_cancelled_fingerprint(self):
        models, long_runs = [], [1]

        def load(_path, _device):
            models.append(LockingModel(lambda: [0.5, 0.5, 0.5, 0.5], long_runs))
            model = models[-1]
            return lambda _audio_path: voice_pyannote.DecodedAudio(model, 16000, 1, 16000 * 60)

        diarization = {"job_id": 1, "speakers": [{"label": "s0", "turns": [{"index": 0, "start": 0.0, "end": 2.0}]}]}
        with tempfile.TemporaryDirectory() as root:
            provider = voice_pyannote.PyannoteProvider(environ=_env(root), load_embedder=load, device_probe=lambda: "cpu")
            duration = _cancel_when(_Started(models), lambda: provider.fingerprint(CONFIG, diarization, "/a.wav", 0))
            self.assertLess(duration, 10)
            result = provider.fingerprint(CONFIG, diarization, "/a.wav", 0)
        self.assertEqual(len(models), 2, "the interrupted embedder is reloaded before the next job")
        self.assertEqual(len(result["fingerprints"]), 2, "the speaker and its turn")


class _Started:
    """An event-like view: set once the first loaded model has started its inference."""

    def __init__(self, models):
        self.models = models

    def wait(self, timeout):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if self.models and self.models[0].started.is_set():
                return True
            time.sleep(0.01)
        return False


if __name__ == "__main__":
    unittest.main()
