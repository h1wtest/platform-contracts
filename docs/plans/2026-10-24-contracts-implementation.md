# C01 Versioned platform contracts implementation plan

Goal: strict v2 events and internal HTTP plus business-v1 schemas, generated Go/Rust/TypeScript definitions and codecs, public/private OpenAPI and deterministic compatibility inventory without changing immutable v1 schemas.

Architecture: retain v1 packages byte-for-byte; add generated eventsv2/businessv1/internalhttpv2 packages backed by one closed JSON Schema constraint subset, rejecting unsupported keywords during generation. Domain payloads are enumerated, unknown properties forbidden. Shared business DTOs live in businessv1. Secrets occur only in typed decrypted notification command definitions and encrypted envelope bodies, never public payloads.

Constraints: only this isolated repo; existing pinned Go/Rust/Python toolchains/dependencies; no tests/build/lint/format execution in worker; parent executes validation after integration; no commits/publication/cloud.

1. Author behavioral Go and Python regressions covering envelope roundtrip, UUIDv7, unknown/missing/null fields, metadata, producer/topic ownership, encrypted-command isolation, exact money/version constraints and generator unsupported-keyword rejection.
2. Add canonical schema/business-v1.json with closed shared types and domain-specific business schemas/endpoint inventory, internal-http-v2.json for identity/profile/commerce/notification owner calls, events-v2.json for enumerated payloads and topic metadata.
3. Implement strict recursive Go schema codec supporting only generated constraints; generate DTO MarshalJSON/UnmarshalJSON/Validate methods and event envelope exact public runtime API. Payload validators retain all nested constraints and duplicate-key rejection.
4. Generate Rust modules and TypeScript definitions/runtime validators from the identical rules; export new Rust modules alongside unchanged v1 API. Emit public/private OpenAPI 3.1 and SHA256 release manifest.
5. Integrate generation, Docker, Makefile and CI drift/compatibility inputs. Document v2 topics, ownership, commands, migration and validation commands.

Parent commands: make generate; go test ./...; cargo test --manifest-path rust/Cargo.toml; uv run --no-project python -m unittest discover -s tests; npx tsc --strict --target es2022 --module esnext --noEmit gen/typescript/*.ts. Generation drift and immutable-v1 compatibility checked against pinned base. No command is represented as passing until parent executes it.
