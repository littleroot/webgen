# Nausicaä

Write component views and styles in `.html` files. Nausicaä generates
[`webapi`][2] package Go code to construct the components for the
js/wasm architecture.

Inspired by [tomato][1].

For documentation, see below.

For command line help text, including examples, run `nausicaa -h`.

## Install

```
go get github.com/littleroot/nausicaa/cmd/nausicaa
```

## Documentation

### Contents

- [Basics](#basics): An overview
- [The `ref` attribute](#the-ref-attribute): Obtain a reference to an element
- [The `<include>` element](#the-include-element): Composition of components
- [The `Roots` method](#the-roots-method): Access the top-level element(s) of a component

### Basics

Nausicaä generates Go code corresponding to component views
specified in input `.html` files. The generated Go code uses the [`webapi`][2]
package and its subpackages. A component file can optionally include CSS for
the component in a top-level `<style>` element at the end of the file. Nausicaä
generates a single CSS file that is the concatenation of styles from all
input files.

Consider a simple component in `FooBar.html`.

```html
<div class="FooBar"></div>

<style>
.FooBar { font-family: "Inter"; }
</style>
```

Nausicaä generates the Go type for the component and its constructor.
The Go type name is derived from the name of the `.html` file.
If you would like the type and its constructor to be exported, begin the
filename with an uppercase letter, akin to naming an exported
type in Go. (Hint: Use title-case or camel-case for the filenames to generate
idiomatic Go code.)

```go
type FooBar struct {
	roots []*dom.Element
}

func NewFooBar() *FooBar {
	div0 := _document.CreateElement("div", nil)
	div0.SetAttribute("class", "FooBar")
	return &Foo{
		roots: []*dom.Element{div0},
	}
}

func (v *FooBar) Roots() []*dom.Element {
	return v.roots
}
```

As mentioned earlier, Nausicaä also generates CSS output that is the concatenation
of styles from all input component files (in this example, just the single file).

```css
.FooBar { font-family: "Inter"; }
```

Use the `--outviews` and `--outcss` flags to specify the location
to write the generated Go and generates CSS, respectively.

### The `ref` attribute

Refs allow access to an element of your component from Go code. For instance,
you might want a reference to an element in your component in order to set
its `textContent` dynamically or to add an event listener.

```html
<div class="Notification">
	<span ref="Message"></span>
</div>
```

The generated Go type has a field that is a reference to the element. The
field's name is the `ref` attribute's value. Begin the `ref` attribute
value with an uppercase letter to produce an exported field or with a
lowercase letter to produce an unexported field.

```go
type Notification struct {
	Message *html.HTMLSpanElement
	roots   []*dom.Element
}
```

You can then access the element from your application code.

```go
text := "Email archived."

n := NewNotification()
n.Message.SetTextContent(&text)
```

### The `<include>` element

The `<include>` element can be used to include another component by its filepath
in the current component. For example:

```html
<div>
	<include path="path/to/Other.html" />
</div>
```

The contents of `path/to/Other.html` will replace the `<include />` element
in the generated code.

The `path` attribute is required. It can either be a relative path (not starting with `/`)
or an absolute path (starting with `/`). If an absolute path is used, it is rooted at the path specified by the `--root` flag.

Additionally, a [`ref`](#the-ref-attribute) attribute may be specified.

### The `Roots` method

The generated component types satisfy this Go interface.

```go
interface {
	Roots() []*dom.Element
}
```

The `Roots` method returns the top-level elements in a component.
(In many cases, there may only be a single top-level element, in which case
the list will have a length of 1.)

Examples:

```go
// Append the Select component inside a <form>.
form := webapi.GetDocument().CreateElement("ul", nil)
sel := NewSelect()
form.AppendChild(&sel.Roots()[0].Node)
```

```go
// Append the <li> elements from a component as children to a <ul> element.
ul := webapi.GetDocument().CreateElement("ul", nil)
items := NewItems()
for _, r := range items.Roots() {
	ul.AppendChild(&r.Node)
}
```

## License

MIT

[1]: https://github.com/donjaime/tomato
[2]: https://github.com/gowebapi/webapi
