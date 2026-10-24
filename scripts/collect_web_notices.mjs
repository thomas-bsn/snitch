import fs from 'node:fs';
import path from 'node:path';
const lock = JSON.parse(fs.readFileSync('package-lock.json'));
fs.mkdirSync('/notices/web', {recursive:true});
for (const [directory, meta] of Object.entries(lock.packages)) {
  if (!directory || meta.dev) continue;
  const destination = `/notices/web/${directory.replaceAll('/', '_')}@${meta.version}`;
  fs.mkdirSync(destination, {recursive:true});
  const files = fs.readdirSync(directory).filter(name => /^(LICENSE|NOTICE|COPYRIGHT)/i.test(name));
  if (!files.length) throw Error(`Missing license: ${directory}`);
  for (const file of files) fs.copyFileSync(path.join(directory,file),path.join(destination,file));
}
