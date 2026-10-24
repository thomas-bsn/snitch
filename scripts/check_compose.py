"""Validate this distribution's Forgeyard contract (not a replacement Forgeyard parser)."""
from copy import deepcopy
from pathlib import Path
import re
from ruamel.yaml import YAML
root = Path(__file__).resolve().parents[1]

def check(d):
    assert set(d) <= {'services','volumes','name','version'} | {k for k in d if k.startswith('x-')}
    services, volumes = d['services'], d['volumes']
    assert set(services) == {'postgres','migrate','app','embed'}
    assert all(re.fullmatch('[a-z0-9-]{1,32}', n) for n in services)
    assert all(v is None or v == {} for v in volumes.values())
    exposed = []
    allowed = {'image','command','entrypoint','user','environment','volumes','depends_on','healthcheck','restart','x-forgeyard'}
    for name, s in services.items():
        assert set(s) <= allowed
        assert isinstance(s.get('image'),str) and s['image'].startswith(('ghcr.io/','${'))
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
