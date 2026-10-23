import copy
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('compatibility', Path(__file__).resolve().parents[1] / 'scripts/check_compatibility.py')
compatibility = importlib.util.module_from_spec(spec)
spec.loader.exec_module(compatibility)

class CompatibilityTests(unittest.TestCase):
    def setUp(self):
        import json
        self.schema = json.loads((Path(__file__).resolve().parents[1] / 'schema/events-v1.json').read_text())

    def test_existing_wire_contract_is_accepted(self):
        self.assertEqual(compatibility.breaking_changes(self.schema, copy.deepcopy(self.schema)), [])

    def test_removing_required_payload_and_weakening_secret_separation_are_breaking(self):
        changed = copy.deepcopy(self.schema)
        changed['$defs']['EmailVerification']['required'].remove('code')
        self.assertTrue(compatibility.breaking_changes(self.schema, changed))
        changed = copy.deepcopy(self.schema)
        changed['oneOf'][1].pop('not')
        self.assertTrue(compatibility.breaking_changes(self.schema, changed))

    def test_required_envelope_metadata_and_known_event_type_cannot_disappear(self):
        changed = copy.deepcopy(self.schema)
        changed['required'].remove('event_id')
        self.assertTrue(compatibility.breaking_changes(self.schema, changed))
        changed = copy.deepcopy(self.schema)
        changed['properties']['event_type']['enum'].remove('identity.user_registered')
        self.assertTrue(compatibility.breaking_changes(self.schema, changed))

    def test_unlisted_root_constraints_are_breaking(self):
        for key, value in {'not': {}, 'anyOf': [False], '$ref': '#/$defs/UserRegistered', 'const': {}}.items():
            with self.subTest(keyword=key):
                changed = copy.deepcopy(self.schema)
                changed[key] = value
                self.assertTrue(compatibility.breaking_changes(self.schema, changed))

    def test_only_schema_annotations_are_ignored_not_named_properties(self):
        changed = copy.deepcopy(self.schema)
        changed['description'] = 'Clarified documentation'
        changed['$defs']['UserRegistered']['title'] = 'Registered user'
        self.assertEqual(compatibility.breaking_changes(self.schema, changed), [])
        changed['$defs']['UserRegistered']['properties']['title'] = {'type': 'string'}
        self.assertTrue(compatibility.breaking_changes(self.schema, changed))
