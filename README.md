<!--
SPDX-FileCopyrightText: 2024 Philip Eklöf

SPDX-License-Identifier: MIT
-->

# argparse

This package is an extension of [spf13/pflag](https://pkg.go.dev/github.com/spf13/pflag#FlagSet) with common CLI parsing features:

- Required and non-empty flags/arguments.
- Mutually exclusive flags.
- Allowed string choices and regex validation.
- Positional arguments.
- Commands, where the first positional argument selects a commands
  implementation, with any remaining arguments being passed to it.

## Example (positional arguments)

```go
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

#### Execution example

`file-ext-renamer --help`:
```
usage: file-ext-renamer [option].. dir [extension]..

arguments:
  dir         directory where files will be renamed
  extension   rename files with this file extension (all files if not given)

options:
  -h, --help                   display this help text and exit
  -l, --log-format string      log output format (options: text, json) (default "text")
  -n, --new-extension string   rename files to this file extension
```

## Example (commands)

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"pxy.se/go/argparse"
)

func main() {
	p := argparse.NewArgParser("calc")

	var jsonEncode bool
	var cmd argparse.Command
	var cmdName string
	var cmdOpts []string

	p.BoolVarP(&jsonEncode, "json", "j", false, "enable JSON output")
	p.CommandInit(&cmd, &cmdName, &cmdOpts)
	p.Command("sum", "sum integers", &sumCmd{})
	p.Command("echo", "echo a message", &echoCmd{})

	if err := p.ParseCurrentArgs(); err != nil {
		fmt.Fprintf(os.Stderr, "error: usage: %v\n", err)
		os.Exit(2)
	}

	if err := cmd.Run(cmdName, cmdOpts); err != nil {
		fmt.Fprintf(os.Stderr, "%s: error: %v\n", cmdName, err)
		os.Exit(2)
	}

	res := cmd.Result()

	if len(res.Error) > 0 {
		for _, err := range res.Error {
			fmt.Fprintf(os.Stderr, "%s: error: %v\n", cmdName, err)
		}
		os.Exit(2)
	}

	if jsonEncode {
		out, err := json.Marshal(res.Data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: error: %v\n", cmdName, err)
			os.Exit(2)
		}
		fmt.Printf("%s\n", out)
	} else {
		fmt.Printf("%s\n", res.Data)
	}
}


type sumResult struct {
	Sum int `json:"sum"`
}

func (r *sumResult) String() string { return fmt.Sprintf("%d", r.Sum) }

type sumCmd struct {
	argparse.BaseCommand
	n []string
}

func (c *sumCmd) Args() {
	c.ArgP.StringPosNVar(&c.n, "n", "numbers to sum (repeatable)", 1, -1)
}

func (c *sumCmd) Command() (any, []error) {
	total := 0
	for _, s := range c.n {
		v, err := strconv.Atoi(s)
		if err != nil {
			return nil, []error{fmt.Errorf("invalid number %q: %w", s, err)}
		}
		total += v
	}
	return &sumResult{Sum: total}, nil
}


type echoResult struct {
	Message string `json:"message"`
}

func (r *echoResult) String() string { return r.Message }

type echoCmd struct {
	argparse.BaseCommand
	msg string
}

func (c *echoCmd) Args() {
	c.ArgP.StringVarP(&c.msg, "message", "m", "", "message to echo")
	c.ArgP.StringDenyEmpty(&c.msg, "message")
	c.ArgP.Required("message")
}

func (c *echoCmd) Command() (any, []error) {
	return &echoResult{Message: c.msg}, nil
}
```

#### Execution examples
`calc --help`:
```
usage: calc [option].. <command> [command option]..

commands:
  sum    sum integers
  echo   echo a message

options:
  -h, --help   display this help text and exit
  -j, --json   enable JSON output
```

`calc -j sum 2 3 5`:
```
{"result":{"sum":10}}
```

`calc sum 2 3 5`:
```
10
```

`calc -j sum 2 '' 5`:
```
{"error":["invalid number \"\": strconv.Atoi: parsing \"\": invalid syntax"]}
```

`calc sum 2 '' 5`:
```
sum: error: invalid number "": strconv.Atoi: parsing "": invalid syntax
```

`calc -j echo --message hello`
```
{"result":{"message":"hello"}}
```

#### Notes

- The above example code could easily be split into different packages, e.g. a main package, an internal/cmd package for the commands, and the actual workflow in a pkg/workflow package.
- Each command typically embeds `argparse.BaseCommand`.
- `Args()` is where the command registers its flags on `c.ArgP`.
- `Command()` performs the work and returns:
  - A pointer to a value implementing `argparse.CommandResultData`, or a **nil pointer** to indicate "no result".
  - A slice of errors (will be reported under `"error"` when marshaled to JSON)
