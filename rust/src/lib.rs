pub mod businessv1;
mod businessv1_types;
pub mod internalhttpv2;
mod internalhttpv2_types;
pub mod eventsv2;
mod eventsv2_types;
mod eventsv2_registry;
mod validation_v2;
pub mod http;
mod http_validation;
mod types;
pub use types::*;

use base64::{engine::general_purpose::STANDARD, Engine};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use serde_json::Value;
use uuid::Uuid;

pub const IDENTITY_TOPIC: &str = "identity.events.v1";
pub const COMMAND_TOPIC: &str = "notification.commands.v1";
pub const PARTITIONS: i32 = 6;

#[derive(Debug, Clone)]
pub struct Envelope {
    pub schema_version: u32,
    pub event_id: Uuid,
    pub event_type: String,
    pub aggregate_id: Uuid,
    pub occurred_at: DateTime<Utc>,
    pub traceparent: Option<String>,
    pub payload: Option<Value>,
    pub encrypted_payload: Option<String>,
}

#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct WireEnvelope {
    schema_version: u32,
    event_id: Uuid,
    event_type: String,
    aggregate_id: Uuid,
    occurred_at: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    traceparent: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    payload: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    encrypted_payload: Option<String>,
}

pub fn sensitive(event_type: &str) -> bool {
    event_type.starts_with("notification.")
}
pub fn topic(event_type: &str) -> Result<&'static str, String> {
    types::payload_definition(event_type).ok_or("unknown event type")?;
    Ok(if sensitive(event_type) {
        COMMAND_TOPIC
    } else {
        IDENTITY_TOPIC
    })
}

pub fn valid_traceparent(value: &str) -> bool {
    let bytes = value.as_bytes();
    bytes.len() == 55
        && &bytes[..3] == b"00-"
        && bytes[35] == b'-'
        && bytes[52] == b'-'
        && bytes.iter().enumerate().all(|(i, b)| {
            i == 2 || i == 35 || i == 52 || b.is_ascii_digit() || (b'a'..=b'f').contains(b)
        })
        && bytes[3..35].iter().any(|b| *b != b'0')
        && bytes[36..52].iter().any(|b| *b != b'0')
}

impl Envelope {
    pub fn validate(&self) -> Result<(), String> {
        if self.schema_version != 1 {
            return Err("unsupported schema version".into());
        }
        topic(&self.event_type)?;
        if self
            .traceparent
            .as_deref()
            .is_some_and(|trace| !valid_traceparent(trace))
        {
            return Err("invalid traceparent".into());
        }
        if sensitive(&self.event_type) {
            if self.payload.is_some() {
                return Err("sensitive plaintext prohibited".into());
            }
            let encoded = self
                .encrypted_payload
                .as_deref()
                .ok_or("encrypted payload required")?;
            let bytes = STANDARD
                .decode(encoded)
                .map_err(|_| "invalid encrypted payload base64")?;
            if bytes.len() < 28 {
                return Err("encrypted payload requires nonce and authentication tag".into());
            }
        } else {
            if self.encrypted_payload.is_some() {
                return Err("public encrypted payload prohibited".into());
            }
            validate_payload(
                &self.event_type,
                self.payload.as_ref().ok_or("public payload required")?,
            )?;
        }
        Ok(())
    }
}

impl Serialize for Envelope {
    fn serialize<S: Serializer>(&self, serializer: S) -> Result<S::Ok, S::Error> {
        self.validate().map_err(serde::ser::Error::custom)?;
        use serde::ser::SerializeMap;
        let mut map =
            serializer.serialize_map(Some(6 + usize::from(self.traceparent.is_some())))?;
        map.serialize_entry("schema_version", &self.schema_version)?;
        map.serialize_entry("event_id", &self.event_id)?;
        map.serialize_entry("event_type", &self.event_type)?;
        map.serialize_entry("aggregate_id", &self.aggregate_id)?;
        map.serialize_entry("occurred_at", &self.occurred_at)?;
        if let Some(trace) = &self.traceparent {
            map.serialize_entry("traceparent", trace)?;
        }
        if let Some(payload) = &self.payload {
            map.serialize_entry("payload", payload)?;
        }
        if let Some(encrypted) = &self.encrypted_payload {
            map.serialize_entry("encrypted_payload", encrypted)?;
        }
        map.end()
    }
}

impl<'de> Deserialize<'de> for Envelope {
    fn deserialize<D: Deserializer<'de>>(deserializer: D) -> Result<Self, D::Error> {
        let raw = Value::deserialize(deserializer)?;
        let object = raw
            .as_object()
            .ok_or_else(|| serde::de::Error::custom("envelope must be an object"))?;
        if object.contains_key("payload") == object.contains_key("encrypted_payload")
            || object.get("traceparent").is_some_and(|v| v.is_null())
        {
            return Err(serde::de::Error::custom(
                "exactly one non-null payload representation required",
            ));
        }
        for key in ["event_id", "aggregate_id"] {
            let text = object
                .get(key)
                .and_then(Value::as_str)
                .ok_or_else(|| serde::de::Error::custom("missing UUID metadata"))?;
            if !valid_uuid(text) {
                return Err(serde::de::Error::custom("invalid UUID metadata"));
            }
        }
        let wire: WireEnvelope = serde_json::from_value(raw).map_err(serde::de::Error::custom)?;
        let envelope = Self {
            schema_version: wire.schema_version,
            event_id: wire.event_id,
            event_type: wire.event_type,
            aggregate_id: wire.aggregate_id,
            occurred_at: wire.occurred_at,
            traceparent: wire.traceparent,
            payload: wire.payload,
            encrypted_payload: wire.encrypted_payload,
        };
        envelope.validate().map_err(serde::de::Error::custom)?;
        Ok(envelope)
    }
}

// The schema is the sole source of payload shapes, including decrypted command DTOs.
pub fn validate_payload(event_type: &str, payload: &Value) -> Result<(), String> {
    let name = types::payload_definition(event_type).ok_or("unknown event type")?;
    static SCHEMA: std::sync::LazyLock<Value> = std::sync::LazyLock::new(|| {
        serde_json::from_str(CANONICAL_SCHEMA).expect("generated canonical schema")
    });
    let definition = &SCHEMA["$defs"][name];
    let object = payload.as_object().ok_or("payload must be an object")?;
    for required in definition["required"]
        .as_array()
        .ok_or("invalid schema required fields")?
    {
        if !object.contains_key(required.as_str().ok_or("invalid schema field")?) {
            return Err("missing payload field".into());
        }
    }
    for (key, value) in object {
        let rule = &definition["properties"][key];
        match rule["type"].as_str() {
            Some("boolean") if value.is_boolean() => {}
            Some("string") => {
                let text = value.as_str().ok_or("invalid string payload field")?;
                if text.chars().count() < rule["minLength"].as_u64().unwrap_or(0) as usize {
                    return Err("empty payload field".into());
                }
                match rule["format"].as_str() {
                    Some("uuid") if !valid_uuid(text) => return Err("invalid payload UUID".into()),
                    Some("date-time") if DateTime::parse_from_rfc3339(text).is_err() => {
                        return Err("invalid payload timestamp".into())
                    }
                    _ => {}
                }
            }
            _ => return Err("unexpected or invalid payload field".into()),
        }
    }
    Ok(())
}

fn valid_uuid(text: &str) -> bool {
    text.len() == 36
        && text.bytes().enumerate().all(|(index, byte)| {
            if matches!(index, 8 | 13 | 18 | 23) {
                byte == b'-'
            } else {
                byte.is_ascii_hexdigit()
            }
        })
        && Uuid::parse_str(text).is_ok()
}
