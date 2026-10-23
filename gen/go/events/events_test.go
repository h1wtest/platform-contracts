package events

import (
	"encoding/json"
	"testing"
)

const registered = `{"schema_version":1,"event_id":"00000000-0000-4000-8000-000000000001","event_type":"identity.user_registered","aggregate_id":"00000000-0000-4000-8000-000000000002","occurred_at":"2026-10-23T00:00:00Z","payload":{"user_id":"00000000-0000-4000-8000-000000000002","email":"person@example.test"}}`

func TestPublicEnvelopeRoundTripAndTypedPayload(t *testing.T) {
	var envelope Envelope
	if err := json.Unmarshal([]byte(registered), &envelope); err != nil { t.Fatal(err) }
	var payload UserRegistered
	if err := envelope.DecodePayload(&payload); err != nil { t.Fatal(err) }
	if payload.Email != "person@example.test" || payload.UserID != envelope.AggregateID { t.Fatal("payload did not survive decoding") }
	wire, err := json.Marshal(envelope)
	if err != nil { t.Fatal(err) }
	var decoded Envelope
	if err := json.Unmarshal(wire, &decoded); err != nil { t.Fatal(err) }
	if decoded.EventID != envelope.EventID { t.Fatal("stable event id changed") }
}

func TestRejectsAmbiguousSecretAndMalformedEnvelopes(t *testing.T) {
	invalid := []string{
		`{"schema_version":1,"event_id":"00000000-0000-4000-8000-000000000001","event_type":"notification.password_reset","aggregate_id":"00000000-0000-4000-8000-000000000002","occurred_at":"2026-10-23T00:00:00Z","payload":{"user_id":"00000000-0000-4000-8000-000000000002","email":"a@b.test","token":"secret","expires_at":"2026-10-23T01:00:00Z"}}`,
		registered[:len(registered)-1] + `,"encrypted_payload":"AAAA"}`,
		`{"schema_version":2,"event_id":"bad","event_type":"unknown","aggregate_id":"bad","occurred_at":"2026-10-23T00:00:00Z","payload":{}}`,
		registered[:len(registered)-2] + `,"verification_code":"secret"}}`,
	}
	for _, wire := range invalid {
		var envelope Envelope
		if err := json.Unmarshal([]byte(wire), &envelope); err == nil { t.Fatalf("accepted invalid envelope: %s", wire) }
	}
}

func TestSensitiveEnvelopeAllowsOnlyCiphertextAndValidatesPlaintextSeparately(t *testing.T) {
	wire := `{"schema_version":1,"event_id":"00000000-0000-4000-8000-000000000001","event_type":"notification.password_reset","aggregate_id":"00000000-0000-4000-8000-000000000002","occurred_at":"2026-10-23T00:00:00Z","encrypted_payload":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}`
	var envelope Envelope
	if err := json.Unmarshal([]byte(wire), &envelope); err != nil { t.Fatal(err) }
	if envelope.DecodePayload(&PasswordReset{}) == nil { t.Fatal("ciphertext treated as plaintext") }
	if ValidatePayload("notification.password_reset", json.RawMessage(`{"user_id":"00000000-0000-4000-8000-000000000002","email":"a@b.test","token":"secret","expires_at":"2026-10-23T01:00:00Z"}`)) != nil { t.Fatal("valid command rejected") }
	if ValidatePayload("notification.password_reset", json.RawMessage(`{"user_id":"00000000-0000-4000-8000-000000000002","email":"a@b.test","expires_at":"2026-10-23T01:00:00Z"}`)) == nil { t.Fatal("missing token accepted") }
}
