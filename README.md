# Fo

Fo is an experimental language which adds functional programming features to Go.
The name is short for "Functional Go".

## Table of Contents

<!-- TOC depthFrom:2 depthTo:3 updateOnSave:false -->

- [Background](#background)
- [Playground](#playground)
- [Installation](#installation)
- [Command Line Usage](#command-line-usage)
- [Examples](#examples)
- [Language Features](#language-features)
  - [Generic Named Types](#generic-named-types)
  - [Generic Functions](#generic-functions)
  - [Generic Methods](#generic-methods)

<!-- /TOC -->

## Background

Go already supports many features that functional programmers might want:
closures, first-class functions, errors as values, etc. The main feature
(and in fact only feature for now) that Fo adds is type polymorphism via
generics. Generics encourage functional programming techniques by making it
possible to write flexible higher-order functions and type-agnostic data
structures.

At this time, Fo should be thought of primarily as an experiment or proof of
concept. It shows what Go looks like and feels like with some new language
features and allows us to explore how those features interact and what you can
build with them. The syntax and implementation are not finalized. The language
is far from appropriate for use in real applications, though that will hopefully
change in the future.

## Playground

If you want to give Fo a try without installing anything, you can visit
[The Fo Playground](https://play.folang.org/).

## Installation

The Fo compiler is written in Go, so you can install it like any other Go
program:

```
go get -u github.com/albrow/fo
```

## Command Line Usage

For now, the CLI for Fo is extremely simple and only works on one file at a
time. There is only one command, `run`, and it works like this:

```
fo run <filename>
```

`<filename>` should be a source file ending in .fo which contains a `main`
function.

## Examples

You can see some example programs showing off various features of the language
in the
[examples directory](https://github.com/albrow/fo/tree/master/examples) of this
repository.

## Language Features

In terms of syntax and semantics, Fo is a superset of Go. That means that any
valid Go program is also a valid Fo program (provided you change the file
extension from ".go" to ".fo").

### Generic Named Types

#### Declaration

Fo extends the Go grammar for type declarations to allow the declaration of
generic types. Generic types expect one or more "type parameters", which are
placeholders for arbitrary types to be specified later.

The extended grammar looks like this (some definitions omitted/simplified):

```
TypeDecl   = "type" identifier [ TypeParams ] Type .
TypeParams = "[" identifier { "," identifier } "]" .
```

In other words, type parameters should follow the type name and are surrounded
by square brackets. Multiple type parameters are separated by a comma (e.g.,
`type A[T, U, V]`).

Here's the syntax for a generic `Box` which can hold a value of any arbitrary
type:

```go
type Box[T] struct {
	v T
}
```

Type parameters are scoped to the type definition and can be used in place of
any type. The following are all examples of valid type declarations:

```go
type A[T] []T

type B[T, U] map[T]U

type C[T, U] func(T) U

type D[T] struct{
	a T
	b A[T]
}
```

In general, any named type can be made generic. The only exception is that Fo
does not currently allow generic interface types.

#### Usage

When using a generic type, you must supply a number of "type arguments", which
are specific types that will take the place of the type parameters in the type
definition. The combination of a generic type name and its corresponding type
arguments is called a "type argument expression". The grammar looks like this
(some definitions omitted/simplified):

```
TypeArgExpr = Type TypeArgs .
TypeArgs    = "[" Type { "," Type } "]" .
```

Like type parameters, type arguments follow the type name and are surrounded by
square brackets, and multiple type arguments are separated by a comma (e.g.
`A[string, int, bool]`). In general, type argument expressions can be used
anywhere you would normally use a type.

Here's how we would use the `Box` type we declared above to initialize a `Box`
which holds a `string` value:

```go
x := Box[string]{ v: "foo" }
```

Fo does not currently support inference of type arguments, so they must always
be specified.

### Generic Functions

#### Declaration

The syntax for declaring a generic function is similar to named types. The
grammar looks like this:

```
FunctionDecl = "func" FunctionName [ TypeParams ] Signature [ FunctionBody ] .
TypeParams   = "[" identifier { "," identifier } "]"
```

As you might expect, type parameters follow the function name. Both the function
signature and body can make use of the given type parameters, and the type
parameters will be replaced with type arguments when the function is used.

Here's how you would declare a `MapSlice` function which applies a given
function `f` to each element of `list` and returns the results.

```go
func MapSlice[T](f func(T) T, list []T) []T {
	result := make([]T, len(list))
	for i, val := range list {
		result[i] = f(val)
	}
	return result
}
```

#### Usage

Just like generic named types, to use a generic function, you must supply the
type arguments. If you actually want to call the function, just add the function
arguments after the type arguments. The grammar looks like this (some
definitions omitted/simplified):

```
CallExpr = FunctionName ( TypeArgs ) Arguments .
TypeArgs = "[" Type { "," Type } "]" .
```

Here's how you would call the `MapSlice` function we defined above:

```go
func incr(n int) int {
	return n+1
}

// ...

MapSlice[int](incr, []int{1, 2, 3})
```

### Generic Methods

#### Declaration

Fo supports a special syntax for methods with a generic receiver type. You can
optionally include the type parameters of the receiver type and those type
parameters can be used in the function signature and body. Whenever a generic
method of this form is called, the type arguments of the receiver type are
passed through to the method definition.

The grammar for methods with generic receiver types looks like this (some
definitions omitted/simplified):

```
MethodDecl = "func" Receiver MethodName [ TypeParams ] Signature [ FunctionBody ] .
Receiver   = "(" [ ReceiverName ] Type [ TypeParams ] ")" .
TypeParams = "[" identifier { "," identifier } "]" .
```

Here's how we would define a method on the `Box` type defined above which makes
use of receiver type parameters:

```go
type Box[T] struct {
  v T
}

func (b Box[T]) Val() T {
  return b.v
}
```

You can also omit the type parameters of the receiver type if they are not
needed. For example, here's how we would define a `String` method which does not
depend on the type parameters of the `Box`:

```go
func (b Box) String() string {
  return fmt.Sprint(b.v)
}
```

A method with a generic receiver can define additional type parameters, just
like a function. Here's an example of a method on `Box` which requires
additional type parameters.

```go
func (b Box[T]) Map[U] (f func(T) U) Box[U] {
  return Box[U]{
    v: f(b.v),
  }
}
```

#### Usage

When calling methods with a generic receiver type, you do not need to specify
the type arguments of the receiver. Here's an example of calling the `Val`
method we defined above:

```go
x := Box[string]{ v: "foo" }
x.Val()
```

However, if the method declaration includes additional type parameters, you
still need to specify them. Here's how we would call the `Map` function
defined above to convert a `Box[int]` to a `Box[string]`.

```go
y := Box[int] { v: 42 }
z := y.Map[string](strconv.Itoa)
```
