#!/usr/bin/env python3
"""Reserve the next Worker version (ADR 0036) as a Git tag `worker/vX.Y.Z` on the commit being published.

Source of truth: the Git tags of the repository, listed with pagination. Two families count, because both
end up as an image tag `vX.Y.Z` of snitch-worker: `worker/vX.Y.Z` (automatic, main) and `vX.Y.Z` (manual release).
The tag is created through the Git refs API, which refuses an existing ref: that refusal is the lock between
concurrent publications. A tag already on the commit is reused, so a re-run of the same commit gets the same version.
Prints the version on stdout. Any API error fails the run: a version is never guessed.
"""
import json
import os
import re
import sys
import urllib.error
import urllib.request

SEMVER = re.compile(r"^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$")
FIRST = (0, 1, 0)
ATTEMPTS = 10


def parse(name):
    m = SEMVER.match(name)
    return tuple(int(x) for x in m.groups()) if m else None


def fmt(v):
    return "v%d.%d.%d" % v


def next_version(tag_names):
    """Highest known version with PATCH+1, or v0.1.0 when there is none."""
    known = [v for v in (parse(n) for n in tag_names) if v]
    if not known:
        return FIRST
    major, minor, patch = max(known)
    return (major, minor, patch + 1)


class Github:
    def __init__(self, repo, token):
        self.base = "https://api.github.com/repos/" + repo
        self.token = token

    def call(self, method, url, body=None):
        req = urllib.request.Request(url, method=method, data=json.dumps(body).encode() if body else None)
        req.add_header("Authorization", "Bearer " + self.token)
        req.add_header("Accept", "application/vnd.github+json")
        req.add_header("X-GitHub-Api-Version", "2022-11-28")
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                return resp.status, json.load(resp), resp.headers.get("Link", "")
        except urllib.error.HTTPError as err:
            return err.code, None, ""

    def refs(self, prefix):
        """All tag refs under refs/tags/<prefix>, following pagination. Returns {tag name: commit sha}."""
        out, url = {}, "%s/git/matching-refs/tags/%s?per_page=100" % (self.base, prefix)
        while url:
            status, data, link = self.call("GET", url)
            if status != 200:
                raise SystemExit("listing tags %s failed: HTTP %s" % (prefix, status))
            for ref in data:
                out[ref["ref"][len("refs/tags/"):]] = ref["object"]["sha"]
            m = re.search(r'<([^>]+)>;\s*rel="next"', link)
            url = m.group(1) if m else None
        return out

    def create(self, tag, sha):
        status, _, _ = self.call("POST", self.base + "/git/refs", {"ref": "refs/tags/" + tag, "sha": sha})
        if status == 201:
            return True
        if status == 422:  # already exists: someone else took this version
            return False
        raise SystemExit("creating tag %s failed: HTTP %s" % (tag, status))


def reserve(gh, sha):
    for _ in range(ATTEMPTS):
        auto = gh.refs("worker/v")
        # Idempotence: a re-run of the same commit keeps its version.
        for name, tag_sha in auto.items():
            if tag_sha == sha and parse(name[len("worker/"):]):
                return name[len("worker/"):]
        manual = gh.refs("v")
        names = [n[len("worker/"):] for n in auto] + list(manual)
        version = fmt(next_version(names))
        if gh.create("worker/" + version, sha):
            return version
    raise SystemExit("no free Worker version after %d attempts" % ATTEMPTS)


def guard_release(gh, tag, sha):
    """A manual release tag must not reuse a version an automatic publication already took for another commit."""
    if not parse(tag):
        raise SystemExit("release tag %s is not vX.Y.Z" % tag)
    taken = gh.refs("worker/" + tag).get("worker/" + tag)
    if taken and taken != sha:
        raise SystemExit("version %s already published by the automatic Worker release on %s" % (tag, taken[:7]))


def main():
    gh = Github(os.environ["GITHUB_REPOSITORY"], os.environ["GH_TOKEN"])
    sha = os.environ["GITHUB_SHA"]
    if os.environ.get("REF_TYPE") == "tag":
        guard_release(gh, os.environ["REF_NAME"], sha)
        print(os.environ["REF_NAME"])
    else:
        print(reserve(gh, sha))


if __name__ == "__main__":
    sys.exit(main())
