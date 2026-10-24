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

    def test_protocol_status_cookie_and_anonymous_catalog(self):
        import json
        module = importlib.util.module_from_spec(SPEC)
        SPEC.loader.exec_module(module)
        schema = json.loads((ROOT/'schema/business-v1.json').read_text())
        doc = module.openapi(schema, True)
        self.assertEqual(doc['components']['securitySchemes']['Session']['name'], 'sso_session')
        self.assertEqual(doc['paths']['/api/v1/catalog']['get']['security'], [])
        for path, method, codes in [('/api/v1/orders','post',[202]),('/api/v1/admin/catalog/products','post',[201]),('/api/v1/admin/oauth/clients','post',[201]),('/api/v1/admin/commands/{id}/resume','post',[200,202])]:
            operation = doc['paths'][path][method]
            self.assertEqual(operation['x-success-statuses'], codes)
            self.assertEqual(sorted(int(code) for code in operation['responses'] if int(code)<300), codes)
        for operation in schema['x-endpoints']:
            self.assertTrue(operation['x-success-statuses'])
            self.assertTrue(all(code in (200,201,202,204) for code in operation['x-success-statuses']))
        fixture={'$defs':{},'x-endpoints':[{'method':'DELETE','path':'/fixture','response':'Unused','x-success-statuses':[204]}]}
        self.assertEqual(module.openapi(fixture,True)['paths']['/fixture']['delete']['responses']['204'],{'description':'Success'})
        command=doc['paths']['/api/v1/admin/commands/{id}']['get']
        self.assertEqual(command['x-permission-policy'],'original_actor+original_action_permission+fresh_mfa')
        self.assertNotIn('x-permission',command)
        self.assertEqual(doc['paths']['/api/v1/admin/templates/{event_type}/preview']['post']['x-permission'],'notifications:templates:write')

    def test_new_strict_boundary_models(self):
        import json
        schema = json.loads((ROOT/'schema/internal-http-v2.json').read_text())
        prepare = schema['$defs']['PrepareActionRequest']
        self.assertEqual(prepare['properties']['subscription_id'], {'type':'string','format':'uuid'})
        self.assertNotIn('subscription_id',prepare['required'])
        outcome = schema['$defs']['SubscriptionActionOutcome']
        self.assertEqual(set(outcome['required']),set(outcome['properties']))
        self.assertEqual(outcome['properties']['state']['enum'],['APPLIED','REJECTED'])
        self.assertEqual(schema['$defs']['TemplatePreviewRequest']['required'],['locales'])
        self.assertIn('variables',schema['$defs']['TemplateRevision']['required'])
        for name, rule in schema['$defs'].items():
            module = importlib.util.module_from_spec(SPEC)
            SPEC.loader.exec_module(module)
            module.check_rule(rule)
        endpoint=next(e for e in schema['x-endpoints'] if e['path']=='/internal/v2/subscription-action-outcomes/{id}')
        self.assertEqual(endpoint['callers'],['billing'])

if __name__ == '__main__':
    unittest.main()
