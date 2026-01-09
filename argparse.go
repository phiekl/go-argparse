// SPDX-FileCopyrightText: 2024 Philip Eklöf
//
// SPDX-License-Identifier: MIT

// Package argparse extends spf13/pflag with common CLI parsing features:
//
//   - Required flags/arguments.
//   - Mutually exclusive flags.
//   - Allowed string choices and regex validation.
//   - Positional arguments.
package argparse

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/pflag"
)

// ArgParser embeds pflag.FlagSet and adds post-parse validation and positional
// argument support.
//
// Create with NewArgParser. Call ParseArgs or ParseCurrentArgs (not FlagSet.Parse)
// to apply validations.
type ArgParser struct {
	pflag.FlagSet

	// Error stores the last parse error returned by Parse/ParseArgs.
	Error error
	// Name is the program name used in help and error messages.
	Name string

	allowedRegexps     []allowedRegexp
	allowedOptions     []allowedOption
	pos                []pos
	posN               *posN
	mutuallyExclusives [][]string
	required           []string
}

type allowedOption struct {
	name    string
	target  *string
	options []string
}

func (a *allowedOption) check() error {
	if !slices.Contains(a.options, *a.target) {
		return fmt.Errorf(
			"%s: invalid value: %q is not among options: %q", a.name, *a.target, a.options,
		)
	}
	return nil
}

type allowedRegexp struct {
	name   string
	target *string
	regexp *regexp.Regexp
}

func (a *allowedRegexp) check() error {
	if !a.regexp.MatchString(*a.target) {
		return fmt.Errorf(
			"%s: invalid value: %q is not matching regexp %q", a.name, *a.target, a.regexp,
		)
	}
	return nil
}

type pos struct {
	target *string
	name   string
	usage  string
}

type posN struct {
	target *[]string
	name   string
	usage  string
	minN   int
	maxN   int
}

// NewArgParser creates a new parser and registers -h/--help.
func NewArgParser(name string) *ArgParser {
	p := ArgParser{
		Name: name,
	}
	p.Init(name, pflag.ContinueOnError)
	p.BoolP(
		"help",
		"h",
		false,
		"display this help text and exit",
	)
	return &p
}

// MutuallyExclusive declares that at most one of the named flags may be set.
// The constraint is enforced by ParseArgs.
func (p *ArgParser) MutuallyExclusive(names ...string) {
	for _, name := range names {
		if flag := p.Lookup(name); flag == nil {
			p.die("mutually exclusive: undefined flag: %s", name)
		}
	}
	if p.Parsed() {
		p.die("mutually exclusive: %v: cannot define post-parse", names)
	}
	p.mutuallyExclusives = append(p.mutuallyExclusives, names)
}

// ParseCurrentArgs calls ParseArgs with os.Args[1:].
func (p *ArgParser) ParseCurrentArgs() error {
	return p.ParseArgs(os.Args[1:])
}

// ParseArgs calls FlagSet's Parse(), parsing arguments as usual. Positional
// arguments and checks such as required arguments are verified afterwards.
func (p *ArgParser) ParseArgs(args []string) error {
	if err := p.Parse(args); err != nil {
		p.Error = err
		return err
	}
	if help, _ := p.GetBool("help"); help {
		p.generateHelp()
	}
	if err := p.parseNargs(); err != nil {
		return err
	}
	if err := p.parseRequired(); err != nil {
		return err
	}
	if err := p.parseMutuallyExclusive(); err != nil {
		return err
	}
	if err := p.parseAllowed(); err != nil {
		return err
	}
	return nil
}

// Required marks a flag as required.
// The constraint is enforced by ParseArgs.
func (p *ArgParser) Required(name string) {
	if flag := p.Lookup(name); flag == nil {
		p.die("required: undefined flag: %s", name)
	}
	if p.Parsed() {
		p.die("required: %s: cannot define post-parse", name)
	}
	p.required = append(p.required, name)
}

