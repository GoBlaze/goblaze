package goblaze

import "testing"

func TestParam(t *testing.T) {
	ctx := &Ctx{
		pnames:  [32]string{"name", "age"},
		pvalues: [32]string{"John", "30"},
	}

	if got := ctx.Param("name"); got != "John" {
		t.Errorf("Param('name') = %v, want %v", got, "John")
	}

	if got := ctx.Param("unknown"); got != "" {
		t.Errorf("Param('unknown') = %v, want %v", got, "")
	}

	if got := ctx.Param("age"); got != "30" {
		t.Errorf("Param('age') = %v, want %v", got, "30")
	}
}
