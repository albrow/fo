package main

import "fmt"

func main() {

	l := New__int()
	e4 := l.PushBack(4)
	e1 := l.PushFront(1)
	l.InsertBefore(3, e4)
	l.InsertAfter(2, e1)

	for e := l.Front(); e != nil; e = e.Next() {
		fmt.Println(e.Value)
	}
}

type Element__int struct {
	next, prev *Element__int

	list *List__int

	Value int
}

func (e *Element__int) Next() *Element__int {
	if p := e.next; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

func (e *Element__int) Prev() *Element__int {
	if p := e.prev; e.list != nil && p != &e.list.root {
		return p
	}
	return nil
}

type List__int struct {
	root Element__int
	len  int
}

func (l *List__int) Init() *List__int {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.len = 0
	return l
}

func New__int() *List__int {
	return (&List__int{}).Init()
}

func (l *List__int) Len() int { return l.len }

func (l *List__int) Front() *Element__int {
	if l.len == 0 {
		return nil
	}
	return l.root.next
}

func (l *List__int) Back() *Element__int {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

func (l *List__int) lazyInit() {
	if l.root.next == nil {
		l.Init()
	}
}

func (l *List__int) insert(e, at *Element__int) *Element__int {
	n := at.next
	at.next = e
	e.prev = at
	e.next = n
	n.prev = e
	e.list = l
	l.len++
	return e
}

func (l *List__int) insertValue(v int, at *Element__int) *Element__int {
	return l.insert(&Element__int{Value: v}, at)
}

func (l *List__int) remove(e *Element__int) *Element__int {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--
	return e
}

func (l *List__int) Remove(e *Element__int) interface {
} {
	if e.list == l {

		l.remove(e)
	}
	return e.Value
}

func (l *List__int) PushFront(v int) *Element__int {
	l.lazyInit()
	return l.insertValue(v, &l.root)
}

func (l *List__int) PushBack(v int) *Element__int {
	l.lazyInit()
	return l.insertValue(v, l.root.prev)
}

func (l *List__int) InsertBefore(v int, mark *Element__int) *Element__int {
	if mark.list != l {
		return nil
	}

	return l.insertValue(v, mark.prev)
}

func (l *List__int) InsertAfter(v int, mark *Element__int) *Element__int {
	if mark.list != l {
		return nil
	}

	return l.insertValue(v, mark)
}

func (l *List__int) MoveToFront(e *Element__int) {
	if e.list != l || l.root.next == e {
		return
	}

	l.insert(l.remove(e), &l.root)
}

func (l *List__int) MoveToBack(e *Element__int) {
	if e.list != l || l.root.prev == e {
		return
	}

	l.insert(l.remove(e), l.root.prev)
}

func (l *List__int) MoveBefore(e, mark *Element__int) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.insert(l.remove(e), mark.prev)
}

func (l *List__int) MoveAfter(e, mark *Element__int) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.insert(l.remove(e), mark)
}

func (l *List__int) PushBackList(other *List__int) {
	l.lazyInit()
	for i, e := other.Len(), other.Front(); i > 0; i, e = i-1, e.Next() {
		l.insertValue(e.Value, l.root.prev)
	}
}

func (l *List__int) PushFrontList(other *List__int) {
	l.lazyInit()
	for i, e := other.Len(), other.Back(); i > 0; i, e = i-1, e.Prev() {
		l.insertValue(e.Value, &l.root)
	}
}
