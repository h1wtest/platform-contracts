package eventsv2

import (
	"encoding/json"
	"fmt"
	"platform-contracts/gen/go/wire"
	"time"
)

const Partitions = 6
const IdentityTopic = "identity.events.v2"
const CommandTopic = "notification.commands.v2"

type Envelope struct {
	SchemaVersion    int             `json:"schema_version"`
	EventID          string          `json:"event_id"`
	EventType        string          `json:"event_type"`
	Producer         string          `json:"producer"`
	AggregateType    string          `json:"aggregate_type"`
	AggregateID      string          `json:"aggregate_id"`
	AggregateVersion int64           `json:"aggregate_version"`
	OccurredAt       time.Time       `json:"occurred_at"`
	CorrelationID    string          `json:"correlation_id,omitempty"`
	CausationID      string          `json:"causation_id,omitempty"`
	Traceparent      string          `json:"traceparent,omitempty"`
	Payload          json.RawMessage `json:"payload,omitempty"`
	EncryptedPayload string          `json:"encrypted_payload,omitempty"`
}

func Sensitive(eventType string) bool { r, ok := eventRules[eventType]; return ok && r.Encrypted }
func Topic(eventType string) string   { return eventRules[eventType].Topic }
func (e Envelope) Topic() string      { return Topic(e.EventType) }
func ValidatePayload(eventType string, data []byte) error {
	r, ok := eventRules[eventType]
	if !ok {
		return fmt.Errorf("unknown event type")
	}
	return registry.Validate(r.Definition, data)
}
func DecodePayload(eventType string, data []byte, target any) error {
	if err := ValidatePayload(eventType, data); err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
func (e Envelope) Validate() error {
	r, ok := eventRules[e.EventType]
	if !ok {
		return fmt.Errorf("unknown event type")
	}
	if e.SchemaVersion != 2 || !wire.Format("uuid-v7", e.EventID) || !wire.Format("uuid", e.AggregateID) || e.AggregateVersion < 1 {
		return fmt.Errorf("invalid event metadata")
	}
	if e.Producer != r.Producer || e.AggregateType != r.AggregateType {
		return fmt.Errorf("event producer or aggregate mismatch")
	}
	if (r.Encrypted || e.AggregateType == "external_event") && (e.AggregateID != e.EventID || e.AggregateVersion != 1) {
		return fmt.Errorf("independent command/event stream requires event ID and version one")
	}
	_, offset := e.OccurredAt.Zone()
	if e.OccurredAt.IsZero() || offset != 0 {
		return fmt.Errorf("UTC event timestamp required")
	}
	if e.CorrelationID != "" && !wire.Format("uuid", e.CorrelationID) || e.CausationID != "" && !wire.Format("uuid", e.CausationID) || e.Traceparent != "" && !wire.Format("traceparent", e.Traceparent) {
		return fmt.Errorf("invalid event tracing metadata")
	}
	if r.Encrypted {
		if len(e.Payload) != 0 || !wire.Format("encrypted", e.EncryptedPayload) {
			return fmt.Errorf("encrypted command required; plaintext prohibited")
		}
	} else {
		if e.EncryptedPayload != "" || len(e.Payload) == 0 {
			return fmt.Errorf("exactly one public payload required")
		}
		if err := ValidatePayload(e.EventType, e.Payload); err != nil {
			return err
		}
		if r.AggregateField != "" {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(e.Payload, &fields); err != nil {
				return err
			}
			var owner string
			if err := json.Unmarshal(fields[r.AggregateField], &owner); err != nil || owner != e.AggregateID {
				return fmt.Errorf("payload aggregate identity mismatch")
			}
		}
	}
	return nil
}
func (e Envelope) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	type alias Envelope
	return json.Marshal(alias(e))
}
func (e *Envelope) UnmarshalJSON(data []byte) error {
	// A closed metadata registry also detects duplicate keys recursively before typed decode.
	var header struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	if err := envelopeRegistry.Validate(header.EventType, data); err != nil {
		return err
	}
	type alias Envelope
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	candidate := Envelope(a)
	if err := candidate.Validate(); err != nil {
		return err
	}
	*e = candidate
	return nil
}

var envelopeRegistry = func() wire.Registry {
	var schema struct {
		Properties map[string]*wire.Rule `json:"properties"`
		Required   []string              `json:"required"`
	}
	if err := json.Unmarshal([]byte(CanonicalSchema), &schema); err != nil {
		panic(err)
	}
	result := wire.Schema(CanonicalSchema)
	for typ, r := range eventRules {
		props := make(map[string]*wire.Rule, len(schema.Properties))
		for k, v := range schema.Properties {
			if k != "payload" && k != "encrypted_payload" {
				props[k] = v
			}
		}
		required := append([]string(nil), schema.Required...)
		if r.Encrypted {
			props["encrypted_payload"] = schema.Properties["encrypted_payload"]
			required = append(required, "encrypted_payload")
		} else {
			props["payload"] = result[r.Definition]
			required = append(required, "payload")
		}
		result[typ] = &wire.Rule{Type: json.RawMessage(`"object"`), Properties: props, Required: required}
	}
	return result
}()
