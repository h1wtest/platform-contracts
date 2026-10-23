// Package events implements the canonical JSON event contract. Payload DTOs are schema-generated.
package events

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "regexp"
    "strings"
    "time"
)

const IdentityTopic = "identity.events.v1"
const CommandTopic = "notification.commands.v1"
const Partitions = 6

// Envelope carries exactly one payload representation. Sensitive commands never carry plaintext.
type Envelope struct {
    SchemaVersion int `json:"schema_version"`
    EventID string `json:"event_id"`
    EventType string `json:"event_type"`
    AggregateID string `json:"aggregate_id"`
    OccurredAt time.Time `json:"occurred_at"`
    Traceparent string `json:"traceparent,omitempty"`
    Payload json.RawMessage `json:"payload,omitempty"`
    EncryptedPayload string `json:"encrypted_payload,omitempty"`
}

type wireEnvelope Envelope

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var tracePattern = regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)

func Sensitive(eventType string) bool { return strings.HasPrefix(eventType, "notification.") }
func Topic(eventType string) (string, error) {
    if _, ok := payloadDefinitions[eventType]; !ok { return "", errors.New("unknown event type") }
    if Sensitive(eventType) { return CommandTopic, nil }
    return IdentityTopic, nil
}

func (e Envelope) Validate() error {
    if e.SchemaVersion != 1 || !uuidPattern.MatchString(e.EventID) || !uuidPattern.MatchString(e.AggregateID) || e.OccurredAt.IsZero() { return errors.New("invalid envelope metadata") }
    if _, err := Topic(e.EventType); err != nil { return err }
    if e.Traceparent != "" {
        if !tracePattern.MatchString(e.Traceparent) || e.Traceparent[3:35] == strings.Repeat("0", 32) || e.Traceparent[36:52] == strings.Repeat("0", 16) { return errors.New("invalid traceparent") }
    }
    if Sensitive(e.EventType) {
        if len(e.Payload) != 0 || e.EncryptedPayload == "" { return errors.New("command requires exclusively encrypted_payload") }
        decoded, err := base64.StdEncoding.Strict().DecodeString(e.EncryptedPayload)
        if err != nil || len(decoded) < 28 { return errors.New("invalid encrypted payload framing") }
        return nil
    }
    if e.EncryptedPayload != "" || len(e.Payload) == 0 { return errors.New("public event requires exclusively payload") }
    return ValidatePayload(e.EventType, e.Payload)
}

func (e Envelope) MarshalJSON() ([]byte, error) {
    if err := e.Validate(); err != nil { return nil, err }
    return json.Marshal(wireEnvelope(e))
}

func (e *Envelope) UnmarshalJSON(data []byte) error {
    var fields map[string]json.RawMessage
    if err := strictJSON(data, &fields); err != nil { return err }
    for _, key := range []string{"schema_version", "event_id", "event_type", "aggregate_id", "occurred_at"} {
        raw, ok := fields[key]
        if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) { return errors.New("missing envelope metadata") }
    }
    _, payload := fields["payload"]
    _, encrypted := fields["encrypted_payload"]
    if payload == encrypted { return errors.New("exactly one payload field required") }
    if raw, ok := fields["traceparent"]; ok {
        var trace string
        if err := json.Unmarshal(raw, &trace); err != nil || trace == "" { return errors.New("invalid traceparent") }
    }
    var wire wireEnvelope
    if err := strictJSON(data, &wire); err != nil { return err }
    candidate := Envelope(wire)
    if err := candidate.Validate(); err != nil { return err }
    *e = candidate
    return nil
}

func (e Envelope) DecodePayload(destination any) error {
    if err := e.Validate(); err != nil { return err }
    if Sensitive(e.EventType) { return errors.New("decrypt command before decoding its payload") }
    return strictJSON(e.Payload, destination)
}

type property struct { Type string `json:"type"`; Format string `json:"format"`; MinLength int `json:"minLength"` }
type definition struct { Required []string `json:"required"`; Properties map[string]property `json:"properties"` }
var definitions = func() map[string]definition {
    var schema struct { Definitions map[string]definition `json:"$defs"` }
    if err := json.Unmarshal([]byte(canonicalSchema), &schema); err != nil { panic("invalid generated canonical schema") }
    return schema.Definitions
}()

// ValidatePayload also validates decrypted command plaintext; it does not mutate payload bytes.
func ValidatePayload(eventType string, data json.RawMessage) error {
    name, ok := payloadDefinitions[eventType]
    if !ok { return errors.New("unknown event type") }
    def := definitions[name]
    var fields map[string]json.RawMessage
    if err := strictJSON(data, &fields); err != nil || fields == nil { return errors.New("payload must be an object") }
    for _, key := range def.Required { if _, ok := fields[key]; !ok { return fmt.Errorf("missing payload field %s", key) } }
    for key, raw := range fields {
        rule, ok := def.Properties[key]
        if !ok { return errors.New("unexpected payload field") }
        if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) { return errors.New("null payload field") }
        if rule.Type == "boolean" {
            var b bool
            if err := json.Unmarshal(raw, &b); err != nil { return errors.New("invalid boolean payload field") }
            continue
        }
        var value string
        if err := json.Unmarshal(raw, &value); err != nil || len([]rune(value)) < rule.MinLength { return errors.New("invalid string payload field") }
        if rule.Format == "uuid" && !uuidPattern.MatchString(value) { return errors.New("invalid payload UUID") }
        if rule.Format == "date-time" { if _, err := time.Parse(time.RFC3339Nano, value); err != nil { return errors.New("invalid payload timestamp") } }
    }
    return nil
}

func strictJSON(data []byte, destination any) error {
    decoder := json.NewDecoder(bytes.NewReader(data))
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(destination); err != nil { return err }
    if err := decoder.Decode(new(any)); err != io.EOF { return errors.New("trailing JSON data") }
    return nil
}
