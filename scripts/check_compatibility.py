#!/usr/bin/env python3
"""Reject in-place v1 schema changes except explicitly descriptive annotations."""
import json
import sys
from pathlib import Path

ANNOTATIONS = {'title', 'description', '$comment', 'examples'}
SCHEMA_MAPS = {'$defs', 'definitions', 'properties', 'patternProperties', 'dependentSchemas'}
SCHEMA_LISTS = {'allOf', 'anyOf', 'oneOf', 'prefixItems'}
SCHEMA_CHILDREN = {'not', 'if', 'then', 'else', 'items', 'contains', 'additionalProperties', 'unevaluatedProperties', 'propertyNames', 'unevaluatedItems', 'contentSchema'}


def semantic_schema(schema):
    if not isinstance(schema, dict):
        return schema
    result = {}
    for key, value in schema.items():
        if key in ANNOTATIONS:
            continue
        if key in SCHEMA_MAPS and isinstance(value, dict):
            result[key] = {name: semantic_schema(child) for name, child in value.items()}
        elif key in SCHEMA_LISTS and isinstance(value, list):
            result[key] = [semantic_schema(child) for child in value]
        elif key in SCHEMA_CHILDREN:
            result[key] = semantic_schema(value)
        else:
            # Unknown keywords and application extensions are immutable, not ignored.
            # Do not strip annotation-looking keys from const/enum data or field names.
            result[key] = value
    return result


def breaking_changes(previous, current):
    if json.dumps(semantic_schema(previous), sort_keys=True) != json.dumps(semantic_schema(current), sort_keys=True):
        return ['v1 wire contract changed; publish a new schema/topic version']
    return []


def main():
    if len(sys.argv) != 3:
        raise SystemExit('usage: check_compatibility.py BASE_SCHEMA CURRENT_SCHEMA')
    previous, current = (json.loads(Path(path).read_text()) for path in sys.argv[1:])
    errors = breaking_changes(previous, current)
    if errors:
        raise SystemExit('\n'.join(errors))


if __name__ == '__main__':
    main()
