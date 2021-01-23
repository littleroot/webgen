package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/littleroot/webgen"
)

const (
	permDir  = 0775 // drwxrwxr-x.
	permFile = 0664 // -rw-rw-r--.
)

var (
	stderr = log.New(os.Stderr, "", 0)
)

const usage = `
Generate webapi package Go code for the js/wasm architecture from components
defined in HTML.

Usage:
   webgen [--outcss=<file>] [--outviews=<file>] [--package=<name>]
          [--root=<dir>] <input-file>...
   webgen (-h | --help)

Flags:
   -h --help           Print help and exit
   --outcss=<file>     Write CSS output to specified file instead of stdout
   --outviews=<file>   Write views output to specified file instead of stdout
   --package=<name>    Package name to use in output (default: "views")
   --root=<dir>        Root directory for absolute paths in <include />
                       elements (default: ".")

Example:
   webgen --package=ui \
          --outviews=ui.go \
          --outcss=public/components.css \
          components/*.html
`

var (
	fHelp        bool
	fOutViews    string
	fOutCSS      string
	fPackageName string
	fRoot        string
)

func printUsage() {
	stderr.Printf("%s", strings.TrimSpace(usage))
}

func main() {
	log.SetFlags(log.Lshortfile)

	flag.BoolVar(&fHelp, "help", false, "")
	flag.BoolVar(&fHelp, "h", false, "")
	flag.StringVar(&fOutViews, "outviews", "", "")
	flag.StringVar(&fOutCSS, "outcss", "", "")
	flag.StringVar(&fPackageName, "package", "views", "")
	flag.StringVar(&fRoot, "root", ".", "")

	flag.Usage = printUsage
	flag.Parse()

	if fHelp {
		printUsage()
		os.Exit(0)
	}

	args := flag.Args()

	if len(args) == 0 {
		printUsage()
		os.Exit(2)
	}

	outViews := os.Stdout
	outCSS := os.Stdout

	if fOutViews != "" {
		outViews = createFile(fOutViews)
		defer outViews.Close()
	}
	if fOutCSS != "" {
		outViews = createFile(fOutCSS)
		defer outViews.Close()
	}

	opts := webgen.Options{
		Package: fPackageName,
		Root:    fRoot,
	}

	views, css, err := webgen.Generate(args, opts)
	if err != nil {
		stderr.Printf("%s", err)
		os.Exit(1)
	}

	if _, err := outViews.Write(views); err != nil {
		stderr.Printf("write output views: %s", err)
	}
	if _, err := outCSS.Write(css); err != nil {
		stderr.Printf("write output CSS: %s", err)
	}
}

func createFile(p string) *os.File {
	err := os.MkdirAll(filepath.Base(p), permDir)
	if err != nil {
		stderr.Printf("%s", err)
		os.Exit(1)
	}
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permFile)
	if err != nil {
		stderr.Printf("%s", err)
		os.Exit(1)
	}
	return f
}
