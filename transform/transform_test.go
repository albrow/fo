package transform

import (
	"bytes"
	"strings"
	"testing"

	"github.com/albrow/fo/format"
	"github.com/albrow/fo/parser"
	"github.com/albrow/fo/token"
	"github.com/aryann/difflib"
)

func TestTransformStructTypeUnused(t *testing.T) {
	src := `package main

type T struct::(U) {}

func main() { }
`

	expected := `package main

func main() {}
`

	testParseFile(t, src, expected)
}
func TestTransformStructTypeLiterals(t *testing.T) {
	src := `package main

type Box struct::(T) {
	val T
}

type Tuple struct::(T, U) {
	first T
	second U
}

func main() {
	a := Box::(string){}
	b := &Box::(int){}
	c := []Box::(string){}
	d := [2]Box::(int){}
	e := map[string]Box::(string){}

	myTuple := Tuple::(int, string) {
		first: 2,
		second: "foo",
	}
}
`

	expected := `package main

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
	a := Box__string{}
	b := &Box__int{}
	c := []Box__string{}
	d := [2]Box__int{}
	e := map[string]Box__string{}

	myTuple := Tuple__int__string{
		first:  2,
		second: "foo",
	}
}
`
	testParseFile(t, src, expected)
}

func TestTransformStructTypeFuncArgs(t *testing.T) {
	src := `package main

type Either struct::(T, U) {
	left T
	right U
}

func getData() Either::(int, string) {
	return Either::(int, string){}
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
	return Either__int__string{}
}

func handleEither(e Either__error__string) {
}

func main() {}
`

	testParseFile(t, src, expected)
}

func TestTransformStructTypeMethodReceiver(t *testing.T) {
	src := `package main

type Maybe struct::(T) {
	val T
	valid bool
}

func (m Maybe::(string)) IsValid() bool {
	return m.valid
}

func main() { }
`

	expected := `package main

type Maybe__string struct {
	val   string
	valid bool
}

func (m Maybe__string) IsValid() bool {
	return m.valid
}

func main() {}
`

	testParseFile(t, src, expected)
}

func TestTransformStructTypeSwitch(t *testing.T) {
	src := `package main

type Box struct::(T) {
	val T
}

func main() { 
	var x interface{} = Box::(int){}
	switch x.(type) {
	case Box::(int):
	case Box::(string):
	}
}
`

	expected := `package main

type Box__string struct {
	val string
}
type Box__int struct {
	val int
}

func main() {
	var x interface{} = Box__int{}
	switch x.(type) {
	case Box__int:
	case Box__string:
	}
}
`

	testParseFile(t, src, expected)
}

func TestTransformStructTypeAssert(t *testing.T) {
	src := `package main

type Box struct::(T) {
	val T
}

func main() { 
	var x interface{} = Box::(int){}
	_ = x.(Box::(int))
	_ = x.(Box::(string))
}
`

	expected := `package main

type Box__string struct {
	val string
}
type Box__int struct {
	val int
}

func main() {
	var x interface{} = Box__int{}
	_ = x.(Box__int)
	_ = x.(Box__string)
}
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
		diff := difflib.Diff(strings.Split(expected, "\n"), strings.Split(output.String(), "\n"))
		diffStrings := ""
		for _, d := range diff {
			diffStrings += d.String() + "\n"
		}
		t.Fatalf(
			"output of Transform did not match expected\n\n%s",
			diffStrings,
		)
	}
}
