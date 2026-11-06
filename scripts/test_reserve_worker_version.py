import unittest
from reserve_worker_version import Github, guard_release, next_version, reserve, fmt


class Fake(Github):
    """In-memory Git tags; `race` lets a rival take the proposed version once before our create."""

    def __init__(self, tags, race=0):
        self.tags, self.race = dict(tags), race

    def refs(self, prefix):
        return {k: v for k, v in self.tags.items() if k.startswith(prefix)}

    def create(self, tag, sha):
        if self.race:
            self.race -= 1
            self.tags[tag] = "rival"
            return False
        if tag in self.tags:
            return False
        self.tags[tag] = sha
        return True


class Reserve(unittest.TestCase):
    def test_first(self):
        self.assertEqual(reserve(Fake({}), "a"), "v0.1.0")

    def test_patch_increment_ignores_noise(self):
        self.assertEqual(fmt(next_version(["v0.1.0", "v0.1.9", "v0.1.10", "latest", "v1.0.0-rc1", "sha-abc"])), "v0.1.11")

    def test_manual_release_counts(self):
        self.assertEqual(reserve(Fake({"worker/v0.1.1": "x", "v0.2.0": "y"}), "a"), "v0.2.1")

    def test_rerun_same_commit_same_version(self):
        gh = Fake({"worker/v0.1.0": "a", "worker/v0.1.1": "b"})
        self.assertEqual(reserve(gh, "a"), "v0.1.0")
        self.assertEqual(len(gh.tags), 2)

    def test_concurrent_publications_get_distinct_versions(self):
        gh = Fake({}, race=1)
        self.assertEqual(reserve(gh, "a"), "v0.1.1")
        self.assertEqual(gh.tags["worker/v0.1.0"], "rival")

    def test_never_overwrites(self):
        gh = Fake({"worker/v0.1.0": "x"})
        reserve(gh, "a")
        self.assertEqual(gh.tags["worker/v0.1.0"], "x")

    def test_release_guard(self):
        gh = Fake({"worker/v0.1.0": "x"})
        with self.assertRaises(SystemExit):
            guard_release(gh, "v0.1.0", "other")
        guard_release(gh, "v0.1.0", "x")
        guard_release(gh, "v0.3.0", "other")
        with self.assertRaises(SystemExit):
            guard_release(gh, "v0.1.0-rc1", "x")


if __name__ == "__main__":
    unittest.main()
