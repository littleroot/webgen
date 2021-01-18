package nausicaa

import (
	"testing"
)

func TestToUppperFirstRune(t *testing.T) {
	testcases := []struct {
		in, expect string
	}{
		{"fooBar", "FooBar"},
		{"FooBar", "FooBar"},
		{"", ""},
		{"f", "F"},
	}

	for _, tt := range testcases {
		t.Run(tt.in, func(t *testing.T) {
			got := toUppperFirstRune(tt.in)
			if got != tt.expect {
				t.Errorf("expected: %s, got: %s", tt.expect, got)
			}
		})
	}
}

func TestVarNamer(t *testing.T) {
	namer := newVarNames()

	Equal(t, "div0", namer.next("div"))
	Equal(t, "div1", namer.next("div"))
	Equal(t, "span0", namer.next("span"))
	Equal(t, "img0", namer.next("img"))
	Equal(t, "div2", namer.next("div"))
	Equal(t, "img1", namer.next("img"))
}

func Equal(t *testing.T, expect, got string) {
	t.Helper()
	if got != expect {
		t.Errorf("expected: %s, got: %s", expect, got)
	}
}
