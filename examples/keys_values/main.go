package main

import "fmt"

func keys__string__int(m map[string]int) []string {
	result := make([]string, len(m))
	i := 0
	for key := range m {
		result[i] = key
		i++
	}
	return result
}

func values__string__int(m map[string]int) []int {
	result := make([]int, len(m))
	i := 0
	for _, val := range m {
		result[i] = val
		i++
	}
	return result
}

func main() {
	m := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}

	fmt.Println(keys__string__int(m))

	fmt.Println(values__string__int(m))

}
