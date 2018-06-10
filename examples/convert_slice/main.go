package main

import "fmt"

func convertSlice__int__uint(s []int) []uint {
	result := make([]uint, len(s))
	for i, v := range s {
		result[i] = uint(v)
	}
	return result
}

func main() {
	ints := []int{1, 2, 3}
	uints := convertSlice__int__uint(ints)
	fmt.Printf("slice was converted from %T to %T\n", ints, uints)

	fmt.Println(uints)

}
