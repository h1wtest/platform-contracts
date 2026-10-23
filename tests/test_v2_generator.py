import importlib.util
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location('generate_v2', ROOT / 'scripts/generate_v2.py')

class V2GeneratorTests(unittest.TestCase):
    def test_unsupported_constraints_fail_closed(self):
        module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(module)
        for rule in ({'type':'string','unknown':True}, {'type':'object','additionalProperties':True}, {'type':'string','format':'unimplemented'}, {'type':'string','pattern':'unimplemented'}):
            with self.assertRaises(ValueError):
                module.check_rule(rule)

    def test_every_public_operation_has_typed_response(self):
        import json
        schema = json.loads((ROOT/'schema/business-v1.json').read_text())
        for operation in schema['x-endpoints']:
            self.assertIn(operation['response'], schema['$defs'])
            if 'request' in operation:
                self.assertIn(operation['request'], schema['$defs'])

if __name__ == '__main__':
    unittest.main()
