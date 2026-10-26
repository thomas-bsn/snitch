"""Isolated real deployment acceptance test. No host media, no model downloads.
Pass a locally built app image. Output contains metrics only, never service logs.
"""
import http.cookiejar
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
    # No POSTGRES_PASSWORD and no MASTER_KEY: the deployment must start without them. An explicit password is
    # tested separately (existing installations keep working).
    base_env = {k: v for k, v in os.environ.items() if k not in ('POSTGRES_PASSWORD', 'MASTER_KEY')}
    env = dict(base_env, SNITCH_IMAGE=image,
               PUBLIC_URL='http://127.0.0.1:8080', COOKIE_SECURE='false', CLIPS_ROOTS='', EMBEDDINGS_MODEL='')
    explicit_password = 'synthetic:@/?#%+ '+secrets.token_hex(16)
    explicit_env = dict(env, POSTGRES_PASSWORD=explicit_password)
    # An empty env file keeps a developer's local .env out of the test: the variables above are the only ones.
    empty_env = temp / 'empty.env'
    empty_env.write_text('')
    override = temp / 'local.yaml'
    override.write_text('''services:
  secrets:
    mem_limit: 64m
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
    good, bad, explicit = 'snitch-test-'+secrets.token_hex(5), 'snitch-fail-'+secrets.token_hex(5), 'snitch-explicit-'+secrets.token_hex(5)
    projects = [good, bad, explicit]
    environments = {good: env, bad: env, explicit: explicit_env}
    def docker(*args, check=True, environment=env):
        result = subprocess.run(['docker', *args], env=environment, cwd=root, capture_output=True, text=True)
        if check and result.returncode:
            raise RuntimeError('Docker operation failed (output withheld to protect bootstrap credentials): '+args[0])
        return result
    def compose(project, *args, fail=False, check=True):
        files = ['--env-file', str(empty_env), '-f', str(root/'compose.yaml'), '-f', str(override)]
        if fail:
            files += ['-f', str(failure)]
        return docker('compose', '-p', project, *files, *args, check=check, environment=environments[project])
    def state(project, service):
        identifier = compose(project, 'ps', '-aq', service).stdout.strip()
        return json.loads(docker('inspect', identifier).stdout)[0]
    def up(project):
        return compose(project, 'up', '-d', '--wait', '--wait-timeout', '150', check=False)
    def stored_password(project):
        return compose(project, 'exec', '-T', 'app', 'cat', '/data/secrets/postgres_password').stdout
    def fingerprint(project):
        # Hash only: the password itself is never printed.
        return compose(project, 'exec', '-T', 'app', 'sha256sum', '/data/secrets/postgres_password').stdout.split()[0]
    def api(project, path, payload=None):
        base = 'http://'+compose(project, 'port', 'app', '8080').stdout.strip()
        data = None if payload is None else json.dumps(payload).encode()
        req = urllib.request.Request(base+path, data=data, headers={'Content-Type':'application/json', 'Origin':env['PUBLIC_URL']})
        return urllib.request.urlopen(req, timeout=15)
    def stored_key(project):
        return compose(project, 'exec', '-T', 'app', 'cat', '/data/secrets/master_key').stdout
    def key_fingerprint(project):
        # Hash only: the key itself is never printed.
        return compose(project, 'exec', '-T', 'app', 'sha256sum', '/data/secrets/master_key').stdout.split()[0]
    def login(project, password):
        # A session of its own, with cookie and CSRF token, to call the administration.
        opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
        base = 'http://'+compose(project, 'port', 'app', '8080').stdout.strip()
        def call(method, path, payload=None):
            headers = {'Content-Type':'application/json', 'Origin':env['PUBLIC_URL']}
            if method != 'GET':
                headers['X-CSRF-Token'] = csrf[0]
            data = None if payload is None else json.dumps(payload).encode()
            return json.load(opener.open(urllib.request.Request(base+path, data=data, method=method, headers=headers), timeout=15))
        csrf = ['']
        call('POST', '/api/auth/login', {'username':'synthetic-admin', 'password':password})
        csrf[0] = call('GET', '/api/auth/me')['csrf_token']
        return call
    def install(project):
        logs = compose(project, 'logs', '--no-log-prefix', 'app').stdout
        code = next(json.loads(line)['code'] for line in logs.splitlines() if 'one-time installation code' in line)
        password = secrets.token_urlsafe(24)
        assert api(project, '/api/setup', {'code':code, 'username':'synthetic-admin', 'password':password}).status == 201
        return password
    def wait_initialized(project):
        for _ in range(60):
            try:
                assert json.load(api(project, '/api/setup'))['initialized']
                return
            except (OSError, AssertionError):
                time.sleep(1)
        raise AssertionError('Installation did not persist over restart')
    def psql(project, password):
        # Through the service name, not loopback: the image trusts local connections, but not the network ones.
        return compose(project, 'exec', '-T', '-e', 'PGPASSWORD='+password, 'postgres', 'psql', '-h', 'postgres', '-U', 'receipts', '-d', 'receipts', '-tAc', 'select 1', check=False)
    try:
        started = up(good)
        if started.returncode:
            # Diagnostic without secrets: the tail of the `up` output (pull progress and container events, never
            # service logs), the service states, and the logs of the services that never print the one-time
            # installation code (never the app logs).
            print('--- up (tail)', file=sys.stderr)
            print('\n'.join((started.stdout + started.stderr).splitlines()[-25:]), file=sys.stderr)
            print(compose(good, 'ps', '-a', '--format', '{{.Service}} {{.State}} {{.Health}} {{.ExitCode}}', check=False).stdout, file=sys.stderr)
            for service in ('postgres', 'migrate', 'embed'):
                print('--- '+service, file=sys.stderr)
                print(compose(good, 'logs', '--no-log-prefix', '--tail', '15', service, check=False).stdout, file=sys.stderr)
            raise RuntimeError('Docker operation failed (output withheld to protect bootstrap credentials): compose up')
        assert state(good, 'secrets')['State']['ExitCode'] == 0
        assert state(good, 'migrate')['State']['ExitCode'] == 0
        # Fresh install without any password variable: the password was produced, and the database enforces it.
        generated = stored_password(good)
        assert len(generated) >= 32 and generated == generated.strip(), 'password not produced'
        assert psql(good, 'wrong-'+generated[:8]).returncode != 0, 'database accepted a wrong password'
        assert psql(good, generated).stdout.strip() == '1', 'database refused the produced password'
        # The database is not published to the host, only reachable on the deployment network.
        assert not state(good, 'postgres')['NetworkSettings']['Ports'].get('5432/tcp'), 'database published to the host'
        password_fingerprint = fingerprint(good)
        assert '<html' in api(good, '/').read().decode().lower()
        setup = json.load(api(good, '/api/setup'))
        assert not setup['initialized'] and 'display_name' not in setup
        admin_password = install(good)
        # No MASTER_KEY given: the application generated its key and keeps it in the secrets volume, which the
        # database service cannot read (private file) and which is not part of the database.
        key = stored_key(good).strip()
        assert len(key) == 44, 'master key not generated'
        assert compose(good, 'exec', '-T', '-u', 'postgres', 'postgres', 'cat', '/run/snitch-secrets/master_key', check=False).returncode != 0, 'database service can read the master key'
        key_hash = key_fingerprint(good)
        compose(good, 'stop', 'embed')
        assert api(good, '/api/health/ready').status == 200
        compose(good, 'restart', 'app')
        wait_initialized(good)
        compose(good, 'start', 'embed')
        # Second start (containers recreated, volumes kept): same password, same data, installation still done.
        compose(good, 'down', '--remove-orphans')
        assert up(good).returncode == 0, 'second start failed'
        assert fingerprint(good) == password_fingerprint, 'password changed over a restart'
        assert key_fingerprint(good) == key_hash, 'master key changed over a restart'
        wait_initialized(good)
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
        # Master key: encrypts the secrets, exportable by the administrator, absent from a dump of the database, and
        # its loss does not prevent the start: the secrets to re-enter are listed.
        admin = login(good, admin_password)
        admin('PUT', '/api/admin/settings', {'client_id':'synthetic-app', 'client_secret':'synthetic-discord-secret', 'registration_approval':True})
        exported = admin('POST', '/api/admin/master-key/export')
        assert exported['master_key'] == key and exported['warning'] == 'master_key_not_replayable', 'exported key differs from the kept one'
        dump = compose(good, 'exec', '-T', 'postgres', 'pg_dump', '-U', 'receipts', '-d', 'receipts', '-a').stdout
        assert key not in dump and 'synthetic-discord-secret' not in dump, 'key or secret readable in a dump of the database'
        compose(good, 'exec', '-T', 'app', 'rm', '/data/secrets/master_key')
        compose(good, 'restart', 'app')
        wait_initialized(good)
        assert state(good, 'app')['State']['Running'], 'application did not start without its key'
        admin = login(good, admin_password)
        settings = admin('GET', '/api/admin/settings')
        assert settings['unreadable_secrets'] == ['discord_client_id', 'discord_client_secret'] and settings['master_key_origin'] == 'generated', 'unreadable secrets not listed'
        assert stored_key(good).strip() != key, 'a new key was not generated'
        admin('PUT', '/api/admin/settings', {'client_id':'synthetic-app', 'client_secret':'synthetic-discord-secret', 'registration_approval':True})
        assert admin('GET', '/api/admin/settings')['unreadable_secrets'] == [], 're-entering the secrets did not work'
        # The produced password never appears in any service log.
        assert generated not in compose(good, 'logs', '--no-log-prefix').stdout, 'password written to a log'
        assert key not in compose(good, 'logs', '--no-log-prefix').stdout, 'master key written to a log'
        generated = None
        compose(good, 'stop', 'app')
        assert state(good, 'app')['State']['ExitCode'] == 0, 'SIGTERM did not shut down cleanly'
        compose(bad, 'up', '-d', '--wait', '--wait-timeout', '90', fail=True, check=False)
        assert state(bad, 'migrate')['State']['ExitCode'] != 0
        assert not state(bad, 'app')['State']['Running'], 'Failed migration allowed API start'
        assert state(bad, 'migrate')['RestartCount'] == 0
        # Existing installation with an explicit password: honoured, kept over a restart, and still working once
        # the variable is removed (the password is kept in the volume).
        assert up(explicit).returncode == 0, 'explicit password start failed'
        assert stored_password(explicit) == explicit_password, 'explicit password not honoured'
        assert psql(explicit, explicit_password).stdout.strip() == '1', 'database refused the explicit password'
        install(explicit)
        assert up(explicit).returncode == 0
        compose(explicit, 'down', '--remove-orphans')
        assert up(explicit).returncode == 0, 'explicit password restart failed'
        wait_initialized(explicit)
        compose(explicit, 'down', '--remove-orphans')
        environments[explicit] = env
        assert up(explicit).returncode == 0, 'restart without the variable failed'
        wait_initialized(explicit)
        assert stored_password(explicit) == explicit_password, 'password changed when the variable was removed'
        print('Compose acceptance passed: fresh install without variables, produced password (enforced, kept over restart, not published, not logged), generated master key (kept over restart, exported, not in the database, loss reported and recoverable), explicit password (honoured, restart, variable removed), persistence, optional embeddings, clean stop, migration failure gate')
        if len(sys.argv) > 2:
            Path(sys.argv[2]).write_text(json.dumps(report, indent=2)+'\n')
    finally:
        for project in projects:
            compose(project, 'down', '-v', '--remove-orphans', check=False)
