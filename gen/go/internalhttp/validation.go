package internalhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"
	"unicode/utf8"

	"platform-contracts/gen/go/events"
)

type wireRule struct {
	Type       json.RawMessage      `json:"type"`
	Ref        string               `json:"$ref"`
	Properties map[string]*wireRule `json:"properties"`
	Required   []string             `json:"required"`
	Items      *wireRule            `json:"items"`
	Enum       []json.RawMessage    `json:"enum"`
	Format     string               `json:"format"`
	MinLength  *int                 `json:"minLength"`
	MaxLength  *int                 `json:"maxLength"`
	Minimum    *int64               `json:"minimum"`
}

var wireDefinitions = func() map[string]*wireRule {
	var schema struct {
		Defs map[string]*wireRule `json:"$defs"`
	}
	if err := json.Unmarshal([]byte(CanonicalSchema), &schema); err != nil {
		panic("invalid generated HTTP schema")
	}
	return schema.Defs
}()

var httpUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validateHTTP(name string, data []byte) error {
	rule, ok := wireDefinitions[name]
	if !ok {
		return fmt.Errorf("unknown HTTP definition %s", name)
	}
	return validateWire(rule, data)
}

func validateWire(rule *wireRule, data []byte) error {
	if rule == nil {
		return fmt.Errorf("missing HTTP schema rule")
	}
	if rule.Ref != "" {
		if rule.Ref == "events-v1.json" {
			var envelope events.Envelope
			return json.Unmarshal(data, &envelope)
		}
		const prefix = "#/$defs/"
		if len(rule.Ref) <= len(prefix) || rule.Ref[:len(prefix)] != prefix {
			return fmt.Errorf("unsupported HTTP schema reference")
		}
		return validateHTTP(rule.Ref[len(prefix):], data)
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("trailing HTTP JSON")
	}
	var types []string
	var single string
	if json.Unmarshal(rule.Type, &single) == nil {
		types = []string{single}
	} else if json.Unmarshal(rule.Type, &types) != nil {
		return fmt.Errorf("invalid HTTP schema type")
	}
	allowed := func(kind string) bool {
		for _, t := range types {
			if t == kind {
				return true
			}
		}
		return false
	}
	if value == nil {
		if allowed("null") {
			return nil
		}
		return fmt.Errorf("non-null HTTP field required")
	}
	if len(rule.Enum) > 0 {
		found := false
		for _, candidate := range rule.Enum {
			var expected string
			if json.Unmarshal(candidate, &expected) != nil {
				return fmt.Errorf("invalid HTTP schema enum")
			}
			if text, ok := value.(string); ok && text == expected {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("invalid HTTP enum value")
		}
	}
	switch v := value.(type) {
	case map[string]any:
		if !allowed("object") {
			return fmt.Errorf("HTTP object not allowed")
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return err
		}
		for _, required := range rule.Required {
			if _, ok := fields[required]; !ok {
				return fmt.Errorf("missing HTTP field %s", required)
			}
		}
		for field, raw := range fields {
			child, ok := rule.Properties[field]
			if !ok {
				return fmt.Errorf("unexpected HTTP field %s", field)
			}
			if err := validateWire(child, raw); err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
		}
	case []any:
		if !allowed("array") {
			return fmt.Errorf("HTTP array not allowed")
		}
		var items []json.RawMessage
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		for _, item := range items {
			if err := validateWire(rule.Items, item); err != nil {
				return err
			}
		}
	case string:
		if !allowed("string") {
			return fmt.Errorf("HTTP string not allowed")
		}
		length := utf8.RuneCountInString(v)
		if rule.MinLength != nil && length < *rule.MinLength || rule.MaxLength != nil && length > *rule.MaxLength {
			return fmt.Errorf("invalid HTTP string length")
		}
		if rule.Format == "uuid" && !httpUUID.MatchString(v) {
			return fmt.Errorf("invalid HTTP UUID")
		}
		if rule.Format == "date-time" {
			if _, err := time.Parse(time.RFC3339Nano, v); err != nil {
				return fmt.Errorf("invalid HTTP timestamp")
			}
		}
	case bool:
		if !allowed("boolean") {
			return fmt.Errorf("HTTP boolean not allowed")
		}
	case json.Number:
		n, err := v.Int64()
		if !allowed("integer") || err != nil {
			return fmt.Errorf("invalid HTTP integer")
		}
		if rule.Minimum != nil && n < *rule.Minimum {
			return fmt.Errorf("HTTP integer below minimum")
		}
	default:
		return fmt.Errorf("unsupported HTTP value")
	}
	return nil
}
