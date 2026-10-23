package internalhttp

import (
	"encoding/json"
	"strings"
	"testing"
)

const actorJSON = `{"active":true,"user_id":"00000000-0000-4000-8000-000000000001","email":"a@example.test","session_id":"00000000-0000-4000-8000-000000000002","roles":[],"permissions":[],"mfa_verified":false}`

func TestIntrospectionRejectsMalformedSuccessfulDependencyResponse(t *testing.T) {
	for _, raw := range []string{
		`{"active":true,"user_id":"not-a-uuid"}`,
		`{"active":true,"user_id":"00000000-0000-4000-8000-000000000001","email":"a","session_id":"00000000-0000-4000-8000-000000000002","roles":null,"permissions":[],"mfa_verified":false}`,
		actorJSON[:len(actorJSON)-1] + `,"actor_id":"injected"}`,
	} {
		var value IntrospectResponse
		if json.Unmarshal([]byte(raw), &value) == nil {
			t.Fatalf("accepted malformed actor: %s", raw)
		}
	}
	var actor IntrospectResponse
	if err := json.Unmarshal([]byte(actorJSON), &actor); err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(actor); err != nil {
		t.Fatal(err)
	}
	actor.UserID = "bad"
	if _, err := json.Marshal(actor); err == nil {
		t.Fatal("serialized invalid UUID")
	}
}

func TestHTTPShapePresenceNullabilityEnumsAndBounds(t *testing.T) {
	cases := []struct {
		raw         string
		destination any
	}{
		{`{"require_mfa":null}`, &IntrospectRequest{}},
		{`{"reason":null}`, &BanRequest{}},
		{`{"status":"other"}`, &ReplayResponse{}},
		{`{"corrected_event":{}}`, &ReplayRequest{}},
		{`{"corrected_event":null}`, &ReplayRequest{}},
		{`{"email":"a","email_login_alerts":true,"email_security_alerts":true,"telegram_login_alerts":false,"telegram_security_alerts":false}`, &NotificationSettings{}},
		{`null`, &DeliveriesResponse{}},
		{`{"unexpected":true}`, &EmptyRequest{}},
		{`{"reason":"` + strings.Repeat("a", 501) + `"}`, &BanRequest{}},
	}
	for _, test := range cases {
		if json.Unmarshal([]byte(test.raw), test.destination) == nil {
			t.Fatalf("accepted invalid shape: %s", test.raw)
		}
	}
	var settings NotificationSettings
	if err := json.Unmarshal([]byte(`{"email":"a","telegram_chat_id":null,"email_login_alerts":true,"email_security_alerts":true,"telegram_login_alerts":false,"telegram_security_alerts":false}`), &settings); err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(settings); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{}`), &BanRequest{}); err != nil {
		t.Fatal(err)
	}
}

func TestDeliveryRejectsEnumMinimumAndMissingNullableFields(t *testing.T) {
	valid := `{"id":"00000000-0000-4000-8000-000000000001","user_id":"00000000-0000-4000-8000-000000000002","channel":"email","status":"pending","attempts":0,"next_attempt_at":"2026-10-23T00:00:00Z","last_error":null,"message_id":"x","created_at":"2026-10-23T00:00:00Z","delivered_at":null,"ambiguous":false}`
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(valid), &fields); err != nil {
		t.Fatal(err)
	}
	for key, replacement := range map[string]json.RawMessage{"channel": json.RawMessage(`"sms"`), "status": json.RawMessage(`"unknown"`), "attempts": json.RawMessage(`-1`), "last_error": nil} {
		changed := make(map[string]json.RawMessage)
		for k, v := range fields {
			changed[k] = v
		}
		if replacement == nil {
			delete(changed, key)
		} else {
			changed[key] = replacement
		}
		raw, err := json.Marshal(changed)
		if err != nil {
			t.Fatal(err)
		}
		if json.Unmarshal(raw, &Delivery{}) == nil {
			t.Fatalf("accepted invalid %s", key)
		}
	}
	if err := json.Unmarshal([]byte(valid), &Delivery{}); err != nil {
		t.Fatal(err)
	}
}
