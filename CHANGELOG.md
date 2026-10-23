# Changelog

## Unreleased — canonical platform contracts (2026-10-23)

- Replace obsolete protobuf/gRPC contracts with versioned JSON Schema 2020-12 event and internal HTTP contracts consumed by Rust and Go implementations.
- Generate strict Go/Rust DTOs from canonical definitions; validate envelope identity, version, payload rules and sensitive/public separation. Define six-partition stable aggregate keys and AES-GCM command framing/AAD.
- Cover introspection, administrative user commands, notification settings, delivery/replay, quarantine/replay and structured errors with shared internal HTTP DTOs. Document server-authenticated TLS plus caller credentials without claiming mTLS.
- Add deterministic generation, conservative v1 compatibility checks, behavioral serialization/encryption regressions, pinned Go 1.26.10/Rust 1.99.0 CI and dependency/secret scanning.
- Added manual workflow dispatch, explicit pinned Go installation, locked Cargo dependency resolution and full-history redacted secret checks before dependencies/build; actual remote results remain separate acceptance evidence.
- Standardized Python Make entrypoints on uv; require an explicit nonblank compatibility baseline before invocation and preserve schema-path argument boundaries. Added Make behavior regressions without weakening compatibility checks.
- Enforced strict canonical HTTP decoding/encoding through ordinary Go JSON and Rust serde APIs, including required nullable presence, optional non-nullable values, UUIDs, enums, bounds and unknown fields. Added malformed introspection and cross-language HTTP regressions; handler authorization remains separate.
- Compatibility now compares every semantic schema keyword and runs for PR, push and manual dispatch using trusted event-specific predecessors; added real Git history gate regressions. Only explicitly descriptive schema annotations may change in place.
- Pinned generated Rust formatting to Rust 1.99.0 rustfmt and install that component in CI; the scratch artifact now includes complete Go/Rust contract library sources, validators and manifests, without binaries or build outputs.
- Earlier local controller generation and Go/Rust/Python checks passed before the later strict HTTP codec and compatibility-gate fixes. Recompute source provenance and rerun contract/consumer checks before publication; no earlier digest or pass is claimed for these later changes.
- Consumer CI requires an approved immutable contracts commit SHA with source provenance; the local content digest is not a remote revision. Remote CI, OIDC certification and production acceptance are not claimed.
