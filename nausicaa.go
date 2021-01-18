package nausicaa

import (
	"bytes"
	"fmt"
	"go/format"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

type TagAndVarName struct {
	TagName string
	VarName string
}

type stack struct {
	s []TagAndVarName
}

func (st *stack) push(v TagAndVarName) {
	st.s = append(st.s, v)
}

func (st *stack) pop() TagAndVarName {
	v := st.s[len(st.s)-1]
	st.s = st.s[:len(st.s)-1]
	return v
}

func (st *stack) len() int {
	return len(st.s)
}

func (st *stack) peek() (TagAndVarName, bool) {
	if st.len() == 0 {
		return TagAndVarName{}, false
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
		err := g.generateOneFile(p, newOrderedSet())
		if err != nil {
			return nil, nil, err
		}
	}

	// return g.viewsBuf.Bytes(), g.cssBuf.Bytes(), nil

	// Run through gofmt-style formatting.
	views, err := format.Source(g.viewsBuf.Bytes())
	if err != nil {
		panic(err) // code bug: we may have generated bad code
	}
	css, err := format.Source(g.cssBuf.Bytes())
	if err != nil {
		panic(err) // code bug: we may have generated bad code
	}
	return views, css, nil
}

func (g *generator) generateOneFile(path string, history *orderedSet) error {
	_, ok := g.seen[path]
	if ok {
		return nil // already generated
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	err = g.generateComponent(f, path, history)
	if err != nil {
		return err
	}

	g.seen[path] = struct{}{}
	return nil
}

var disallowedRefs = map[string]struct{}{
	"roots": {},
}

type TagAndVarAndTypeName struct {
	TagName  string
	VarName  string
	TypeName string
}

func (g *generator) generateComponent(in io.Reader, path string, history *orderedSet) (err error) {
	if history.has(path) {
		return fmt.Errorf("cyclical include") // TODO
	}

	history.add(path)
	defer history.remove(path)

	typeName := componentTypeName(filepath.Base(path))
	funcName := constructorFuncName(typeName)

	var funcBuf bytes.Buffer
	fmt.Fprintf(&funcBuf, "func %s() *%s {", funcName, typeName)

	z := html.NewTokenizer(in)
	namer := newVarNames()

	var names stack                               // also used to record depth
	var insideStyle bool                          // whether we break out inside top-level <style>
	refs := make(map[string]TagAndVarAndTypeName) // ref attribute value -> names
	addNewline := false                           // cosmetic whitespace
	var roots []string                            // roots var names

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
				// text node without parent
				// TODO: log a warning?
				continue
			}
			text := formatTextContent(z.Text())
			if len(text) == 0 {
				continue
			}
			parent, _ := names.peek()
			strName := namer.next("stringliteral")
			fmt.Fprintf(&funcBuf, "const %s = %q\n", strName, text)
			fmt.Fprintf(&funcBuf, "%s.SetTextContent(&%s)\n", parent.VarName, strName)

		case html.StartTagToken:
			tn, hasAttr := z.TagName()
			tagName := string(tn)
			varName := namer.next(tagName)

			if tagName == "style" && names.len() == 0 {
				names.push(TagAndVarName{tagName, varName})
				insideStyle = true
				break tokenizeView
			}

			names.push(TagAndVarName{tagName, varName})
			if !addNewline {
				addNewline = true
			} else {
				fmt.Fprint(&funcBuf, "\n")
			}

			if tagName == "include" {
				var foundPathAttr bool
				var refAttrVal string
				var includeTypeName string

				err := attrsFunc(z, hasAttr, func(k, v []byte) error {
					isRef := equalsRef(k)
					isPath := equalsPath(k)

					// validate attributes
					if !isRef && !isPath {
						return fmt.Errorf("<include> specifies invalid attribute %q", k)
					}

					if isRef {
						if _, ok := disallowedRefs[string(v)]; ok {
							return fmt.Errorf("ref name %q disallowed (reserved for internal use)", v)
						}
						refAttrVal = string(v)
					} else {
						foundPathAttr = true

						v := string(v)
						var includePath string
						if filepath.IsAbs(v) {
							includePath = filepath.Join(g.opts.Root, v)
						} else {
							includePath = filepath.Join(filepath.Dir(path), v)
						}
						err := g.generateOneFile(includePath, history)
						if err != nil {
							return err
						}
						// ... successfully included; append it
						includeTypeName = componentTypeName(filepath.Base(includePath))
						includeConstructorFuncName := constructorFuncName(includeTypeName)
						fmt.Fprintf(&funcBuf, "%s := %s()\n", varName, includeConstructorFuncName)
					}
					return nil
				})
				if err != nil {
					return err
				}

				if !foundPathAttr {
					return fmt.Errorf("missing required \"path\" attribute in <include>")
				}
				if refAttrVal != "" {
					ex, ok := refs[refAttrVal]
					if ok {
						return fmt.Errorf("ref %q appears multiple times in component (previous occurence in <%s>)", refAttrVal, ex.TagName)
					}
					refs[refAttrVal] = TagAndVarAndTypeName{tagName, varName, includeTypeName}
				}

				continue
			}

			fmt.Fprintf(&funcBuf, "%s := _document.CreateElement(%q, nil)\n", varName, tagName)
			attrsFunc(z, hasAttr, func(k, v []byte) error {
				if equalsRef(k) {
					v := string(v)
					if _, ok := disallowedRefs[v]; ok {
						return fmt.Errorf("ref name %q disallowed (reserved for internal use)", v)
					}
					ex, ok := refs[v]
					if ok {
						return fmt.Errorf("ref %q appears multiple times in component (previous occurence in <%s>)", v, ex.TagName)
					}
					refs[v] = TagAndVarAndTypeName{tagName, varName, ""}
					return nil
				}
				fmt.Fprintf(&funcBuf, "%s.SetAttribute(%q, %q)\n", varName, k, v)
				return nil
			})

		case html.EndTagToken:
			curr := names.pop()
			parent, ok := names.peek()
			if !ok {
				// no parent; record as root
				roots = append(roots, curr.VarName)
				continue
			}
			if curr.TagName == "include" {
				fmt.Fprintf(&funcBuf, "for _, r := range %s.roots {\n", curr.VarName)
				fmt.Fprintf(&funcBuf, "%s.AppendChild(&r.Node)\n", parent.VarName)
				fmt.Fprintf(&funcBuf, "}\n")
			} else {
				fmt.Fprintf(&funcBuf, "%s.AppendChild(&%s.Node)\n", parent.VarName, curr.VarName)
			}

		case html.SelfClosingTagToken:
			// TODO

		case html.CommentToken:
			// ignore
		case html.DoctypeToken:
			// ignore
		}
	}

	fmt.Fprintf(&funcBuf, "\n\nreturn &%s{\n", typeName)
	for k, r := range refs {
		fmt.Fprintf(&funcBuf, "%s: %s,\n", k, r.VarName)
	}
	fmt.Fprintf(&funcBuf, "roots: []*dom.Element{%s},\n", strings.Join(roots, ", "))
	fmt.Fprint(&funcBuf, "}")
	fmt.Fprint(&funcBuf, "}\n\n")

	var typeBuf bytes.Buffer
	fmt.Fprintf(&typeBuf, "type %s struct {", typeName)
	for k, v := range refs {
		typeName := "*dom.Element" // TODO: make this more specific (like *html.HTMLDomElement)
		if v.TypeName != "" {
			typeName = "*" + v.TypeName
		}
		fmt.Fprintf(&typeBuf, "%s %s\n", k, typeName)
	}
	fmt.Fprint(&typeBuf, "roots []*dom.Element\n")
	fmt.Fprint(&typeBuf, "}\n\n")

	// Add view output to the overall output.
	io.Copy(&g.viewsBuf, &typeBuf)
	io.Copy(&g.viewsBuf, &funcBuf)

	if insideStyle {
		// TODO: write the CSS filename to make it easy to know where
		// the generated CSS originates from.
		var css bytes.Buffer

		fmt.Fprintf(&css, "}\n\n")
		io.Copy(&g.cssBuf, &css)
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

func (v *varNames) next(kind string) string {
	n := v.m[kind]
	v.m[kind]++
	return fmt.Sprintf("%s%d", kind, n)
}

func equalsRef(k []byte) bool {
	return len(k) == 3 &&
		k[0] == 'r' &&
		k[1] == 'e' &&
		k[2] == 'f'
}

func equalsPath(k []byte) bool {
	return len(k) == 4 &&
		k[0] == 'p' &&
		k[1] == 'a' &&
		k[2] == 't' &&
		k[3] == 'h'
}

var (
	newline = []byte{'\n'}
	slash   = []byte{'/'}
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

func attrsFunc(z *html.Tokenizer, hasAttr bool, f func(k, v []byte) error) error {
	for hasAttr {
		var k, v []byte
		k, v, hasAttr = z.TagAttr()
		if err := f(k, v); err != nil {
			return err
		}
	}
	return nil
}

type KV struct {
	K string
	V string
}

func attrs(z *html.Tokenizer, hasAttr bool) []KV {
	var kvs []KV
	attrsFunc(z, hasAttr, func(k, v []byte) error {
		kvs = append(kvs, KV{string(k), string(v)})
		return nil
	})
	return kvs
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
	_document = webapi.GetDocument()
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
