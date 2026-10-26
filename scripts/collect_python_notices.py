"""Copy installed distributions' original license texts, not just homepage links."""
from importlib.metadata import distributions
from pathlib import Path
import shutil
out = Path('/usr/share/licenses/snitch/python')
for dist in distributions():
    if dist.metadata['Name'] == 'snitch-client':
        continue
    files = [f for f in dist.files or [] if any(p.lower().startswith(('license', 'notice', 'copying', 'copyright')) for p in f.parts)]
    extra = Path('/licenses-extra') / dist.metadata['Name']  # texts copied from upstream for wheels that ship none
    if not files and extra.is_dir():
        target = out / (dist.metadata['Name']+'@'+dist.version)
        target.mkdir(parents=True, exist_ok=True)
        for source in extra.iterdir():
            shutil.copyfile(source, target / source.name)
        continue
    if not files:
        raise RuntimeError('Missing license: '+dist.metadata['Name'])
    for file in files:
        source = Path(dist.locate_file(file))
        if source.is_file():
            target = out / (dist.metadata['Name']+'@'+dist.version) / str(file)
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(source,target)
