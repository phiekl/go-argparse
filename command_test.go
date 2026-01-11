// SPDX-FileCopyrightText: 2026 Philip Eklöf
//
// SPDX-License-Identifier: MIT

package argparse

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// ---- helpers ----

type testResult struct{ S string }

func (r *testResult) String() string { return r.S }

type nonPtrResult struct{ S string }

func (r nonPtrResult) String() string { return r.S }

// command impl used for BaseCommand.Run tests
type testCmd struct {
	BaseCommand
	argsCalled    int
	commandCalled int

	res  any
	errs []error
}

func newTestCmd(res any, errs []error) *testCmd {
	tc := &testCmd{res: res, errs: errs}
	tc.Bind(tc)
	return tc
}

func (t *testCmd) Args() { t.argsCalled++ }

func (t *testCmd) Command() (any, []error) {
	t.commandCalled++
	return t.res, t.errs
}

// ---- tests ----

func TestBaseCommand_Bind_Run_ErrOnMissingImpl(t *testing.T) {
	var bc BaseCommand
	if err := bc.Run("x", nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBaseCommand_Run_SetsNameAndCapturesResultAndErrors(t *testing.T) {
	res := &testResult{S: "ok"}
	errs := []error{errors.New("e1"), errors.New("e2")}

	cmd := newTestCmd(res, errs)

	if err := cmd.Run("mycmd", nil); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if got := cmd.Name(); got != "mycmd" {
		t.Fatalf("Name = %q, want %q", got, "mycmd")
	}
	if cmd.argsCalled != 1 {
		t.Fatalf("Args called %d times, want 1", cmd.argsCalled)
	}
	if cmd.commandCalled != 1 {
		t.Fatalf("Command called %d times, want 1", cmd.commandCalled)
	}

	gotRes := cmd.Result()
	if gotRes.Data == nil {
		t.Fatalf("Result.Data is nil, want non-nil")
	}
	tr, ok := gotRes.Data.(*testResult)
	if !ok {
		t.Fatalf("Result.Data has type %T, want *testResult", gotRes.Data)
	}
	if tr.S != "ok" {
		t.Fatalf("Result.Data.String() = %q, want %q", tr.String(), "ok")
	}
	if len(gotRes.Error) != 2 || gotRes.Error[0].Error() != "e1" || gotRes.Error[1].Error() != "e2" {
		t.Fatalf("Result.Error = %#v, want [e1 e2]", gotRes.Error)
	}
}

func TestBaseCommand_captureResult_ErrOnNilInterface(t *testing.T) {
	var bc BaseCommand
	if err := bc.captureResult(nil, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBaseCommand_captureResult_ErrOnNonPointer(t *testing.T) {
	var bc BaseCommand
	if err := bc.captureResult(nonPtrResult{S: "x"}, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestBaseCommand_captureResult_NilPointerMeansNoData(t *testing.T) {
	var bc BaseCommand
	var r *testResult = nil

	if err := bc.captureResult(r, []error{errors.New("e")}); err != nil {
		t.Fatalf("captureResult returned error: %v", err)
	}
	if bc.result.Data != nil {
		t.Fatalf("Data = %v, want nil", bc.result.Data)
	}
	if len(bc.result.Error) != 1 || bc.result.Error[0].Error() != "e" {
		t.Fatalf("Error = %#v, want [e]", bc.result.Error)
	}
}

func TestBaseCommand_captureResult_ErrOnPointerNotImplementingCommandResultData(t *testing.T) {
	var bc BaseCommand
	x := new(int) // *int does not implement CommandResultData

	if err := bc.captureResult(x, nil); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCommandResult_MarshalJSON(t *testing.T) {
	r := CommandResult{
		Data:  &testResult{S: "hello"},
		Error: []error{errors.New("a"), errors.New("b")},
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// decode into map to avoid depending on field ordering
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v; json=%s", err, string(b))
	}

	// error should be []any{"a","b"}
	ev, ok := m["error"]
	if !ok {
		t.Fatalf("missing key 'error' in %v", m)
	}
	es, ok := ev.([]any)
	if !ok || len(es) != 2 || es[0] != "a" || es[1] != "b" {
		t.Fatalf("'error' = %#v, want [\"a\",\"b\"]", ev)
	}

	// result should be present. Since CommandResultData is an interface and testResult
	// has exported field S, it should marshal as {"S":"hello"}.
	rv, ok := m["result"]
	if !ok {
		t.Fatalf("missing key 'result' in %v", m)
	}
	rm, ok := rv.(map[string]any)
	if !ok || rm["S"] != "hello" {
		t.Fatalf("'result' = %#v, want map with S=hello", rv)
	}
}

func TestCommandResult_MarshalJSON_OmitsEmptyFields(t *testing.T) {
	// no errors, no result
	r := CommandResult{}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("json.Unmarshal: %v; json=%s", err, string(b))
	}

	// With omitempty, both keys should be absent. (result omitted because Data is nil)
	if _, ok := m["error"]; ok {
		t.Fatalf("did not expect 'error' in %v", m)
	}
	if _, ok := m["result"]; ok {
		t.Fatalf("did not expect 'result' in %v", m)
	}
}

func TestBaseCommand_captureResult_StoresConcreteDataInterface(t *testing.T) {
	var bc BaseCommand
	res := &testResult{S: "x"}

	if err := bc.captureResult(res, nil); err != nil {
		t.Fatalf("captureResult returned error: %v", err)
	}

	// Ensure the stored interface points to the same concrete value.
	if bc.result.Data == nil {
		t.Fatalf("Data is nil")
	}
	if reflect.ValueOf(bc.result.Data).Pointer() != reflect.ValueOf(res).Pointer() {
		t.Fatalf("stored Data does not reference the same pointer")
	}
}
