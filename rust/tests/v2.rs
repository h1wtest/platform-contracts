use platform_contracts::{businessv1,eventsv2};
use serde_json::{json,Value};
fn registered()->Value{json!({"schema_version":2,"event_id":"019a13cf-0000-7000-8000-000000000001","event_type":"identity.user_registered","producer":"identity","aggregate_type":"user","aggregate_id":"019a13cf-0000-7000-8000-000000000002","aggregate_version":1,"occurred_at":"2026-10-24T00:00:00Z","payload":{"user_id":"019a13cf-0000-7000-8000-000000000002","email":"user@example.test","locale":"en","registration_id":"019a13cf-0000-7000-8000-000000000003","referral_claim_id":null}})}
#[test]fn envelope_roundtrip_and_closed_payload(){let value=registered();let event:eventsv2::Envelope=serde_json::from_value(value.clone()).unwrap();assert_eq!(event.topic().unwrap(),"identity.events.v2");assert_eq!(serde_json::to_value(event).unwrap(),value);let mut invalid=value;invalid["payload"]["unknown"]=json!(true);assert!(serde_json::from_value::<eventsv2::Envelope>(invalid).is_err());}
#[test]fn plaintext_secret_is_rejected(){let mut value=registered();value["event_type"]=json!("notification.password_reset");value["aggregate_type"]=json!("delivery_command");value["payload"]=json!({"user_id":"019a13cf-0000-7000-8000-000000000002","email":"u@example.test","token":"secret","expires_at":"2026-10-24T00:00:00Z","locale":"en"});assert!(serde_json::from_value::<eventsv2::Envelope>(value).is_err());}
#[test]fn duplicate_fields_are_not_hidden(){let text=serde_json::to_string(&registered()).unwrap().replace("\"schema_version\":2","\"schema_version\":2,\"schema_version\":2");assert!(serde_json::from_str::<eventsv2::Envelope>(&text).is_err());}
#[test]fn exact_money_and_nullable_patch(){assert!(businessv1::validate("ManualSettlement",&json!({"amount":"1.000000","network":"tron","reference":"proof"})).is_ok());for amount in ["1.0","01.000000","9000000.000001"]{assert!(businessv1::validate("ManualSettlement",&json!({"amount":amount,"network":"tron","reference":"proof"})).is_err());}let patch:businessv1::ProfileChanges=serde_json::from_str("{\"display_name\":null}").unwrap();assert_eq!(patch.display_name,Some(None));assert_eq!(serde_json::to_value(patch).unwrap(),json!({"display_name":null}));}
#[test]
fn serialized_envelope_rejects_wrong_aggregate_owner() {
 let mut raw=registered();raw["aggregate_id"]=json!("019a13cf-0000-7000-8000-000000000099");
 let bytes=serde_json::to_vec(&raw).unwrap();
 assert!(serde_json::from_slice::<eventsv2::Envelope>(&bytes).is_err());
}

