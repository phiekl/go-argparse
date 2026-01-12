// SPDX-FileCopyrightText: 2024 Philip Eklöf
//
// SPDX-License-Identifier: MIT

package argparse

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

func testError(t *testing.T, err error, expected string) {
	if err == nil {
		t.Fatalf("expected parsing error")
	}
	if err.Error() != expected {
		t.Fatalf("expected parsing error %q, got %q\n", expected, err)
	}
}

func testExec(t *testing.T, rcx int, stdoutx, stderrx string) {
	cmd := exec.Command(os.Args[0], "-test.run="+t.Name())
	cmd.Env = append(os.Environ(), "TEST_EXIT_RUN=1")

	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd.Stdout = &stdoutBuffer
	cmd.Stderr = &stderrBuffer

	rc := 0
	if err := cmd.Run(); err != nil {
		e, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("unknown exec error: %v\n", err)
		}
		rc = e.ExitCode()
	}

	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()

	if rc != rcx {
		t.Errorf("expected exit code %d, got %d", rcx, rc)
	}

	if stdout != stdoutx {
		t.Errorf("stdout : %q", stdout)
		t.Logf("stdoutx: %q", stdoutx)
	}
	if stderr != stderrx {
		t.Errorf("stderr : %q", stderr)
		t.Logf("stderrx: %q", stderrx)
	}
}

func testExecRun() bool {
	return os.Getenv("TEST_EXIT_RUN") == "1"
}

func testPanic(t *testing.T, r any, msgx string) {
	if r == nil {
		t.Fatal("expected panic")
	}

	msg, ok := r.(string)
	if !ok {
		t.Fatalf("expected panic of type string, got %T", r)
	}

	if msg != msgx {
		t.Errorf("panic msg : %q", msg)
		t.Logf("panic msgx: %q", msgx)
	}
}

func testNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("unexpected parsing error: %v", err)
	}
}

func TestHelp_NoArguments(t *testing.T) {
	if testExecRun() {
		p := NewArgParser("testprog")
		var a, b string
		p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
		p.StringPosVar(&b, "b-test", "usage-b")
		args := []string{}
		_ = p.ParseArgs(args)
		return
	}

	stderrx := "usage: testprog [option].. b-test\n"
	stderrx += "\n"
	stderrx += "arguments:\n"
	stderrx += "  b-test   usage-b\n"
	stderrx += "\n"
	stderrx += "options:\n"
	stderrx += "  -h, --help            display this help text and exit\n"
	stderrx += "  -a, --a-test string   usage-a (default \"default-a\")\n"
	testExec(t, 1, "", stderrx)
}

func TestHelp_Requested_FlagsOnly(t *testing.T) {
	if testExecRun() {
		p := NewArgParser("testprog")
		var a string
		p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
		args := []string{"--help"}
		_ = p.ParseArgs(args)
		return
	}

	stdoutx := "usage: testprog [option]..\n"
	stdoutx += "\n"
	stdoutx += "options:\n"
	stdoutx += "  -h, --help            display this help text and exit\n"
	stdoutx += "  -a, --a-test string   usage-a (default \"default-a\")\n"
	testExec(t, 0, stdoutx, "")
}

func TestHelp_Requested_PosArg(t *testing.T) {
	if testExecRun() {
		p := NewArgParser("testprog")
		var a string
		p.StringPosVar(&a, "a-test", "usage-a")
		args := []string{"--help"}
		_ = p.ParseArgs(args)
		return
	}

	stdoutx := "usage: testprog [option].. a-test\n"
	stdoutx += "\n"
	stdoutx += "arguments:\n"
	stdoutx += "  a-test   usage-a\n"
	stdoutx += "\n"
	stdoutx += "options:\n"
	stdoutx += "  -h, --help   display this help text and exit\n"
	testExec(t, 0, stdoutx, "")
}

func TestHelp_Requested_PosNArgMin0Infinite(t *testing.T) {
	if testExecRun() {
		p := NewArgParser("testprog")
		var a string
		p.StringPosVar(&a, "a-test", "usage-a")
		var b []string
		p.StringPosNVar(&b, "b-test", "usage-b", 0, -1)
		args := []string{"--help"}
		_ = p.ParseArgs(args)
		return
	}

	stdoutx := "usage: testprog [option].. a-test [b-test]..\n"
	stdoutx += "\n"
	stdoutx += "arguments:\n"
	stdoutx += "  a-test   usage-a\n"
	stdoutx += "  b-test   usage-b\n"
	stdoutx += "\n"
	stdoutx += "options:\n"
	stdoutx += "  -h, --help   display this help text and exit\n"
	testExec(t, 0, stdoutx, "")
}

