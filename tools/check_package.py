#!/usr/bin/env python3
"""Static checks for the AI implementation guide package. Only standard library."""
import json
import pathlib
import re
import sys

root = pathlib.Path(__file__).resolve().parents[1]
manifest = json.loads((root / 'tickets/manifest.json').read_text(encoding='utf8'))
assert manifest['task_count'] == len(manifest['tasks']) == 44, 'wrong ticket count'
assert len({t['id'] for t in manifest['tasks']}) == 44, 'duplicate ticket ID'
for index, t in enumerate(manifest['tasks']):
    assert (root / t['ticket']).is_file(), t['ticket']
    assert t['depends_on'] == ([manifest['tasks'][index-1]['id']] if index else []), t['id']
    content = (root / t['ticket']).read_text(encoding='utf8')
    for word in ['要完成的实现', '必须新增的测试', '验证步骤', '完成判据']:
        assert word in content, (t['id'], word)
    for p in t['spec_refs']:
        assert (root / p).exists(), (t['id'], p)

for file in root.rglob('*.json'):
    json.loads(file.read_text(encoding='utf8'))
for phase in range(13):
    assert (root / f'phases/P{phase:02}.md').is_file()

# Check links that point at local Markdown files. Ignore external links and templates.
bad=[]
for file in root.rglob('*.md'):
    if '__pycache__' in str(file): continue
    txt=file.read_text(encoding='utf8')
    for link in re.findall(r'\[[^\]]+\]\(([^)]+)\)',txt):
        if link.startswith(('http:', 'https:', '#')):continue
        target=(file.parent/link.split('#')[0]).resolve()
        if not target.exists():bad.append((str(file.relative_to(root)),link))
assert not bad, f'broken local links: {bad}'
print(f'PASS package static checks: {len(manifest["tasks"])} tickets, 13 phases, valid JSON and Markdown links')
