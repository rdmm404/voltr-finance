package errors

import (
	stderrors "errors"
	"testing"
)

func TestErrorKindsAndSafeNormalization(t *testing.T) {
	validation := Validation("bad input")
	if !IsKind(validation, KindValidation) || CodeOf(validation) != CodeValidation {
		t.Fatalf("validation error = %#v", validation)
	}
	cause := stderrors.New("sql secret")
	internal := Normalize(cause)
	if !IsKind(internal, KindInternal) || MessageOf(internal) != "internal error" || !stderrors.Is(internal, cause) {
		t.Fatalf("internal error = %#v", internal)
	}
}
