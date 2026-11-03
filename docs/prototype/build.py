import base64, json, os, sys

tpl, assets, icons, out = sys.argv[1:5]
def uri(path, mime):
    with open(path, 'rb') as f:
        return f'data:{mime};base64,' + base64.b64encode(f.read()).decode()

sp = {}
for name in sorted(os.listdir(os.path.join(assets, 'sp'))):
    key = name.rsplit('.', 1)[0]
    sp[key] = uri(os.path.join(assets, 'sp', name), 'image/gif')
alld = os.path.join(assets, 'all')
for name in sorted(os.listdir(alld)):
    key, ext = name.rsplit('.', 1)
    sp[key] = uri(os.path.join(alld, name), 'image/gif' if ext == 'gif' else 'image/png')
cry = {}
for name in sorted(os.listdir(os.path.join(assets, 'cry'))):
    cry[name.rsplit('.', 1)[0]] = uri(os.path.join(assets, 'cry', name), 'audio/ogg')
icon = {}
for k in ('claude', 'codex'):
    with open(os.path.join(icons, k + '.svg')) as f:
        icon[k] = f.read().strip().replace('<title>Claude Code</title>', '').replace('<title>Codex</title>', '')

data = json.dumps({'sp': sp, 'cry': cry, 'icon': icon})
html = open(tpl).read().replace('__ASSETS__', data)
open(out, 'w').write(html)
print(len(html) // 1024, 'KB')
