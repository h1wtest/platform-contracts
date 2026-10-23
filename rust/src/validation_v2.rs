use base64::{engine::general_purpose::STANDARD, Engine};
use serde_json::Value;

pub fn format(kind: &str, text: &str) -> bool {
    match kind {
        "" => true,
        "uuid" | "uuid-v7" => {
            let bytes = text.as_bytes();
            bytes.len() == 36
                && bytes.iter().enumerate().all(|(i, b)| {
                    if matches!(i, 8 | 13 | 18 | 23) {
                        *b == b'-'
                    } else {
                        b.is_ascii_digit() || (b'a'..=b'f').contains(b)
                    }
                })
                && matches!(bytes[19], b'8' | b'9' | b'a' | b'b')
                && (kind != "uuid-v7" || bytes[14] == b'7')
                && uuid::Uuid::parse_str(text).is_ok()
        }
        "date-time" => text.ends_with('Z') && chrono::DateTime::parse_from_rfc3339(text).is_ok(),
        "sha256" => {
            text.len() == 64
                && text
                    .bytes()
                    .all(|b| b.is_ascii_digit() || (b'a'..=b'f').contains(&b))
        }
        "decimal" | "usdt6" | "usdt6-total" => {
            let mut parts = text.split('.');
            let integer = parts.next().unwrap_or("");
            let fraction = parts.next();
            if parts.next().is_some()
                || integer.is_empty()
                || integer.len() > 1 && integer.starts_with('0')
                || !integer.bytes().all(|b| b.is_ascii_digit())
            {
                return false;
            }
            if let Some(f) = fraction {
                if f.is_empty() || !f.bytes().all(|b| b.is_ascii_digit()) {
                    return false;
                }
            }
            if kind == "decimal" {
                return true;
            }
            fraction.is_some_and(|f| f.len() == 6)
                && text
                    .replace('.', "")
                    .parse::<i64>()
                    .is_ok_and(|n| kind == "usdt6-total" || n <= 9_000_000_000_000)
        }
        "traceparent" => crate::valid_traceparent(text),
        "encrypted" => STANDARD.decode(text).is_ok_and(|v| v.len() >= 28),
        _ => false,
    }
}

pub fn validate(schema: &str, name: &str, value: &Value) -> Result<(), String> {
    // Each module supplies one immutable canonical schema; no remote resolver exists.
    static REGISTRIES: std::sync::LazyLock<
        std::sync::Mutex<std::collections::HashMap<String, std::sync::Arc<Value>>>,
    > = std::sync::LazyLock::new(|| std::sync::Mutex::new(std::collections::HashMap::new()));
    let root = {
        let mut registries = REGISTRIES.lock().map_err(|_| "schema registry poisoned")?;
        if let Some(root) = registries.get(schema) {
            root.clone()
        } else {
            let root = std::sync::Arc::new(
                serde_json::from_str::<Value>(schema).map_err(|e| e.to_string())?,
            );
            registries.insert(schema.into(), root.clone());
            root
        }
    };
    check(&root, &root["$defs"][name], value)
}

fn check(root: &Value, rule: &Value, value: &Value) -> Result<(), String> {
    if rule.is_null() {
        return Err("unknown contract definition".into());
    }
    if let Some(reference) = rule["$ref"].as_str() {
        let name = reference
            .strip_prefix("#/$defs/")
            .ok_or("unsupported schema reference")?;
        return check(root, &root["$defs"][name], value);
    }
    if let Some(choices) = rule["anyOf"].as_array() {
        return if choices.iter().any(|r| check(root, r, value).is_ok()) {
            Ok(())
        } else {
            Err("invalid nullable reference".into())
        };
    }
    let allowed = |kind: &str| {
        rule["type"].as_str() == Some(kind)
            || rule["type"]
                .as_array()
                .is_some_and(|a| a.iter().any(|v| v.as_str() == Some(kind)))
    };
    if value.is_null() {
        return if allowed("null")
            && rule["enum"]
                .as_array()
                .is_none_or(|values| values.contains(&Value::Null))
        {
            Ok(())
        } else {
            Err("null prohibited".into())
        };
    }
    if let Some(expected) = rule.get("const") {
        if value != expected {
            return Err("invalid constant".into());
        }
    }
    if let Some(values) = rule["enum"].as_array() {
        if !values.contains(value) {
            return Err("invalid enum".into());
        }
    }
    match value {
        Value::Object(fields) => {
            if !allowed("object") {
                return Err("object prohibited".into());
            }
            if let Some(required) = rule["required"].as_array() {
                for field in required {
                    if !fields.contains_key(field.as_str().ok_or("invalid required field")?) {
                        return Err("missing contract field".into());
                    }
                }
            }
            for (name, value) in fields {
                let child = rule["properties"]
                    .get(name)
                    .ok_or("unexpected contract field")?;
                check(root, child, value).map_err(|e| format!("{name}: {e}"))?;
            }
        }
        Value::Array(items) => {
            if !allowed("array") {
                return Err("array prohibited".into());
            }
            if rule["minItems"]
                .as_u64()
                .is_some_and(|n| items.len() < (n as usize))
                || rule["maxItems"]
                    .as_u64()
                    .is_some_and(|n| items.len() > (n as usize))
            {
                return Err("invalid array length".into());
            }
            for item in items {
                check(root, &rule["items"], item)?;
            }
        }
        Value::String(text) => {
            if !allowed("string") {
                return Err("string prohibited".into());
            }
            let length = text.chars().count() as u64;
            if rule["minLength"].as_u64().is_some_and(|n| length < n)
                || rule["maxLength"].as_u64().is_some_and(|n| length > n)
                || !format(rule["format"].as_str().unwrap_or(""), text)
            {
                return Err("invalid string constraint".into());
            }
        }
        Value::Bool(_) => {
            if !allowed("boolean") {
                return Err("boolean prohibited".into());
            }
        }
        Value::Number(number) => {
            let n = number.as_i64().ok_or("integer required")?;
            if !allowed("integer")
                || rule["minimum"].as_i64().is_some_and(|m| n < m)
                || rule["maximum"].as_i64().is_some_and(|m| n > m)
            {
                return Err("integer outside bounds".into());
            }
        }
        Value::Null => unreachable!(),
    }
    Ok(())
}

