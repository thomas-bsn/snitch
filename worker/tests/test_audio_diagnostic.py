"""Contract/adapter tests; real neural inference lives in test_audio_acoustic.py."""
import json
import os
import subprocess
import tempfile
import unittest
import wave
from pathlib import Path

from snitch_worker import audio_diagnostic as audio
from snitch_worker import voice_manifest


class AudioDiagnosticTests(unittest.TestCase):
    def test_missing_model_never_means_no_speech(self):
        with tempfile.TemporaryDirectory() as root:
            result = audio.diagnose('missing.media', 3, root)
        self.assertEqual((result['status'], result['cause']), ('failed', 'model_missing'))
        self.assertEqual(result['intervals'], [])

    def test_per_channel_evidence_and_uncertainty(self):
        # A short neural hit is uncertainty, never a safe exclusion.
        self.assertEqual(audio.classify([[0.01, 0.3, 0.01]], .096, 0)['status'], 'ambiguous')
        self.assertEqual(audio.classify([[0.01] * 10], .32, 0)['status'], 'no_speech')
        result = audio.classify([[0.0] * 10, [0.8] * 10], .32, 1.5)
        self.assertEqual(result['status'], 'speech')
        self.assertEqual(result['intervals'], [{'start': 1.5, 'end': 1.82}])
        self.assertEqual(audio.classify([[0.01] * 10], .32, None)['status'], 'ambiguous')

    def test_decode_real_stream_index_and_channels(self):
        with tempfile.TemporaryDirectory() as root:
            source, target = Path(root)/'multi.mkv', Path(root)/'decoded.wav'
            subprocess.run(['ffmpeg', '-v', 'error', '-f', 'lavfi', '-i', 'color=s=16x16:d=1',
                '-f', 'lavfi', '-i', 'anullsrc=r=16000:cl=stereo', '-map', '0:v', '-map', '1:a',
                '-t', '1', '-c:a', 'pcm_s16le', str(source)], check=True)
            audio.decode(str(source), str(target), 1)
            with wave.open(str(target)) as wav:
                self.assertEqual((wav.getnchannels(), wav.getframerate(), wav.getnframes()), (2, 16000, 16000))
            with self.assertRaises(audio.DiagnosticError):
                audio.decode(str(source), str(target), 7)


class FakeScorer:
    """Deterministic injected scorer for orchestration; never evidence about real detector quality."""

    def __init__(self, value):
        self.value, self.resets = value, 0

    def reset_states(self):
        self.resets += 1

    def __call__(self, _window, rate):
        assert rate == 16000
        return self.value


class OrchestrationTests(unittest.TestCase):
    def media(self, root):
        path = Path(root) / 'two.mkv'
        subprocess.run(['ffmpeg', '-v', 'error', '-f', 'lavfi', '-i', 'color=s=16x16:d=1', '-f', 'lavfi', '-i',
                        'sine=frequency=200:sample_rate=16000:duration=1', '-f', 'lavfi', '-i',
                        'anullsrc=r=16000:cl=stereo', '-map', '0:v', '-map', '1:a', '-map', '2:a', '-t', '1',
                        '-c:a', 'pcm_s16le', str(path)], check=True)
        return str(path)

    def task(self, index, **overrides):
        from snitch_worker.api.snitch_client.models import WorkerAudioDiagnosticInput
        values = dict(plan_id=3, track_index=index, input_version='a' * 64, detector_version=audio.VERSION,
                      detector_config_version=audio.CONFIG_VERSION, start_seconds=1.25, duration_seconds=1.0)
        values.update(overrides)
        return type('Task', (), {'audio_diagnostic': WorkerAudioDiagnosticInput(**values)})()

    def run_step(self, media, index, scorer, **overrides):
        from unittest.mock import patch
        from snitch_worker import runner
        with patch.object(runner, 'download_media', lambda _c, _t, _l, destination: Path(destination).write_bytes(Path(media).read_bytes())):
            return runner.run_audio_diagnostic(None, self.task(index, **overrides), 'lease', f'audio.diagnostic.{index}', scorer)

    def test_real_stream_index_offset_once_and_every_channel_reset(self):
        with tempfile.TemporaryDirectory() as root:
            media = self.media(root)
            scorer = FakeScorer(.9)
            result = self.run_step(media, 2, scorer)
            self.assertEqual(result['status'], 'speech')
            self.assertEqual(result['intervals'][0]['start'], 1.25)
            self.assertEqual(scorer.resets, 2)  # stereo: two channels never share recurrent state
            self.assertEqual(self.run_step(media, 1, FakeScorer(.01), start_seconds=0.)['status'], 'no_speech')

    def test_unsupported_or_mismatched_plan_is_a_job_failure_not_an_observation(self):
        with tempfile.TemporaryDirectory() as root:
            media = self.media(root)
            with self.assertRaises(ValueError):
                self.run_step(media, 1, FakeScorer(.0), detector_version='another-detector')
            with self.assertRaises(ValueError):
                self.run_step(media, 1, FakeScorer(.0), track_index=2)

    def test_unknown_offset_or_decode_error_is_never_no_speech(self):
        with tempfile.TemporaryDirectory() as root:
            media = self.media(root)
            result = audio.diagnose(media, 1, offset=None, detector=FakeScorer(.0))
            self.assertEqual((result['status'], result['cause']), ('ambiguous', 'timeline_unknown'))
            self.assertEqual(audio.diagnose(media, 9, detector=FakeScorer(.0))['cause'], 'decode_failed')

    def test_capability_is_declared_only_where_the_detector_can_run(self):
        from unittest.mock import patch
        from snitch_worker import runner
        with patch.object(audio, 'available', return_value=False):
            self.assertNotIn('audio.diagnostic.*', runner.supported_steps())
        with patch.object(audio, 'available', return_value=True):
            self.assertIn('audio.diagnostic.*', runner.supported_steps())


