package wire

import "testing"

func TestCumulativeUSDTDoesNotReuseSaleCeiling(t *testing.T) {
	for _, value := range []string{"0.000000", "18000000.000000", "9223372036854.775807"} {
		if !Format("usdt6-total", value) {
			t.Fatalf("rejected cumulative amount %s", value)
		}
	}
	for _, value := range []string{"-1.000000", "018000000.000000", "1.00000", "1.0000000", "1e6", "9223372036854.775808", "9223372036855.000000"} {
		if Format("usdt6-total", value) {
			t.Fatalf("accepted invalid cumulative amount %s", value)
		}
	}
	if Format("usdt6", "18000000.000000") {
		t.Fatal("individual sale ceiling was weakened")
	}
	if !Format("usdt6", "9000000.000000") {
		t.Fatal("valid maximum sale rejected")
	}
}

func TestSerializedCanonicalIntegerTokens(t *testing.T) {
	registry := Schema(`{"$defs":{"Integer":{"type":"object","additionalProperties":false,"required":["value"],"properties":{"value":{"type":"integer","minimum":-9223372036854775808,"maximum":9223372036854775807}}}}}`)
	for _, token := range []string{"1.0000000000000001", "9007199254740991.1", "1e0", "1E+0", "0.0", "-0.0", "1e-999"} {
		if err := registry.Validate("Integer", []byte(`{"value":`+token+`}`)); err == nil {
			t.Fatalf("accepted %s", token)
		}
	}
	for _, token := range []string{"-9223372036854775808", "9223372036854775807", "9007199254740991", "-9007199254740991"} {
		if err := registry.Validate("Integer", []byte(`{"value":`+token+`}`)); err != nil {
			t.Fatalf("rejected %s: %v", token, err)
		}
	}
}
