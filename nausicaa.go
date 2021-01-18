package nausicaa

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

type stack struct {
	s []string
}

func (st *stack) push(v string) {
	st.s = append(st.s, v)
}

func (st *stack) pop() string {
	v := st.s[len(st.s)-1]
	st.s = st.s[:len(st.s)-1]
	return v
}

func (st *stack) len() int {
	return len(st.s)
}

func (st *stack) peek() (string, bool) {
	if st.len() == 0 {
		return "", false
	}
	return st.s[len(st.s)-1], true
}

type orderedSet struct {
	m map[string]struct{}
	s []string
}

func newOrderedSet() *orderedSet {
	return &orderedSet{
		m: make(map[string]struct{}),
	}
}

func (o *orderedSet) add(v string) {
	_, ok := o.m[v]
	if ok {
		return
	}
	o.m[v] = struct{}{}
	o.s = append(o.s, v)
}

func (o *orderedSet) remove(v string) {
	_, ok := o.m[v]
	if !ok {
		return
	}

	delete(o.m, v)

	var i int
	for i = range o.s {
		if o.s[i] == v {
			break
		}
	}
	copy(o.s[i:], o.s[i+1:])
	o.s[len(o.s)-1] = ""
	o.s = o.s[:len(o.s)-1]
}

func (o *orderedSet) has(v string) bool {
	_, ok := o.m[v]
	return ok
}

type Options struct {
	Package string // output package name
	Root    string // root directory for absolute paths in <include /> elements
}

func Generate(inputFiles []string, opts Options) (viewOuts, cssOut []byte, err error) {
	g := &generator{
		opts: opts,
		seen: make(map[string]struct{}),
	}
	return g.run(inputFiles)
}

type generator struct {
	opts Options

	seen             map[string]struct{}
	viewsBuf, cssBuf bytes.Buffer
}

func (g *generator) run(input []string) ([]byte, []byte, error) {
	err := viewsHeaderTpl.Execute(&g.viewsBuf, g.opts.Package)
	if err != nil {
		panic(err) // code bug: check template args?
	}

	for _, p := range input {
		err := g.generateOneFile(p)
		if err != nil {
			return nil, nil, err
		}
	}

	// // Run through gofmt-style formatting.
	// views, err := format.Source(g.viewsBuf.Bytes())
	// if err != nil {
	// 	panic(err) // code bug: we may have generated bad code
	// }
	// css, err := format.Source(g.cssBuf.Bytes())
	// if err != nil {
	// 	panic(err) // code bug: we may have generated bad code
	// }

	return g.viewsBuf.Bytes(), g.cssBuf.Bytes(), nil
}

