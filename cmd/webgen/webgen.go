package main

import (
	"flag"
	"fmt"
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
          [--root=<dir>] (<input-file> | <input-directory>)...
   webgen (-h | --help)

Flags:
   -h --help           Print help and exit
   --outcss=<file>     Write CSS output to specified file instead of stdout
   --outviews=<file>   Write views output to specified file instead of stdout
   --package=<name>    Package name to use in output (default: "views")
   --root=<dir>        Root directory for absolute paths in <include />
                       elements (default: ".")

Example:
   # Recursively find all *.html files in the "components" directory and use
   # them as input. Write output to "ui.go" and "public/components.css" with
   # package name "ui" for the Go code.
   webgen --package=ui \
          --outviews=ui.go \
          --outcss=public/components.css \
          components
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

	if err := run(args); err != nil {
		stderr.Printf("%s", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	outViews := os.Stdout
	outCSS := os.Stdout

	if fOutViews != "" {
		outViews = createFile(fOutViews)
		defer outViews.Close()
	}
	if fOutCSS != "" {
		outCSS = createFile(fOutCSS)
		defer outCSS.Close()
	}

	opts := webgen.Options{
		Package: fPackageName,
		Root:    fRoot,
	}

	var inFiles []string
	dedup := make(map[string]struct{})
	maybeAdd := func(p string) {
		if _, ok := dedup[p]; ok {
			return // already present
		}
		dedup[p] = struct{}{}
		inFiles = append(inFiles, p)
	}

	for _, a := range args {
		info, err := os.Stat(a)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := filepath.Walk(a, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil // will be handled by recursive walk
				}
				if filepath.Ext(p) != ".html" {
					return nil
				}
				if _, ok := dedup[p]; ok {
					return nil // already present
				}
				inFiles = append(inFiles, p)
				return nil
			}); err != nil {
				return err
			}
		} else {
			// assume it's a file.
			// we also don't check for a .html extension since this is an
			// explicitly provided command line argument.
			maybeAdd(a)
		}
	}

	views, css, err := webgen.Generate(inFiles, opts)
	if err != nil {
		return err
	}

	if _, err := outViews.Write(views); err != nil {
		return fmt.Errorf("write output views: %s", err)
	}
	if _, err := outCSS.Write(css); err != nil {
		return fmt.Errorf("write output css: %s", err)
	}

	return nil
}

func createFile(p string) *os.File {
	err := os.MkdirAll(filepath.Dir(p), permDir)
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
