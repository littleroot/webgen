package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/littleroot/nausicaa"
)

const (
	permDir  = 0775 // drwxrwxr-x.
	permFile = 0664 // -rw-rw-r--.
)

var (
	stderr = log.New(os.Stderr, "", 0)
)

const usage = `
usage: nausicaa [-h | --help] [--outfile=<path>] [--package=<name>]
                [--root=<dir>] <input>...

Flags:

   -h | --help        Print help and exit
   --outfile=<path>   Write output to specified file instead of stdout
   --package=<name>   Package name to use in output (default "views")
   --root=<dir>       Root directory for absolute path includes (default ".")

Examples:

   nausicaa button.html segmented_control.html
   nausicaa $(find ./components -name '*.html')
   nausicaa --package=ui --outfile=my/pkg/ui/ui.go file.html
`

var (
	fHelp        bool
	fOutfile     string
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
	flag.StringVar(&fOutfile, "outfile", "", "")
	flag.StringVar(&fPackageName, "package", "", "")
	flag.StringVar(&fRoot, "root", ".", "")
	flag.Usage = printUsage
	flag.Parse()

	args := flag.Args()

	if fHelp {
		printUsage()
		os.Exit(0)
	}
	if len(args) == 0 {
		printUsage()
		os.Exit(2)
	}

	out := os.Stdout

	if fOutfile != "" {
		var err error
		out, err = os.OpenFile(fOutfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permFile)
		if err != nil {
			stderr.Printf("open %s: %s", fOutfile, err)
			os.Exit(1)
		}
		defer out.Close()
	}

	opts := &nausicaa.Options{
		Package: fPackageName,
		Root:    fRoot,
	}
	err := nausicaa.Generate(out, args, opts)
	if err != nil {
		stderr.Printf("%s", err)
		os.Exit(1)
	}
}
