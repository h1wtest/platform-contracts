import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('http_generator', Path(__file__).resolve().parents[1] / 'scripts/generate_http.py')
generator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(generator)


class HTTPGeneratorTests(unittest.TestCase):
    def test_unsupported_constraints_fail_instead_of_generating_weaker_codecs(self):
        for rule in [
            {'type':'string', 'pattern':'secret'},
            {'type':'object', 'properties':{}, 'additionalProperties':True},
            {'type':'string', 'format':'email'},
            {'$ref':'other-contract.json'},
            {'type':'object','additionalProperties':False,'properties':{'child':{'type':'integer','maximum':10}}},
        ]:
            with self.subTest(rule=rule), self.assertRaises(ValueError):
                generator.check_rule(rule)
