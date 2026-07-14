package patch

import "testing"

func TestFieldRepresentsAbsentSetAndClear(t *testing.T) {
	absent := Unchanged[string]()
	if absent.Present() || absent.Value() != nil {
		t.Fatalf("absent=%+v", absent)
	}
	set := Set("value")
	if !set.Present() || set.Value() == nil || *set.Value() != "value" {
		t.Fatalf("set=%+v", set)
	}
	clear := Clear[string]()
	if !clear.Present() || clear.Value() != nil {
		t.Fatalf("clear=%+v", clear)
	}
}
