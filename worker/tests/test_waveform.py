import array
import tempfile
import unittest
import wave
from pathlib import Path

from snitch_worker import waveform


def write_wav(path, samples, width=2, rate=16000):
    with wave.open(str(path), "wb") as out:
        out.setnchannels(1)
        out.setsampwidth(width)
        out.setframerate(rate)
        if width == 2:
            out.writeframes(array.array("h", samples).tobytes())
        else:
            out.writeframes(bytes(len(samples)))


class ComputePeaksTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.root = Path(self.directory.name)

    def tearDown(self):
        self.directory.cleanup()

    def test_loud_half_and_silent_half_produce_bounded_peaks(self):
        samples = [0] * 16000 + [16384, -32768] * 8000
        path = self.root / "prepared.wav"
        write_wav(path, samples)

        peaks = waveform.compute_peaks(str(path), count=40)

        self.assertLessEqual(len(peaks), 40)
        self.assertGreater(len(peaks), 0)
        self.assertTrue(all(isinstance(value, int) and 0 <= value <= 255 for value in peaks))
        self.assertEqual(peaks[0], 0)
        self.assertEqual(peaks[-1], 255)

    def test_empty_audio_has_no_peaks(self):
        path = self.root / "empty.wav"
        write_wav(path, [])

        self.assertEqual(waveform.compute_peaks(str(path)), [])

    def test_non_16_bit_audio_has_no_peaks(self):
        path = self.root / "eight.wav"
        write_wav(path, [0] * 100, width=1)

        self.assertEqual(waveform.compute_peaks(str(path)), [])

    def test_attach_peaks_adds_key_without_touching_segments(self):
        path = self.root / "prepared.wav"
        write_wav(path, [1000] * 320)
        result = {"segments": [{"start": 0.0}]}

        waveform.attach_peaks(result, str(path))

        self.assertIn("peaks", result)
        self.assertEqual(result["segments"], [{"start": 0.0}])


if __name__ == "__main__":
    unittest.main()


class UnreadableAudioTests(unittest.TestCase):
    def test_unreadable_file_has_no_peaks_and_does_not_fail(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "not-audio.wav"
            path.write_bytes(b"synthetic, not a wave file")

            self.assertEqual(waveform.compute_peaks(str(path)), [])