func (g *generator) generateOneFile(path string) error {
	_, ok := g.seen[path]
	if ok {
		return nil // already generated
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	err = g.generateComponent(f, path, newOrderedSet())
	if err != nil {
		return err
	}

	g.seen[path] = struct{}{}
	return nil
}

func (g *generator) generateComponent(in io.Reader, path string, history *orderedSet) (err error) {
	if history.has(path) {
		panic("import cycle") // TODO
	}

	history.add(path)
	defer history.remove(path)

	typeName := componentTypeName(filepath.Base(path))
	funcName := constructorFuncName(typeName)

	var typeBuf, funcBuf bytes.Buffer
	typeBuf.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
	funcBuf.WriteString(fmt.Sprintf("func %s() {\n", funcName))

	namer := newVarNames()
	_ = namer

	z := html.NewTokenizer(in)
	var names stack
	var inStyle bool

tokenizeView:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				break tokenizeView
			}
			return z.Err()

		case html.TextToken:
			if names.len() == 0 {
				// TODO: log a warning?
				continue
			}
			text := formatTextContent(z.Text())
			if len(text) == 0 {
				continue
			}
			parentName, _ := names.peek()
			strName := namer.next("_stringliteral")
			fmt.Fprintf(&funcBuf, "const %s = %q\n", strName, text)
			fmt.Fprintf(&funcBuf, "%s.SetTextContent(&%s)\n", parentName, strName)

		case html.StartTagToken:
			tn, _ := z.TagName()

			name := namer.next(string(tn))
			names.push(name)

			if isStyleTag(tn) {
				inStyle = true
				break tokenizeView
			}

			if isIncludeTag(tn) {
			}

			fmt.Fprintf(&funcBuf, "%s = __document.CreateElement(%q, nil)\n", name, tn)
			// TODO: attrs

		case html.EndTagToken:
			name := names.pop()
			parentName, ok := names.peek()
			if ok {
				fmt.Fprintf(&funcBuf, "%s.AppendChild(&%s.Node)\n", parentName, name)
			}

		case html.SelfClosingTagToken:
			// TODO

		case html.CommentToken:
			// ignore
		case html.DoctypeToken:
			// ignore
		}
	}

	typeBuf.WriteString("}\n\n")
	funcBuf.WriteString("}\n\n")

	// Add view output to the overall output.
	io.Copy(&g.viewsBuf, &typeBuf)
	io.Copy(&g.viewsBuf, &funcBuf)

	if inStyle {
		var css bytes.Buffer

		// TODO: write the CSS filename to make it easy to know where
		// the generated CSS originates from.
		io.Copy(&g.cssBuf, &css)
		g.cssBuf.WriteString("\n\n")
	}

	return nil
}

// varNames returns successive variable names to use in a component.
type varNames struct {
	m map[string]int
}

func newVarNames() varNames {
	return varNames{
		m: make(map[string]int),
	}
}

func (v *varNames) next(tagName string) string {
	n := v.m[tagName]
	v.m[tagName]++
	return fmt.Sprintf("%s%d", tagName, n)
}

func isIncludeTag(tn []byte) bool {
	return len(tn) == 7 &&
		tn[0] == 'i' &&
		tn[1] == 'n' &&
		tn[2] == 'c' &&
		tn[3] == 'l' &&
		tn[4] == 'u' &&
		tn[5] == 'd' &&
		tn[6] == 'e'
}

func isStyleTag(tn []byte) bool {
	return len(tn) == 5 &&
		tn[0] == 's' &&
		tn[1] == 't' &&
		tn[2] == 'y' &&
		tn[3] == 'l' &&
		tn[4] == 'e'
}

var (
	newline = []byte{'\n'}
)

func componentTypeName(filename string) string {
	// Remove what we assume to be the extension.
	idx := strings.LastIndex(filename, ".")
	if idx != -1 {
		filename = filename[:idx]
	}

	return filename
}

func toUppperFirstRune(n string) string {
	r, i := utf8.DecodeRuneInString(n)
	if i == 0 {
		return n
	}
	return string([]rune{unicode.ToUpper(r)}) + n[i:]
}

func isExportedName(name string) bool {
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

func constructorFuncName(typeName string) string {
	if isExportedName(typeName) {
		return "New" + typeName
	}
	return "new" + toUppperFirstRune(typeName)
}

func attrsFunc(z *html.Tokenizer, f func(k, v []byte)) {
	for {
		k, v, more := z.TagAttr()
		if !more {
			break
		}
		f(k, v)
	}
}

const viewsHeader = `
package {{.}}

// Code generated by nausicaa. DO NOT EDIT.

import (
	"github.com/gowebapi/webapi"
	"github.com/gowebapi/webapi/dom"
)

type (
	_ *webapi.Document // prevent unused import errors
	_ *dom.Element
)

var (
	__document = webapi.GetDocument()
)
`

var viewsHeaderTpl = template.Must(template.New("").Parse(viewsHeader))

func isSpaceExceptNBSP(r rune) bool {
	if r == 0xA0 { // NBSP
		return false
	}
	return unicode.IsSpace(r)
}

func formatTextContent(b []byte) []byte {
	b = bytes.ReplaceAll(b, newline, nil)
	b = bytes.TrimFunc(b, isSpaceExceptNBSP)
	return b
}
