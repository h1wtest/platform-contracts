// Package wire validates the closed schema subset used by generated v2 contracts.
package wire

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Rule struct {
	Type       json.RawMessage   `json:"type"`
	Ref        string            `json:"$ref"`
	AnyOf      []*Rule           `json:"anyOf"`
	Properties map[string]*Rule  `json:"properties"`
	Required   []string          `json:"required"`
	Items      *Rule             `json:"items"`
	Enum       []json.RawMessage `json:"enum"`
	Const      json.RawMessage   `json:"const"`
	Format     string            `json:"format"`
	MinLength  *int              `json:"minLength"`
	MaxLength  *int              `json:"maxLength"`
	Minimum    *int64            `json:"minimum"`
	Maximum    *int64            `json:"maximum"`
	MinItems   *int              `json:"minItems"`
	MaxItems   *int              `json:"maxItems"`
}
type Registry map[string]*Rule

func Schema(text string) Registry {
	var s struct {
		Defs Registry `json:"$defs"`
	}
	if err := json.Unmarshal([]byte(text), &s); err != nil {
		panic(err)
	}
	return s.Defs
}
func (r Registry) Validate(name string, data []byte) error {
	if len(data) > 1048576 {
		return fmt.Errorf("contract body too large")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	v, err := value(d)
	if err != nil {
		return err
	}
	if _, err = d.Token(); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	return r.check(r[name], v)
}

// Decode recursively rather than through map unmarshalling: duplicate keys are invalid wire.
func value(d *json.Decoder) (any, error) {
	t, e := d.Token()
	if e != nil {
		return nil, e
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return t, nil
	}
	if delim == '{' {
		m := map[string]any{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return nil, e
			}
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("invalid key")
			}
			if _, ok = m[key]; ok {
				return nil, fmt.Errorf("duplicate field %s", key)
			}
			v, e := value(d)
			if e != nil {
				return nil, e
			}
			m[key] = v
		}
		_, e = d.Token()
		return m, e
	}
	if delim == '[' {
		a := []any{}
		for d.More() {
			v, e := value(d)
			if e != nil {
				return nil, e
			}
			a = append(a, v)
		}
		_, e = d.Token()
		return a, e
	}
	return nil, fmt.Errorf("invalid JSON delimiter")
}

var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var digest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var money = regexp.MustCompile(`^(0|[1-9][0-9]{0,6})\.[0-9]{6}$`)
var aggregateMoney = regexp.MustCompile(`^(0|[1-9][0-9]{0,12})\.[0-9]{6}$`)
var trace = regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)

func Format(kind, text string) bool {
	switch kind {
	case "":
		return true
	case "uuid":
		return uuid.MatchString(text)
	case "uuid-v7":
		return uuid.MatchString(text) && text[14] == '7'
	case "sha256":
		return digest.MatchString(text)
	case "usdt6":
		if !money.MatchString(text) {
			return false
		}
		n, e := strconv.ParseInt(strings.ReplaceAll(text, ".", ""), 10, 64)
		return e == nil && n <= 9000000000000
	case "usdt6-total":
		if !aggregateMoney.MatchString(text) {
			return false
		}
		_, err := strconv.ParseInt(strings.ReplaceAll(text, ".", ""), 10, 64)
		return err == nil
	case "decimal":
		return regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`).MatchString(text)
	case "date-time":
		t, e := time.Parse(time.RFC3339Nano, text)
		return e == nil && !t.IsZero() && strings.HasSuffix(text, "Z")
	case "traceparent":
		return trace.MatchString(text) && text[3:35] != strings.Repeat("0", 32) && text[36:52] != strings.Repeat("0", 16)
	case "encrypted":
		raw, e := base64.StdEncoding.Strict().DecodeString(text)
		return e == nil && len(raw) >= 28
	default:
		return false
	}
}
func (r Registry) check(rule *Rule, v any) error {
	if rule == nil {
		return fmt.Errorf("unknown contract definition")
	}
	if len(rule.AnyOf) > 0 {
		for _, candidate := range rule.AnyOf {
			if r.check(candidate, v) == nil {
				return nil
			}
		}
		return fmt.Errorf("invalid nullable reference")
	}
	if rule.Ref != "" {
		const p = "#/$defs/"
		if !strings.HasPrefix(rule.Ref, p) {
			return fmt.Errorf("unsupported schema ref")
		}
		return r.check(r[strings.TrimPrefix(rule.Ref, p)], v)
	}
	var types []string
	var one string
	if json.Unmarshal(rule.Type, &one) == nil {
		types = []string{one}
	} else if json.Unmarshal(rule.Type, &types) != nil {
		return fmt.Errorf("invalid schema type")
	}
	allowed := func(k string) bool {
		for _, t := range types {
			if t == k {
				return true
			}
		}
		return false
	}
	if v == nil {
		if allowed("null") {
			if len(rule.Enum) > 0 {
				for _, candidate := range rule.Enum {
					if bytes.Equal(candidate, []byte("null")) {
						return nil
					}
				}
				return fmt.Errorf("null outside enum")
			}
			return nil
		}
		return fmt.Errorf("null prohibited")
	}
	if rule.Const != nil {
		b, _ := json.Marshal(v)
		if !bytes.Equal(b, rule.Const) {
			return fmt.Errorf("invalid constant")
		}
	}
	if len(rule.Enum) > 0 {
		b, _ := json.Marshal(v)
		found := false
		for _, x := range rule.Enum {
			if bytes.Equal(b, x) {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("invalid enum")
		}
	}
	switch x := v.(type) {
	case map[string]any:
		if !allowed("object") {
			return fmt.Errorf("object prohibited")
		}
		for _, k := range rule.Required {
			if _, ok := x[k]; !ok {
				return fmt.Errorf("missing field %s", k)
			}
		}
		for k, v := range x {
			child, ok := rule.Properties[k]
			if !ok {
				return fmt.Errorf("unknown field %s", k)
			}
			if e := r.check(child, v); e != nil {
				return fmt.Errorf("%s: %w", k, e)
			}
		}
	case []any:
		if !allowed("array") {
			return fmt.Errorf("array prohibited")
		}
		if rule.MinItems != nil && len(x) < *rule.MinItems || rule.MaxItems != nil && len(x) > *rule.MaxItems {
			return fmt.Errorf("invalid array length")
		}
		for _, v := range x {
			if e := r.check(rule.Items, v); e != nil {
				return e
			}
		}
	case string:
		if !allowed("string") {
			return fmt.Errorf("string prohibited")
		}
		n := utf8.RuneCountInString(x)
		if rule.MinLength != nil && n < *rule.MinLength || rule.MaxLength != nil && n > *rule.MaxLength || !Format(rule.Format, x) {
			return fmt.Errorf("invalid string constraint")
		}
	case bool:
		if !allowed("boolean") {
			return fmt.Errorf("boolean prohibited")
		}
	case json.Number:
		n, e := x.Int64()
		if !allowed("integer") || e != nil {
			return fmt.Errorf("integer required")
		}
		if rule.Minimum != nil && n < *rule.Minimum || rule.Maximum != nil && n > *rule.Maximum {
			return fmt.Errorf("integer outside bounds")
		}
	default:
		return fmt.Errorf("invalid value")
	}
	return nil
}