// Every generated deserialize uses this visitor, so duplicate keys cannot be hidden
// by a serde_json::Value map before schema validation.
pub struct UniqueValue(pub Value);
impl<'de> serde::Deserialize<'de> for UniqueValue {
    fn deserialize<D: serde::Deserializer<'de>>(d: D) -> Result<Self, D::Error> {
        struct Visitor;
        impl<'de> serde::de::Visitor<'de> for Visitor {
            type Value = UniqueValue;
            fn expecting(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
                f.write_str("strict JSON value")
            }
            fn visit_bool<E: serde::de::Error>(self, v: bool) -> Result<Self::Value, E> {
                Ok(UniqueValue(Value::Bool(v)))
            }
            fn visit_i64<E: serde::de::Error>(self, v: i64) -> Result<Self::Value, E> {
                Ok(UniqueValue(Value::Number(v.into())))
            }
            fn visit_u64<E: serde::de::Error>(self, v: u64) -> Result<Self::Value, E> {
                Ok(UniqueValue(Value::Number(v.into())))
            }
            fn visit_f64<E: serde::de::Error>(self, _: f64) -> Result<Self::Value, E> {
                Err(E::custom("floating-point wire numbers prohibited"))
            }
            fn visit_str<E: serde::de::Error>(self, v: &str) -> Result<Self::Value, E> {
                Ok(UniqueValue(Value::String(v.into())))
            }
            fn visit_string<E: serde::de::Error>(self, v: String) -> Result<Self::Value, E> {
                Ok(UniqueValue(Value::String(v)))
            }
            fn visit_none<E: serde::de::Error>(self) -> Result<Self::Value, E> {
                Ok(UniqueValue(Value::Null))
            }
            fn visit_unit<E: serde::de::Error>(self) -> Result<Self::Value, E> {
                Ok(UniqueValue(Value::Null))
            }
            fn visit_seq<A: serde::de::SeqAccess<'de>>(
                self,
                mut a: A,
            ) -> Result<Self::Value, A::Error> {
                let mut out = Vec::new();
                while let Some(UniqueValue(v)) = a.next_element()? {
                    out.push(v);
                }
                Ok(UniqueValue(Value::Array(out)))
            }
            fn visit_map<A: serde::de::MapAccess<'de>>(
                self,
                mut a: A,
            ) -> Result<Self::Value, A::Error> {
                let mut out = serde_json::Map::new();
                while let Some(k) = a.next_key::<String>()? {
                    if out.contains_key(&k) {
                        return Err(serde::de::Error::custom("duplicate JSON key"));
                    }
                    let UniqueValue(v) = a.next_value()?;
                    out.insert(k, v);
                }
                Ok(UniqueValue(Value::Object(out)))
            }
        }
        d.deserialize_any(Visitor)
    }
}

pub fn present_nullable<'de, D, T>(d: D) -> Result<Option<Option<T>>, D::Error>
where
    D: serde::Deserializer<'de>,
    T: serde::Deserialize<'de>,
{
    <Option<T> as serde::Deserialize>::deserialize(d).map(Some)
}
