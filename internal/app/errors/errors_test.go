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

	wrapped := WrapInternal("load budget report", internal)
	operation, causeType := Diagnostic(wrapped)
	if operation != "load budget report" || causeType != "*errors.errorString" || !stderrors.Is(wrapped, cause) {
		t.Fatalf("diagnostic operation=%q causeType=%q error=%#v", operation, causeType, wrapped)
	}
}
