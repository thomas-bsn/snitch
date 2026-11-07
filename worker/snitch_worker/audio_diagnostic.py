"""Pinned offline neural speech diagnostics (ADR 0048). No implicit acquisition.

Model installation is a separate explicit operation. Inference decodes every
channel independently, uses bounded 32ms windows and cooperates with lease loss.
Scores are evidence, not calibrated probabilities or speaker identification.
"""
from __future__ import annotations

import argparse
import array
import hashlib
import math
import os
import subprocess
import sys
from pathlib import Path
import tempfile
import urllib.request
import wave
from typing import Callable

from snitch_worker import cancellation

REPO = 'snakers4/silero-vad'
REVISION = '7e30209a3e901f9842f81b225f3e93d8199902b1'
SHA256 = 'e1122837f4154c511485fe0b9c64455f7b929c96fbb8d79fbdb336383ebd3720'
MODEL = 'silero_vad.jit'
VERSION = 'silero-v6.2.1-' + REVISION
CONFIG_VERSION = 'channels-16k-512-hi050-lo020-min096-v1'
URL = f'https://raw.githubusercontent.com/{REPO}/{REVISION}/src/silero_vad/data/{MODEL}'
LICENSE = '''MIT License

Copyright (c) 2020-present Silero Team

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
'''


class DiagnosticError(Exception):
    def __init__(self, cause: str):
        self.cause = cause
        super().__init__(cause)


def model_directory(environ=None):
    from snitch_worker.voice_models import model_path
    return model_path(os.environ if environ is None else environ, REPO, REVISION)


def verified(path: str) -> bool:
    try:
        return hashlib.sha256(Path(path, MODEL).read_bytes()).hexdigest() == SHA256
    except OSError:
        return False


def load(path: str):
    if not Path(path, MODEL).exists():
        raise DiagnosticError('model_missing')
    if not verified(path):
        raise DiagnosticError('model_checksum_mismatch')
    try:
        import torch
        model = torch.jit.load(str(Path(path, MODEL)), map_location='cpu').eval()
        with torch.inference_mode():
            value = float(model(torch.zeros(512), 16000).item())
        if not math.isfinite(value):
            raise ValueError('nonfinite inference')
        model.reset_states()
        return model
    except (ImportError, OSError, RuntimeError, ValueError) as error:
        raise DiagnosticError('runtime_incompatible') from error


def available() -> bool:
    """Whether this image can run the detector at all (Torch is installed). It says nothing about the model."""
    import importlib.util
    return importlib.util.find_spec('torch') is not None


def ready(environ=None) -> bool:
    """Local proof only; never downloads. Full inference proof is installation's job."""
    path = model_directory(environ)
    return Path(path, '.snitch-ready').exists() and verified(path)


def install(path: str, *, source: str | None = None, download: bool = False,
            report: Callable[[str], None] = lambda _: None):
    """Explicit acquisition, checksum BEFORE loading executable JIT, then real CPU proof.

    An interrupted/corrupt copy never gets a readiness marker. An operator can
    import the exact artifact from removable storage with ``source`` offline.
    """
    target = Path(path)
    target.mkdir(parents=True, exist_ok=True)
    marker = target / '.snitch-ready'
    marker.unlink(missing_ok=True)
    if not verified(path):
        if source is None and not download:
            raise DiagnosticError('model_missing')
        report('downloading')
        with tempfile.NamedTemporaryFile(dir=target, delete=False) as handle:
            temporary = Path(handle.name)
            try:
                with (open(source, 'rb') if source else urllib.request.urlopen(URL, timeout=60)) as incoming:
                    content = incoming.read(2272527)
                if len(content) != 2272526 or hashlib.sha256(content).hexdigest() != SHA256:
                    raise DiagnosticError('model_checksum_mismatch')
                handle.write(content)
                handle.flush()
                os.replace(temporary, target / MODEL)
            finally:
                temporary.unlink(missing_ok=True)
    report('verifying')
    load(path)
    (target / 'LICENSE.silero').write_text(LICENSE)
    marker.write_text(REVISION)
    report('ready')


def decode(source: str, target: str, index: int):
    if isinstance(index, bool) or not isinstance(index, int) or index < 0:
        raise DiagnosticError('invalid_track_index')
    try:
        cancellation.run_process(['ffmpeg', '-nostdin', '-v', 'error', '-y', '-protocol_whitelist',
            'file,pipe', '-i', source, '-map', f'0:{index}', '-vn', '-ar', '16000',
            '-c:a', 'pcm_s16le', target], timeout=7200, stdout=-3, stderr=-3)
    except (OSError, subprocess.SubprocessError) as error:
        # JobCancelled is a BaseException, deliberately not caught here.
        raise DiagnosticError('decode_failed') from error


