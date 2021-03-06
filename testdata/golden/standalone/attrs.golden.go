package ui

// Code generated by webgen. DO NOT EDIT.

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

// source: testdata/standalone/attrs.html

type attrs struct {
	roots []*dom.Element
}

func newAttrs() *attrs {
	input0 := _document.CreateElement("input", nil)
	input0.SetAttribute("type", "text")
	input0.SetAttribute("class", "foo")
	return &attrs{
		roots: []*dom.Element{input0},
	}
}

func (v *attrs) Roots() []*dom.Element {
	return v.roots
}
