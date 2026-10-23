use platform_contracts::{validate_payload, Envelope};
use serde_json::json;

fn public() -> serde_json::Value {
    json!({"schema_version":1,"event_id":"00000000-0000-4000-8000-000000000001","aggregate_id":"00000000-0000-4000-8000-000000000002","event_type":"identity.user_registered","occurred_at":"2026-10-23T00:00:00Z","payload":{"user_id":"00000000-0000-4000-8000-000000000002","email":"person@example.test"}})
}

#[test]
fn public_contract_roundtrips_without_changing_stable_id() {
    let value = public();
    let event: Envelope = serde_json::from_value(value.clone()).unwrap();
    assert_eq!(serde_json::to_value(event).unwrap(), value);
}

#[test]
fn rejects_both_payload_fields_unknown_versions_and_public_secrets() {
    let mut both = public();
    both["encrypted_payload"] = json!("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=");
    assert!(serde_json::from_value::<Envelope>(both).is_err());
    let mut version = public();
    version["schema_version"] = json!(2);
    assert!(serde_json::from_value::<Envelope>(version).is_err());
    let mut leak = public();
    leak["payload"]["code"] = json!("secret");
    assert!(serde_json::from_value::<Envelope>(leak).is_err());
}

#[test]
fn commands_require_encryption_and_decrypted_payload_validation() {
    let mut command = public();
    command["event_type"] = json!("notification.email_verification");
    assert!(serde_json::from_value::<Envelope>(command.clone()).is_err());
    command.as_object_mut().unwrap().remove("payload");
    command["encrypted_payload"] = json!("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=");
    assert!(serde_json::from_value::<Envelope>(command).is_ok());
    assert!(validate_payload("notification.email_verification", &json!({"user_id":"00000000-0000-4000-8000-000000000002","email":"a@example.test","code":"123456","expires_at":"2026-10-23T01:00:00Z"})).is_ok());
    assert!(validate_payload(
        "notification.email_verification",
        &json!({"user_id":"00000000-0000-4000-8000-000000000002","email":"a@example.test"})
    )
    .is_err());
}
