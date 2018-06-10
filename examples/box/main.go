package main

import (
	"fmt"
	"strconv"
)

type (
	Box__int struct {
		v int
	}
	Box__string struct {
		v string
	}
)

func (b Box__int) Val() int {
	return b.v
}
func (b Box__string) Val() string {
	return b.v
}

func (b Box__int) Map__string(f func(int) string) Box__string {
	return Box__string{
		v: f(b.v),
	}
}

func main() {

	x := Box__string{v: "foo"}
	fmt.Printf("x is of type Box[%T] and has value: %q\n", x.Val(), x.Val())

	y := Box__int{v: 42}
	fmt.Printf("y is of type Box[%T] and has value: %v\n", y.Val(), y.Val())

	z := y.Map__string(strconv.Itoa)
	fmt.Printf("z is of type Box[%T] and has value: %q\n", z.Val(), z.Val())

}
