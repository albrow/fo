package transform

import (
	"bytes"
	"strings"
	"testing"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/format"
	"github.com/albrow/fo/importer"
	"github.com/albrow/fo/parser"
	"github.com/albrow/fo/token"
	"github.com/albrow/fo/types"
	"github.com/aryann/difflib"
)

func TestTransformStructTypeUnused(t *testing.T) {
	src := `package main

type T[U] struct {}

func f[T](x T) {}

func (T[U]) f0() {}

func (T) f1() {}

func main() { }
`

	expected := `package main

func main() {}
`

	testParseFile(t, src, expected)
}

func TestTransformStructTypeLiterals(t *testing.T) {
	src := `package main

type Box[T] struct {
	val T
}

type Tuple[T, U] struct {
	first T
	second U
}

type Map[T, U] struct {
	m map[T]U
}

func main() {
	var _ = Box[string]{}
	var _ = &Box[int]{}
	var _ = []Box[string]{}
	var _ = [2]Box[int]{}
	var _ = map[string]Box[string]{}

	var _ = Map[string, int]{}

	var _ = Tuple[int, string] {
		first: 2,
		second: "foo",
	}
}
`

	expected := `package main

type (
	Box__int struct {
		val int
	}
	Box__string struct {
		val string
	}
)

type Tuple__int__string struct {
	first  int
	second string
}

type Map__string__int struct {
	m map[string]int
}

func main() {
	var _ = Box__string{}
	var _ = &Box__int{}
	var _ = []Box__string{}
	var _ = [2]Box__int{}
	var _ = map[string]Box__string{}

	var _ = Map__string__int{}

	var _ = Tuple__int__string{
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

type Box[T] struct{
	val T
}

func main() {
	var _ = Box[bytes.Buffer]{}
}
`

	expected := `package main

import "bytes"

type Box__bytes_Buffer struct {
	val bytes.Buffer
}

func main() {
	var _ = Box__bytes_Buffer{}
}
`
	testParseFile(t, src, expected)
}

func TestTransformStructTypeFuncArgs(t *testing.T) {
	src := `package main

type Either[T, U] struct {
	left T
	right U
}

func getData() Either[int, string] {
	return Either[int, string]{}
}

func handleEither(e Either[error, string]) {
}

func main() { }
`

	expected := `package main

type (
	Either__error__string struct {
		left  error
		right string
	}
	Either__int__string struct {
		left  int
		right string
	}
)

func getData() Either__int__string {
	return Either__int__string{}
}

func handleEither(e Either__error__string) {
}

func main() {}
`

	testParseFile(t, src, expected)
}

func TestTransformStructTypeSwitch(t *testing.T) {
	src := `package main

type Box[T] struct {
	val T
}

func main() {
	var x interface{} = Box[int]{}
	switch x.(type) {
	case Box[int]:
	case Box[string]:
	}
}
`

	expected := `package main

type (
	Box__int struct {
		val int
	}
	Box__string struct {
		val string
	}
)

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

type Box[T] struct {
	val T
}

func main() {
	var x interface{} = Box[int]{}
	_ = x.(Box[int])
	_ = x.(Box[string])
}
`

	expected := `package main

type (
	Box__int struct {
		val int
	}
	Box__string struct {
		val string
	}
)

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

import "fmt"

func Print[T](t T) {
	fmt.Println(t)
}

func MakeSlice[T]() []T {
	return make([]T, 0)
}

func main() {
	Print[int](5)
	Print[int](42)
	Print[string]("foo")
	MakeSlice[string]()
}
`

	expected := `package main

import "fmt"

func Print__int(t int) {
	fmt.Println(t)
}
func Print__string(t string) {
	fmt.Println(t)
}

func MakeSlice__string() []string {
	return make([]string, 0)
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

type Tuple[T, U] struct {
	first T
	second U
}

type BoxedTuple[T, U] struct {
	val Tuple[T, U]
}

type BoxedTupleString[T] struct {
	val Tuple[string, T]
}

func main() {
	var _ = BoxedTuple[string, int]{
		val: Tuple[string, int]{
			first: "foo",
			second: 42,
		},
	}
	var _ = BoxedTuple[float64, int]{}
	var _ = BoxedTupleString[bool]{
		val: Tuple[string, bool] {
			first: "abc",
			second: true,
		},
	}
	var _ = BoxedTupleString[uint]{}
}
`

	expected := `package main

type (
	Tuple__float64__int struct {
		first  float64
		second int
	}
	Tuple__string__bool struct {
		first  string
		second bool
	}
	Tuple__string__int struct {
		first  string
		second int
	}
	Tuple__string__uint struct {
		first  string
		second uint
	}
)

type (
	BoxedTuple__float64__int struct {
		val Tuple__float64__int
	}
	BoxedTuple__string__int struct {
		val Tuple__string__int
	}
)

type (
	BoxedTupleString__bool struct {
		val Tuple__string__bool
	}
	BoxedTupleString__uint struct {
		val Tuple__string__uint
	}
)

func main() {
	var _ = BoxedTuple__string__int{
		val: Tuple__string__int{
			first:  "foo",
			second: 42,
		},
	}
	var _ = BoxedTuple__float64__int{}
	var _ = BoxedTupleString__bool{
		val: Tuple__string__bool{
			first:  "abc",
			second: true,
		},
	}
	var _ = BoxedTupleString__uint{}
}
`

	testParseFile(t, src, expected)
}

