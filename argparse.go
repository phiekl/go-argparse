// SPDX-FileCopyrightText: 2024 Philip Eklöf
//
// SPDX-License-Identifier: MIT

// Package argparse extends spf13/pflag with common CLI parsing features:
//
//   - Required and non-empty flags/arguments.
//   - Mutually exclusive flags.
//   - Allowed string choices and regex validation.
//   - Positional arguments.
//   - Commands, where the first positional argument selects a commands
//     implementation, with any remaining arguments being passed to it.
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
	command            *Command
	commandArgs        []commandArg
	commandName        *string
	commandOptions     *[]string
	denyEmpty          []string
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

type commandArg struct {
	name        string
	description string
	impl        Command
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

// CommandInit registers the variables populated by command parsing:
// the selected command implementation, the selected command name, and any
// remaining arguments after the command name.
//
// Must be called before Command and before parsing.
func (p *ArgParser) CommandInit(commandTarget *Command, nameTarget *string, optionsTarget *[]string) {
	prefix := "CommandInit(): cannot be defined"

	if p.command != nil {
		panic(fmt.Sprintf("%s as CommandInit() has already been defined", prefix))
	}
	p.command = commandTarget
	p.commandName = nameTarget
	p.commandOptions = optionsTarget
}

// Command registers a named subcommand.
//
// The command name is required and is parsed like a primary positional argument.
// Call Command once per supported subcommand. CommandInit must be called first.
func (p *ArgParser) Command(name string, description string, command Command) {
	prefix := fmt.Sprintf("Command(%q): cannot be defined", name)

	if name == "" {
		panic(fmt.Sprintf("%s with empty name", prefix))
	}
	if p.command == nil {
		panic(fmt.Sprintf("%s as CommandInit() has not been defined", prefix))
	}
	if len(p.pos) > 0 {
		panic(fmt.Sprintf("%s as StringPosVar() has been defined", prefix))
	}
	if p.posN != nil {
		panic(fmt.Sprintf("%s as StringPosNVar() has been defined", prefix))
	}

	for _, commandArg := range p.commandArgs {
		if commandArg.name == name {
			panic(fmt.Sprintf("%s as Command(%q) is already defined", prefix, name))
		}
	}

	// Bind the Command implementation to its BaseCommand.
	command.Bind(command)

	p.commandArgs = append(p.commandArgs, commandArg{name, description, command})
}

// MutuallyExclusive declares that at most one of the named flags may be set.
// The constraint is enforced by ParseArgs.
func (p *ArgParser) MutuallyExclusive(names ...string) {
	prefix := fmt.Sprintf("MutuallyExclusive(%q): cannot be defined", names)

	if len(names) < 2 {
		panic(fmt.Sprintf("%s with less than two names", prefix))
	}

	if dupes := sliceDuplicates(names); len(dupes) > 0 {
		panic(fmt.Sprintf("%s with duplicate values", prefix))
	}

	for _, name := range names {
		if flag := p.Lookup(name); flag == nil {
			panic(fmt.Sprintf("%s for undefined flag %q", prefix, name))
		}
	}

	if p.Parsed() {
		panic(fmt.Sprintf("%s post-parse", prefix))
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
	if len(args) == 0 && p.requiredArgs() {
		p.generateHelp(1)
	}
	if help, _ := p.GetBool("help"); help {
		p.generateHelp(0)
	}
	if err := p.parseCommand(); err != nil {
		return err
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
	if err := p.parseDenyEmpty(); err != nil {
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
	prefix := fmt.Sprintf("Required(%q): cannot be defined", name)

	if name == "" {
		panic(fmt.Sprintf("%s with empty name", prefix))
	}

	if flag := p.Lookup(name); flag == nil {
		panic(fmt.Sprintf("%s for undefined flag", prefix))
	}

	if p.Parsed() {
		panic(fmt.Sprintf("%s post-parse", prefix))
	}

	p.required = append(p.required, name)
}

// StringAllowOptions restricts a string flag or positional argument to one of
// the provided options. Enforced by ParseArgs.
func (p *ArgParser) StringAllowOptions(target *string, name string, options []string) {
	prefix := fmt.Sprintf("StringAllowOptions(%q): cannot be defined", name)

	if name == "" {
		panic(fmt.Sprintf("%s with empty name", prefix))
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
			panic(fmt.Sprintf("%s for undefined flag", prefix))
		}
		if flag.Value.Type() != "string" {
			panic(fmt.Sprintf("%s for a flag that is not a string value", prefix))
		}
	}

	if p.Parsed() {
		panic(fmt.Sprintf("%s post-parse", prefix))
	}

	p.allowedOptions = append(p.allowedOptions, allowedOption{name, target, options})
}

// StringAllowRegexp restricts a string flag or positional argument to values
// matching re. Enforced by ParseArgs.
func (p *ArgParser) StringAllowRegexp(target *string, name string, re string) {
	prefix := fmt.Sprintf("StringAllowRegexp(%q): cannot be defined", name)

	if name == "" {
		panic(fmt.Sprintf("%s with empty name", prefix))
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
			panic(fmt.Sprintf("%s for undefined flag", prefix))
		}
		if flag.Value.Type() != "string" {
			panic(fmt.Sprintf("%s for a flag that is not a string value", prefix))
		}
	}

	rec, err := regexp.Compile(re)
	if err != nil {
		panic(fmt.Sprintf("%s due to: %v", prefix, err))

	}

	if p.Parsed() {
		panic(fmt.Sprintf("%s post-parse", prefix))
	}

	p.allowedRegexps = append(p.allowedRegexps, allowedRegexp{name, target, rec})
}

// StringDenyEmpty marks the named string flag or positional argument as
// required to have a non-empty value (i.e. not "").
// Enforced by ParseArgs.
func (p *ArgParser) StringDenyEmpty(target *string, name string) {
	prefix := fmt.Sprintf("StringDenyEmpty(%q): cannot be defined", name)

	if name == "" {
		panic(fmt.Sprintf("%s with empty name", prefix))
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
			panic(fmt.Sprintf("%s for undefined flag", prefix))
		}
		if flag.Value.Type() != "string" {
			panic(fmt.Sprintf("%s for a flag that is not a string value", prefix))
		}
	}

	if p.Parsed() {
		panic(fmt.Sprintf("%s post-parse", prefix))
	}

	p.denyEmpty = append(p.denyEmpty, name)
}

// StringPosNVar defines a variable number of string positional arguments. minN
// is the minimum number of arguments that are allowed, and maxN the maximum
// number. minN must be less or equal to maxN, unless maxN is -1, which means
// that an inifinite number of positional arguments may be supplied.
func (p *ArgParser) StringPosNVar(target *[]string, name, usage string, minN, maxN int) {
	prefix := fmt.Sprintf("StringPosNVar(%q): cannot be defined", name)

	if name == "" {
		panic(fmt.Sprintf("%s with empty name", prefix))
	}

	if p.command != nil {
		panic(fmt.Sprintf("%s as CommandInit() has been defined", prefix))
	}

	if minN < 0 {
		panic(fmt.Sprintf("%s with minN(%d) < 0", prefix, minN))
	}
	if maxN == 0 {
		panic(fmt.Sprintf("%s with maxN(0)", prefix))
	}
	if maxN < -1 {
		panic(fmt.Sprintf("%s with maxN(%d) < -1", prefix, maxN))
	}
	if maxN != -1 && minN > maxN {
		panic(fmt.Sprintf("%s with minN(%d) > maxN(%d)", prefix, minN, maxN))
	}

	if p.posN != nil {
		panic(fmt.Sprintf("%s as StringPosNVar(%q) is already defined", prefix, p.posN.name))
	}

	for _, pos := range p.pos {
		if pos.name == name {
			panic(fmt.Sprintf("%s as StringPosVar(%q) is already defined", prefix, name))
		}
	}

	p.posN = &posN{target, name, usage, minN, maxN}
}

// StringPosVar defines a required single string positional argument.
// Call multiple times to define multiple fixed positional arguments.
func (p *ArgParser) StringPosVar(target *string, name, usage string) {
	prefix := fmt.Sprintf("StringPosVar(%q): cannot be defined", name)

	if name == "" {
		panic(fmt.Sprintf("%s with empty name", prefix))
	}

	if p.command != nil {
		panic(fmt.Sprintf("%s as Command() has already been defined", prefix))
	}

	if flag := p.Lookup(name); flag != nil {
		panic(fmt.Sprintf("%s as already defined as flag", prefix))
	}

	for _, pos := range p.pos {
		if pos.name == name {
			panic(fmt.Sprintf("%s as StringPosVar(%q) is already defined", prefix, name))
		}
		if pos.target == target {
			panic(fmt.Sprintf("%s using the same target as StringPosVar(%q)", prefix, pos.name))
		}
	}

	if p.posN != nil {
		panic(fmt.Sprintf("%s as StringPosNVar(%q) is already defined", prefix, p.posN.name))
	}

	p.pos = append(p.pos, pos{target, name, usage})
}

func (p *ArgParser) generateHelp(rc int) {
	commandLen := 0
	posArgs := ""
	posLen := 0

	for _, command := range p.commandArgs {
		if len(command.name) > commandLen {
			commandLen = len(command.name)
		}
	}

	for _, pos := range p.pos {
		posArgs = posArgs + " " + pos.name
		if len(pos.name) > posLen {
			posLen = len(pos.name)
		}
	}

	if p.posN != nil {
		if p.posN.minN == 0 && p.posN.maxN == -1 {
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

	out := ""

	if commandLen > 0 {
		if p.commandOptions == nil {
			out += fmt.Sprintf("usage: %s [option].. <command>\n\n", p.Name)
		} else {
			out += fmt.Sprintf("usage: %s [option].. <command> [command option]..\n\n", p.Name)
		}
		format := fmt.Sprintf("  %%-%ds   %%s\n", commandLen)
		out += "commands:\n"
		for _, command := range p.commandArgs {
			out += fmt.Sprintf(format, command.name, command.description)
		}
		out += "\n"
	} else {
		out += fmt.Sprintf("usage: %s [option]..%s\n\n", p.Name, posArgs)
	}

	if posLen > 0 {
		format := fmt.Sprintf("  %%-%ds   %%s\n", posLen)
		out += "arguments:\n"
		for _, pos := range p.pos {
			out += fmt.Sprintf(format, pos.name, pos.usage)
		}
		if p.posN != nil {
			out += fmt.Sprintf(format, p.posN.name, p.posN.usage)
		}
		out += "\n"
	}

	out += "options:\n"
	out += p.FlagUsages()

	if rc == 0 {
		fmt.Printf("%s", out)
	} else {
		fmt.Fprintf(os.Stderr, "%s", out)
	}

	os.Exit(rc)
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

func (p *ArgParser) parseCommand() error {
	nargs := p.Args()

	if len(nargs) > 0 && p.command == nil && len(p.pos) == 0 && p.posN == nil {
		return fmt.Errorf("no positional arguments expected")
	}

	if p.command == nil {
		// parseNargs() will process the arguments instead.
		return nil
	}

	if len(nargs) == 0 {
		return fmt.Errorf("missing command")
	}

	commandName := nargs[0]
	found := false
	for _, command := range p.commandArgs {
		if command.name == commandName {
			*p.command = command.impl
			*p.commandName = commandName
			found = true
		}
	}
	if !found {
		return fmt.Errorf("invalid command: %s", commandName)
	}
	nargs = nargs[1:]

	if len(nargs) == 0 {
		return nil
	}

	if p.commandOptions == nil {
		return fmt.Errorf("options not allowed for command: %s", commandName)
	}

	*p.commandOptions = nargs
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
	if p.command != nil {
		// parseComand() should already have processed all nargs.
		return nil
	}

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

func (p *ArgParser) parseDenyEmpty() error {
	var empty []string
	for _, name := range p.denyEmpty {
		flag := p.Lookup(name)
		if flag == nil {
			found := false
			for _, pos := range p.pos {
				if pos.name == name {
					found = true
					if *pos.target == "" {
						empty = append(empty, name)
					}
					break
				}
			}
			if !found {
				return fmt.Errorf("wtf")
			}
		} else if flag.Value.String() == "" {
			empty = append(empty, name)
		}
	}
	if len(empty) == 1 {
		return fmt.Errorf("flag/argument is empty: %s", empty[0])
	} else if len(empty) > 1 {
		return fmt.Errorf("flags/arguments are empty: %s", strings.Join(empty, ", "))
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

func (p *ArgParser) requiredArgs() bool {
	if p.command != nil {
		return true
	}
	if len(p.pos) > 0 {
		return true
	}
	if p.posN != nil && p.posN.minN > 0 {
		return true
	}
	if len(p.required) > 0 {
		return true
	}
	return false
}

func sliceDuplicates(items []string) []string {
	count := make(map[string]int, len(items))
	for _, s := range items {
		count[s]++
	}
	var dups []string
	for s, n := range count {
		if n > 1 {
			dups = append(dups, s)
		}
	}
	return dups
}
