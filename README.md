# Nausicaä

Write component views and styles in `.html` files. Nausicaä generates
Go code to construct the components for the js/wasm architecture.

Inspired by [tomato][1].

For documentation, see below.

## Install

```
go get github.com/littleroot/nausicaa/cmd/nausicaa
```

## Documentation

### Basics

The `nausicaa` command outputs Go code to construct your components specified
as HTML. The generated Go code uses the [`webapi`][2] package. Component
files can also specify styles for the component in a top-level `<style>` element
at the end of the file.

Consider a simple component in `FooBar.html`:

```html
<div class="FooBar"></div>

<style>
.FooBar { font-family: "Inter"; }
</style>
```

Running `nausicaa` generates the Go type for the component and its constructor.
The Go type name is derived from the filename of the component's `.html` file.
If you would like the type and its constructor to be exported, begin the
filename with an uppercase letter, akin to how you would name an exported
type in Go. (Hint: Use title-case or camel-case for the filenames to generate
idiomatic Go code.)

```go
type FooBar struct {
	Roots []*dom.Element
}

func NewFooBar() *FooBar {
	div0 := _document.CreateElement("div", nil)
	div0.SetAttribute("class", "FooBar")
	return &Foo{
		Roots: []*dom.Element{div0},
	}
}
```

It also generates CSS output that is the concatenation of
styles from all input component files (in this case, just the single file).

```css
.FooBar { font-family: "Inter"; }
```

### The `ref` attribute

TODO

### The `<include>` element

TODO

## License

MIT

[1]: https://github.com/donjaime/tomato
[2]: https://github.com/gowebapi/webapi
