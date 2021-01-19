package nausicaa

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/diff"
)

func TestGenerateStandalone(t *testing.T) {
	files := []string{
		"attrs",
		"Exported",
		"multipleRoots",
		"nested",
		"ref",
		"selfClosing",
		"specificElement",
		"style",
		"styleOnly",
		"textContent",
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
			path := filepath.Join("testdata", "standalone", f+".html")

			expectv, err := ioutil.ReadFile(filepath.Join("testdata", "golden", "standalone", f+".golden.go"))
			Ok(t, err)
			expectc, err := ioutil.ReadFile(filepath.Join("testdata", "golden", "standalone", f+".golden.css"))
			Ok(t, err)

			gotv, gotc, err := g.run([]string{path})
			Ok(t, err)
			EqualBytes(t, expectv, gotv, bytes.TrimSpace)
			EqualBytes(t, expectc, gotc, bytes.TrimSpace)
		})
	}
}

func TestGenerateInclude(t *testing.T) {
	files := [][2]string{
		{"absolutePath", "testdata"},
		{"includeMultipleRoots", ""},
		{"multilevel", ""},
		{"ref", ""},
		{"relativePath", ""},
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
		t.Run(f[0], func(t *testing.T) {
			g.reset()
			if f[1] == "" {
				g.opts.Root = ""
			} else {
				g.opts.Root = f[1]
			}
			path := filepath.Join("testdata", "include", f[0]+".html")

			expectv, err := ioutil.ReadFile(filepath.Join("testdata", "golden", "include", f[0]+".golden.go"))
			Ok(t, err)

			gotv, _, err := g.run([]string{path})
			Ok(t, err)
			EqualBytes(t, expectv, gotv, bytes.TrimSpace)
		})
	}
}

func TestGenerateError(t *testing.T) {
	testcases := []struct {
		filename string
		err      string
	}{
		// TODO: do something correct for this test case, then uncomment.
		// right now, format.Source() panics.
		// {"badHTML", ""},
		{"cycle0Include", "cycle in include paths (cycle0Include.html -> cycle1Include.html -> cycle2Include.html -> cycle0Include.html)"},
		{"disallowedRefNameKeyword", `ref name "select" disallowed (Go keyword)`},
		{"disallowedRefNameRoots", `ref name "roots" disallowed (reserved for internal use)`},
		{"invalidAttrInclude", `<include> specifies invalid attribute "foo"`},
		{"missingPathAttrInclude", `missing required "path" attribute in <include>`},
		{"repeatedRef", `ref name "foo" present multiple times (previous occurence in <div>)`},
		{"topLevelInclude", `top-level <include> disallowed  (hint: nest in <div> or <span>)`},
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

	for _, tt := range testcases {
		t.Run(tt.filename, func(t *testing.T) {
			g.reset()
			path := filepath.Join("testdata", "error", tt.filename+".html")

			_, _, err := g.run([]string{path})
			if !strings.HasSuffix(err.Error(), tt.err) {
				t.Errorf("expected err to end with: %q, got: %q", tt.err, err.Error())
			}
		})
	}
}

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
