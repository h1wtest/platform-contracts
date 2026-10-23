"""Compatibility Make arguments fail closed and preserve actual schema paths."""
import json
from pathlib import Path
import shlex
import subprocess
import sys
import tempfile
import unittest

MAKEFILE = Path(__file__).resolve().parents[1] / 'Makefile'


class MakeCompatibilityTests(unittest.TestCase):
    def test_missing_or_whitespace_base_schema_fails_before_python(self):
        for base in ('', '   '):
            with self.subTest(base=base), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                recorder = root / 'recorder.py'
                recorder.write_text('from pathlib import Path\nPath("invoked").touch()\n')
                python = f'{shlex.quote(sys.executable)} {shlex.quote(str(recorder))}'
                result = subprocess.run(['make', '-f', str(MAKEFILE), 'compatibility', f'PYTHON={python}', f'BASE_SCHEMA={base}'], cwd=root, capture_output=True, text=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn('BASE_SCHEMA is required', result.stderr)
                self.assertFalse((root / 'invoked').exists())

    def test_schema_paths_preserve_spaces_quotes_and_shell_metacharacters(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'compatibility').touch()
            recorder = root / 'recorder.py'
            recorder.write_text('import json, pathlib, sys\npathlib.Path("invocation.json").write_text(json.dumps(sys.argv[1:]))\n')
            python = f'{shlex.quote(sys.executable)} {shlex.quote(str(recorder))}'
            base = 'previous schemas/owner\'s "events"; $HOME.json'
            current = 'current schemas/owner\'s "http"; $HOME.json'
            for extra, expected in (([], 'schema/events-v1.json'), ([f'CURRENT_SCHEMA={current.replace("$", "$$")}'], current)):
                with self.subTest(current=expected):
                    subprocess.run(['make', '-f', str(MAKEFILE), 'compatibility', f'PYTHON={python}', f'BASE_SCHEMA={base.replace("$", "$$")}', *extra], cwd=root, check=True, capture_output=True)
                    self.assertEqual(json.loads((root / 'invocation.json').read_text()), ['scripts/check_compatibility.py', base, expected])


if __name__ == '__main__':
    unittest.main()
