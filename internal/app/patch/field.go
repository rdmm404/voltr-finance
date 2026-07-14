// Package patch defines application-owned tri-state mutation fields.
package patch

// Field represents an update that is absent, sets a value, or clears it.
// A present field with a nil value is the clear state.
type Field[T any] struct {
	present bool
	value   *T
}

func Unchanged[T any]() Field[T]  { return Field[T]{} }
func Set[T any](value T) Field[T] { return Field[T]{present: true, value: &value} }
func Clear[T any]() Field[T]      { return Field[T]{present: true} }

func (f Field[T]) Present() bool { return f.present }
func (f Field[T]) Value() *T     { return f.value }
