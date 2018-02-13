package transform

import (
	"bytes"
	"testing"

	"github.com/albrow/fo/format"
	"github.com/albrow/fo/parser"
	"github.com/albrow/fo/token"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestTransformStructTypeLiterals(t *testing.T) {
	src := `package main

import "fmt"

type Box struct::(T) {
	val T
}

type Tuple struct::(T, U) {
	first T
	second U
}

type Unused struct::(T) {
	val T
}

func main() {
	x := Box::(string) { val: "foo" }
	y := Box::(int) { val: 2 }
	myTuple := Tuple::(int, string) {
		first: 2,
		second: "foo",
	}
}
`

	expected := `package main

import "fmt"

type Box__int struct {
	val int
}
type Box__string struct {
	val string
}

type Tuple__int__string struct {
	first  int
	second string
}

func main() {
	x := Box__string{val: "foo"}
	y := Box__int{val: 2}
	myTuple := Tuple__int__string{
		first:  2,
		second: "foo",
	}
}
`
	testParseFile(t, src, expected)
}

func TestTransformStructTypeInsideFuncType(t *testing.T) {
	src := `package main

type Either struct::(T, U) {
	left T
	right U
}

func getData() Either::(int, string) {
}

func handleEither(e Either::(error, string)) {
}

func main() { }
`

	expected := `package main

type Either__error__string struct {
	left  error
	right string
}
type Either__int__string struct {
	left  int
	right string
}

func getData() Either__int__string {
}

func handleEither(e Either__error__string,) {
}

func main() {}
`

	testParseFile(t, src, expected)
}

func testParseFile(t *testing.T, src string, expected string) {
	t.Helper()
	fset := token.NewFileSet()
	orig, err := parser.ParseFile(fset, "struct_type_literals", src, 0)
	if err != nil {
		t.Fatalf("ParseFile returned error: %s", err.Error())
	}
	transformed, err := File(fset, orig)
	if err != nil {
		t.Fatalf("Transform returned error: %s", err.Error())
	}
	output := bytes.NewBuffer(nil)
	if err := format.Node(output, fset, transformed); err != nil {
		t.Fatalf("format.Node returned error: %s", err.Error())
	}
	if output.String() != expected {
		diff := diffmatchpatch.New()
		t.Fatalf(
			"output of Transform did not match expected\n\n%s",
			diff.DiffPrettyText(diff.DiffMain(expected, output.String(), false)),
		)
	}
}
