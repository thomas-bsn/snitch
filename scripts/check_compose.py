"""Validate this distribution's Forgeyard contract (not a replacement Forgeyard parser)."""
from copy import deepcopy
from pathlib import Path
import json
import re
from ruamel.yaml import YAML
root = Path(__file__).resolve().parents[1]
UPSTREAM = json.loads((root/'deploy/upstream-images.json').read_text())
VARIABLES = {'postgres':'POSTGRES_IMAGE','embed':'EMBED_IMAGE'}
APP_IMAGE = r'ghcr\.io/[a-z0-9._/-]+(:[A-Za-z0-9._-]+)?(@sha256:[0-9a-f]{64})?|\$\{[A-Z_]+:-ghcr\.io/[a-z0-9._/-]+:[A-Za-z0-9._-]+\}'

def check_image(name, image):
    assert isinstance(image,str)
    if name in VARIABLES:
        # Upstream images: exactly the reference pinned by digest in upstream-images.json, nothing else.
        pinned = UPSTREAM[name]
        assert re.search(r'@sha256:[0-9a-f]{64}$',pinned)
        assert image in (pinned, '${'+VARIABLES[name]+':-'+pinned+'}')
    else:
        assert re.fullmatch(APP_IMAGE,image)

def check(d):
    assert set(d) <= {'services','volumes','name','version'} | {k for k in d if k.startswith('x-')}
    services, volumes = d['services'], d['volumes']
    assert set(services) == {'secrets','postgres','migrate','app','embed'}
    assert set(volumes) == {'database','embeddings','media','secrets'}
    assert all(re.fullmatch('[a-z0-9-]{1,32}', n) for n in services)
    assert all(v is None or v == {} for v in volumes.values())
    exposed = []
    allowed = {'image','command','entrypoint','user','environment','volumes','depends_on','healthcheck','restart','x-forgeyard'}
    for name, s in services.items():
        assert set(s) <= allowed
        check_image(name,s.get('image'))
        assert isinstance(s.get('environment',{}),dict)
        for key in ('command','entrypoint'):
            assert key not in s or isinstance(s[key],(list,str))
        for mount in s.get('volumes',[]):
            if isinstance(mount,str):
                parts = mount.split(':')
                assert len(parts) in (2,3) and (len(parts)==2 or parts[2]=='ro')
                source,target = parts[:2]
            else:
                assert set(mount) <= {'type','source','target','read_only'} and mount['type']=='volume'
                source,target = mount['source'],mount['target']
            assert source in volumes and target.startswith('/')
        assert re.fullmatch(r'no|always|unless-stopped|on-failure(?::[1-9][0-9]*)?',s.get('restart','unless-stopped'))
        health = s.get('healthcheck',{})
        assert set(health) <= {'test','interval','timeout','start_period','retries','disable'}
        if 'test' in health:
            assert isinstance(health['test'],str) or (isinstance(health['test'],list) and health['test'][0] in ('CMD','CMD-SHELL','NONE'))
        for dependency, condition in s.get('depends_on',{}).items():
            assert dependency in services and dependency != name
            assert set(condition)=={'condition'}
            assert condition['condition'] in ('service_started','service_healthy','service_completed_successfully')
            if condition['condition']=='service_healthy':
                assert services[dependency].get('healthcheck',{}).get('test')
        meta = s.get('x-forgeyard',{})
        assert set(meta) <= {'memory','backup','expose'}
        assert isinstance(meta.get('memory',512),int) and meta.get('memory',512)>0
        if 'backup' in meta:
            assert meta['backup']=={'type':'postgres'}
        if 'expose' in meta:
            assert set(meta['expose'])=={'port'}
            exposed.append((name,meta['expose']['port']))
    assert exposed==[('app',8080)]
    assert services['migrate']['depends_on']=={'postgres':{'condition':'service_healthy'}}
    assert services['migrate']['restart']=='no'
    assert services['app']['depends_on']=={'postgres':{'condition':'service_healthy'},'migrate':{'condition':'service_completed_successfully'}}
    assert services['postgres']['x-forgeyard']['backup']=={'type':'postgres'}
    assert 'media:/data/media' in services['app'].get('volumes', [])
    check_database_password(services)

