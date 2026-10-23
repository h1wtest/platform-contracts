package wire

import "encoding/json"

// Optional distinguishes an omitted PATCH field, explicit null, and a value.
// Generated omitzero tags omit only Present=false, never an explicit null.
type Optional[T any] struct {
	Present bool
	Null    bool
	Value   T
}

func (v Optional[T]) IsZero() bool { return !v.Present }
func (v Optional[T]) MarshalJSON() ([]byte, error) {
	if v.Null {
		return []byte("null"), nil
	}
	return json.Marshal(v.Value)
}
func (v *Optional[T]) UnmarshalJSON(data []byte) error {
	v.Present = true
	if string(data) == "null" {
		v.Null = true
		var zero T
		v.Value = zero
		return nil
	}
	v.Null = false
	return json.Unmarshal(data, &v.Value)
}