// StringAllowOptions restricts a string flag or positional argument to one of
// the provided options. Enforced by ParseArgs.
func (p *ArgParser) StringAllowOptions(target *string, name string, options []string) {
	if name == "" {
		p.die("allow options: cannot be defined with empty name")
	}

	found := false
	for _, pos := range p.pos {
		if pos.name == name {
			found = true
			break
		}
	}

	if !found {
		flag := p.Lookup(name)
		if flag == nil {
			p.die("allow options: undefined flag: %s", name)
		}
		if flag.Value.Type() != "string" {
			p.die("allow options: %s: flag is not for a string value", name)
		}
		if p.Parsed() {
			p.die("allow options: %s: cannot define post-parse", name)
		}
	}

	p.allowedOptions = append(p.allowedOptions, allowedOption{name, target, options})
}

// StringAllowRegexp restricts a string flag or positional argument to values
// matching re. Enforced by ParseArgs.
func (p *ArgParser) StringAllowRegexp(target *string, name string, re string) {
	if name == "" {
		p.die("allow regexp: cannot be defined with empty name")
	}

	found := false
	for _, pos := range p.pos {
		if pos.name == name {
			found = true
			break
		}
	}

	if !found {
		flag := p.Lookup(name)
		if flag == nil {
			p.die("allow regexp: undefined flag: %s", name)
		}
		if flag.Value.Type() != "string" {
			p.die("allow regexp: %s: flag is not for a string value", name)
		}
		if p.Parsed() {
			p.die("allow regexp: %s: cannot define post-parse", name)
		}
	}

	rec, err := regexp.Compile(re)
	if err != nil {
		p.die("allow regexp: %s: %v", name, err)

	}
	p.allowedRegexps = append(p.allowedRegexps, allowedRegexp{name, target, rec})
}

// StringPosNVar defines a variable number of string positional arguments. minN
// is the minimum number of arguments that are allowed, and maxN the maximum
// number. minN must be less or equal to maxN, unless maxN is -1, which means
// that an inifinite number of positional arguments may be supplied.
func (p *ArgParser) StringPosNVar(target *[]string, name, usage string, minN, maxN int) {
	prefix := fmt.Sprintf("%s: varying positional argument", p.Name)

	if name == "" {
		p.die("%s cannot be defined with empty name", prefix)
	}

	prefix = fmt.Sprintf("%s %q cannot be defined", prefix, name)

	if minN < 0 {
		p.die("%s with minN(%d) < 0", prefix, minN)
	}
	if maxN == 0 {
		p.die("%s with maxN(%d) == 0", prefix, maxN)
	}
	if maxN < -1 {
		p.die("%s with maxN(%d) < -1", prefix, maxN)
	}
	if maxN != -1 && minN > maxN {
		p.die("%s with minN(%d) > maxN(%d)", prefix, minN, maxN)
	}
	if p.posN != nil {
		p.die("%s when a varying positional argument is already defined: %s", p.posN.name)
	}

	for _, pos := range p.pos {
		if pos.name == name {
			p.die("%s when a positional argument with the same name is already defined", prefix)
		}
	}

	p.posN = &posN{target, name, usage, minN, maxN}
}

// StringPosVar defines a required single string positional argument.
// Call multiple times to define multiple fixed positional arguments.
func (p *ArgParser) StringPosVar(target *string, name, usage string) {
	prefix := fmt.Sprintf("%s: positional argument", p.Name)

	if name == "" {
		p.die("%s cannot be defined with empty name", prefix)
	}
	if flag := p.Lookup(name); flag != nil {
		p.die("%s already redefined as flag: %s", prefix, name)
	}
	for _, pos := range p.pos {
		if pos.name == name {
			p.die("%s already defined: %s", prefix, name)
		}
		if pos.target == target {
			p.die(
				"%s %q cannot be defined using the same target as positional argument %q",
				prefix, name, pos.name,
			)
		}
	}
	if p.posN != nil {
		p.die(
			"%s cannot be defined when a varying positional argument is already defined: %s",
			p.Name, p.posN.name,
		)
	}

	p.pos = append(p.pos, pos{target, name, usage})
}

func (p *ArgParser) die(format string, args ...any) {
	var new []interface{}
	new = append(new, p.Name)
	new = append(new, args...)
	panic(fmt.Sprintf("%s: "+format, new...))
}

