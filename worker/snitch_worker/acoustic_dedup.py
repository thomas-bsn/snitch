"""Acoustic duplicate evidence adapter (issue #343, ADR 0053).

Pure and contractual: it reads decoded PCM16 WAV artifacts and returns evidence. It knows
nothing about ASR text, Personnes, Locuteurs locaux, Corrections or the orchestrator, and it
never installs or downloads anything. Text equality and timestamp proximity are only used to
shortlist candidates; a `duplicate` decision always rests on samples.

Method (strategy ladder, details and limits in docs/acoustic-dedup.md):

* ``exact_pcm``  identical decoded samples over a run of blocks. Enabled.
* ``gain``       one positive gain, unclipped, residual <= 1 LSB per block. Enabled: it passed
                 every positive and rejected every hard negative of the fixture matrix.
* ``codec`` / mixtures are only *evaluated* (correlation metrics) and always end in
  ``uncertain_keep``: no codec family or clean-vs-mix rule has been validated.

Anything not established (unknown timeline, other sample rate, short, silent, budget) is
``uncertain_keep``, never a successful deduplication.
"""

from __future__ import annotations

import array
import itertools
import sys
import time
import wave
from dataclasses import dataclass, field
from typing import Any, Callable

SCHEMA_VERSION = 1
ADAPTER_VERSION = "1"
# Strategies allowed to produce `duplicate`, with their rule version. The server derives
# the plan's merge version from this table: changing a rule changes the input version.
ENABLED_STRATEGIES: dict[str, str] = {"exact_pcm": "1", "gain": "1"}
EVALUATED_ONLY = ("codec", "mixture")


@dataclass(frozen=True)
class DedupConfig:
    block_ms: int = 10
    min_run_seconds: float = 0.25
    min_peak: int = 32  # a block below this carries no evidence either way
    max_lag_seconds: float = 0.1
    gain_tolerance: float = 1.0  # LSB, per-block peak residual
    min_gain_distance: float = 0.01
    distinct_correlation: float = 0.3
    max_compare_seconds: float = 120.0
    max_pairs: int = 200
    anchors: int = 5

    @classmethod
    def from_dict(cls, raw: dict[str, Any] | None) -> "DedupConfig":
        raw = raw or {}
        known = {k: raw[k] for k in cls.__dataclass_fields__ if k in raw}
        unknown = set(raw) - set(known)
        if unknown:
            raise ValueError(f"unknown dedup config keys: {sorted(unknown)}")
        return cls(**known)


class Cancelled(Exception):
    pass


@dataclass
class Stats:
    pairs: int = 0
    skipped_pairs: int = 0
    compared_samples: int = 0
    decode_ms: float = 0.0
    elapsed_ms: float = 0.0
    peak_window_samples: int = 0
    candidates: dict[str, int] = field(default_factory=dict)


def _r(value: float) -> float:
    return round(float(value), 6)


def _event_key(event: dict[str, Any]) -> tuple:
    return (event["track_index"], event.get("job_id") or 0, event["start"], event["end"], event["id"])


def _public(event: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": event["id"],
        "track_index": event["track_index"],
        "job_id": event.get("job_id"),
        "start": _r(event["start"]),
        "end": _r(event["end"]),
    }


def _result(left, right, decision, kind, reasons, **extra) -> dict[str, Any]:
    strategy = kind if kind in ENABLED_STRATEGIES else None
    out = {
        "left": _public(left),
        "right": _public(right),
        "decision": decision,
        "evidence_kind": kind,
        "strategy": strategy,
        "strategy_version": ENABLED_STRATEGIES.get(kind),
        "reason_codes": sorted(set(reasons)),
        "lag_seconds": None,
        "gain": None,
        "matched": [],
        "coverage": {"left": 0.0, "right": 0.0},
        "metrics": {},
    }
    out.update(extra)
    return out


def _keep(left, right, *reasons, decision="uncertain_keep", kind="none", **extra):
    return _result(left, right, decision, kind, list(reasons), **extra)


def _load(event: dict[str, Any], first: int, last: int, stats: Stats):
    """Samples [first, last) of the event's file as array('h'), zero-padded outside the file."""
    started = time.perf_counter()
    with wave.open(event["path"], "rb") as handle:
        channels, width, rate, frames = handle.getnchannels(), handle.getsampwidth(), handle.getframerate(), handle.getnframes()
        if channels != 1 or width != 2:
            raise ValueError("unsupported_format")
        lo, hi = max(0, first), min(frames, last)
        handle.setpos(lo)
        data = array.array("h")
        data.frombytes(handle.readframes(max(0, hi - lo)))
        if sys.byteorder == "big":  # WAV is little-endian
            data.byteswap()
    out = array.array("h", bytes(2 * max(0, lo - first))) + data
    out.extend(array.array("h", bytes(2 * max(0, last - first - len(out)))))
    stats.decode_ms += (time.perf_counter() - started) * 1000
    stats.compared_samples += last - first
    stats.peak_window_samples = max(stats.peak_window_samples, last - first)
    return out


