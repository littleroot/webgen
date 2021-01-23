# Nausicaä

Write component views and styles in `.html` files. Nausicaä generates
Go code to construct the components for the js/wasm architecture.

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
```

As mentioned earlier, Nausicaä also generates CSS output that is the concatenation
of styles from all input component files (in this case, just the single file).

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

TODO

### The `Roots` method

TODO

## License

MIT

[1]: https://github.com/donjaime/tomato
[2]: https://github.com/gowebapi/webapi
