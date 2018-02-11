package transform

import (
	"bytes"
	"testing"

	"github.com/albrow/fo/parser"
	"github.com/albrow/fo/printer"
	"github.com/albrow/fo/token"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestTransformStructTypeLiterals(t *testing.T) {
	src := `package main

import "fmt"

type Box struct::(T) {
	val T
}

type Tuple struct::(T, V) {
	first T
	second V
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
	first	int
	second	string
}

func main() {
	x := Box__string{val: "foo"}
	y := Box__int{val: 2}
	myTuple := Tuple__int__string{
		first:	2,
		second:	"foo",
	}
}
`

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
	if err := printer.Fprint(output, fset, transformed); err != nil {
		t.Fatalf("printer.Fprint returned error: %s", err.Error())
	}
	if output.String() != expected {
		diff := diffmatchpatch.New()
		t.Fatalf(
			"output of Transform did not match expected\n\n%s",
			diff.DiffPrettyText(diff.DiffMain(expected, output.String(), false)),
		)
	}
}
