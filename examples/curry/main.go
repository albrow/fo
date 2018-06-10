package main

import "fmt"

func curry2__int__int__int(f func(int, int) int) func(int) func(int) int {
	return func(p1 int) func(int) int {
		return func(p2 int) int {
			return f(p1, p2)
		}
	}
}

var add = curry2__int__int__int(
	func(a, b int) int {
		return a + b
	},
)

func main() {

	incr := add(1)
	fmt.Println(incr(1))

	fmt.Println(incr(10))

	decr := add(-1)
	fmt.Println(decr(1))

	fmt.Println(decr(10))

}
