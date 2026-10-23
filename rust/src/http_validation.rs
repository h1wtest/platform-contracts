use serde_json::Value;

static SCHEMA: std::sync::LazyLock<Value> = std::sync::LazyLock::new(|| {
    serde_json::from_str(crate::http::CANONICAL_SCHEMA).expect("generated canonical HTTP schema")
});

pub(crate) fn validate(name: &str, value: &Value) -> Result<(), String> {
    let rule = SCHEMA["$defs"].get(name).ok_or("unknown HTTP definition")?;
    validate_rule(rule, value)
}

fn validate_rule(rule: &Value, value: &Value) -> Result<(), String> {
    if let Some(reference) = rule["$ref"].as_str() {
        if reference == "events-v1.json" {
            serde_json::from_value::<crate::Envelope>(value.clone()).map_err(|e| e.to_string())?;
            return Ok(());
        }
        return validate(
            reference
                .strip_prefix("#/$defs/")
                .ok_or("unsupported HTTP reference")?,
            value,
        );
    }
    let allowed = |kind: &str| {
        rule["type"].as_str() == Some(kind)
            || rule["type"]
                .as_array()
                .is_some_and(|types| types.iter().any(|t| t.as_str() == Some(kind)))
    };
    if value.is_null() {
        return if allowed("null") {
            Ok(())
        } else {
            Err("non-null HTTP field required".into())
        };
    }
    if rule["enum"]
        .as_array()
        .is_some_and(|values| !values.contains(value))
    {
        return Err("invalid HTTP enum value".into());
    }
    match value {
        Value::Object(fields) => {
            if !allowed("object") {
                return Err("HTTP object not allowed".into());
            }
            if let Some(required) = rule["required"].as_array() {
                for field in required {
                    if !fields.contains_key(field.as_str().ok_or("invalid HTTP schema field")?) {
                        return Err("missing HTTP field".into());
                    }
                }
            }
            for (field, child) in fields {
                let child_rule = rule["properties"]
                    .get(field)
                    .ok_or("unexpected HTTP field")?;
                validate_rule(child_rule, child).map_err(|e| format!("{field}: {e}"))?;
            }
        }
        Value::Array(items) => {
            if !allowed("array") {
                return Err("HTTP array not allowed".into());
            }
            for item in items {
                validate_rule(&rule["items"], item)?;
            }
        }
        Value::String(text) => {
            if !allowed("string") {
                return Err("HTTP string not allowed".into());
            }
            let length = text.chars().count() as u64;
            if rule["minLength"].as_u64().is_some_and(|min| length < min)
                || rule["maxLength"].as_u64().is_some_and(|max| length > max)
            {
                return Err("invalid HTTP string length".into());
            }
            match rule["format"].as_str() {
                Some("uuid") if !crate::valid_uuid(text) => return Err("invalid HTTP UUID".into()),
                Some("date-time") if chrono::DateTime::parse_from_rfc3339(text).is_err() => {
                    return Err("invalid HTTP timestamp".into())
                }
                _ => {}
            }
        }
        Value::Bool(_) => {
            if !allowed("boolean") {
                return Err("HTTP boolean not allowed".into());
            }
        }
        Value::Number(number) => {
            let n = number.as_i64().ok_or("invalid HTTP integer")?;
            if !allowed("integer") {
                return Err("HTTP integer not allowed".into());
            }
            if rule["minimum"].as_i64().is_some_and(|min| n < min) {
                return Err("HTTP integer below minimum".into());
            }
        }
        Value::Null => unreachable!(),
    }
    Ok(())
}
