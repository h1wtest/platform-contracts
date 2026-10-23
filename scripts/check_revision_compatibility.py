#!/usr/bin/env python3
"""Compare both v1 schemas with the trusted event-specific predecessor."""
import json
import re
import subprocess
import sys
from pathlib import Path

from check_compatibility import breaking_changes


def git(*args):
    return subprocess.check_output(['git', *args], text=True).strip()


def main():
    if len(sys.argv) != 3:
        raise SystemExit('usage: check_revision_compatibility.py EVENT_NAME BASE_SHA')
    event, baseline = sys.argv[1:]
    if event not in ('push', 'pull_request', 'workflow_dispatch'):
        raise SystemExit('Unsupported compatibility event')
    if event == 'workflow_dispatch' or event == 'push' and baseline == '0' * 40:
        parents = git('rev-list', '--parents', '-n', '1', 'HEAD').split()[1:]
        baseline = parents[0] if parents else ''
    elif not re.fullmatch('[0-9a-fA-F]{40}', baseline):
        raise SystemExit('Trusted compatibility baseline SHA is required')
    if baseline:
        git('cat-file', '-e', baseline + '^{commit}')
    paths = {p.as_posix() for p in Path('schema').glob('*.json')}
    if baseline:
        paths.update(git('ls-tree', '-r', '--name-only', baseline, '--', 'schema').splitlines())
    for path in sorted(paths):
        name = Path(path).stem
        if not Path(path).is_file():
            raise SystemExit(f'Published schema removed: {path}')
        current = json.loads(Path(path).read_text())
        if not baseline:
            print(f'Initial {name} contract: repository root has no predecessor.')
            continue
        # A missing schema is initial publication only after the baseline commit
        # itself has been verified. Other Git failures are never skipped.
        entries = git('ls-tree', '--name-only', baseline, '--', path).splitlines()
        if path not in entries:
            print(f'Initial {name} contract: no schema at verified baseline {baseline}.')
            continue
        previous = json.loads(git('show', f'{baseline}:{path}'))
        errors = breaking_changes(previous, current)
        if errors:
            raise SystemExit(f'{path}: ' + '; '.join(errors))


if __name__ == '__main__':
    main()
