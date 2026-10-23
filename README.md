# platform-contracts

Canonical contracts are `schema/events-v1.json` and
`schema/internal-http-v1.json` (JSON Schema 2020-12). Kafka and HTTP integration
use JSON, not protobuf/gRPC. Go imports are `platform-contracts/gen/go/events`
and `platform-contracts/gen/go/internalhttp`. The Rust crate is `platform-contracts`
at `rust/`, imported as `platform_contracts`, with public internal HTTP module
`platform_contracts::http`.

## Reproducible generation and compatibility

Python 3.10+ (stdlib only), uv and the supported Go toolchain generate both languages from `$defs`. Python Make recipes use `uv run --no-project python` (`PYTHON` may explicitly select another local interpreter):

```sh
make generate
git diff --exit-code -- gen/go/events/types.go rust/src/types.rs gen/go/internalhttp/types.go rust/src/http.rs
make test
make compatibility BASE_SCHEMA=/path/to/previous/events-v1.json
make compatibility BASE_SCHEMA=/path/to/previous/internal-http-v1.json CURRENT_SCHEMA=schema/internal-http-v1.json
```

`generate` rewrites Go/Rust DTOs and formats generated output with Go `gofmt` and Rust 1.99.0 `rustfmt` (install its component first); `test` runs real Go, Rust and Python suites (all three toolchains plus make required). `compatibility` is read-only and requires a nonblank `BASE_SCHEMA`; omitted `CURRENT_SCHEMA` selects `schema/events-v1.json`. Run it separately for both canonical schemas against the corresponding approved baseline. Schema paths preserve spaces/quotes/shell metacharacters; a literal `$` in a command-line Make variable must be written `$$`. Missing baselines fail before Python invocation; no implicit downloaded or same-file baseline is substituted.

The supported CI toolchains are Go **1.26.10** and Rust **1.99.0**. Generation runs
the Go toolchain's `gofmt` and explicitly pinned Rust 1.99.0 `rustfmt` for canonical
DTO output without network access once those tools are installed. Runtime validators
read the generated schema definition, so field validation and DTOs share one source.
Behavior tests exercise real serialized envelopes and strict secret separation.
CI scans complete Git history with redacted output before installing dependencies,
regenerates, checks both language builds/tests with the pinned Go and Rust toolchains,
audits dependencies and builds a commit-tagged, non-executable schema artifact without
publishing. Cargo commands require the checked-in lockfile. Push, PR and manual dispatch
events use the same gate; initial publication uses push/PR because GitHub only enables
manual dispatch after the workflow exists on the default branch.
The scratch image contains both canonical schemas, Go package sources (including
validators), all Rust library sources and both language manifests/lockfile. It has
no executable, toolchain or build outputs; the Go sources also include contract
regression tests. Runtime validation dependencies are not omitted from the artifact.
The local integration controller executed the earlier contract generation and Go/Rust/Python checks successfully. Strict HTTP codecs and predecessor compatibility gates were subsequently updated; rerun generation and all contract/consumer checks and recompute the canonical source digest before publication. Earlier results do not verify these later changes. Remote GitHub CI has not been run or published in this remediation.

v1 wire shapes are immutable: strict consumers reject unknown fields. Rename,
removal, optional additions, payload-rule changes, or new event kinds require
a new schema/topic version and coordinated consumer migration. The compatibility
gate compares the complete schema, excluding only `title`, `description`,
`$comment`, and `examples` annotations at schema locations. Property names and
enum/const data are never treated as annotations. PRs use their target SHA,
pushes use the trusted pre-push SHA, and manual dispatch uses the selected
commit's first parent. A new branch push also uses its first parent. Missing
baseline commits fail closed; a verified predecessor without a schema or a
repository root commit is explicit initial publication.

Service CI consuming this repository must select an approved immutable 40-character commit SHA, not default `main`, a branch or a moving tag. Identity and user-notification use repository variable `PLATFORM_CONTRACTS_REVISION` or dispatch input `contracts_revision`, validate it before checkout and record both service/contracts revisions. Publish and verify contracts first; after its reviewed merge, set both repository variables to the actual contracts `main` SHA before pushing consumer branches. A local uncommitted worktree cannot be substituted for a published commit. These compatibility checks do not certify OIDC or imply production readiness.

## Internal HTTP boundaries

`internal-http-v1.json` declares the identity introspection/admin APIs, profile
notification settings, notification delivery listing/replay, and owner-scoped
quarantine listing/replay. Its `x-endpoints` inventory binds methods and paths to
request/query/response definitions. Producers and consumers import generated
DTOs instead of duplicating service-local wire structs or response maps.
Actor authentication is forwarded via headers and always revalidated by identity;
no internal body can select an arbitrary actor ID. All routes require service
authentication and sensitive connections require TLS.

Generated names include `IntrospectRequest`, `IntrospectResponse`,
`SearchUsersRequest`, `BanRequest`, `BanResponse`, `UserSummary`, `UsersResponse`,
`NotificationSettings`, `Delivery`, `DeliveriesResponse`, `QuarantineRecord`,
`QuarantineResponse`, `ReplayRequest`, `ReplayResponse`, and `ErrorResponse`.
Optional request fields are pointers/`Option` and omitted when absent. Required
nullable response fields remain present as JSON null. Array responses are raw
arrays, except identity users wrapped as `{users:[...]}`. `corrected_event` is
an optional canonical event envelope; replay never accepts a caller-selected
actor. Delivery/quarantine lists expose metadata only, not decrypted bodies.
Both schemas and all four generated outputs are covered by generation and
compatibility CI. Ordinary Go `json.Unmarshal`/`json.Marshal` and Rust serde
enforce HTTP field presence, nullability, unknown-field rejection, UUID/time
formats, string bounds, enums and integer minima. Required nullable fields must
be present; optional non-nullable fields reject explicit null. The generator
rejects unsupported constraints rather than silently generating weaker codecs.
Live handlers retain semantic authorization and state-transition validation.