#[test]
fn bounded_root_arrays_validate_serde_and_construction() {
 let user=json!({"id":"019a13cf-0000-7000-8000-000000000002","email":"u@example.test","is_verified":true,"is_banned":false,"ban_reason":null,"created_at":"2026-10-24T00:00:00Z","mfa_enabled":false,"roles":[]});
 let valid=Value::Array(vec![user.clone();1000]);
 let users:businessv1::UserSummaries=serde_json::from_value(valid).unwrap();
 assert_eq!(users.as_slice().len(),1000);assert!(serde_json::to_vec(&users).is_ok());
 let over=Value::Array(vec![user;1001]);assert!(serde_json::from_value::<businessv1::UserSummaries>(over).is_err());
 let one:businessv1::UserSummary=serde_json::from_value(users.as_slice().first().map(|v|serde_json::to_value(v).unwrap()).unwrap()).unwrap();
 assert!(businessv1::UserSummaries::new(vec![one;1001]).is_err());
}
#[test]
fn nullable_locale_enum_allows_null_but_not_unknown() {
 assert!(serde_json::from_str::<businessv1::ProfileChanges>("{\"website_locale\":null}").is_ok());
 assert!(serde_json::from_str::<businessv1::ProfileChanges>("{\"website_locale\":\"xx\"}").is_err());
}
#[test]
fn optional_uuid_metadata_is_checked_when_serializing() {
 let mut envelope:eventsv2::Envelope=serde_json::from_value(registered()).unwrap();
 envelope.correlation_id=Some(uuid::Uuid::nil());assert!(serde_json::to_vec(&envelope).is_err());
 envelope.correlation_id=None;envelope.causation_id=Some(uuid::Uuid::nil());assert!(serde_json::to_vec(&envelope).is_err());
}
#[test]
fn event_version_supports_full_int64_not_rounded_numbers() {
 for version in [9_007_199_254_740_992_i64,i64::MAX] {
  let mut value=registered();value["aggregate_version"]=json!(version);
  let envelope:eventsv2::Envelope=serde_json::from_value(value).unwrap();
  let bytes=serde_json::to_vec(&envelope).unwrap();let result:eventsv2::Envelope=serde_json::from_slice(&bytes).unwrap();assert_eq!(result.aggregate_version,version);
 }
 let bytes=serde_json::to_string(&registered()).unwrap().replace("\"aggregate_version\":1","\"aggregate_version\":9223372036854775808");
 assert!(serde_json::from_str::<eventsv2::Envelope>(&bytes).is_err());
}

#[test]
fn serialized_integer_subset_rejects_fraction_and_exponent_tokens() {
 for token in ["1.0000000000000001","9007199254740991.1","1e0","1E+0","0.0","-0.0","1e-999"] {
  let raw=format!("{{\"expected_version\":{},\"idempotency_key\":\"key\"}}",token);
  assert!(serde_json::from_str::<businessv1::VersionCommand>(&raw).is_err(),"accepted {}",token);
 }
 for token in ["-9223372036854775808","9223372036854775807","9007199254740991","-9007199254740991"] {
  let value:Value=serde_json::from_str(token).unwrap();
  assert_eq!(serde_json::to_string(&value).unwrap(),token);
 }
}

#[test]
fn preview_and_immutable_outcome_consume_strict_producer_bytes() {
 use platform_contracts::internalhttpv2;
 let preview=r#"{"locales":{"en":{"subject":"Preview","body":"{{reset_url}}"},"ru":{"subject":"Просмотр","body":"{{reset_url}}"}}}"#;
 assert!(serde_json::from_str::<businessv1::TemplatePreviewRequest>(preview).is_ok());
 assert!(serde_json::from_str::<internalhttpv2::TemplatePreviewResponse>(preview).is_ok());
 for raw in [r#"{"locales":null}"#,r#"{"locales":{"en":{"subject":"x","body":"x"}}}"#] {assert!(serde_json::from_str::<businessv1::TemplatePreviewRequest>(raw).is_err());}
 let outcome=r#"{"action_id":"019a13cf-0000-7000-8000-000000000001","order_id":"019a13cf-0000-7000-8000-000000000002","user_id":"019a13cf-0000-7000-8000-000000000003","state":"REJECTED","subscription_id":null,"subscription_version":null,"period_start":null,"period_end":null,"reason":"conflict"}"#;
 let value:internalhttpv2::SubscriptionActionOutcome=serde_json::from_str(outcome).unwrap();
 assert_eq!(serde_json::to_value(value).unwrap()["subscription_version"],Value::Null);
 assert!(serde_json::from_str::<internalhttpv2::SubscriptionActionOutcome>(&outcome.replace("REJECTED","PENDING")).is_err());
 assert!(serde_json::from_str::<internalhttpv2::SubscriptionActionOutcome>(&outcome.replace(",\"reason\":\"conflict\"","")).is_err());
}
