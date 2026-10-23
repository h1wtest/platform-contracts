use crate::eventsv2_registry::event_rule;
pub use crate::eventsv2_types::*;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use serde_json::Value;
use uuid::Uuid;

pub const IDENTITY_TOPIC: &str = "identity.events.v2";
pub const COMMAND_TOPIC: &str = "notification.commands.v2";
pub const PARTITIONS: i32 = 6;
#[derive(Debug, Clone)]
pub struct Envelope {
    pub schema_version: u32,
    pub event_id: Uuid,
    pub event_type: String,
    pub producer: String,
    pub aggregate_type: String,
    pub aggregate_id: Uuid,
    pub aggregate_version: i64,
    pub occurred_at: DateTime<Utc>,
    pub correlation_id: Option<Uuid>,
    pub causation_id: Option<Uuid>,
    pub traceparent: Option<String>,
    pub payload: Option<Value>,
    pub encrypted_payload: Option<String>,
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
struct Wire {
    schema_version: u32,
    event_id: Uuid,
    event_type: String,
    producer: String,
    aggregate_type: String,
    aggregate_id: Uuid,
    aggregate_version: i64,
    occurred_at: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    correlation_id: Option<Uuid>,
    #[serde(skip_serializing_if = "Option::is_none")]
    causation_id: Option<Uuid>,
    #[serde(skip_serializing_if = "Option::is_none")]
    traceparent: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    payload: Option<Value>,
    #[serde(skip_serializing_if = "Option::is_none")]
    encrypted_payload: Option<String>,
}
pub fn sensitive(t: &str) -> bool {
    event_rule(t).is_some_and(|r| r.4)
}
pub fn topic(t: &str) -> Result<&'static str, String> {
    event_rule(t)
        .map(|r| r.3)
        .ok_or("unknown event type".into())
}
pub fn validate_payload(t: &str, payload: &Value) -> Result<(), String> {
    let r = event_rule(t).ok_or("unknown event type")?;
    crate::eventsv2_types::validate(r.0, payload)
}
impl Envelope {
    pub fn topic(&self) -> Result<&'static str, String> {
        topic(&self.event_type)
    }
    pub fn validate(&self) -> Result<(), String> {
        let r = event_rule(&self.event_type).ok_or("unknown event type")?;
        if self.schema_version != 2
            || self.aggregate_version < 1
            || !crate::validation_v2::format("uuid-v7", &self.event_id.to_string())
            || !crate::validation_v2::format("uuid", &self.aggregate_id.to_string())
        {
            return Err("invalid envelope metadata".into());
        }
        for id in [self.correlation_id, self.causation_id]
            .into_iter()
            .flatten()
        {
            if !crate::validation_v2::format("uuid", &id.to_string()) {
                return Err("invalid optional UUID metadata".into());
            }
        }
        if self.producer != r.1 || self.aggregate_type != r.2 {
            return Err("producer/aggregate mismatch".into());
        }
        if (r.4 || self.aggregate_type == "external_event")
            && (self.aggregate_id != self.event_id || self.aggregate_version != 1)
        {
            return Err("independent event stream requires event ID and version one".into());
        }
        if self
            .traceparent
            .as_ref()
            .is_some_and(|t| !crate::validation_v2::format("traceparent", t))
        {
            return Err("invalid traceparent".into());
        }
        if r.4 {
            if self.payload.is_some()
                || !self
                    .encrypted_payload
                    .as_ref()
                    .is_some_and(|p| crate::validation_v2::format("encrypted", p))
            {
                return Err("purpose-encrypted command required".into());
            }
        } else {
            if self.encrypted_payload.is_some() {
                return Err("encrypted public event prohibited".into());
            }
            validate_payload(
                &self.event_type,
                self.payload.as_ref().ok_or("missing public payload")?,
            )?;
            if !r.5.is_empty() {
                let owner = self
                    .payload
                    .as_ref()
                    .and_then(|payload| payload.get(r.5))
                    .and_then(Value::as_str);
                let aggregate = self.aggregate_id.to_string();
                if owner != Some(aggregate.as_str()) {
                    return Err("payload aggregate identity mismatch".into());
                }
            }
        }
        Ok(())
    }
}
impl Serialize for Envelope {
    fn serialize<S: serde::Serializer>(&self, s: S) -> Result<S::Ok, S::Error> {
        self.validate().map_err(serde::ser::Error::custom)?;
        Wire {
            schema_version: self.schema_version,
            event_id: self.event_id,
            event_type: self.event_type.clone(),
            producer: self.producer.clone(),
            aggregate_type: self.aggregate_type.clone(),
            aggregate_id: self.aggregate_id,
            aggregate_version: self.aggregate_version,
            occurred_at: self.occurred_at,
            correlation_id: self.correlation_id,
            causation_id: self.causation_id,
            traceparent: self.traceparent.clone(),
            payload: self.payload.clone(),
            encrypted_payload: self.encrypted_payload.clone(),
        }
        .serialize(s)
    }
}
impl<'de> Deserialize<'de> for Envelope {
    fn deserialize<D: serde::Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        let crate::validation_v2::UniqueValue(raw) =
            crate::validation_v2::UniqueValue::deserialize(d)?;
        let object = raw
            .as_object()
            .ok_or_else(|| serde::de::Error::custom("envelope object required"))?;
        for name in ["event_id", "aggregate_id", "correlation_id", "causation_id"] {
            if name == "event_id" || name == "aggregate_id" || object.contains_key(name) {
                let text = object
                    .get(name)
                    .and_then(Value::as_str)
                    .ok_or_else(|| serde::de::Error::custom("non-null UUID required"))?;
                if !crate::validation_v2::format(
                    if name == "event_id" {
                        "uuid-v7"
                    } else {
                        "uuid"
                    },
                    text,
                ) {
                    return Err(serde::de::Error::custom("invalid UUID"));
                }
            }
        }
        if object.contains_key("payload") == object.contains_key("encrypted_payload")
            || object.get("payload").is_some_and(Value::is_null)
            || object.get("encrypted_payload").is_some_and(Value::is_null)
            || object.get("traceparent").is_some_and(Value::is_null)
        {
            return Err(serde::de::Error::custom(
                "exactly one non-null body required",
            ));
        }
        let timestamp = object
            .get("occurred_at")
            .and_then(Value::as_str)
            .ok_or_else(|| serde::de::Error::custom("timestamp required"))?;
        if !crate::validation_v2::format("date-time", timestamp) {
            return Err(serde::de::Error::custom("UTC timestamp required"));
        }
        let w: Wire = serde_json::from_value(raw).map_err(serde::de::Error::custom)?;
        let e = Self {
            schema_version: w.schema_version,
            event_id: w.event_id,
            event_type: w.event_type,
            producer: w.producer,
            aggregate_type: w.aggregate_type,
            aggregate_id: w.aggregate_id,
            aggregate_version: w.aggregate_version,
            occurred_at: w.occurred_at,
            correlation_id: w.correlation_id,
            causation_id: w.causation_id,
            traceparent: w.traceparent,
            payload: w.payload,
            encrypted_payload: w.encrypted_payload,
        };
        e.validate().map_err(serde::de::Error::custom)?;
        Ok(e)
    }
}