## Envelope

Required metadata: `schema_version: 1`, UUID `event_id`, `event_type`, UUID
`aggregate_id`, RFC3339 `occurred_at`. Optional `traceparent` is W3C version 00
with nonzero trace/span IDs; it is propagated unchanged to Kafka headers and
delivery work. Exactly one of `payload` or `encrypted_payload` is present:

* `identity.events.v1`: public `identity.*` events carry a strict typed `payload`.
* `notification.commands.v1`: verification/reset commands carry only
  `encrypted_payload`, standard base64 of **12-byte random nonce || AES-256-GCM
  ciphertext || 16-byte authentication tag**. `DELIVERY_ENCRYPTION_KEY` is standard
  base64 of 32 bytes, available only to identity and notification delivery workers;
  it is distinct from identity-only `MFA_ENCRYPTION_KEY`. AAD is canonical lower-case
  hyphenated `event_id` UTF-8. Retries retain the stable ID and original ciphertext.

Sensitive plaintext conforms to `EmailVerification` or `PasswordReset` in
`$defs`; consumers authenticate/decrypt first, then validate it. Never put
tokens, codes, ciphertext-decrypted bodies, or full command envelopes into audit
or delivery history. Both topics have exactly **6 partitions**. Commands retain
at most 24 hours in the broker; broker credentials must separate public-event
consumers from command consumers.

Go `json.Unmarshal` and Rust serde decode validate envelopes; encoding also
validates. Go exports `ValidatePayload`, `Sensitive`, `Topic`, and
`Envelope.DecodePayload`; Rust exports `validate_payload`, `sensitive`, `topic`,
`Envelope`, and generated payload types. Payload fields/types are listed in the
canonical schema. All payloads deny unknown fields, preventing accidental
secret additions to public events.

## Producer durability

Identity `events::enqueue(tx, crypto, event_type, aggregate_id, payload,
traceparent)` validates the shared contract and encrypts sensitive payloads
**before SQL insertion** in the same transaction as authoritative state.
Per-aggregate transaction advisory locks serialize insertion/commit order.
Migration `0005_leased_outbox.sql` refuses pending legacy outbox intents: drain
the old producer/relay before cutover. Processed legacy history remains untouched;
no incompatible intent is silently discarded or published. Fresh deployments
need no legacy action.

The relay claims the earliest pending event for each aggregate with a persisted
60-second UUID-fenced lease and `SKIP LOCKED`. Other aggregates can progress;
later events for a leased/retrying aggregate cannot overtake its head. Expired
claims are recoverable. Acknowledgement and retry updates require the live
lease token. Total Kafka work is bounded to 10 seconds per attempt. Persisted
equal-jitter exponential backoff caps at 300 seconds; authoritative events
remain pending rather than being dropped after an arbitrary retry count.
Processed SQL rows are removed after 24 hours (hourly maintenance).

Identity's application outbox calls the broker-independent
`domain::events::Publisher` port. Its `adapters::kafka::KafkaPublisher` implementation
owns rskafka transport, topic verification, record encoding, and broker acceptance.
`KAFKA_TLS_CA_FILE` must contain the broker CA; `KAFKA_SASL_USERNAME` and
`KAFKA_SASL_PASSWORD` select the service-specific SCRAM-SHA256 principal. TLS
certificate/hostname verification is mandatory, and missing credentials or trust
roots never downgrade to anonymous/plaintext transport. Advertised broker DNS
names must match certificate SANs. Identity produces both topics; public-only
consumer principals must not receive command-topic access.

The Kafka record key is canonical `aggregate_id` UUID text, partitioned by
32-bit FNV-1a over those UTF-8 bytes modulo 6. This is stable across process
restarts, unlike randomized language hashers. Changing partition count or
partitioner requires a coordinated topic version. A stale worker/network may
produce an ambiguous duplicate after lease loss; exactly-once Kafka publication
is not promised. Consumers persist unique inbox `event_id` with local changes
and acknowledge only after commit. Their inbox prevents duplicate effects.
Identity readiness depends on SQL, not Kafka; SQL backlog metrics retain
`is_processed`, `created_at`, and `retry_count` columns.

SQL lease/secret tests live in `identity-service/tests/outbox_security.rs` and were exercised by the controller against real isolated PostgreSQL and an authenticated TLS/SCRAM Kafka broker. They are ignored by default because external dependencies are required. Run the SQL tests serially against a fresh **separate** outbox database; run the relay case against another fresh database and real six-partition topics. Do not combine these with auth tests in one shared pending-outbox database. Exact commands and the purpose-separated signing/encryption environment are in identity-service's README. Controller evidence also includes 36 selected identity SQL regressions, rekey, actual relay, six signing/rotation checks and strict HTTPS/browser OIDC integration; those are cross-service acceptance evidence, not standalone contract tests. No broker mock or successful fake delivery is counted as acceptance.