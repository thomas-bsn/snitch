import io
import json
import logging
import time
import unittest

from snitch_worker import log


def read(stream: io.StringIO) -> list[dict]:
    return [json.loads(line) for line in stream.getvalue().splitlines() if line.strip()]


class StructuredLogTests(unittest.TestCase):
    def tearDown(self):
        log.configure("INFO", stream=io.StringIO())

    def test_shared_schema_with_service_event_and_context(self):
        stream = io.StringIO()
        log.configure("INFO", stream=stream)
        log.emit(logging.INFO, "job_claimed", job_id=7, clip_id=3)
        [line] = read(stream)
        self.assertEqual(line["service"], "worker")
        self.assertEqual(line["event"], "job_claimed")
        self.assertEqual(line["level"], "INFO")
        self.assertEqual((line["job_id"], line["clip_id"]), (7, 3))
        self.assertIn("timestamp", line)

    def test_log_level_controls_volume(self):
        expected = {"ERROR": 1, "WARN": 2, "INFO": 3, "DEBUG": 4, "": 3, "nonsense": 3, "debug": 4}
        for level, count in expected.items():
            stream = io.StringIO()
            log.configure(level, stream=stream)
            for severity in (logging.DEBUG, logging.INFO, logging.WARNING, logging.ERROR):
                log.emit(severity, "probe")
            self.assertEqual(len(read(stream)), count, level)

    def test_warning_is_written_as_warn(self):
        stream = io.StringIO()
        log.configure("WARN", stream=stream)
        log.emit(logging.WARNING, "x")
        self.assertEqual(read(stream)[0]["level"], "WARN")

    def test_forbidden_keys_and_raw_exceptions_never_appear_at_any_level(self):
        stream = io.StringIO()
        log.configure("DEBUG", stream=stream)
        try:
            raise RuntimeError("provider said token=hf_secret at /tmp/private/audio.wav")
        except RuntimeError as error:
            log.emit(
                logging.DEBUG, "probe", error=error, Authorization="Bearer abc", lease_token="lt_secret", hf_token="hf_x",
                cookie="c=1", transcript="private words", embedding=[0.1], nested={"token": "nested-secret"}, job_id=4,
            )
            log.get_logger().error("raw", exc_info=True)
        output = stream.getvalue()
        for secret in ("Bearer abc", "lt_secret", "hf_x", "c=1", "private words", "nested-secret", "hf_secret", "/tmp/private"):
            self.assertNotIn(secret, output)
        self.assertEqual(read(stream)[0]["job_id"], 4)

    def test_raw_error_text_under_error_keys_is_replaced_by_code_and_category(self):
        for level in ("DEBUG", "INFO", "WARN", "ERROR"):
            stream = io.StringIO()
            log.configure(level, stream=stream)
            sql = 'duplicate key value violates unique constraint "users_email_key" Key (email)=(ann@example.test)'
            provider = "provider said invalid key sk-synthetic-123 for https://api.example.test/v1?sig=abc"
            log.emit(logging.ERROR, "trap", error=sql, err=provider, reason=provider, api_key="sk-synthetic-123", job_id=9)
            log.emit(logging.ERROR, "trap_failure", **log.failure(RuntimeError(provider)))
            log.emit(logging.ERROR, "trap_failure", **log.failure(TimeoutError(provider)))
            output = stream.getvalue()
            for leaked in ("users_email_key", "ann@example.test", "sk-synthetic", "sig=abc", "provider said"):
                self.assertNotIn(leaked, output)
            lines = read(stream)
            self.assertEqual(lines[0]["job_id"], 9)
            self.assertEqual((lines[1]["error_code"], lines[1]["error_category"]), ("operation_error", "internal"))
            self.assertEqual((lines[2]["error_code"], lines[2]["error_category"]), ("timeout", "timeout"))

    def test_non_string_values_under_error_keys_are_replaced_by_code_and_category(self):
        stream = io.StringIO()
        log.configure("DEBUG", stream=stream)
        log.emit(logging.ERROR, "trap", error=RuntimeError("provider said sk-synthetic-123"), detail={"body": "ann@example.test"},
                 reason=["sig=abc"], exception=TimeoutError("slow"))
        output = stream.getvalue()
        for leaked in ("sk-synthetic", "ann@example.test", "sig=abc", "provider said", "RuntimeError"):
            self.assertNotIn(leaked, output)
        line = read(stream)[0]
        for key in ("error", "detail", "reason"):
            self.assertEqual(line[key], {"code": "operation_error", "category": "internal"})
        self.assertEqual(line["exception"], {"code": "timeout", "category": "timeout"})


