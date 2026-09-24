package api

import "encoding/json"

// Optional preserves the difference between an omitted JSON property and a
// property explicitly set to null. A pointer alone cannot represent all three
// states needed by PATCH: omitted, null, and a concrete value.
type Optional[T any] struct {
	Present bool
	Value   *T
}

// UnmarshalJSON implements the json.Unmarshaler interface for the Optional type.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	// UnmarshalJSON is called only when the property is present in the request.
	o.Present = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	o.Value = &value
	return nil
}
