import io
import json
import logging
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


if __name__ == "__main__":
    unittest.main()