def _wav_rate(path: str) -> tuple[int, int, int, int]:
    with wave.open(path, "rb") as handle:
        return handle.getframerate(), handle.getnchannels(), handle.getsampwidth(), handle.getnframes()


def _runs(status: list[int], block: int, rate: int, config: DedupConfig) -> list[tuple[int, int]]:
    """Maximal [start, end) sample runs without a failing block and with enough evidence.

    status: 0 = differs, 1 = equal but uninformative (silence), 2 = equal and informative.
    Uninformative (equal, near-silent) blocks never count as evidence, but a run keeps the
    equal silence adjacent to proven content: those samples are identical too.
    """
    runs, i, n = [], 0, len(status)
    need = config.min_run_seconds * rate / block
    while i < n:
        if status[i] == 0:
            i += 1
            continue
        j = i
        while j < n and status[j] != 0:
            j += 1
        informative = [k for k in range(i, j) if status[k] == 2]
        if informative and len(informative) >= need:
            runs.append((i * block, j * block))
        i = j
    return runs


def _block_status_exact(left, right, block: int, config: DedupConfig) -> list[int]:
    status = []
    for start in range(0, len(left) - block + 1, block):
        a, b = left[start : start + block], right[start : start + block]
        if a != b:
            status.append(0)
        elif max(map(abs, a)) >= config.min_peak and len(set(a)) > 8:
            status.append(2)
        else:
            status.append(1)
    return status


def _numpy():
    try:
        import numpy

        return numpy
    except ImportError:  # the base worker has no numpy: transformed rules then keep
        return None


def _best_lag(np, x, y_padded, pad: int) -> tuple[int, float]:
    """Lag (samples, y relative to x) in [-pad, pad] maximizing the normalized correlation."""
    n = len(x)
    xf = x.astype("float64")
    yf = y_padded.astype("float64")
    size = 1 << (n + len(yf)).bit_length()
    corr = np.fft.irfft(np.fft.rfft(yf, size) * np.conj(np.fft.rfft(xf, size)), size)
    # corr[k] = sum_t y[t + k] x[t]; window k=0 is y_padded offset 0 => lag = k - pad
    values = corr[: 2 * pad + 1]
    k = int(np.argmax(values))
    energy_x = float(xf @ xf)
    seg = yf[k : k + n]
    energy_y = float(seg @ seg)
    norm = (float(values[k]) / (energy_x * energy_y) ** 0.5) if energy_x > 0 and energy_y > 0 else 0.0
    return k - pad, norm


