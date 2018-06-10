package main

import (
	"fmt"
	"strconv"
)

func pipe3__string__int__int__string(f0 func(string) (int, error),
	f1 func(int) (int, error),
	f2 func(int) (string, error)) func(string) (string, error) {
	return func(p0 string) (string, error) {
		p1, err := f0(p0)
		if err != nil {
			var r string
			return r, err
		}
		p2, err := f1(p1)
		if err != nil {
			var r string
			return r, err
		}
		return f2(p2)
	}
}

func neverError__int__string(f func(int) string) func(int) (string, error) {
	return func(t int) (string, error) {
		return f(t), nil
	}
}

func decr(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("cannot decrement %d", n)
	}
	return n - 1, nil
}

var decrString = pipe3__string__int__int__string(
	strconv.Atoi,
	decr,

	neverError__int__string(strconv.Itoa),
)

func main() {
	fmt.Printf("decrString has type: %T\n", decrString)

	var err error
	result, err := decrString("5")
	fmt.Println(result, err)

	_, err = decrString("apple")
	fmt.Println(err)

	_, err = decrString("0")
	fmt.Println(err)

}
