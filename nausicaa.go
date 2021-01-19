package nausicaa

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

type tagAndVarName struct {
	TagName string
	VarName string
}

type stack struct {
	s []tagAndVarName
}

func (st *stack) push(v tagAndVarName) {
	st.s = append(st.s, v)
}

func (st *stack) pop() tagAndVarName {
	v := st.s[len(st.s)-1]
	st.s = st.s[:len(st.s)-1]
	return v
}

func (st *stack) len() int {
	return len(st.s)
}

func (st *stack) peek() (tagAndVarName, bool) {
	if st.len() == 0 {
		return tagAndVarName{}, false
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

func (o *orderedSet) forEach(f func(string)) {
	for _, v := range o.s {
		f(v)
	}
}

type Options struct {
	Package string // output package name
	Root    string // root directory for absolute paths in <include /> elements
}

func Generate(inputFiles []string, opts Options) (viewsOut, cssOut []byte, err error) {
	g := &generator{
		opts:      opts,
		generated: make(map[string]struct{}),
	}
	return g.run(inputFiles)
}

type generator struct {
	opts Options

	generated        map[string]struct{}
	viewsBuf, cssBuf bytes.Buffer
}

func (g *generator) run(input []string) ([]byte, []byte, error) {
	err := viewsHeaderTpl.Execute(&g.viewsBuf, g.opts.Package)
	if err != nil {
		panic(err) // code bug: check template args?
	}

	fmt.Fprint(&g.cssBuf, "/* Code generated by nausicaa. DO NOT EDIT. */\n\n")

	for _, p := range input {
		err := g.generateOneFile(p, newOrderedSet())
		if err != nil {
			return nil, nil, err
		}
	}

	// Run through gofmt-style formatting.
	views, err := format.Source(g.viewsBuf.Bytes())
	if err != nil {
		panic(err) // code bug: we may have generated bad code
	}

	return views, g.cssBuf.Bytes(), nil
}

func (g *generator) generateOneFile(path string, history *orderedSet) error {
	_, ok := g.generated[path]
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

	g.generated[path] = struct{}{}
	return nil
}

func isDisallowedRefName(name string) bool {
	return token.IsKeyword(name) || name == "roots"
}

type tagAndVarAndTypeName struct {
	TagName  string
	VarName  string
	TypeName string
}

func errDisallowedRefName(path, ref string) error {
	return fmt.Errorf("%s: ref name %q disallowed", path, ref)
}

func errRepeatedRefName(path, ref, prevTagName string) error {
	return fmt.Errorf("%s: ref name %q present multiple times (previous occurence in <%s>)", path, ref, prevTagName)
}

func (g *generator) generateComponent(in io.Reader, path string, history *orderedSet) (err error) {
	if history.has(path) {
		var cycle []string
		history.forEach(func(v string) {
			cycle = append(cycle, filepath.Base(v))
		})
		cycle = append(cycle, filepath.Base(path))
		return fmt.Errorf("%s: cycle in include paths (%s)", path, strings.Join(cycle, " -> "))
	}

	history.add(path)
	defer history.remove(path)

	typeName := componentTypeName(filepath.Base(path))
	funcName := constructorFuncName(typeName)

	var funcBuf bytes.Buffer
	fmt.Fprintf(&funcBuf, "func %s() *%s {\n", funcName, typeName)

	z := html.NewTokenizer(in)
	namer := newVarNames()

	var names stack                               // also used to record depth
	var insideStyle bool                          // whether we break out inside top-level <style>
	refs := make(map[string]tagAndVarAndTypeName) // ref attribute value -> names
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
				names.push(tagAndVarName{tagName, varName})
				insideStyle = true
				break tokenizeView
			}

			names.push(tagAndVarName{tagName, varName})

			err := g.handleStartToken(&funcBuf, z, path, tagName, varName, hasAttr, refs, history)
			if err != nil {
				return err
			}

		case html.EndTagToken:
			curr := names.pop()
			g.handleEndToken(&funcBuf, curr.TagName, curr.VarName, &names,
				func(root string) { roots = append(roots, root) })

		case html.SelfClosingTagToken:
			tn, hasAttr := z.TagName()
			tagName := string(tn)
			varName := namer.next(tagName)

			err := g.handleStartToken(&funcBuf, z, path, tagName, varName, hasAttr, refs, history)
			if err != nil {
				return err
			}

			g.handleEndToken(&funcBuf, tagName, varName, &names,
				func(root string) { roots = append(roots, root) })

		case html.CommentToken, html.DoctypeToken:
			// ignore
		}
	}

	writeReturn(&funcBuf, typeName, refs, roots)
	fmt.Fprint(&funcBuf, "\n}\n\n")

	// Add this view's output to the overall output.
	writeTypeDefinition(&g.viewsBuf, path, typeName, refs)
	io.Copy(&g.viewsBuf, &funcBuf)

	if insideStyle {
		if z.Next() != html.TextToken {
			return fmt.Errorf("%s: cannot find <style> text", path)
		}
		fmt.Fprintf(&g.cssBuf, "/* source: %s */\n%s\n\n", path, bytes.TrimSpace(z.Text()))
		// NOTE: We dont't check for the end </style> tag.
	}

	return nil
}

func (g *generator) handleStartToken(w io.Writer, z *html.Tokenizer,
	path, tagName, varName string, hasAttr bool,
	refs map[string]tagAndVarAndTypeName, history *orderedSet) error {

	if tagName == "include" {
		return g.handleStartInclude(w, z, path, tagName, varName, hasAttr, refs, history)
	}
	return g.handleStartRegular(w, z, path, tagName, varName, hasAttr, refs)
}

