"""Isolated real deployment acceptance test. No host media, no model downloads.
Pass a locally built app image. Output contains metrics only, never service logs.
"""
import base64
import json
import os
from pathlib import Path
import secrets
import subprocess
import sys
import tempfile
import time
import urllib.request

root = Path(__file__).resolve().parents[1]
images = json.loads((root / 'deploy/upstream-images.json').read_text())
image = sys.argv[1] if len(sys.argv) > 1 else 'snitch-smoke:local'
report = {}
with tempfile.TemporaryDirectory(prefix='snitch-compose-') as temp:
    temp = Path(temp)
    env = dict(os.environ, SNITCH_IMAGE=image, POSTGRES_IMAGE=images['postgres'], EMBED_IMAGE=images['embed'],
               POSTGRES_PASSWORD='synthetic:@/?#%+ '+secrets.token_hex(16), MASTER_KEY=base64.b64encode(secrets.token_bytes(32)).decode(),
               PUBLIC_URL='http://127.0.0.1:8080', COOKIE_SECURE='false', DISPLAY_NAME='Synthetic distribution', CLIPS_ROOTS='', EMBEDDINGS_MODEL='')
    override = temp / 'local.yaml'
    override.write_text('''services:
  postgres:
    mem_limit: 512m
  migrate:
    mem_limit: 256m
  app:
    mem_limit: 384m
    ports: ["127.0.0.1::8080"]
  embed:
    mem_limit: 1536m
''')
    failure = temp / 'failure.yaml'
    failure.write_text('services:\n  migrate:\n    command: [deliberately-invalid-command]\n')
    def docker(*args, check=True):
        result = subprocess.run(['docker', *args], env=env, cwd=root, capture_output=True, text=True)
        if check and result.returncode:
            raise RuntimeError('Docker operation failed (output withheld to protect bootstrap credentials): '+args[0])
        return result
    def compose(project, *args, fail=False, check=True):
        files = ['-f', str(root/'compose.yaml'), '-f', str(override)]
        if fail:
            files += ['-f', str(failure)]
        return docker('compose', '-p', project, *files, *args, check=check)
    def state(project, service):
        identifier = compose(project, 'ps', '-aq', service).stdout.strip()
        return json.loads(docker('inspect', identifier).stdout)[0]
    projects = ['snitch-test-'+secrets.token_hex(5), 'snitch-fail-'+secrets.token_hex(5)]
    try:
        good, bad = projects
        compose(good, 'up', '-d', '--wait', '--wait-timeout', '150')
        assert state(good, 'migrate')['State']['ExitCode'] == 0
        address = compose(good, 'port', 'app', '8080').stdout.strip()
        base = 'http://'+address
        def request(path, payload=None):
            data = None if payload is None else json.dumps(payload).encode()
            req = urllib.request.Request(base+path, data=data, headers={'Content-Type':'application/json', 'Origin':env['PUBLIC_URL']})
            return urllib.request.urlopen(req, timeout=15)
        assert '<html' in request('/').read().decode().lower()
        setup = json.load(request('/api/setup'))
        assert not setup['initialized'] and setup['display_name'] == 'Synthetic distribution'
        logs = compose(good, 'logs', '--no-log-prefix', 'app').stdout
        code = next(json.loads(line)['code'] for line in logs.splitlines() if 'one-time installation code' in line)
        assert request('/api/setup', {'code':code, 'username':'synthetic-admin', 'password':secrets.token_urlsafe(24)}).status == 201
        compose(good, 'stop', 'embed')
        assert request('/api/health/ready').status == 200
        compose(good, 'restart', 'app')
        base = 'http://'+compose(good, 'port', 'app', '8080').stdout.strip()
        for _ in range(60):
            try:
                assert json.load(request('/api/setup'))['initialized']
                break
            except (OSError, AssertionError):
                time.sleep(1)
        else:
            raise AssertionError('Installation did not persist over restart')
        compose(good, 'start', 'embed')
        # Public Ollama API is unavailable on host; inspect its model store instead.
        models = compose(good, 'exec', '-T', 'embed', 'sh', '-c', 'find /root/.ollama/models -type f 2>/dev/null || true').stdout
        assert not models.strip(), 'Unexpected model files'
        time.sleep(10)
        samples = []
        for _ in range(5):
            ids = [state(good, name)['Id'] for name in ('app', 'postgres', 'embed')]
            raw = docker('stats', '--no-stream', '--format', '{{json .}}', *ids).stdout
            samples.append([json.loads(line) for line in raw.splitlines()])
            time.sleep(2)
        report['idle_samples'] = samples
        replay = compose(good, 'run', '--rm', '--no-deps', '--entrypoint', '/bin/sh', 'migrate', '-c', '/snitch migrate >/dev/null 2>&1 && cat /sys/fs/cgroup/memory.peak').stdout.strip()
        report['migration_replay_peak_bytes'] = int(replay)
        report['protocol'] = {'warmup_seconds':10, 'samples':5, 'between_samples_seconds':2, 'model_loaded':False}
        report['images'] = json.loads(docker('image', 'inspect', image, images['postgres'], images['embed']).stdout)
        # Only explicitly selected nonsecret image metadata is written below.
        report['images'] = [{k:item.get(k) for k in ('RepoTags','RepoDigests','Architecture','Size')} for item in report['images']]
        report['volumes'] = {}
        for name, path in [('postgres','/var/lib/postgresql'), ('embed','/root/.ollama')]:
            report['volumes'][name] = compose(good, 'exec', '-T', name, 'du', '-sk', path).stdout.strip()
        compose(good, 'stop', 'app')
        assert state(good, 'app')['State']['ExitCode'] == 0, 'SIGTERM did not shut down cleanly'
        compose(bad, 'up', '-d', '--wait', '--wait-timeout', '90', fail=True, check=False)
        assert state(bad, 'migrate')['State']['ExitCode'] != 0
        assert not state(bad, 'app')['State']['Running'], 'Failed migration allowed API start'
        assert state(bad, 'migrate')['RestartCount'] == 0
        print('Compose acceptance passed: fresh install, reserved password, persistence, optional embeddings, clean stop, migration failure gate')
        if len(sys.argv) > 2:
            Path(sys.argv[2]).write_text(json.dumps(report, indent=2)+'\n')
    finally:
        for project in projects:
            compose(project, 'down', '-v', '--remove-orphans', check=False)