func TestHelp_Requested_PosNArgMin0Max3(t *testing.T) {
	if testExecRun() {
		p := NewArgParser("testprog")
		var a string
		p.StringPosVar(&a, "a-test", "usage-a")
		var b []string
		p.StringPosNVar(&b, "b-test", "usage-b", 0, 3)
		args := []string{"--help"}
		_ = p.ParseArgs(args)
		return
	}

	stdoutx := "usage: testprog [option].. a-test [b-test [b-test [b-test]]]\n"
	stdoutx += "\n"
	stdoutx += "arguments:\n"
	stdoutx += "  a-test   usage-a\n"
	stdoutx += "  b-test   usage-b\n"
	stdoutx += "\n"
	stdoutx += "options:\n"
	stdoutx += "  -h, --help   display this help text and exit\n"
	testExec(t, 0, stdoutx, "")
}

func TestHelp_Requested_PosNArgMin1Max3(t *testing.T) {
	if testExecRun() {
		p := NewArgParser("testprog")
		var a string
		p.StringPosVar(&a, "a-test", "usage-a")
		var b []string
		p.StringPosNVar(&b, "b-test", "usage-b", 1, 3)
		args := []string{"--help"}
		_ = p.ParseArgs(args)
		return
	}

	stdoutx := "usage: testprog [option].. a-test b-test [b-test [b-test]]\n"
	stdoutx += "\n"
	stdoutx += "arguments:\n"
	stdoutx += "  a-test   usage-a\n"
	stdoutx += "  b-test   usage-b\n"
	stdoutx += "\n"
	stdoutx += "options:\n"
	stdoutx += "  -h, --help   display this help text and exit\n"
	testExec(t, 0, stdoutx, "")
}

func TestHelp_Requested_PosNArgMin3MaxInfinite(t *testing.T) {
	if testExecRun() {
		p := NewArgParser("testprog")
		var a string
		p.StringPosVar(&a, "a-test", "usage-a")
		var b []string
		p.StringPosNVar(&b, "b-test", "usage-b", 3, -1)
		args := []string{"--help"}
		_ = p.ParseArgs(args)
		return
	}

	stdoutx := "usage: testprog [option].. a-test b-test b-test b-test..\n"
	stdoutx += "\n"
	stdoutx += "arguments:\n"
	stdoutx += "  a-test   usage-a\n"
	stdoutx += "  b-test   usage-b\n"
	stdoutx += "\n"
	stdoutx += "options:\n"
	stdoutx += "  -h, --help   display this help text and exit\n"
	testExec(t, 0, stdoutx, "")
}

