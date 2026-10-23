package businessv1

import (
	"encoding/json"
	"testing"
)

func TestMoneyAndVersionAreExact(t *testing.T) {
	good := `{"rate_bps":100,"expected_version":1,"idempotency_key":"key","reason":"operator"}`
	var rate RateUpdate
	if err := json.Unmarshal([]byte(good), &rate); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{"rate_bps":10001,"expected_version":1,"idempotency_key":"key","reason":"operator"}`, `{"rate_bps":100,"expected_version":9007199254740992,"idempotency_key":"key","reason":"operator"}`, `{"rate_bps":100,"expected_version":1,"idempotency_key":"key","reason":"operator","extra":true}`} {
		if json.Unmarshal([]byte(bad), &rate) == nil {
			t.Fatal("accepted invalid rate")
		}
	}
	for _, amount := range []string{`0.000001`, `9000000.000000`} {
		if err := Validate("ManualSettlement", []byte(`{"amount":"`+amount+`","network":"tron","reference":"proof"}`)); err != nil {
			t.Fatal(err)
		}
	}
	for _, amount := range []string{`1`, `1.0`, `01.000000`, `9000000.000001`, `-1.000000`, `1e6`} {
		if Validate("ManualSettlement", []byte(`{"amount":"`+amount+`","network":"tron","reference":"proof"}`)) == nil {
			t.Fatal("accepted invalid amount", amount)
		}
	}
}
func TestPatchRetainsOmittedNullAndValue(t *testing.T) {
	var omitted, cleared, set ProfileChanges
	if err := json.Unmarshal([]byte(`{}`), &omitted); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"display_name":null}`), &cleared); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"display_name":"Ada"}`), &set); err != nil {
		t.Fatal(err)
	}
	if omitted.DisplayName.Present || !cleared.DisplayName.Present || !cleared.DisplayName.Null || set.DisplayName.Value != "Ada" {
		t.Fatal("PATCH semantics lost")
	}
	for _, tt := range []struct {
		v    ProfileChanges
		want string
	}{{omitted, `{}`}, {cleared, `{"display_name":null}`}, {set, `{"display_name":"Ada"}`}} {
		got, err := json.Marshal(tt.v)
		if err != nil || string(got) != tt.want {
			t.Fatalf("%s != %s (%v)", got, tt.want, err)
		}
	}
}
func TestUnknownAndNullBooleanAreRejected(t *testing.T) {
	for _, raw := range []string{`{"email_subscription":null}`, `{"bogus":true}`, `{"email_subscription":true,"email_subscription":false}`} {
		var patch PreferencesChanges
		if json.Unmarshal([]byte(raw), &patch) == nil {
			t.Fatal("accepted invalid PATCH", raw)
		}
	}
}

func TestPublicUsersArrayIsNotInternalWrapper(t *testing.T) {
	var users UserSummaries
	if err := json.Unmarshal([]byte(`[]`), &users); err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal([]byte(`{"users":[]}`), &users) == nil {
		t.Fatal("public list accepted internal wrapper")
	}
	var internal UsersResponse
	if err := json.Unmarshal([]byte(`{"users":[]}`), &internal); err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal([]byte(`[]`), &internal) == nil {
		t.Fatal("internal list accepted public array")
	}
}
