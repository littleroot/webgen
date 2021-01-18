# Nausicaä

Inspired by [tomato][1].

## Usage

```
Usage:
   nausicaa [--outcss=<file>] [--outviews=<file>] [--package=<name>]
            [--root=<dir>] <input-file>...
   nausicaa (-h | --help)

Flags:
   -h --help           Print help and exit
   --outcss=<file>     Write CSS output to specified file instead of stdout
   --outviews=<file>   Write view output to specified file instead of stdout
   --package=<name>    Package name to use in output (default: "views")
   --root=<dir>        Root directory for absolute paths in <include />
                       elements (default: ".")

Examples:
   nausicaa Button.html SegmentedControl.html
   nausicaa $(find ./components -name '*.html')
   nausicaa --package=ui --outfile=my/pkg/ui/ui.go Select.html
```

## License

MIT

[1]: https://github.com/donjaime/tomato
