"""Actual pinned Silero inference, opt-in with SNITCH_TEST_VAD_DIRECTORY.

Offline test execution: model acquisition must happen explicitly beforehand.
No network, voice service or real library input is used here.
"""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

from snitch_worker import audio_diagnostic as audio


@unittest.skipUnless(os.environ.get('SNITCH_TEST_VAD_DIRECTORY'), 'explicitly installed Silero required')
class AcousticTests(unittest.TestCase):
    def setUp(self):
        self.root = tempfile.TemporaryDirectory()
        self.addCleanup(self.root.cleanup)
        self.voice = str(Path(__file__).parent/'fixtures/acoustic/voice.wav')
        self.model = os.environ['SNITCH_TEST_VAD_DIRECTORY']

    def make(self, name, args):
        path = str(Path(self.root.name, name))
        subprocess.run(['ffmpeg', '-v', 'error', '-y', *args, path], check=True)
        return path

    def test_both_orders_and_misleading_metadata(self):
        for reverse in (False, True):
            path = self.make(f'order{reverse}.mkv', ['-f','lavfi','-i','color=s=16x16:d=8',
                '-i',self.voice,'-f','lavfi','-i','sine=frequency=150:sample_rate=16000:duration=8',
                '-map','0:v','-map','2:a' if reverse else '1:a','-map','1:a' if reverse else '2:a',
                '-metadata:s:a:0','title=Game','-metadata:s:a:1','title=Microphone',
                '-c:a','pcm_s16le','-t','8'])
            outcomes = [audio.diagnose(path,i,self.model)['status'] for i in (1,2)]
            self.assertEqual(outcomes, ['no_speech','speech'] if reverse else ['speech','no_speech'])

    def test_game_dialogue_silence_low_gain_failure(self):
        game = self.make('dialogue.wav', ['-i',self.voice,'-f','lavfi','-i',
            'sine=frequency=150:sample_rate=16000:duration=8', '-filter_complex',
            '[1:a]volume=0.05[t];[0:a][t]amix=inputs=2:duration=first', '-ar','16000','-c:a','pcm_s16le'])
        silence = self.make('silence.wav',['-f','lavfi','-i','anullsrc=r=16000:cl=stereo','-t','1','-c:a','pcm_s16le'])
        quiet = self.make('quiet.wav',['-i',self.voice,'-af','volume=0.0005','-c:a','pcm_s16le'])
        self.assertEqual(audio.diagnose(game,0,self.model)['status'], 'speech')
        self.assertEqual(audio.diagnose(silence,0,self.model)['status'], 'no_speech')
        self.assertEqual(audio.diagnose(quiet,0,self.model)['status'], 'ambiguous')
        self.assertEqual(audio.diagnose(game,8,self.model)['cause'], 'decode_failed')

    def test_opposite_phase_stereo_and_offline_install(self):
        stereo = self.make('antiphase.wav',['-i',self.voice,'-af',
            'pan=stereo|c0=c0|c1=-1*c0','-c:a','pcm_s16le'])
        self.assertEqual(audio.diagnose(stereo,0,self.model)['status'], 'speech')
        target = str(Path(self.root.name,'imported'))
        audio.install(target,source=str(Path(self.model,audio.MODEL)))
        self.assertEqual(audio.diagnose(self.voice,0,target)['status'],'speech')
        Path(target,audio.MODEL).write_bytes(b'corrupt')
        self.assertEqual(audio.diagnose(self.voice,0,target)['cause'],'model_checksum_mismatch')

    def test_order_path_installs_then_detector_reads_it_offline(self):
        """The Worker's order path with the real installer (artifact imported from the explicit local copy)."""
        from snitch_worker import voice_manifest, voice_models
        environ = {voice_manifest.EMBEDDING_PATH_ENV: str(Path(self.root.name, 'embedding'))}
        reports = []
        manager = voice_models.ModelManager(environ, lambda *_: None, lambda *_: None,
            install_vad=lambda path, **_: audio.install(path, source=str(Path(self.model, audio.MODEL))))
        manager.prepare({'model_id': 'silero-vad-v6.2.1', 'kind': 'vad', 'repo': audio.REPO,
                         'revision': audio.REVISION, 'role': 'ordered'}, reports.append)
        self.assertEqual(reports[-1]['state'], 'ready')
        self.assertTrue(audio.ready(environ))
        self.assertTrue(manager.holds({'kind': 'vad', 'repo': audio.REPO, 'revision': audio.REVISION}))
        result = audio.diagnose(self.voice, 0, audio.model_directory(environ), offset=2.5)
        self.assertEqual(result['status'], 'speech')
        self.assertGreaterEqual(result['intervals'][0]['start'], 2.5)  # stream offset applied once
