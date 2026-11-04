"""Read-only vault access. Paths and links never escape the selected vault."""
import json
import os
from pathlib import Path
import sys
from urllib.parse import unquote

LIMIT = 1024 * 1024

def root_path(value):
    root = Path(value).expanduser().resolve()
    if not root.is_dir():
        raise ValueError('Vault 폴더를 찾을 수 없습니다.')
    return root

def notes(root):
    found = []
    for directory, dirs, files in os.walk(root, followlinks=False):
        dirs[:] = sorted(d for d in dirs if not d.startswith('.') and not Path(directory, d).is_symlink())
        for name in sorted(files):
            p = Path(directory, name)
            if name.lower().endswith('.md') and not p.is_symlink():
                found.append(p.relative_to(root).as_posix())
                if len(found) >= 20000:
                    return sorted(found)
    return sorted(found)

def safe_note(root, name):
    p = (root / name).resolve()
    if not p.is_relative_to(root) or p.suffix.lower() != '.md':
        raise ValueError('Vault 내부의 Markdown 노트만 열 수 있습니다.')
    return p

def read(root, name):
    p = safe_note(root, name)
    if p.stat().st_size > LIMIT:
        raise ValueError('1 MiB 이하의 노트만 표시합니다.')
    return p.read_text(encoding='utf-8')

def resolve(root, target, current):
    target = unquote(target).split('#', 1)[0].split('?', 1)[0]
    if not target:
        return current
    if not target.lower().endswith('.md'):
        target += '.md'
    candidates = [target.lstrip('/')] if target.startswith('/') else [str(Path(current).parent / target), target]
    for name in candidates:
        p = safe_note(root, name)
        if p.is_file():
            return p.relative_to(root).as_posix()
    matches = [n for n in notes(root) if n == target or n.endswith('/' + target)]
    if len(matches) == 1:
        return matches[0]
    raise ValueError('동일한 이름의 노트가 여러 개입니다. 경로로 검색해 주세요.' if matches else '연결된 노트를 찾을 수 없습니다.')

def request(action, args):
    if action == 'vaults':
        config = Path.home() / 'Library/Application Support/obsidian/obsidian.json'
        if not config.exists():
            return []
        return [v['path'] for v in json.loads(config.read_text()).get('vaults', {}).values() if Path(v['path']).is_dir()]
    root = root_path(args[0])
    if action == 'search':
        query = args[1].casefold().strip()
        all_notes = notes(root)
        names = [n for n in all_notes if not query or query in n.casefold()]
        results = names[:200] if query else names
        if query and len(results) < 200:
            named = set(names)
            for name in all_notes:
                if name in named:
                    continue
                try:
                    if query in read(root, name).casefold():
                        results.append(name)
                except (OSError, UnicodeError, ValueError):
                    continue
                if len(results) == 200:
                    break
        return {'root': str(root), 'notes': results, 'total': len(all_notes), 'limited': (bool(query) and len(results) == 200) or len(all_notes) == 20000}
    if action == 'read':
        name = resolve(root, args[1], args[2])
        return {'path': name, 'text': read(root, name)}
    raise ValueError('Unknown action')

if __name__ == '__main__':
    try:
        print(json.dumps(request(sys.argv[1], sys.argv[2:]), ensure_ascii=False))
    except Exception as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