def classify(scores: list[list[float]], duration: float, offset: float | None,
             *, degraded: bool = False) -> dict:
    result = {'status': 'ambiguous', 'cause': 'insufficient_evidence', 'intervals': []}
    if not scores or not all(scores) or duration <= 0 or any(not math.isfinite(s) or s < 0 or s > 1 for c in scores for s in c):
        return result
    if offset is None or not math.isfinite(offset) or offset < 0:
        result['cause'] = 'timeline_unknown'
        return result
    intervals = []
    uncertain = degraded
    for channel in scores:
        start = None
        for i, value in enumerate(channel + [0.]):
            uncertain |= .2 <= value < .5
            if value >= .5 and start is None:
                start = i
            if value < .5 and start is not None:
                if i - start >= 3:
                    intervals.append({'start': offset + start * .032,
                                      'end': offset + min(duration, i * .032)})
                else:
                    uncertain = True
                start = None
    for interval in sorted(intervals, key=lambda v: v['start']):
        if result['intervals'] and interval['start'] <= result['intervals'][-1]['end']:
            result['intervals'][-1]['end'] = max(interval['end'], result['intervals'][-1]['end'])
        else:
            result['intervals'].append(interval)
    if result['intervals']:
        result.update(status='speech', cause='neural_speech')
    elif not uncertain:
        result.update(status='no_speech', cause='complete_low_neural_evidence')
    return result


def diagnose(source: str, index: int, model_path: str | None = None, *,
             offset: float | None = 0, expected_duration: float | None = None,
             detector=None) -> dict:
    """Public adapter; injectable scorer is for deterministic orchestration tests.

    Duration/offset come from the selected stream, never the container. Unknown
    offsets and incomplete decoding cannot produce a reliable negative.
    """
    base = {'detector_version': VERSION, 'detector_config_version': CONFIG_VERSION}
    try:
        model = detector if detector is not None else load(model_path or model_directory())
        with tempfile.TemporaryDirectory(prefix='snitch-diagnostic-') as temporary:
            target = str(Path(temporary, 'decoded.wav'))
            decode(source, target, index)
            with wave.open(target, 'rb') as wav:
                channels, frames = wav.getnchannels(), wav.getnframes()
                duration = frames / 16000
                if wav.getsampwidth() != 2 or channels < 1 or channels > 32 or duration > 14400:
                    raise DiagnosticError('unsupported_audio')
                scores = []
                degraded = False
                for channel in range(channels):
                    cancellation.check()
                    model.reset_states()
                    wav.rewind()
                    evidence = []
                    peak = 0
                    clipped = 0
                    while raw := wav.readframes(512):
                        cancellation.check()
                        samples = array.array('h', raw)[channel::channels]
                        if sys.byteorder != 'little':
                            samples.byteswap()
                        peak = max(peak, max(map(abs, samples), default=0))
                        clipped += sum(abs(s) >= 32760 for s in samples)
                        if detector is None:
                            import torch
                            chunk = torch.tensor(samples, dtype=torch.float32) / 32768
                            if len(chunk) < 512:
                                chunk = torch.nn.functional.pad(chunk, (0, 512 - len(chunk)))
                            with torch.inference_mode():
                                value = float(model(chunk, 16000).item())
                        else:
                            value = float(model(list(samples) + [0] * (512 - len(samples)), 16000))
                        evidence.append(value)
                    scores.append(evidence)
                    degraded |= (0 < peak < 128) or clipped > max(1, frames * .01)
                degraded |= expected_duration is not None and abs(expected_duration - duration) > max(.1, duration * .01)
                return base | classify(scores, duration, offset, degraded=degraded)
    except DiagnosticError as error:
        return base | {'status': 'failed', 'cause': error.cause, 'intervals': []}
    except (OSError, ValueError, RuntimeError, wave.Error):
        return base | {'status': 'failed', 'cause': 'inference_failed', 'intervals': []}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['install'])
    parser.add_argument('--directory', default=model_directory())
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--source')
    group.add_argument('--download', action='store_true')
    args = parser.parse_args()
    install(args.directory, source=args.source, download=args.download, report=print)
