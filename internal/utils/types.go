package utils

import "encoding/json"

type Partial[T any] struct {
	Set   bool
	Value T
}

func (n Partial[T]) MarshalJSON() ([]byte, error) {
	if !n.Set {
		var zero T
		return json.Marshal(zero)
	}
	return json.Marshal(n.Value)
}

func (n *Partial[T]) UnmarshalJSON(data []byte) error {
	n.Set = true
	return json.Unmarshal(data, &n.Value)
}

// func (n Partial[T]) JSONSchemaAlias() any {
// 	var v *T
// 	return v
// }

func NewPartial[T any](value T) Partial[T] {
	return Partial[T]{
		Set:   true,
		Value: value,
	}
}
