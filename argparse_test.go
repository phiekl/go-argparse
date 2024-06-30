// SPDX-FileCopyrightText: 2024 Philip Eklöf
//
// SPDX-License-Identifier: MIT

package argparse

import (
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

func testNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("unexpected parsing error: %v", err)
	}
}

func TestMutuallyExclusiveFail(t *testing.T) {
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

func TestMutuallyExclusiveOK(t *testing.T) {
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

func TestParseFlagFail(t *testing.T) {
	p := NewArgParser("testprog")
	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	args := []string{"-b", "test"}
	err := p.ParseArgs(args)
	testError(t, err, "unknown shorthand flag: 'b' in -b")
}

func TestParseFlagOK(t *testing.T) {
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

func TestRequiredFail(t *testing.T) {
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

func TestRequiredOK(t *testing.T) {
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

func TestStringAllowOptionsFail(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringAllowOptions(&a, "a-test", []string{"test1", "test2", "test3"})
	args := []string{"-a", "test4"}
	err := p.ParseArgs(args)
	testError(t, err, "a-test: invalid value: \"test4\" is not among options: [\"test1\" \"test2\" \"test3\"]")
}

func TestStringAllowOptionsOK(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringAllowOptions(&a, "a-test", []string{"test1", "test2", "test3"})
	args := []string{"-a", "test2"}
	err := p.ParseArgs(args)
	testNoError(t, err)
}

func TestStringAllowRegexpFail(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringAllowRegexp(&a, "a-test", "^a")
	args := []string{"-a", "b"}
	err := p.ParseArgs(args)
	testError(t, err, "a-test: invalid value: \"b\" is not matching regexp \"^a\"")
}

func TestStringAllowRegexpOK(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringVarP(&a, "a-test", "a", "default-a", "usage-a")
	p.StringAllowRegexp(&a, "a-test", "^a")
	args := []string{"-a", "abc"}
	err := p.ParseArgs(args)
	testNoError(t, err)
}

func TestStringPosVarFail(t *testing.T) {
	p := NewArgParser("testprog")

	var a string
	p.StringPosVar(&a, "a", "usage-a")
	var b string
	p.StringPosVar(&b, "b", "usage-b")
	args := []string{"x"}
	err := p.ParseArgs(args)
	testError(t, err, "insufficient number of positional arguments, see --help")
}

func TestStringPosVarOK(t *testing.T) {
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

func TestStringPosNVarFailTooFew(t *testing.T) {
	p := NewArgParser("testprog")

	var a []string
	p.StringPosNVar(&a, "a", "usage-a", 1, -1)
	args := []string{}
	err := p.ParseArgs(args)
	testError(t, err, "no \"a\" positional argument(s) provided, see --help")
}

func TestStringPosNVarFailTooMany(t *testing.T) {
	p := NewArgParser("testprog")

	var a []string
	p.StringPosNVar(&a, "a", "usage-a", 1, 3)
	args := []string{"a", "b", "c", "d"}
	err := p.ParseArgs(args)
	testError(t, err, "got 4 \"a\" positional argument(s), expected 3 at most, see --help")
}

func TestStringPosNVarOKInfiniteNoneProvided(t *testing.T) {
	p := NewArgParser("testprog")

	var a []string
	p.StringPosNVar(&a, "a", "usage-a", 0, -1)
	args := []string{}
	err := p.ParseArgs(args)
	testNoError(t, err)
	if len(a) != 0 {
		t.Fatalf("a: expected zero-length, got: %d", len(a))
	}
}

func TestStringPosNVarOKInfiniteProvided(t *testing.T) {
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

func TestStringPosNVarOKMin1Max2Provided1(t *testing.T) {
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

func TestStringPosNVarOKMin1Max2Provided2(t *testing.T) {
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

func TestStringPosNVarOKMin3Max3(t *testing.T) {
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
