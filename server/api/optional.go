package api

import "encoding/json"

type Optional[T any] struct {
	Present bool
	Value   *T
}

// UnmarshalJSON implements the json.Unmarshaler interface for the Optional type.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
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
