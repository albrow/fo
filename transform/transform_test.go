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

func::(T) f(x T) {}

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

type Map struct::(T, U) {
	m map[T]U
}

func main() {
	a := Box::(string){}
	b := &Box::(int){}
	c := []Box::(string){}
	d := [2]Box::(int){}
	e := map[string]Box::(string){}

	f := Map::(string, int){}

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

type Map__string__int struct {
	m map[string]int
}

func main() {
	a := Box__string{}
	b := &Box__int{}
	c := []Box__string{}
	d := [2]Box__int{}
	e := map[string]Box__string{}

	f := Map__string__int{}

	myTuple := Tuple__int__string{
		first:  2,
		second: "foo",
	}
}
`
	testParseFile(t, src, expected)
}

func TestTransformStructTypeSelectorUsage(t *testing.T) {
	src := `package main

import "bytes"

type Box struct::(T) {
	val T
}

func main() {
	x := Box::(bytes.Buffer){}
}
`

	expected := `package main

import "bytes"

type Box__bytes_Buffer struct {
	val bytes.Buffer
}

func main() {
	x := Box__bytes_Buffer{}
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

type Box__int struct {
	val int
}
type Box__string struct {
	val string
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

type Box__int struct {
	val int
}
type Box__string struct {
	val string
}

func main() {
	var x interface{} = Box__int{}
	_ = x.(Box__int)
	_ = x.(Box__string)
}
`

	testParseFile(t, src, expected)
}

func TestTransformFuncDecl(t *testing.T) {
	src := `package main

func::(T) Print(t T) {
	fmt.Println(t)
}

func::(T) MakeSlice() []T {
	return make([]T)
}

func main() {
	Print::(int)(5)
	Print::(int)(42)
	Print::(string)("foo")
	MakeSlice::(string)()
}
`

	expected := `package main

func Print__int(t int) {
	fmt.Println(t)
}
func Print__string(t string) {
	fmt.Println(t)
}

func MakeSlice__string() ([]string) {
	return make([]string)
}

func main() {
	Print__int(5)
	Print__int(42)
	Print__string("foo")
	MakeSlice__string()
}
`

	testParseFile(t, src, expected)
}

func TestTransformStructTypeInherited(t *testing.T) {
	src := `package main

type Tuple struct::(T, U) {
	first T
	second U
}

type BoxedTuple struct::(T, U) {
	val Tuple::(T, U)
}

type BoxedTupleString struct::(T) {
	val Tuple::(string, T)
}

func main() {
	a := BoxedTuple::(string, int){
		val: Tuple::(string, int){
			first: "foo",
			second: 42,
		},
	}
	b := BoxedTuple::(float64, int){}
	c := BoxedTupleString::(bool){
		val: Tuple::(string, bool) {
			first: "abc",
			second: true,
		},
	}
	d := BoxedTupleString::(uint){}
}
`

	expected := `package main

type Tuple__float64__int struct {
	first  float64
	second int
}
type Tuple__string__bool struct {
	first  string
	second bool
}
type Tuple__string__int struct {
	first  string
	second int
}
type Tuple__string__uint struct {
	first  string
	second uint
}

type BoxedTuple__float64__int struct {
	val Tuple__float64__int
}
type BoxedTuple__string__int struct {
	val Tuple__string__int
}

type BoxedTupleString__bool struct {
	val Tuple__string__bool
}
type BoxedTupleString__uint struct {
	val Tuple__string__uint
}

func main() {
	a := BoxedTuple__string__int{
		val: Tuple__string__int{
			first:  "foo",
			second: 42,
		},
	}
	b := BoxedTuple__float64__int{}
	c := BoxedTupleString__bool{
		val: Tuple__string__bool{
			first:  "abc",
			second: true,
		},
	}
	d := BoxedTupleString__uint{}
}
`

	testParseFile(t, src, expected)
}

func TestTransformFuncDeclInherited(t *testing.T) {
	src := `package main

type Tuple struct::(T, U) {
	first T
	second U
}

func::(T, U) NewTuple(first T, second U) Tuple::(T, U) {
	return Tuple::(T, U){
		first: first,
		second: second,
	}
}

func::(T) NewTupleString(first string, second T) Tuple::(string, T) {
	return Tuple::(string, T) {
		first: first,
		second: second,
	}
}

func main() {
	x := NewTuple::(bool, int64)(true, 42)
	y := NewTupleString::(float64)("foo", 12.34)
}
`

	expected := `package main

type Tuple__bool__int64 struct {
	first  bool
	second int64
}
type Tuple__string__float64 struct {
	first  string
	second float64
}

func NewTuple__bool__int64(first bool, second int64) (Tuple__bool__int64) {
	return Tuple__bool__int64{
		first:  first,
		second: second,
	}
}

func NewTupleString__float64(first string, second float64) (Tuple__string__float64) {
	return Tuple__string__float64{
		first:  first,
		second: second,
	}
}

func main() {
	x := NewTuple__bool__int64(true, 42)
	y := NewTupleString__float64("foo", 12.34)
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
