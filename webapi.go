package nausicaa

import "fmt"

func webapiNames(tagName string) (typeName string, funcName string, ok bool) {
	t, ok := webapiElementToType[tagName]
	if !ok {
		return "", "", false
	}
	typeName = fmt.Sprintf("%s.HTML%sElement", t.Package, t.Type)
	funcName = fmt.Sprintf("%s.HTML%sElementFromJS", t.Package, t.Type)
	return typeName, funcName, true
}

// Obtained from webapi@v0.0.0-20201112202446-44407bcf554b.
var webapiElementToType = map[string]struct {
	Package string
	Type    string
}{
	// "github.com/gowebapi/webapi/html/canvas"
	"canvas": {"canvas", "Canvas"},

	// "github.com/gowebapi/webapi/html/media"
	"audio": {"media", "Audio"},
	"media": {"media", "Media"},
	"track": {"media", "Track"},
	"video": {"media", "Video"},

	// "github.com/gowebapi/webapi/html"
	"a":        {"html", "Anchor"},
	"area":     {"html", "Area"},
	"br":       {"html", "BR"},
	"base":     {"html", "Base"},
	"body":     {"html", "Body"},
	"button":   {"html", "Button"},
	"dl":       {"html", "DList"},
	"data":     {"html", "Data"},
	"datalist": {"html", "DataList"},
	"details":  {"html", "Details"},
	"dialog":   {"html", "Dialog"},
	"dir":      {"html", "Directory"},
	"div":      {"html", "Div"},
	// skip type HTMLElement
	"fieldset": {"html", "FieldSet"},
	"font":     {"html", "Font"},
	"form":     {"html", "Form"},
	"frameset": {"html", "FrameSet"},
	"hr":       {"html", "HR"},
	"head":     {"html", "Head"},
	"h1":       {"html", "Heading"},
	"h2":       {"html", "Heading"},
	"h3":       {"html", "Heading"},
	"h4":       {"html", "Heading"},
	"h5":       {"html", "Heading"},
	"h6":       {"html", "Heading"},
	"html":     {"html", "Html"},
	"img":      {"html", "Image"},
	"input":    {"html", "Input"},
	"li":       {"html", "LI"},
	"label":    {"html", "Label"},
	"legend":   {"html", "Legend"},
	"link":     {"html", "Link"},
	"map":      {"html", "Map"},
	"marquee":  {"html", "Marquee"},
	"menu":     {"html", "Menu"},
	"meta":     {"html", "Meta"},
	"meter":    {"html", "Meter"},
	"mod":      {"html", "Mod"},
	"ol":       {"html", "OList"},
	"optgroup": {"html", "OptGroup"},
	"option":   {"html", "Option"},
	"output":   {"html", "Output"},
	"p":        {"html", "Paragraph"},
	"param":    {"html", "Param"},
	"picture":  {"html", "Picture"},
	"pre":      {"html", "Pre"},
	"progress": {"html", "Progress"},
	"quote":    {"html", "Quote"},
	"script":   {"html", "Script"},
	"select":   {"html", "Select"},
	"slot":     {"html", "Slot"},
	"source":   {"html", "Source"},
	"span":     {"html", "Span"},
	"style":    {"html", "Style"},
	"caption":  {"html", "TableCaption"},
	"td":       {"html", "TableCell"},
	"colgroup": {"html", "TableCol"},
	"table":    {"html", "Table"},
	"tr":       {"html", "TableRow"},
	"tbody":    {"html", "TableSection"},
	"template": {"html", "Template"},
	"textarea": {"html", "TextArea"},
	"time":     {"html", "Time"},
	"title":    {"html", "Title"},
	"ul":       {"html", "UList"},
	// skip type HTMLUnknownElement
}