def check_database_password(services):
    """The database password is produced by the `secrets` job, kept in its volume, read from a file by the
    database, the migration and the application, never required as a variable and never in the compose."""
    job = services['secrets']
    assert job['restart']=='no' and job['command']==['init-secrets']
    assert job['volumes']==['secrets:/data/secrets']
    # An explicit value stays optional and takes precedence (existing installations).
    assert job['environment']=={'POSTGRES_PASSWORD':'${POSTGRES_PASSWORD:-}'}
    pg = services['postgres']
    assert pg['depends_on']=={'secrets':{'condition':'service_completed_successfully'}}
    assert pg['environment']['POSTGRES_PASSWORD_FILE']=='/run/snitch-secrets/postgres_password'
    assert 'secrets:/run/snitch-secrets:ro' in pg['volumes'] and 'database:/var/lib/postgresql' in pg['volumes']
    for name in ('migrate','app'):
        env = services[name]['environment']
        assert env['POSTGRES_PASSWORD_FILE']=='/data/secrets/postgres_password'
        assert 'secrets:/data/secrets:ro' in services[name]['volumes']
    for name, s in services.items():
        for key, value in s.get('environment',{}).items():
            assert key!='PGPASSWORD', 'password passed through a variable instead of the file'
            if name!='secrets':
                assert key!='POSTGRES_PASSWORD', 'password passed as a variable instead of the file'
            assert 'POSTGRES_PASSWORD:?' not in str(value), 'the database password must not be mandatory'
    # The database is reachable only on the deployment network: `ports` is rejected for every service.
    assert all('ports' not in s for s in services.values())

if __name__=='__main__':
    raw = YAML(typ='safe').load((root/'compose.yaml').read_text())
    check(raw)
    fixtures = []
    for key in ('ports','networks','network_mode','env_file','container_name','privileged','cap_add','devices','extends','build','tmpfs','mem_limit'):
        fixtures.append(('forbidden '+key, lambda d,k=key:d['services']['app'].update({k:True})))
    fixtures += [
        ('bind mount',lambda d:d['services']['app'].update(volumes=['/host:/clips:ro'])),
        ('anonymous mount',lambda d:d['services']['app'].update(volumes=['/clips'])),
        ('volume options',lambda d:d['volumes'].update(database={'external':True})),
        ('bad metadata',lambda d:d['services']['app']['x-forgeyard'].update(gpu=True)),
        ('extra exposure',lambda d:d['services']['embed']['x-forgeyard'].update(expose={'port':11434})),
        ('looping job',lambda d:d['services']['migrate'].update(restart='always')),
        ('no migration gate',lambda d:d['services']['app']['depends_on'].pop('migrate')),
        ('no healthcheck',lambda d:d['services']['postgres'].pop('healthcheck')),
        ('unknown dependency',lambda d:d['services']['app']['depends_on'].update(missing={'condition':'service_started'})),
        ('invalid health',lambda d:d['services']['app']['healthcheck'].update(test=['SHELL','true'])),
        ('image without ghcr default',lambda d:d['services']['app'].update(image='${SNITCH_IMAGE:?Set SNITCH_IMAGE}')),
        ('floating tag without digest',lambda d:d['services']['embed'].update(image='${EMBED_IMAGE:-ollama/ollama:0.40.10}')),
        ('latest tag',lambda d:d['services']['postgres'].update(image='${POSTGRES_IMAGE:-pgvector/pgvector:latest}')),
        ('different upstream image',lambda d:d['services']['embed'].update(image='${EMBED_IMAGE:-ollama/ollama:0.40.10@sha256:'+'0'*64+'}')),
        ('wrong variable name',lambda d:d['services']['postgres'].update(image=d['services']['postgres']['image'].replace('POSTGRES_IMAGE','EMBED_IMAGE'))),
        ('mirror instead of upstream',lambda d:d['services']['postgres'].update(image='${POSTGRES_IMAGE:-ghcr.io/thomas-bsn/snitch-postgres:latest}')),
        ('unknown non-ghcr image',lambda d:d['services']['app'].update(image='${SNITCH_IMAGE:-docker.io/library/nginx:latest}')),
        ('unknown non-ghcr literal',lambda d:d['services']['migrate'].update(image='nginx:latest')),
        ('database published to the host',lambda d:d['services']['postgres'].update(ports=['5432:5432'])),
        ('mandatory database password',lambda d:d['services']['secrets']['environment'].update(POSTGRES_PASSWORD='${POSTGRES_PASSWORD:?Set POSTGRES_PASSWORD}')),
        ('password in the database environment',lambda d:d['services']['postgres']['environment'].update(POSTGRES_PASSWORD='literal')),
        ('password passed by variable to the app',lambda d:d['services']['app']['environment'].update(PGPASSWORD='${POSTGRES_PASSWORD:-}')),
        ('password file not read by migrate',lambda d:d['services']['migrate']['environment'].pop('POSTGRES_PASSWORD_FILE')),
        ('database not gated by secrets',lambda d:d['services']['postgres'].pop('depends_on')),
        ('secrets volume writable for database',lambda d:d['services']['postgres'].update(volumes=['database:/var/lib/postgresql','secrets:/run/snitch-secrets'])),
        ('looping secrets job',lambda d:d['services']['secrets'].update(restart='always')),
        ('wrong backup',lambda d:d['services']['postgres']['x-forgeyard'].update(backup={'type':'files'})),
    ]
    for label, mutate in fixtures:
        candidate = deepcopy(raw)
        mutate(candidate)
        try:
            check(candidate)
        except (AssertionError,KeyError,ValueError):
            continue
        raise AssertionError('Accepted invalid fixture: '+label)
    print(f'Forgeyard distribution valid; {len(fixtures)} negative fixtures rejected')
