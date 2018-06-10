package main

import "fmt"

type Set__int map[int]struct {
}

func New__int() Set__int {
	return Set__int{}
}

func NewFromSlice__int(slice []int) Set__int {
	s := New__int()
	for _, v := range slice {
		s.Add(v)
	}
	return s
}

func (s Set__int) Add(vs ...int) {
	for _, v := range vs {
		s[v] = struct {
		}{}
	}
}

func (s Set__int) Remove(v int) {
	delete(s, v)
}

func (s Set__int) Contains(v int) bool {
	_, ok := s[v]
	return ok
}

func (s Set__int) Slice() []int {
	slice := make([]int, len(s))
	i := 0
	for v := range s {
		slice[i] = v
		i++
	}
	return slice
}

func (s Set__int) String() string {
	return fmt.Sprint(s.Slice())
}

func Union__int(a, b Set__int) Set__int {
	result := New__int()
	for v := range a {
		result.Add(v)
	}
	for v := range b {
		result.Add(v)
	}
	return result
}

func Intersect__int(a, b Set__int) Set__int {
	result := New__int()
	for v := range a {
		if b.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

func Diff__int(a, b Set__int) Set__int {
	result := New__int()
	for v := range a {
		if !b.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

func main() {
	a := NewFromSlice__int([]int{
		1,
		2,
		3,
	})
	fmt.Printf("a contains 1: %v\n", a.Contains(1))

	fmt.Printf("a: %s\n", a)

	b := NewFromSlice__int([]int{
		2,
		3,
		4,
	})
	fmt.Printf("b: %s\n", b)

	fmt.Printf("Union: %v\n", Union__int(a, b))

	fmt.Printf("Intersect: %v\n", Intersect__int(a, b))

	fmt.Printf("Diff(a, b): %v\n", Diff__int(a, b))

	fmt.Printf("Diff(b, a): %v\n", Diff__int(b, a))

	a.Remove(2)
	fmt.Printf("a after calling Remove(2): %s\n", a)

	fmt.Printf("a as a slice: %v\n", a.Slice())

}