class ScrubTests(unittest.TestCase):
    def test_tokens_bearer_urls_with_credentials_and_clip_paths_are_masked_in_values(self):
        stream = io.StringIO()
        log.configure("DEBUG", stream=stream)
        log.emit(
            logging.INFO, "probe",
            message="Authorization: Bearer abc.def; hf_SyntheticToken123 and rw_synthetic-token",
            url="https://ann:pw-synth@api.example.test/v1/audio?sig=abc&X-Amz-Signature=def",
            path="/var/folders/xy/snitch-voice-1/original.media",
            clip="/Users/ann/Movies/clip-private-42.mp4",
            wav="prepared.wav",
            lease="lt_synthetic-lease-token",
        )
        output = stream.getvalue()
        for leaked in ("abc.def", "hf_SyntheticToken123", "rw_synthetic-token", "ann:pw-synth", "sig=abc",
                       "X-Amz-Signature", "/var/folders", "original.media", "/Users/ann", "clip-private-42",
                       "prepared.wav", "lt_synthetic-lease-token"):
            self.assertNotIn(leaked, output)
        [line] = read(stream)
        self.assertEqual(line["url"], "https://api.example.test/[masked]")

    def test_masking_table_matches_the_server(self):
        # The same table as app/internal/jobs/logscrub_test.go: the Worker and the server mask alike.
        cases = [
            ("Authorization: Bearer abc.def", "Authorization: Bearer [masked]"),
            ("token hf_SyntheticToken123 ok", "token [masked] ok"),
            ("lease rw_synthetic-token", "lease [masked]"),
            ("https://ann:pw-synth@hub.example/path?sig=abc", "https://hub.example/[masked]"),
            ("https://hub.example", "https://hub.example"),
            ("clip /private/var/synthetic/audio.wav done", "clip [masked_path] done"),
            ("synthetic-recording.mp4 uploaded", "[masked_path] uploaded"),
            ("hf_not_found job 42 claimed for step voice.fingerprint", "hf_not_found job 42 claimed for step voice.fingerprint"),
        ]
        for text, want in cases:
            self.assertEqual(log.scrub(text), want)

    def test_plain_text_and_stable_codes_survive(self):
        self.assertEqual(log.scrub("job 42 claimed for step voice.fingerprint"), "job 42 claimed for step voice.fingerprint")
        self.assertEqual(log.scrub("hf_not_found"), "hf_not_found")


class _Sink:
    def __init__(self, fail=False):
        self.batches = []
        self.fail = fail

    def __call__(self, lines):
        self.batches.append(list(lines))
        if self.fail:
            raise OSError("synthetic network failure")


class LineShipperTests(unittest.TestCase):
    def setUp(self):
        self.addCleanup(log.configure, "INFO", io.StringIO())

    def shipper(self, sink, job_id=7, interval=30.0):
        shipper = log.LineShipper(job_id, sink, interval=interval)
        self.addCleanup(shipper.close)
        log.get_logger().addHandler(shipper)
        self.addCleanup(log.get_logger().removeHandler, shipper)
        return shipper

    def test_sends_batches_of_fifty_lines_as_soon_as_they_are_full(self):
        log.configure("INFO", stream=io.StringIO())
        sink = _Sink()
        self.shipper(sink)
        for index in range(120):
            log.emit(logging.INFO, "step", job_id=7, index=index)
        deadline = time.monotonic() + 3
        while sum(map(len, sink.batches)) < 100 and time.monotonic() < deadline:
            time.sleep(0.01)
        self.assertEqual([len(batch) for batch in sink.batches][:2], [50, 50])

    def test_flushes_pending_lines_every_interval(self):
        log.configure("INFO", stream=io.StringIO())
        sink = _Sink()
        self.shipper(sink, interval=0.05)
        log.emit(logging.INFO, "small", job_id=7)
        deadline = time.monotonic() + 3
        while not sink.batches and time.monotonic() < deadline:
            time.sleep(0.01)
        self.assertEqual(len(sink.batches), 1)
        self.assertEqual(json.loads(sink.batches[0][0])["event"], "small")

    def test_ships_only_lines_of_its_own_job_and_masks_them(self):
        log.configure("DEBUG", stream=io.StringIO())
        sink = _Sink()
        shipper = self.shipper(sink)
        log.emit(logging.INFO, "mine", job_id=7, url="https://u:p@host.example.test/x?sig=abc")
        log.emit(logging.INFO, "other", job_id=8)
        log.emit(logging.INFO, "unscoped")
        shipper.close()
        lines = [json.loads(line) for batch in sink.batches for line in batch]
        self.assertEqual([line["event"] for line in lines], ["mine"])
        self.assertNotIn("sig=abc", json.dumps(lines))
        self.assertNotIn("u:p@", json.dumps(lines))

    def test_close_sends_what_remains_and_a_failed_send_never_raises(self):
        log.configure("INFO", stream=io.StringIO())
        sink = _Sink(fail=True)
        shipper = self.shipper(sink, interval=30.0)
        log.emit(logging.INFO, "last", job_id=7)
        shipper.close()
        self.assertEqual(len(sink.batches), 1)


if __name__ == "__main__":
    unittest.main()
