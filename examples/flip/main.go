package main

import (
	"fmt"
	"strconv"
)

func flip____byte__string____byte(f func([]byte, string) []byte) func(string, []byte) []byte {
	return func(p1 string, p0 []byte) []byte {
		return f(p0, p1)
	}
}

var appendQuote = flip____byte__string____byte(strconv.AppendQuote)

func main() {
	fmt.Printf("original: %T\n", strconv.AppendQuote)

	fmt.Printf("flipped: %T\n", appendQuote)

	buf := []byte("quote: ")
	quote := `"Hello!"`
	result := appendQuote(quote, buf)
	fmt.Println(string(result))

}
