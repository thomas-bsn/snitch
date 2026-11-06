#!/usr/bin/env python3
"""CI performance of the "Verify and distribute" workflow, as Markdown.

  ci_metrics.py [RUN_ID]            jobs of one run (default: $GITHUB_RUN_ID), written to the job summary by CI
  ci_metrics.py --history N         the last N completed runs of the workflow (wall clock, compute minutes)

Reads the GitHub API with $GH_TOKEN (or `gh auth token` when run by hand). Measures only: it fails nothing.
"""
import argparse
import json
import os
import subprocess
import sys
import urllib.request
from datetime import datetime, timezone


def parse_time(value):
    return datetime.fromisoformat(value.replace("Z", "+00:00")) if value else None


def seconds(start, end):
    return max(0, int((end - start).total_seconds())) if start and end else 0


def summarize(jobs):
    """Rows (name, conclusion, seconds, queued seconds) and the totals of a run's jobs."""
    rows, starts, ends, compute, billed = [], [], [], 0, 0
    for job in jobs:
        created, started, completed = (parse_time(job.get(k)) for k in ("created_at", "started_at", "completed_at"))
        if job.get("conclusion") == "skipped" or not started or not completed:
            continue
        duration = seconds(started, completed)
        rows.append((job["name"], job.get("conclusion") or job.get("status"), duration, seconds(created, started)))
        starts.append(started)
        ends.append(completed)
        compute += duration
        billed += -(-duration // 60)  # GitHub rounds every job up to the minute
    wall = seconds(min(starts), max(ends)) if starts else 0
    return sorted(rows, key=lambda r: -r[2]), {"wall": wall, "compute": compute, "billed": billed}


def minutes(value):
    return "%dm%02ds" % divmod(value, 60)


def render(rows, totals):
    out = ["| Job | Result | Duration | Queued |", "|---|---|---:|---:|"]
    out += ["| %s | %s | %s | %s |" % (name, result, minutes(d), minutes(q)) for name, result, d, q in rows]
    out.append("")
    out.append("**Wall clock** %s · **compute** %s · **billed** %d min (skipped jobs excluded)"
               % (minutes(totals["wall"]), minutes(totals["compute"]), totals["billed"]))
    return "\n".join(out)


class Api:
    def __init__(self, repo, token):
        self.base, self.token = "https://api.github.com/repos/" + repo, token

    def get(self, path):
        req = urllib.request.Request(self.base + path)
        req.add_header("Authorization", "Bearer " + self.token)
        req.add_header("Accept", "application/vnd.github+json")
        with urllib.request.urlopen(req, timeout=30) as resp:
            return json.load(resp)

    def jobs(self, run_id):
        return self.get("/actions/runs/%s/jobs?per_page=100&filter=latest" % run_id)["jobs"]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("run_id", nargs="?", default=os.environ.get("GITHUB_RUN_ID"))
    parser.add_argument("--history", type=int)
    parser.add_argument("--repo", default=os.environ.get("GITHUB_REPOSITORY", "thomas-bsn/snitch"))
    parser.add_argument("--workflow", default="verify.yml")
    parser.add_argument("--branch", default="main")
    args = parser.parse_args()
    token = os.environ.get("GH_TOKEN") or subprocess.run(["gh", "auth", "token"], capture_output=True, text=True, check=True).stdout.strip()
    api = Api(args.repo, token)
    if args.history:
        runs = api.get("/actions/workflows/%s/runs?branch=%s&status=success&per_page=%d" % (args.workflow, args.branch, args.history))["workflow_runs"]
        print("| Run | Commit | Event | Wall clock | Compute | Billed |\n|---|---|---|---:|---:|---:|")
        for run in runs:
            _, totals = summarize(api.jobs(run["id"]))
            print("| %s | %s | %s | %s | %s | %d min |" % (run["id"], run["head_sha"][:7], run["event"], minutes(totals["wall"]), minutes(totals["compute"]), totals["billed"]))
        return 0
    if not args.run_id:
        sys.exit("no run id: pass one or set GITHUB_RUN_ID")
    rows, totals = summarize(api.jobs(args.run_id))
    print(render(rows, totals))
    return 0


if __name__ == "__main__":
    sys.exit(main())
