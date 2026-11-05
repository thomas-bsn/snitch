import unittest

import ci_metrics


def job(name, start, end, conclusion="success", created=None):
    return {"name": name, "conclusion": conclusion, "created_at": created or start, "started_at": start, "completed_at": end}


class SummarizeTest(unittest.TestCase):
    def test_wall_clock_is_the_span_not_the_sum(self):
        rows, totals = ci_metrics.summarize([
            job("a", "2026-01-01T10:00:00Z", "2026-01-01T10:03:00Z"),
            job("b", "2026-01-01T10:00:00Z", "2026-01-01T10:04:30Z", created="2026-01-01T09:59:50Z"),
        ])
        self.assertEqual(totals["wall"], 270)
        self.assertEqual(totals["compute"], 450)
        self.assertEqual(totals["billed"], 3 + 5)  # each job rounded up to the minute
        self.assertEqual([r[0] for r in rows], ["b", "a"])  # slowest first
        self.assertEqual(rows[0][3], 10)  # queued time

    def test_skipped_and_unfinished_jobs_do_not_count(self):
        rows, totals = ci_metrics.summarize([
            job("skipped", "2026-01-01T10:00:00Z", "2026-01-01T10:00:00Z", conclusion="skipped"),
            {"name": "running", "status": "in_progress", "started_at": "2026-01-01T10:00:00Z", "completed_at": None},
        ])
        self.assertEqual(rows, [])
        self.assertEqual(totals, {"wall": 0, "compute": 0, "billed": 0})


if __name__ == "__main__":
    unittest.main()
