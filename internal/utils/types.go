package utils

import (
	"encoding/json"
	"log/slog"
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
	slog.Info("unmarshal called", "data", string(data))
	n.Set = true
	return json.Unmarshal(data, &n.Value)
}

func (n Optional[T]) JSONSchemaAlias() any {
	var v T
	return v
}

func NewPartial[T any](value T) Optional[T] {
	return Optional[T]{
		Set:   true,
		Value: value,
	}
}

// type Nullable[T any] struct {
// 	Value *T
// }

// func (n Nullable[T]) MarshalJSON() ([]byte, error) {
// 	if n.Value == nil {
// 		return nil, nil
// 	}

// 	return json.Marshal(*n.Value)
// }

// func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
// 	return json.Unmarshal(data, &n.Value)
// }

// func (n Nullable[T]) JSONSchema() *jsonschema.Schema {
// 	reflector := &jsonschema.Reflector{}
// 	var zero T
// 	baseSchema := reflector.Reflect(zero)
// 	schema := &jsonschema.Schema{
// 		Extras: map[string]any{"type": []string{baseSchema.Type, "null"}},
// 	}

// 	slog.Info("schema called", "schema", schema)
// 	return schema
// }
