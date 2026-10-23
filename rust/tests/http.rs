use platform_contracts::http::*;
use serde_json::{json, Value};

#[test]
fn introspection_requires_complete_valid_actor() {
    assert!(serde_json::from_value::<IntrospectResponse>(
        json!({"active":true,"user_id":"not-a-uuid"})
    )
    .is_err());
    let valid = json!({"active":true,"user_id":"00000000-0000-4000-8000-000000000001","email":"a@example.test","session_id":"00000000-0000-4000-8000-000000000002","roles":[],"permissions":[],"mfa_verified":false});
    let actor: IntrospectResponse = serde_json::from_value(valid.clone()).unwrap();
    assert_eq!(serde_json::to_value(actor).unwrap(), valid);
    for key in [
        "active",
        "user_id",
        "email",
        "session_id",
        "roles",
        "permissions",
        "mfa_verified",
    ] {
        let mut missing = valid.clone();
        missing.as_object_mut().unwrap().remove(key);
        assert!(
            serde_json::from_value::<IntrospectResponse>(missing).is_err(),
            "missing {key}"
        );
        let mut null = valid.clone();
        null[key] = Value::Null;
        assert!(
            serde_json::from_value::<IntrospectResponse>(null).is_err(),
            "null {key}"
        );
    }
}

#[test]
fn optional_fields_are_not_nullable_and_required_nullable_fields_cannot_disappear() {
    assert!(serde_json::from_value::<IntrospectRequest>(json!({"require_mfa":null})).is_err());
    assert!(serde_json::from_value::<BanRequest>(json!({"reason":null})).is_err());
    assert!(serde_json::from_value::<BanRequest>(json!({"reason":"a".repeat(501)})).is_err());
    assert!(serde_json::from_value::<BanRequest>(json!({})).is_ok());
    assert!(serde_json::from_value::<ReplayRequest>(json!({"corrected_event":null})).is_err());
    assert!(serde_json::from_value::<ReplayRequest>(json!({"corrected_event":{}})).is_err());
    let valid = json!({"email":"a","telegram_chat_id":null,"email_login_alerts":true,"email_security_alerts":true,"telegram_login_alerts":false,"telegram_security_alerts":false});
    assert!(serde_json::from_value::<NotificationSettings>(valid.clone()).is_ok());
    let mut missing = valid;
    missing.as_object_mut().unwrap().remove("telegram_chat_id");
    assert!(serde_json::from_value::<NotificationSettings>(missing).is_err());
    assert!(serde_json::from_value::<EmptyRequest>(json!({"actor_id":"injected"})).is_err());
}

#[test]
fn delivery_and_replay_validate_enum_and_minimum_constraints() {
    assert!(serde_json::from_value::<ReplayResponse>(json!({"status":"other"})).is_err());
    let valid = json!({"id":"00000000-0000-4000-8000-000000000001","user_id":"00000000-0000-4000-8000-000000000002","channel":"email","status":"pending","attempts":0,"next_attempt_at":"2026-10-23T00:00:00Z","last_error":null,"message_id":"x","created_at":"2026-10-23T00:00:00Z","delivered_at":null,"ambiguous":false});
    assert!(serde_json::from_value::<Delivery>(valid.clone()).is_ok());
    for (key, value) in [
        ("channel", json!("sms")),
        ("status", json!("other")),
        ("attempts", json!(-1)),
    ] {
        let mut invalid = valid.clone();
        invalid[key] = value;
        assert!(
            serde_json::from_value::<Delivery>(invalid).is_err(),
            "invalid {key}"
        );
    }
    let mut missing = valid;
    missing.as_object_mut().unwrap().remove("last_error");
    assert!(serde_json::from_value::<Delivery>(missing).is_err());
}
