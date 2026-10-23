package internalhttpv2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"platform-contracts/gen/go/events"
	"platform-contracts/gen/go/eventsv2"
)

// QuarantineReplayRequest accepts only a fully validated replacement envelope,
// never an unchecked payload or decrypted command. Owner authorization remains required.
type QuarantineReplayRequest struct {
	Reason         string          `json:"reason"`
	CorrectedEvent json.RawMessage `json:"corrected_event,omitempty"`
}

func (v QuarantineReplayRequest) Validate() error {
	if len(v.Reason) == 0 || len(v.Reason) > 500 {
		return fmt.Errorf("reason required")
	}
	if v.CorrectedEvent != nil {
		var version struct {
			SchemaVersion int `json:"schema_version"`
		}
		if err := json.Unmarshal(v.CorrectedEvent, &version); err != nil {
			return err
		}
		switch version.SchemaVersion {
		case 1:
			var e events.Envelope
			if err := json.Unmarshal(v.CorrectedEvent, &e); err != nil {
				return err
			}
		case 2:
			var e eventsv2.Envelope
			if err := json.Unmarshal(v.CorrectedEvent, &e); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown corrected event schema")
		}
	}
	return nil
}
func (v QuarantineReplayRequest) MarshalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	type alias QuarantineReplayRequest
	return json.Marshal(alias(v))
}
func (v *QuarantineReplayRequest) UnmarshalJSON(data []byte) error {
	// Check duplicate keys in an independently closed shape whose corrected body is
	// dispatched through the strict event codec below.
	decoder := json.NewDecoder(bytes.NewReader(data))
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	if len(fields) < 1 || len(fields) > 2 {
		return fmt.Errorf("invalid replay fields")
	}
	for k := range fields {
		if k != "reason" && k != "corrected_event" {
			return fmt.Errorf("unknown replay field")
		}
	}
	// Duplicate top-level detection is performed by decoding tokens, not a map.
	check := json.NewDecoder(bytes.NewReader(data))
	token, err := check.Token()
	if err != nil || token != json.Delim('{') {
		return fmt.Errorf("object required")
	}
	seen := map[string]bool{}
	for check.More() {
		token, err = check.Token()
		if err != nil {
			return err
		}
		k := token.(string)
		if seen[k] {
			return fmt.Errorf("duplicate field")
		}
		seen[k] = true
		var raw json.RawMessage
		if err = check.Decode(&raw); err != nil {
			return err
		}
	}
	type alias QuarantineReplayRequest
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	candidate := QuarantineReplayRequest(a)
	if err := candidate.Validate(); err != nil {
		return err
	}
	*v = candidate
	return nil
}
