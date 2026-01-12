// SPDX-FileCopyrightText: 2026 Philip Eklöf
//
// SPDX-License-Identifier: MIT

package argparse

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// BaseCommand provides a reusable implementation of the Command interface.
//
// BaseCommand holds common command state (parser, name, result) and delegates
// argument registration and command execution to a bound CommandImpl.
type BaseCommand struct {
	// ArgP is the argument parser instance created for the current Run.
	ArgP *ArgParser

	impl   CommandImpl
	name   string
	result CommandResult
}

// Bind wires an implementation of a Command to its BaseCommand.
//
// The provided impl must also implement CommandImpl; otherwise Bind will panic
// due to the type assertion.
func (c *BaseCommand) Bind(impl Command) {
	c.impl = impl.(CommandImpl)
}

// Name returns the command name as provided to Run.
func (c *BaseCommand) Name() string {
	return c.name
}

// Result returns the most recent command result captured from Run.
func (c *BaseCommand) Result() CommandResult {
	return c.result
}

// Run executes the command with the given name and option tokens.
//
// Run creates a new ArgParser, asks the bound implementation to register its
// arguments via Args, parses opts, invokes the implementation's Command method,
// and captures the returned result and errors into c.Result().
func (c *BaseCommand) Run(name string, opts []string) error {
	if c.impl == nil {
		return fmt.Errorf("command implementation not set")
	}

	c.name = name

	c.ArgP = NewArgParser(name)
	c.impl.Args()

	if err := c.ArgP.ParseArgs(opts); err != nil {
		return err
	}

	res, errs := c.impl.Command()
	if err := c.captureResult(res, errs); err != nil {
		return err
	}
	return nil
}

func (c *BaseCommand) captureResult(res any, errs []error) error {
	c.result.Error = errs

	if res == nil {
		// No result, all good, do not populate.
		return nil
	}

	v := reflect.ValueOf(res)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("command result capture: expected pointer, got %T", res)
	}

	if v.IsNil() {
		// No result, all good, do not populate.
		return nil
	}

	data, ok := res.(CommandResultData)
	if !ok {
		return fmt.Errorf("command result capture: expected CommandResultData, got %T", res)
	}

	c.result.Data = data
	return nil
}

// Command is the public interface for a runnable command.
//
// Implementations typically embed BaseCommand to reuse parsing/execution
// scaffolding, and call Bind to attach the concrete implementation.
type Command interface {
	// Run executes the command with the provided name and option tokens.
	Run(name string, opts []string) error

	// Name returns the current command name (set during Run).
	Name() string

	// Result returns the last captured CommandResult from Run.
	Result() CommandResult

	// Bind attaches the concrete command implementation to the base command logic.
	Bind(Command)
}

// CommandImpl is the internal interface implemented by concrete commands.
//
// It separates command-specific argument definitions (Args) and the actual
// execution (Command) from the shared BaseCommand runner.
type CommandImpl interface {
	// Args registers the command's arguments and options with the active ArgParser.
	Args()

	// Command executes the command and returns a result value and a slice of
	// errors. The result is expected to be a pointer to a value implementing
	// CommandResultData; nil or a nil pointer indicates no result data.
	Command() (any, []error)
}

// CommandResult is the structured output captured from a command execution.
type CommandResult struct {
	// Data is the command's result payload, if any.
	Data CommandResultData

	// Error contains any errors produced during command execution.
	Error []error
}

// MarshalJSON marshals the result into JSON.
//
// Errors are encoded as a slice of strings under the "error" key. The result
// payload is encoded under the "result" key.
func (r CommandResult) MarshalJSON() ([]byte, error) {
	var errs []string
	for _, err := range r.Error {
		errs = append(errs, err.Error())
	}
	return json.Marshal(
		&struct {
			Error  []string          `json:"error,omitempty"`
			Result CommandResultData `json:"result,omitempty"`
		}{
			Error:  errs,
			Result: r.Data,
		},
	)
}

// CommandResultData is the interface implemented by command result payloads.
//
// Implementations should provide a human-readable String representation.
type CommandResultData interface {
	String() string
}
