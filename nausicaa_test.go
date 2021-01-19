package nausicaa

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kylelemons/godebug/diff"
)

func TestGenerate(t *testing.T) {
	files := []string{
		"attrs",
		"Exported",
		"multiple_roots",
		"nested",
		"ref",
		"self_closing",
		"specific_element",
		"style",
		"style_only",
		"text_content",
		"unexported",
	}

	g := generator{
		opts: Options{
			Package: "ui",
		},
		generated: make(map[string]struct{}),
		open: func(name string) (io.ReadCloser, error) {
			return os.Open(name)
		},
	}

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			g.reset()
			path := filepath.Join("testdata", "single", f+".html")

			expectv, err := ioutil.ReadFile(filepath.Join("testdata", "golden", "single", f+".golden.go"))
			Ok(t, err)
			expectc, err := ioutil.ReadFile(filepath.Join("testdata", "golden", "single", f+".golden.css"))
			Ok(t, err)

			gotv, gotc, err := g.run([]string{path})
			Ok(t, err)
			EqualBytes(t, expectv, gotv, bytes.TrimSpace)
			EqualBytes(t, expectc, gotc, bytes.TrimSpace)
		})
	}
}

func TestGenerateError(t *testing.T) {} // TODO

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

func EqualBytes(t *testing.T, expect, got []byte, normalize func([]byte) []byte) {
	t.Helper()
	e := normalize(expect)
	g := normalize(got)
	if !bytes.Equal(e, g) {
		t.Errorf("%s", diff.Diff(string(e), string(g)))
	}
}

func Ok(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}
