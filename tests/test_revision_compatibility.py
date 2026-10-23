import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / 'scripts/check_revision_compatibility.py'


class RevisionCompatibilityTests(unittest.TestCase):
    def test_push_and_dispatch_reject_breaking_published_schema(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            def git(*args):
                return subprocess.check_output(['git', *args], cwd=root, text=True).strip()
            git('init', '-q')
            git('config', 'user.email', 'contracts@example.test')
            git('config', 'user.name', 'Contracts test')
            (root / 'schema').mkdir()
            for name in ('events-v1', 'internal-http-v1'):
                (root / f'schema/{name}.json').write_text(json.dumps({'type':'object'}))
            git('add', '.')
            git('commit', '-qm', 'Initial contract')
            base = git('rev-parse', 'HEAD')
            (root / 'schema/events-v1.json').write_text(json.dumps({'type':'object','not':{}}))
            git('add', '.')
            git('commit', '-qm', 'Incompatible constraint')
            for event, previous in [('push', base), ('push', '0'*40), ('workflow_dispatch', ''), ('pull_request', base)]:
                result = subprocess.run([sys.executable, str(SCRIPT), event, previous], cwd=root, text=True, capture_output=True)
                self.assertNotEqual(result.returncode, 0, event)
                self.assertIn('v1 wire contract changed', result.stderr)
            missing = subprocess.run([sys.executable, str(SCRIPT), 'push', 'f'*40], cwd=root, capture_output=True)
            self.assertNotEqual(missing.returncode, 0)

    def test_root_commit_is_explicit_initial_publication(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for args in [('init','-q'),('config','user.email','contracts@example.test'),('config','user.name','Contracts test')]:
                subprocess.run(['git',*args],cwd=root,check=True,capture_output=True)
            (root/'schema').mkdir()
            for name in ('events-v1', 'internal-http-v1'):
                (root/f'schema/{name}.json').write_text('{}')
            subprocess.run(['git','add','.'],cwd=root,check=True,capture_output=True)
            subprocess.run(['git','commit','-qm','Initial'],cwd=root,check=True,capture_output=True)
            result = subprocess.run([sys.executable,str(SCRIPT),'workflow_dispatch',''],cwd=root,text=True,capture_output=True)
            self.assertEqual(result.returncode,0,result.stderr)
            self.assertIn('Initial',result.stdout)