func TestTransformFuncDeclInherited(t *testing.T) {
	src := `package main

type Tuple[T, U] struct {
	first T
	second U
}

func NewTuple[T, U](first T, second U) Tuple[T, U] {
	return Tuple[T, U]{
		first: first,
		second: second,
	}
}

func NewTupleString[T](first string, second T) Tuple[string, T] {
	return Tuple[string, T] {
		first: first,
		second: second,
	}
}

func main() {
	var _ = NewTuple[bool, int64](true, 42)
	var _ = NewTupleString[float64]("foo", 12.34)
}
`

	expected := `package main

type (
	Tuple__bool__int64 struct {
		first  bool
		second int64
	}
	Tuple__string__float64 struct {
		first  string
		second float64
	}
)

func NewTuple__bool__int64(first bool, second int64) Tuple__bool__int64 {
	return Tuple__bool__int64{
		first:  first,
		second: second,
	}
}

func NewTupleString__float64(first string, second float64) Tuple__string__float64 {
	return Tuple__string__float64{
		first:  first,
		second: second,
	}
}

func main() {
	var _ = NewTuple__bool__int64(true, 42)
	var _ = NewTupleString__float64("foo", 12.34)
}
`

	testParseFile(t, src, expected)
}

func TestTransformMethods(t *testing.T) {
	src := `package main

type A[T] T

func (A[T]) f0() T {
	var x T
	return x
}

func (a A[T]) f1() T {
	return T(a)
}

func (a A[T]) f2[U, V]() (T, U, V) {
	var x U
	var y V
	return T(a), x, y
}

func (*A) f3() {}

func main() {
	var _ = A[string]("")
	var _ = A[bool](true)

	var x A[uint]
	x.f2[float64, int8]()
}
`

	expected := `package main

type (
	A__bool   bool
	A__string string
	A__uint   uint
)

func (A__bool) f0() bool {
	var x bool
	return x
}
func (A__string) f0() string {
	var x string
	return x
}
func (A__uint) f0() uint {
	var x uint
	return x
}

func (a A__bool) f1() bool {
	return bool(a)
}
func (a A__string) f1() string {
	return string(a)
}
func (a A__uint) f1() uint {
	return uint(a)
}

func (a A__T) f2__float64__int8() (T, float64, int8) {
	var x float64
	var y int8
	return T(a), x, y
}

func (*A__bool) f3()   {}
func (*A__string) f3() {}
func (*A__uint) f3()   {}

func main() {
	var _ = A__string("")
	var _ = A__bool(true)

	var x A__uint
	x.f2__float64__int8()
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
	conf := types.Config{}
	conf.Importer = importer.Default()
	pkg, err := conf.Check("transformtest", fset, []*ast.File{orig}, nil)
	if err != nil {
		t.Fatalf("conf.Check returned error: %s", err.Error())
	}
	transformed, err := File(fset, orig, pkg)
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
