package nausicaa

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/net/html"
)

type Options struct {
	Package string // output package name
	Root    string // directory where absolute paths are rooted
}

func Generate(out io.Writer, input []string, opts *Options) error {
	g := &generator{
		out:  out,
		opts: opts,
		seen: make(map[string]struct{}),
	}
	return g.run(input)
}

type generator struct {
	out  io.Writer
	opts *Options
	seen map[string]struct{}
}

func (g *generator) run(input []string) error {
	for _, i := range input {
		if err := g.runOne(i); err != nil {
			return err
		}
	}
	return nil
}

func (g *generator) runOne(input string) error {
	_, ok := g.seen[input]
	if ok {
		return nil
	}
	g.seen[input] = struct{}{}

	f, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("open %s: %s", input, err)
	}
	defer f.Close()

	z := html.NewTokenizer(f)
	_ = z
	return nil
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

var (
	space = []byte{' '}
)