func (p *ArgParser) generateHelp() {
	posArgs := ""
	posLen := 0

	for _, pos := range p.pos {
		posArgs = posArgs + " " + pos.name
		if len(pos.name) > posLen {
			posLen = len(pos.name)
		}
	}

	if p.posN != nil {
		if p.posN.minN == 0 {
			posArgs = posArgs + " [" + p.posN.name + "]"
		}
		for i := 1; i <= p.posN.minN; i++ {
			posArgs = posArgs + " " + p.posN.name
		}
		if p.posN.maxN == -1 {
			posArgs = posArgs + ".."
		} else {
			for i := p.posN.minN; i < p.posN.maxN; i++ {
				posArgs = posArgs + " " + "[" + p.posN.name
			}
			for i := p.posN.minN; i < p.posN.maxN; i++ {
				posArgs = posArgs + "]"
			}
		}
		if len(p.posN.name) > posLen {
			posLen = len(p.posN.name)
		}
	}

	fmt.Printf("usage: %s [flag]..%s\n\n", p.Name, posArgs)

	if posLen > 0 {
		format := fmt.Sprintf("  %%-%ds   %%s\n", posLen)
		fmt.Printf("positional arguments:\n")
		for _, pos := range p.pos {
			fmt.Printf(format, pos.name, pos.usage)
		}
		if p.posN != nil {
			fmt.Printf(format, p.posN.name, p.posN.usage)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("flags:\n")
	fmt.Printf("%s", p.FlagUsages())
	os.Exit(0)
}

func (p *ArgParser) parseAllowed() error {
	for _, allowed := range p.allowedRegexps {
		if err := allowed.check(); err != nil {
			return err
		}
	}
	for _, allowed := range p.allowedOptions {
		if err := allowed.check(); err != nil {
			return err
		}
	}
	return nil
}

func (p *ArgParser) parseMutuallyExclusive() error {
	for _, names := range p.mutuallyExclusives {
		changed := ""
		for _, name := range names {
			flag := p.Lookup(name)
			if flag.Changed {
				if changed != "" {
					return fmt.Errorf("%s and %s are mutually exclusive flags", changed, name)
				}
				changed = name
			}
		}
	}
	return nil
}

func (p *ArgParser) parseNargs() error {
	nargs := p.Args()

	if len(nargs) > 0 && len(p.pos) == 0 && p.posN == nil {
		return fmt.Errorf("no positional arguments expected")
	}

	if len(p.pos) > 0 {
		if len(nargs) < len(p.pos) {
			return fmt.Errorf("insufficient number of positional arguments, see --help")
		}
		for i, v := range nargs[0:len(p.pos)] {
			*p.pos[i].target = v
		}
		nargs = nargs[len(p.pos):]
	}

	if p.posN != nil {
		if len(nargs) < p.posN.minN {
			if len(nargs) == 0 {
				if p.posN.maxN == -1 {
					return fmt.Errorf(
						"no %q positional argument(s) provided, see --help",
						p.posN.name,
					)
				} else {
					return fmt.Errorf(
						"no %q positional argument(s) provided, expected %d, see --help",
						p.posN.name, p.posN.minN,
					)
				}
			}
			return fmt.Errorf(
				"got %d %q positional argument(s), expected %d at least, see --help",
				len(nargs), p.posN.name, p.posN.minN,
			)
		}
		if p.posN.maxN != -1 && len(nargs) > p.posN.maxN {
			return fmt.Errorf(
				"got %d %q positional argument(s), expected %d at most, see --help",
				len(nargs), p.posN.name, p.posN.maxN,
			)
		}
		*p.posN.target = nargs
		nargs = nargs[:0]
	}

	if len(nargs) > 0 {
		return fmt.Errorf("unexpected number of positional arguments")
	}

	return nil
}

func (p *ArgParser) parseRequired() error {
	var required []string
	for _, name := range p.required {
		flag := p.Lookup(name)
		if !flag.Changed {
			required = append(required, name)
		}
	}
	if len(required) == 1 {
		return fmt.Errorf("missing required flag: %s", required[0])
	} else if len(required) > 1 {
		return fmt.Errorf("missing required flags: %s", strings.Join(required, ", "))
	}
	return nil
}
