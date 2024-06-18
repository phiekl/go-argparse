<!--
SPDX-FileCopyrightText: 2024 Philip Eklöf

SPDX-License-Identifier: MIT
-->

# argparse

This package is an extension of [spf13/pflag's
FlagSet](https://pkg.go.dev/github.com/spf13/pflag#FlagSet), supporting
positional arguments, and defining various requirements for arguments such as
required arguments and regular expressions of argument values.


## Example

```
package main

import (
	"pxy.se/go/argparse"
)

type cmdArgs struct {
	logFormat       string
	dir             string
	newExtension    string
	extensionFilter []string
}

func main() {
	args := cmdArgs{}
	p := argparse.NewArgParser("file-ext-renamer")

	p.StringVarP(&args.logFormat,
		"log-format", "l", "text",
		"log output format (options: text, json)",
	)
	p.StringAllowOptions(&args.logFormat,
		"log-format",
		[]string{"json", "text"},
	)

	p.StringVarP(&args.newExtension,
		"new-extension", "n", "",
		"rename files to this file extension",
	)
	p.StringAllowRegexp(&args.newExtension,
		"new-extension",
		"^[0-9a-z]+$",
	)
	p.Required("new-extension")

	p.StringPosVar(&args.dir,
		"dir",
		"directory where files will be renamed",
	)

	p.StringPosNVar(&args.extensionFilter,
		"extension",
		"rename files with this file extension (all files if not given)",
		0, -1,
	)

	err := p.ParseCurrentArgs()
	if err != nil {
		panic(err)
	}
}
```

Running this program with --help would generate:
```
usage: file-ext-renamer [flag].. dir [extension]..

positional arguments:
  dir         directory where files will be renamed
  extension   rename files with this file extension (all files if not given)

flags:
  -h, --help                   display this help text and exit
  -l, --log-format string      log output format (options: text, json) (default "text")
  -n, --new-extension string   rename files to this file extension
```