class InstallationTests(unittest.TestCase):
    def test_no_implicit_acquisition(self):
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as root, patch('urllib.request.urlopen', side_effect=AssertionError('network')):
            with self.assertRaises(audio.DiagnosticError) as raised:
                audio.install(root)
            self.assertEqual(raised.exception.cause, 'model_missing')
            self.assertFalse(audio.ready({voice_manifest.EMBEDDING_PATH_ENV: str(Path(root) / 'embedding')}))
            self.assertEqual(audio.diagnose('x.mkv', 1, root)['cause'], 'model_missing')

    def test_corrupt_artifact_is_refused_before_execution_and_never_ready(self):
        with tempfile.TemporaryDirectory() as root:
            source = Path(root) / 'fake.jit'
            source.write_bytes(b'not the pinned model')
            with self.assertRaises(audio.DiagnosticError) as raised:
                audio.install(str(Path(root) / 'target'), source=str(source))
            self.assertEqual(raised.exception.cause, 'model_checksum_mismatch')
            self.assertFalse(Path(root, 'target', '.snitch-ready').exists())
            self.assertFalse(Path(root, 'target', audio.MODEL).exists())

    def manager(self, root, installer):
        from snitch_worker import voice_models
        return voice_models.ModelManager({voice_manifest.EMBEDDING_PATH_ENV: str(Path(root) / 'embedding')},
                                         lambda *_: None, lambda *_: None, install_vad=installer)

    def order(self, **overrides):
        item = {'model_id': 'silero-vad-v6.2.1', 'kind': 'vad', 'repo': audio.REPO, 'revision': audio.REVISION, 'role': 'ordered'}
        return item | overrides

    def test_order_installs_only_the_pinned_detector_with_closed_causes(self):
        from urllib.error import URLError
        with tempfile.TemporaryDirectory() as root:
            calls = []
            def good(path, **kwargs):
                calls.append(kwargs['download'])
                Path(path).mkdir(parents=True, exist_ok=True)
                Path(path, '.snitch-ready').write_text('ok')
            reports = []
            self.manager(root, good).prepare(self.order(), reports.append)
            self.assertEqual([r['state'] for r in reports][-1], 'ready')
            self.assertEqual(calls, [True])
            for override, installer, cause in (
                ({'revision': 'f' * 40}, good, 'runtime_incompatible'),
                ({}, lambda *_a, **_k: (_ for _ in ()).throw(URLError('offline')), 'hf_network'),
                ({}, lambda *_a, **_k: (_ for _ in ()).throw(audio.DiagnosticError('model_checksum_mismatch')), 'runtime_incompatible'),
            ):
                reports.clear()
                with tempfile.TemporaryDirectory() as other:
                    self.manager(other, installer).prepare(self.order(**override), reports.append)
                self.assertEqual((reports[-1]['state'], reports[-1]['cause_code']), ('failed', cause))

    def test_order_contract_accepts_the_detector_engine_only(self):
        from snitch_worker import model_orders
        order = {'action': 'install', 'kind': 'vad', 'engine': 'silero-jit', 'model_id': 'silero-vad-v6.2.1',
                 'repo': audio.REPO, 'revision': audio.REVISION, 'idempotency_key': 'diagnostic-install-01'}
        self.assertTrue(model_orders.OrderExecutor._well_formed(order))
        self.assertFalse(model_orders.OrderExecutor._well_formed(order | {'engine': 'pyannote'}))
