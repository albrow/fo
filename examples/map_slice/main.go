package main

import (
	"fmt"
	"strings"
)

func mapSlice__int__int(f func(int) int, list []int) []int {
	result := make([]int, len(list))
	for i, val := range list {
		result[i] = f(val)
	}
	return result
}
func mapSlice__int__uint(f func(int) uint, list []int) []uint {
	result := make([]uint, len(list))
	for i, val := range list {
		result[i] = f(val)
	}
	return result
}
func mapSlice__string__string(f func(string) string, list []string) []string {
	result := make([]string, len(list))
	for i, val := range list {
		result[i] = f(val)
	}
	return result
}

func incr(n int) int {
	return n + 1
}

func main() {

	numbers := []int{1, 2, 3}
	fmt.Println(mapSlice__int__int(incr, numbers))

	uints := mapSlice__int__uint(func(n int) uint { return uint(n) }, numbers)
	fmt.Printf("uints has type %T\n", uints)

	strns := []string{"apple", "banana", "carrot"}
	fmt.Println(mapSlice__string__string(strings.ToUpper, strns))

}