func (g *generator) handleStartRegular(w io.Writer, z *html.Tokenizer,
	path, tagName, varName string, hasAttr bool,
	refs map[string]tagAndVarAndTypeName) error {

	fmt.Fprintf(w, "%s := _document.CreateElement(%q, nil)\n", varName, tagName)
	err := attrsFunc(z, hasAttr, func(k, v []byte) error {
		if equalsRef(k) {
			v := string(v)
			if isDisallowedRefName(v) {
				return errDisallowedRefName(path, v)
			}
			ex, ok := refs[v]
			if ok {
				return errRepeatedRefName(path, v, ex.TagName)
			}
			refs[v] = tagAndVarAndTypeName{tagName, varName, ""}
			return nil
		}
		fmt.Fprintf(w, "%s.SetAttribute(%q, %q)\n", varName, k, v)
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (g *generator) handleStartInclude(w io.Writer, z *html.Tokenizer,
	path, tagName, varName string, hasAttr bool,
	refs map[string]tagAndVarAndTypeName, history *orderedSet) error {

	var foundPathAttr bool
	var refAttrVal string
	var includeTypeName string

	err := attrsFunc(z, hasAttr, func(k, v []byte) error {
		isRef := equalsRef(k)
		isPath := equalsPath(k)

		// validate attributes
		if !isRef && !isPath {
			return fmt.Errorf("%s: <include> specifies invalid attribute %q", path, k)
		}

		if isRef {
			val := string(v)
			if isDisallowedRefName(val) {
				return errDisallowedRefName(path, val)
			}
			refAttrVal = val
			return nil
		}

		foundPathAttr = true
		val := string(v)

		var includePath string
		if filepath.IsAbs(val) {
			includePath = filepath.Join(g.opts.Root, val)
		} else {
			includePath = filepath.Join(filepath.Dir(path), val)
		}

		err := g.generateOneFile(includePath, history)
		if err != nil {
			return err
		}

		// ... successfully included; construct it
		includeTypeName = componentTypeName(filepath.Base(includePath))
		includeConstructorFuncName := constructorFuncName(includeTypeName)
		fmt.Fprintf(w, "%s := %s()\n", varName, includeConstructorFuncName)
		return nil
	})

	if err != nil {
		return err
	}

	if !foundPathAttr {
		return fmt.Errorf("%s: missing required \"path\" attribute in <include>", path)
	}
	if refAttrVal != "" {
		ex, ok := refs[refAttrVal]
		if ok {
			return errRepeatedRefName(path, refAttrVal, ex.TagName)
		}
		refs[refAttrVal] = tagAndVarAndTypeName{tagName, varName, includeTypeName}
	}
	return nil
}

func (*generator) handleEndToken(w io.Writer, tagName, varName string, names *stack, addRoot func(string)) {
	parent, ok := names.peek()
	if !ok {
		// no parent; record as root
		addRoot(varName)
		return
	}

	if tagName == "include" {
		fmt.Fprintf(w, "for _, r := range %s.roots {\n", varName)
		fmt.Fprintf(w, "%s.AppendChild(&r.Node)\n", parent.VarName)
		fmt.Fprintf(w, "}\n")
	} else {
		fmt.Fprintf(w, "%s.AppendChild(&%s.Node)\n", parent.VarName, varName)
	}
}

func writeReturn(w io.Writer, typeName string, refs map[string]tagAndVarAndTypeName, roots []string) {
	fmt.Fprintf(w, "return &%s{\n", typeName)
	for k, r := range refs {
		if _, f, ok := webapiNames(r.TagName); ok {
			fmt.Fprintf(w, "%s: %s(%s),\n", k, f, r.VarName)
		} else {
			fmt.Fprintf(w, "%s: %s,\n", k, r.VarName)
		}
	}
	fmt.Fprintf(w, "roots: []*dom.Element{%s},\n", strings.Join(roots, ", "))
	fmt.Fprint(w, "}")
}

func writeTypeDefinition(w io.Writer, path, typeName string, refs map[string]tagAndVarAndTypeName) {
	fmt.Fprintf(w, "// source: %s\n", path)
	fmt.Fprintf(w, "type %s struct {\n", typeName)
	for k, v := range refs {
		typeName := "*dom.Element"
		if v.TypeName != "" {
			typeName = "*" + v.TypeName
		} else if t, _, ok := webapiNames(v.TagName); ok {
			typeName = "*" + t
		}
		fmt.Fprintf(w, "%s %s\n", k, typeName)
	}
	fmt.Fprint(w, "roots []*dom.Element\n")
	fmt.Fprint(w, "}\n\n")
}

// varNames returns successive variable names to use in a component's
// "constructor" function.
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

const viewsHeader = `package {{.}}

// Code generated by nausicaa. DO NOT EDIT.

import (
	"github.com/gowebapi/webapi"
	"github.com/gowebapi/webapi/dom"
	"github.com/gowebapi/webapi/html"
	"github.com/gowebapi/webapi/html/canvas"
	"github.com/gowebapi/webapi/html/media"
)

type (
	_ *webapi.Document // prevent unused import errors
	_ *dom.Element
	_ *html.HTMLDivElement
	_ *canvas.HTMLCanvasElement
	_ *media.HTMLAudioElement
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