func TestMutuallyExclusive_Die_Dupes(t *testing.T) {
	defer func() {
		msgx := "MutuallyExclusive([\"a-test\" \"a-test\"]): cannot be defined with duplicate values"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.MutuallyExclusive("a-test", "a-test")
}

func TestMutuallyExclusive_Die_LessThanTwoNames(t *testing.T) {
	defer func() {
		msgx := "MutuallyExclusive([\"a-test\"]): cannot be defined with less than two names"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.MutuallyExclusive("a-test")
}

func TestMutuallyExclusive_Die_PostParse(t *testing.T) {
	defer func() {
		msgx := "MutuallyExclusive([\"a-test\" \"b-test\"]): cannot be defined post-parse"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a, b string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringVarP(&b, "b-test", "b", "default-b", "usage-b")
	_ = p.ParseArgs([]string{"-a", "test"})
	p.MutuallyExclusive("a-test", "b-test")
}

func TestMutuallyExclusive_Die_Undefined(t *testing.T) {
	defer func() {
		msgx :=
			"MutuallyExclusive([\"a-test\" \"b-test\"]): cannot be defined for undefined flag \"b-test\""
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.MutuallyExclusive("a-test", "b-test")
	_ = p.ParseArgs([]string{"-a", "test"})
}

func TestMutuallyExclusive_Fail(t *testing.T) {
	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	var b string
	p.StringVarP(&b, "b-test", "b", "default-b", "usage-b")
	p.MutuallyExclusive("a-test", "b-test")
	args := []string{"-a", "test", "-b", "test"}
	err := p.ParseArgs(args)
	testError(t, err, "a-test and b-test are mutually exclusive flags")
}

func TestMutuallyExclusive_OK(t *testing.T) {
	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	var b string
	p.StringVarP(&b, "b-test", "b", "default-b", "usage-b")
	p.MutuallyExclusive("a-test", "b-test")
	args := []string{"-a", "test"}
	err := p.ParseArgs(args)
	testNoError(t, err)
}

func TestParseFlag_Fail(t *testing.T) {
	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	args := []string{"-b", "test"}
	err := p.ParseArgs(args)
	testError(t, err, "unknown shorthand flag: 'b' in -b")
}

func TestParseFlag_OK(t *testing.T) {
	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	args := []string{"-a", "test"}
	err := p.ParseArgs(args)
	testNoError(t, err)
	if a != "test" {
		t.Fatalf("a: expected parsed value 'test', got: %q", a)
	}
}

func TestRequired_Die_EmptyName(t *testing.T) {
	defer func() {
		msgx :=
			"Required(\"\"): cannot be defined with empty name"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	p.Required("")
	_ = p.ParseArgs([]string{"-a", "test"})
}

func TestRequired_Die_PostParse(t *testing.T) {
	defer func() {
		msgx :=
			"Required(\"a-test\"): cannot be defined post-parse"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	_ = p.ParseArgs([]string{"-a", "test"})
	p.Required("a-test")
}

func TestRequired_Die_Undefined(t *testing.T) {
	defer func() {
		msgx :=
			"Required(\"a-test\"): cannot be defined for undefined flag"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	p.Required("a-test")
	_ = p.ParseArgs([]string{"-a", "test"})
}

func TestRequired_Fail(t *testing.T) {
	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	var b string
	p.StringVarP(&b, "b-test", "b", "default-b", "usage-b")
	p.Required("b-test")
	args := []string{"-a", "test"}
	err := p.ParseArgs(args)
	testError(t, err, "missing required flag: b-test")
}

func TestRequired_OK(t *testing.T) {
	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	var b string
	p.StringVarP(&b, "b-test", "b", "default-b", "usage-b")
	p.Required("b-test")
	args := []string{"-a", "test", "-b", "test"}
	err := p.ParseArgs(args)
	testNoError(t, err)
}

func TestStringAllowOptions_Die_EmptyName(t *testing.T) {
	defer func() {
		msgx :=
			"StringAllowOptions(\"\"): cannot be defined with empty name"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringAllowOptions(&a, "", []string{"x", "y", "z"})
}

func TestStringAllowOptions_Die_FlagNotString(t *testing.T) {
	defer func() {
		msgx :=
			"StringAllowOptions(\"b-test\"): cannot be defined for a flag that is not a string value"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	var b int
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.IntVarP(&b, "b-test", "b", 1, "usage-b")
	p.StringAllowOptions(&a, "b-test", []string{"x", "y", "z"})
}

func TestStringAllowOptions_Die_PostParse(t *testing.T) {
	defer func() {
		msgx :=
			"StringAllowOptions(\"a-test\"): cannot be defined post-parse"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	_ = p.ParseArgs([]string{"-a", "test"})
	p.StringAllowOptions(&a, "a-test", []string{"x", "y", "z"})
}

func TestStringAllowOptions_Die_Undefined(t *testing.T) {
	defer func() {
		msgx :=
			"StringAllowOptions(\"a-test\"): cannot be defined for undefined flag"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringAllowOptions(&a, "a-test", []string{"x", "y", "z"})
}

func TestStringAllowOptions_Fail(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringAllowOptions(&a, "a-test", []string{"test1", "test2", "test3"})
	args := []string{"-a", "test4"}
	err := p.ParseArgs(args)
	testError(t, err, "a-test: invalid value: \"test4\" is not among options: [\"test1\" \"test2\" \"test3\"]")
}

func TestStringAllowOptions_OK(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringAllowOptions(&a, "a-test", []string{"test1", "test2", "test3"})
	args := []string{"-a", "test2"}
	err := p.ParseArgs(args)
	testNoError(t, err)
}

func TestStringAllowRegexp_Die_EmptyName(t *testing.T) {
	defer func() {
		msgx :=
			"StringAllowRegexp(\"\"): cannot be defined with empty name"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringAllowRegexp(&a, "", ".*")
}

func TestStringAllowRegexp_Die_FlagNotString(t *testing.T) {
	defer func() {
		msgx :=
			"StringAllowRegexp(\"b-test\"): cannot be defined for a flag that is not a string value"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	var b int
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.IntVarP(&b, "b-test", "b", 1, "usage-b")
	p.StringAllowRegexp(&a, "b-test", ".*")
}

func TestStringAllowRegexp_Die_PostParse(t *testing.T) {
	defer func() {
		msgx :=
			"StringAllowRegexp(\"b-test\"): cannot be defined post-parse"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a, b string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringVarP(&b, "b-test", "b", "default-b", "usage-b")
	_ = p.ParseArgs([]string{"-a", "test"})
	p.StringAllowRegexp(&b, "b-test", ".*")
}

func TestStringAllowRegexp_Die_RegexCompile(t *testing.T) {
	defer func() {
		msgx :=
			"StringAllowRegexp(\"a-test\"): cannot be defined due to: error parsing regexp: missing closing ]: `[`"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringAllowRegexp(&a, "a-test", "[")
	_ = p.ParseArgs([]string{"-a", "test"})
}

func TestStringAllowRegexp_Die_Undefined(t *testing.T) {
	defer func() {
		msgx :=
			"StringAllowRegexp(\"a-test\"): cannot be defined for undefined flag"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringAllowRegexp(&a, "a-test", ".*")
}

func TestStringAllowRegexp_Fail(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringAllowRegexp(&a, "a-test", "^a")
	args := []string{"-a", "b"}
	err := p.ParseArgs(args)
	testError(t, err, "a-test: invalid value: \"b\" is not matching regexp \"^a\"")
}

func TestStringAllowRegexp_OK(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringAllowRegexp(&a, "a-test", "^a")
	args := []string{"-a", "abc"}
	err := p.ParseArgs(args)
	testNoError(t, err)
}

func TestStringDenyEmpty_Die_EmptyName(t *testing.T) {
	defer func() {
		msgx :=
			"StringDenyEmpty(\"\"): cannot be defined with empty name"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringDenyEmpty(&a, "")
}

func TestStringDenyEmpty_Die_FlagNotString(t *testing.T) {
	defer func() {
		msgx :=
			"StringDenyEmpty(\"b-test\"): cannot be defined for a flag that is not a string value"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	var b int
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.IntVarP(&b, "b-test", "b", 1, "usage-b")
	p.StringDenyEmpty(&a, "b-test")
}

func TestStringDenyEmpty_Die_PostParse(t *testing.T) {
	defer func() {
		msgx :=
			"StringDenyEmpty(\"b-test\"): cannot be defined post-parse"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a, b string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringVarP(&b, "b-test", "b", "default-b", "usage-b")
	_ = p.ParseArgs([]string{"-a", "test"})
	p.StringDenyEmpty(&b, "b-test")
}

func TestStringDenyEmpty_Die_Undefined(t *testing.T) {
	defer func() {
		msgx :=
			"StringDenyEmpty(\"a-test\"): cannot be defined for undefined flag"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringDenyEmpty(&a, "a-test")
}

func TestStringDenyEmpty_Fail_PosVar(t *testing.T) {
	p := NewArgParser("testprog")

	var a, b, c string
	p.StringPosVar(&a, "a-test", "usage-a")
	p.StringDenyEmpty(&a, "a-test")
	p.StringPosVar(&b, "b-test", "usage-b")
	p.StringDenyEmpty(&b, "b-test")
	p.StringPosVar(&c, "c-test", "usage-c")
	p.StringDenyEmpty(&c, "c-test")
	args := []string{"", "y", ""}
	err := p.ParseArgs(args)
	testError(t, err, "flags/arguments are empty: a-test, c-test")
}

func TestStringDenyEmpty_Fail_StringVarP(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringDenyEmpty(&a, "a-test")
	args := []string{"-a", ""}
	err := p.ParseArgs(args)
	testError(t, err, "flag/argument is empty: a-test")
}

func TestStringDenyEmpty_OK(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringDenyEmpty(&a, "a-test")
	args := []string{"-a", "x"}
	err := p.ParseArgs(args)
	testNoError(t, err)
}

func TestStringDenyEmpty_OK_StringVarP_EmptyDefault(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "", "usage-a")
	p.StringDenyEmpty(&a, "a-test")
	args := []string{}
	err := p.ParseArgs(args)
	testNoError(t, err)
}

func TestStringPosNVar_Die_AlreadyDefinedPosNVar(t *testing.T) {
	defer func() {
		msgx :=
			"StringPosNVar(\"b-test\"): cannot be defined as StringPosNVar(\"a-test\") is already defined"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a, b []string
	p.StringPosNVar(&a, "a-test", "usage-a", 1, 2)
	p.StringPosNVar(&b, "b-test", "usage-b", 1, 2)
}

func TestStringPosNVar_Die_AlreadyDefinedPosVar(t *testing.T) {
	defer func() {
		msgx :=
			"StringPosNVar(\"a-test\"): cannot be defined as StringPosVar(\"a-test\") is already defined"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	var b []string
	p.StringPosVar(&a, "a-test", "usage-a")
	p.StringPosNVar(&b, "a-test", "usage-a", 1, 2)
}

func TestStringPosNVar_Die_EmptyName(t *testing.T) {
	defer func() {
		msgx :=
			"StringPosNVar(\"\"): cannot be defined with empty name"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a []string
	p.StringPosNVar(&a, "", "usage-a", 1, 2)
}

func TestStringPosNVar_Die_MaxNEqualZero(t *testing.T) {
	defer func() {
		msgx :=
			"StringPosNVar(\"a-test\"): cannot be defined with maxN(0)"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a []string
	p.StringPosNVar(&a, "a-test", "usage-a", 1, 0)
}

func TestStringPosNVar_Die_MaxNLessThanMinusOne(t *testing.T) {
	defer func() {
		msgx := "StringPosNVar(\"a-test\"): cannot be defined with maxN(-2) < -1"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a []string
	p.StringPosNVar(&a, "a-test", "usage-a", 1, -2)
}

func TestStringPosNVar_Die_MinNGreaterThanMaxN(t *testing.T) {
	defer func() {
		msgx := "StringPosNVar(\"a-test\"): cannot be defined with minN(2) > maxN(1)"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a []string
	p.StringPosNVar(&a, "a-test", "usage-a", 2, 1)
}

func TestStringPosNVar_Die_MinNLessThanZero(t *testing.T) {
	defer func() {
		msgx := "StringPosNVar(\"a-test\"): cannot be defined with minN(-1) < 0"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a []string
	p.StringPosNVar(&a, "a-test", "usage-a", -1, 2)
}

func TestStringPosNVar_Fail_TooFew(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringPosVar(&a, "a", "usage-a")
	var b []string
	p.StringPosNVar(&b, "b", "usage-b", 1, -1)
	args := []string{"x"}
	err := p.ParseArgs(args)
	testError(t, err, "no \"b\" positional argument(s) provided, see --help")
}

func TestStringPosNVar_Fail_TooMany(t *testing.T) {
	p := NewArgParser("testprog")

	var a []string
	p.StringPosNVar(&a, "a", "usage-a", 1, 3)
	args := []string{"a", "b", "c", "d"}
	err := p.ParseArgs(args)
	testError(t, err, "got 4 \"a\" positional argument(s), expected 3 at most, see --help")
}

func TestStringPosNVar_OK_InfiniteNoneProvided(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringPosVar(&a, "a", "usage-a")
	var b []string
	p.StringPosNVar(&b, "b", "usage-b", 0, -1)
	args := []string{"x"}
	err := p.ParseArgs(args)
	testNoError(t, err)
	if len(b) != 0 {
		t.Fatalf("b: expected zero-length, got: %d", len(a))
	}
}

func TestStringPosNVar_OK_InfiniteProvided(t *testing.T) {
	p := NewArgParser("testprog")

	var a []string
	p.StringPosNVar(&a, "a", "usage-a", 0, -1)
	args := []string{"a", "b"}
	err := p.ParseArgs(args)
	testNoError(t, err)
	if len(a) != 2 {
		t.Fatalf("a: expected length 2, got: %d", len(a))
	}
	if a[0] != "a" {
		t.Fatalf("a[0]: expected parsed value 'a', got: %q", a[0])
	}
	if a[1] != "b" {
		t.Fatalf("a[1]: expected parsed value 'b', got: %q", a[1])
	}
}

func TestStringPosNVar_OK_Min1Max2Provided1(t *testing.T) {
	p := NewArgParser("testprog")

	var a []string
	p.StringPosNVar(&a, "a", "usage-a", 1, 2)
	args := []string{"a"}
	err := p.ParseArgs(args)
	testNoError(t, err)
	if len(a) != 1 {
		t.Fatalf("a: expected length 1, got: %d", len(a))
	}
	if a[0] != "a" {
		t.Fatalf("a[0]: expected parsed value 'a', got: %q", a[0])
	}
}

func TestStringPosNVar_OK_Min1Max2Provided2(t *testing.T) {
	p := NewArgParser("testprog")

	var a []string
	p.StringPosNVar(&a, "a", "usage-a", 1, 2)
	args := []string{"a", "b"}
	err := p.ParseArgs(args)
	testNoError(t, err)
	if len(a) != 2 {
		t.Fatalf("a: expected length 2, got: %d", len(a))
	}
	if a[0] != "a" {
		t.Fatalf("a[0]: expected parsed value 'a', got: %q", a[0])
	}
	if a[1] != "b" {
		t.Fatalf("a[1]: expected parsed value 'b', got: %q", a[1])
	}
}

func TestStringPosNVar_OK_Min3Max3(t *testing.T) {
	p := NewArgParser("testprog")

	var a []string
	p.StringPosNVar(&a, "a", "usage-a", 3, 3)
	args := []string{"a", "b", "c"}
	err := p.ParseArgs(args)
	testNoError(t, err)
	if len(a) != 3 {
		t.Fatalf("a: expected length 3, got: %d", len(a))
	}
	if a[0] != "a" {
		t.Fatalf("a[0]: expected parsed value 'a', got: %q", a[0])
	}
	if a[1] != "b" {
		t.Fatalf("a[1]: expected parsed value 'b', got: %q", a[1])
	}
	if a[2] != "c" {
		t.Fatalf("a[2]: expected parsed value 'c', got: %q", a[2])
	}
}

func TestStringPosVar_Die_AlreadyDefinedAsFlag(t *testing.T) {
	defer func() {
		msgx :=
			"StringPosVar(\"a-test\"): cannot be defined as already defined as flag"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a, b string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringPosVar(&b, "a-test", "usage-a")
}

func TestStringPosVar_Die_AlreadyDefinedPosNVar(t *testing.T) {
	defer func() {
		msgx :=
			"StringPosVar(\"b-test\"): cannot be defined as StringPosNVar(\"a-test\") is already defined"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a []string
	var b string
	p.StringPosNVar(&a, "a-test", "usage-a", 1, 2)
	p.StringPosVar(&b, "b-test", "usage-b")
}

func TestStringPosVar_Die_AlreadyDefinedPosVarName(t *testing.T) {
	defer func() {
		msgx :=
			"StringPosVar(\"a-test\"): cannot be defined as StringPosVar(\"a-test\") is already defined"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a, b string
	p.StringPosVar(&a, "a-test", "usage-a")
	p.StringPosVar(&b, "a-test", "usage-a")
}

func TestStringPosVar_Die_AlreadyDefinedPosVarTarget(t *testing.T) {
	defer func() {
		msgx :=
			"StringPosVar(\"b-test\"): cannot be defined using the same target as StringPosVar(\"a-test\")"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringPosVar(&a, "a-test", "usage-a")
	p.StringPosVar(&a, "b-test", "usage-b")
}

func TestStringPosVar_Die_EmptyName(t *testing.T) {
	defer func() {
		msgx :=
			"StringPosVar(\"\"): cannot be defined with empty name"
		testPanic(t, recover(), msgx)
	}()

	p := NewArgParser("testprog")
	var a string
	p.StringPosVar(&a, "", "usage-a")
}

func TestStringPosVar_Fail(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringPosVar(&a, "a", "usage-a")
	var b string
	p.StringPosVar(&b, "b", "usage-b")
	args := []string{"x"}
	err := p.ParseArgs(args)
	testError(t, err, "insufficient number of positional arguments, see --help")
}

func TestStringPosVar_OK(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringPosVar(&a, "a", "usage-a")
	var b string
	p.StringPosVar(&b, "b", "usage-b")
	args := []string{"x", "y"}
	err := p.ParseArgs(args)
	testNoError(t, err)
	if a != "x" {
		t.Fatalf("a: expected parsed value 'x', got: %q", a)
	}
	if b != "y" {
		t.Fatalf("b: expected parsed value 'y', got: %q", b)
	}
}
