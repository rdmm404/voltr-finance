package utils

import (
	"encoding/json"
)

type Optional[T any] struct {
	Set   bool
	Value T
}

func (n Optional[T]) MarshalJSON() ([]byte, error) {
	if !n.Set {
		var zero T
		return json.Marshal(zero)
	}
	return json.Marshal(n.Value)
}

func (n *Optional[T]) UnmarshalJSON(data []byte) error {
	n.Set = true
	return json.Unmarshal(data, &n.Value)
}

func NewOptional[T any](value T) Optional[T] {
	return Optional[T]{
		Set:   true,
		Value: value,
	}
}