def _compare_pair(left, right, config: DedupConfig, stats: Stats, cancel: Callable[[], None]) -> dict[str, Any]:
    if left["track_index"] == right["track_index"]:
        return _keep(left, right, "same_track")
    if left.get("clip_offset") is None or right.get("clip_offset") is None:
        return _keep(left, right, "unknown_timeline")
    lo = max(left["start"], right["start"])
    hi = min(left["end"], right["end"])
    if hi - lo < config.min_run_seconds:
        # Same words said later, or events too short to compare: nothing to suppress.
        return _keep(left, right, "timeline_disjoint", decision="distinct")
    if hi - lo > config.max_compare_seconds:
        return _keep(left, right, "budget_exceeded")
    try:
        (lrate, lch, lw, _), (rrate, rch, rw, _) = _wav_rate(left["path"]), _wav_rate(right["path"])
    except (wave.Error, EOFError, OSError):
        return _keep(left, right, "undecodable")
    if lch != 1 or rch != 1 or lw != 2 or rw != 2:
        return _keep(left, right, "unsupported_format")
    if lrate != rrate:
        return _keep(left, right, "sample_rate_not_validated", kind="none", metrics={"left_rate": lrate, "right_rate": rrate})
    rate = lrate
    pad = int(round(config.max_lag_seconds * rate))
    block = max(1, rate * config.block_ms // 1000)
    l0 = int(round((lo - left["clip_offset"]) * rate))
    r0 = int(round((lo - right["clip_offset"]) * rate))
    count = int(round((hi - lo) * rate))
    cancel()
    try:
        x = _load(left, l0, l0 + count, stats)
        y_pad = _load(right, r0 - pad, r0 + count + pad, stats)
    except (wave.Error, EOFError, OSError, ValueError):
        return _keep(left, right, "undecodable")
    cancel()

    best: tuple[int, int, list[tuple[int, int]]] | None = None  # total, -|lag|, runs
    best_lag = 0
    # Exact path: stdlib only. Anchor windows are located in the other file by byte search.
    anchor = 512
    energies = sorted(
        ((sum(abs(v) for v in x[s : s + anchor]), s) for s in range(0, max(1, count - anchor + 1), anchor // 2)),
        reverse=True,
    )
    y_bytes = y_pad.tobytes()
    tried: set[int] = set()
    for _, s in energies[: config.anchors]:
        window = x[s : s + anchor]
        if len(window) < anchor or max(map(abs, window)) < config.min_peak:
            continue
        needle = window.tobytes()
        at = y_bytes.find(needle)
        hops = 0
        while at != -1 and hops < 8:
            if at % 2 == 0:
                lag = (at // 2) - s - pad
                if lag not in tried and abs(lag) <= pad:
                    tried.add(lag)
                    start = max(0, -lag)
                    end = min(count, count - lag)
                    ya = y_pad[pad + lag + start : pad + lag + end]
                    xa = x[start : start + len(ya)]
                    runs = _runs(_block_status_exact(xa, ya, block, config), block, rate, config)
                    runs = [(a + start, b + start) for a, b in runs]
                    total = sum(b - a for a, b in runs)
                    key = (total, -abs(lag))
                    if runs and (best is None or key > (best[0], best[1])):
                        best, best_lag = (total, -abs(lag), runs), lag
            at = y_bytes.find(needle, at + 1)
            hops += 1
    kind, gain, reasons = "exact_pcm", None, []
    metrics: dict[str, Any] = {}
    runs = best[2] if best else []

    np = _numpy()
    corr_value = None
    if not runs and np is not None:
        xs = np.frombuffer(x.tobytes(), dtype="<i2").astype("int64")
        ys = np.frombuffer(y_pad.tobytes(), dtype="<i2").astype("int64")
        if int(np.abs(xs).max(initial=0)) < config.min_peak:
            return _keep(left, right, "no_signal")
        lag, corr_value = _best_lag(np, xs, ys, pad)
        metrics = {"normalized_correlation": _r(corr_value), "correlation_lag_samples": lag}
        start = max(0, -lag)
        end = min(count, count - lag)
        ya = ys[pad + lag + start : pad + lag + end]
        xa = xs[start : start + len(ya)]
        n_blocks = len(ya) // block
        if n_blocks:
            xb = xa[: n_blocks * block].reshape(n_blocks, block)
            yb = ya[: n_blocks * block].reshape(n_blocks, block)
            sxx, sxy = (xb * xb).sum(axis=1), (xb * yb).sum(axis=1)
            clipped = (np.abs(xb).max(axis=1) >= 32767) | (np.abs(yb).max(axis=1) >= 32767)
            usable = (sxx > 0) & ~clipped
            ratio = np.where(usable, sxy / np.where(sxx > 0, sxx, 1), 0.0)
            informative_pre = usable & (np.minimum(np.abs(xb).max(axis=1), np.abs(yb).max(axis=1)) >= config.min_peak)
            if informative_pre.any():
                g = float(np.median(ratio[informative_pre]))
                metrics["gain_estimate"] = _r(g)
                if g <= 0:
                    reasons.append("polarity_or_gain_sign_not_validated")
                elif abs(g - 1.0) < config.min_gain_distance:
                    reasons.append("near_unity_gain_not_validated")
                else:
                    residual = np.abs(yb - g * xb).max(axis=1)
                    ok = (residual <= config.gain_tolerance) & ~clipped
                    informative = np.minimum(np.abs(xb).max(axis=1) * min(g, 1.0), np.abs(yb).max(axis=1)) >= config.min_peak
                    status = np.where(~ok, 0, np.where(informative, 2, 1)).tolist()
                    g_runs = _runs(status, block, rate, config)
                    metrics["gain_median_residual_peak"] = _r(float(np.median(residual[informative_pre])))
                    if g_runs:
                        runs = [(a + start, b + start) for a, b in g_runs]
                        kind, gain, best_lag = "gain", _r(g), lag
            if not runs:
                # Informational only: residual of the best unit-gain alignment (codec/mix signature).
                res = ya[: n_blocks * block] - xa[: n_blocks * block]
                xe = float((xa[: n_blocks * block] ** 2).sum())
                metrics["residual_to_signal_ratio"] = _r(float((res**2).sum()) / xe) if xe else None

    if not runs:
        if np is None:
            return _keep(left, right, "numpy_unavailable_transforms_not_evaluated", metrics=metrics)
        if corr_value is not None and corr_value >= 0.9:
            return _keep(left, right, "codec_or_transform_candidate_not_validated", *reasons, kind="codec", metrics=metrics)
        if corr_value is not None and corr_value >= config.distinct_correlation:
            return _keep(left, right, "partial_correlation_not_proof", *reasons, kind="mixture", metrics=metrics)
        return _keep(left, right, "no_acoustic_match", *reasons, decision="distinct", metrics=metrics)

    matched = []
    covered = 0
    for a, b in runs:
        covered += b - a
        matched.append(
            {
                "left_start": _r(left["clip_offset"] + (l0 + a) / rate),
                "left_end": _r(left["clip_offset"] + (l0 + b) / rate),
                "right_start": _r(right["clip_offset"] + (r0 + a + best_lag) / rate),
                "right_end": _r(right["clip_offset"] + (r0 + b + best_lag) / rate),
            }
        )
    seconds = covered / rate
    reasons = [f"{kind}_samples_proven"]
    if seconds + 2 * block / rate < min(left["end"] - left["start"], right["end"] - right["start"]):
        reasons.append("partial_coverage")
    metrics["matched_seconds"] = _r(seconds)
    return _result(
        left,
        right,
        "duplicate",
        kind,
        reasons,
        lag_seconds=_r(best_lag / rate),
        gain=gain,
        matched=matched,
        coverage={
            "left": _r(min(1.0, seconds / max(1e-9, left["end"] - left["start"]))),
            "right": _r(min(1.0, seconds / max(1e-9, right["end"] - right["start"]))),
        },
        metrics=metrics,
    )


def compare(left: dict[str, Any], right: dict[str, Any], config: DedupConfig | None = None, *, cancel: Callable[[], None] | None = None, stats: Stats | None = None) -> dict[str, Any]:
    """Compare two events. The result does not depend on argument order."""
    config = config or DedupConfig()
    first, second = sorted((left, right), key=_event_key)
    return _compare_pair(first, second, config, stats or Stats(), cancel or (lambda: None))


def run(payload: dict[str, Any], *, cancel: Callable[[], None] | None = None) -> dict[str, Any]:
    """Worker adapter entry point (JSON in, JSON out).

    payload: {"schema_version": 1, "config": {...}, "events": [{id, track_index, job_id,
    path, clip_offset, start, end}], "pairs": optional [[id, id]...]}.
    Without pairs, every cross-track pair whose clip ranges overlap is compared.
    Output order and content are canonical: they never depend on the order of events.
    """
    started = time.perf_counter()
    if payload.get("schema_version") != SCHEMA_VERSION:
        raise ValueError("unsupported schema_version")
    config = DedupConfig.from_dict(payload.get("config"))
    events = sorted(payload.get("events", []), key=_event_key)
    ids = [event["id"] for event in events]
    if len(set(ids)) != len(ids):
        raise ValueError("duplicate event id")
    for event in events:
        if not (event["end"] > event["start"] >= 0):
            raise ValueError(f"invalid event range: {event['id']}")
    by_id = {event["id"]: event for event in events}
    if payload.get("pairs") is not None:
        pair_ids = {tuple(sorted((a, b))) for a, b in payload["pairs"]}
        pairs = sorted((tuple(sorted((by_id[a], by_id[b]), key=_event_key)) for a, b in pair_ids), key=lambda p: (_event_key(p[0]), _event_key(p[1])))
    else:
        pairs = [
            (a, b)
            for a, b in itertools.combinations(events, 2)
            if a["track_index"] != b["track_index"] and min(a["end"], b["end"]) > max(a["start"], b["start"])
        ]
    stats = Stats()
    check = cancel or (lambda: None)
    comparisons = []
    for index, (a, b) in enumerate(pairs):
        check()
        if index >= config.max_pairs:
            stats.skipped_pairs += 1
            comparisons.append(_keep(a, b, "pair_budget_exceeded"))
            continue
        stats.pairs += 1
        comparisons.append(_compare_pair(a, b, config, stats, check))
    stats.elapsed_ms = (time.perf_counter() - started) * 1000
    for item in comparisons:
        stats.candidates[item["decision"]] = stats.candidates.get(item["decision"], 0) + 1
    return {
        "schema_version": SCHEMA_VERSION,
        "adapter_version": ADAPTER_VERSION,
        "enabled_strategies": dict(sorted(ENABLED_STRATEGIES.items())),
        "evaluated_only": list(EVALUATED_ONLY),
        "comparisons": comparisons,
        "stats": {
            "pairs": stats.pairs,
            "skipped_pairs": stats.skipped_pairs,
            "compared_samples": stats.compared_samples,
            "decode_ms": _r(stats.decode_ms),
            "elapsed_ms": _r(stats.elapsed_ms),
            "peak_window_samples": stats.peak_window_samples,
            "decisions": dict(sorted(stats.candidates.items())),
        },
    }
