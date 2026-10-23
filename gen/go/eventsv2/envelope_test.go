package eventsv2

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func registered() Envelope {
	return Envelope{SchemaVersion: 2, EventID: "019a13cf-0000-7000-8000-000000000001", EventType: "identity.user_registered", Producer: "identity", AggregateType: "user", AggregateID: "019a13cf-0000-7000-8000-000000000002", AggregateVersion: 1, OccurredAt: time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC), Payload: json.RawMessage(`{"user_id":"019a13cf-0000-7000-8000-000000000002","email":"user@example.test","locale":"en","registration_id":"019a13cf-0000-7000-8000-000000000003","referral_claim_id":null}`)}
}
func TestEnvelopeRoundtripAndTopic(t *testing.T) {
	e := registered()
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var got Envelope
	if err = json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Topic() != "identity.events.v2" || got.AggregateVersion != 1 || got.EventID != e.EventID {
		t.Fatalf("bad roundtrip: %+v", got)
	}
}

func TestSerializedEnvelopeRejectsWrongPayloadAggregate(t *testing.T) {
	event := registered()
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(string(raw), `"aggregate_id":"`+event.AggregateID+`"`, `"aggregate_id":"019a13cf-0000-7000-8000-000000000099"`, 1)
	var decoded Envelope
	if err = json.Unmarshal([]byte(bad), &decoded); err == nil {
		t.Fatal("consumer accepted payload for different aggregate owner")
	}
	event.AggregateID = "019a13cf-0000-7000-8000-000000000099"
	if _, err = json.Marshal(event); err == nil {
		t.Fatal("producer serialized payload for different aggregate owner")
	}
}
func TestEnvelopeRejectsInvalidWire(t *testing.T) {
	good, _ := json.Marshal(registered())
	for _, bad := range []string{
		strings.Replace(string(good), `"producer":"identity"`, `"producer":"billing"`, 1),
		strings.Replace(string(good), `"aggregate_version":1`, `"aggregate_version":0`, 1),
		strings.Replace(string(good), `"schema_version":2`, `"schema_version":1`, 1),
		strings.Replace(string(good), `"referral_claim_id":null`, `"extra":null`, 1),
		strings.Replace(string(good), `"locale":"en"`, `"locale":null`, 1),
		strings.Replace(string(good), `"locale":"en"`, `"locale":"en","locale":"ru"`, 1),
		strings.Replace(string(good), `7000-8000-000000000001`, `4000-8000-000000000001`, 1),
		strings.Replace(string(good), `"schema_version":2`, `"schema_version":2,"unknown":true`, 1),
	} {
		var e Envelope
		if json.Unmarshal([]byte(bad), &e) == nil {
			t.Errorf("accepted invalid wire: %s", bad)
		}
	}
}
func TestSecretsOnlyEncryptedCommands(t *testing.T) {
	e := registered()
	e.EventType = "notification.password_reset"
	e.Payload = json.RawMessage(`{"user_id":"019a13cf-0000-7000-8000-000000000002","email":"u@example.test","token":"secret","expires_at":"2026-10-24T00:00:00Z","locale":"en"}`)
	if e.Validate() == nil {
		t.Fatal("accepted plaintext command")
	}
	if err := ValidatePayload(e.EventType, e.Payload); err != nil {
		t.Fatal(err)
	}
	e.AggregateType = "delivery_command"
	e.AggregateID = e.EventID
	e.Payload = nil
	e.EncryptedPayload = base64.StdEncoding.EncodeToString(make([]byte, 28))
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	if e.Topic() != "notification.commands.v2" {
		t.Fatal(e.Topic())
	}
	e.EventType = "identity.user_registered"
	if e.Validate() == nil {
		t.Fatal("accepted encrypted public event")
	}
}
func TestUnknownEventCannotCarryArbitraryPayload(t *testing.T) {
	e := registered()
	e.EventType = "identity.surprise"
	if e.Validate() == nil {
		t.Fatal("accepted unknown event")
	}
}

func TestEnvelopeInt64VersionBoundary(t *testing.T) {
	for _, version := range []int64{9007199254740992, 9223372036854775807} {
		event := registered()
		event.AggregateVersion = version
		raw, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		var got Envelope
		if err = json.Unmarshal(raw, &got); err != nil || got.AggregateVersion != version {
			t.Fatalf("lost exact version %d: %v", version, err)
		}
	}
	raw, _ := json.Marshal(registered())
	overflow := strings.Replace(string(raw), `"aggregate_version":1`, `"aggregate_version":9223372036854775808`, 1)
	var got Envelope
	if json.Unmarshal([]byte(overflow), &got) == nil {
		t.Fatal("accepted int64 overflow")
	}
}
