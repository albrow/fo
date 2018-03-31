package types

import (
	"fmt"

	"github.com/albrow/fo/ast"
)

// TODO(albrow): test the error case here
// TODO(alborw): wrap this into the ident function or otherwise make it more
//               efficient?
func (check *Checker) typeParams(e *ast.Ident, obj Object) {
	var givenParams []ast.Expr
	if e.TypeParams != nil {
		givenParams = e.TypeParams.List
	}
	switch obj := obj.(type) {
	case *TypeName:
		if typ, ok := obj.Type().(*Named); ok {
			switch underlying := typ.Underlying().(type) {
			case *Struct:
				if len(underlying.typeParams) != len(givenParams) {
					check.error(
						e.Pos(),
						fmt.Sprintf(
							"wrong number of type params for %s (expected %d but got %d)",
							e.Name,
							len(underlying.typeParams),
							len(givenParams),
						),
					)
				}
				return
			}
		}
	case *Func:
		// TODO(albrow): Implement this.
		return
	}
	if len(givenParams) != 0 {
		check.error(
			e.Pos(),
			fmt.Sprintf(
				"wrong number of type params for %s (%T) (expected 0 but got %d)",
				e.Name,
				obj,
				len(givenParams),
			),
		)
	}
}

// TODO(albrow): return a named type (e.g. Box::(string)) to make type errors
// easier to read.
func (check *Checker) generateStructType(e *ast.Ident, s *Struct) *Struct {
	typeMapping := map[string]Type{}
	for i, typ := range e.TypeParams.List {
		if ident, ok := typ.(*ast.Ident); ok {
			var x operand
			check.rawExpr(&x, ident, nil)
			if x.typ != nil {
				typeMapping[s.typeParams[i].String()] = x.typ
			}
		}
	}
	return replaceTypesInStruct(s, typeMapping)
}

// replaceTypes recursively replaces any type parameters starting at root with
// the corresponding concrete type by looking up in typeMapping. typeMapping is
// a map of type parameter identifier to concrete type. replaceTypes works with
// compound types such as maps, slices, and arrays whenever the type parameter
// is part of the type. For example, root can be a []T and replaceTypes will
// correctly replace T with the corresponding concrete type (assuming it is
// included in typeMapping).
func replaceTypes(root Type, typeMapping map[string]Type) Type {
	switch t := root.(type) {
	case *TypeParam:
		if newType, found := typeMapping[t.String()]; found {
			return newType
		}
		// TODO(albrow): handle this error?
		panic(fmt.Errorf("undefined type parameter: %s", t))
	case *Pointer:
		newPointer := *t
		newPointer.base = replaceTypes(t.base, typeMapping)
		return &newPointer
	case *Slice:
		newSlice := *t
		newSlice.elem = replaceTypes(t.elem, typeMapping)
		return &newSlice
	case *Map:
		_, keyParameterized := t.key.(*TypeParam)
		_, valParameterized := t.elem.(*TypeParam)
		if keyParameterized || valParameterized {
			newMap := *t
			newMap.key = replaceTypes(t.key, typeMapping)
			newMap.elem = replaceTypes(t.elem, typeMapping)
			return &newMap
		}
	case *Array:
		newArray := *t
		newArray.elem = replaceTypes(t.elem, typeMapping)
		return &newArray
	case *Chan:
		newChan := *t
		newChan.elem = replaceTypes(t.elem, typeMapping)
		return &newChan
	case *Struct:
		return replaceTypesInStruct(t, typeMapping)
	case *Signature:
		return replaceTypesInSignature(t, typeMapping)
	}
	return root
}

func replaceTypesInStruct(root *Struct, typeMapping map[string]Type) *Struct {
	var fields []*Var
	for _, field := range root.fields {
		newField := *field
		newField.typ = replaceTypes(field.Type(), typeMapping)
		fields = append(fields, &newField)
	}
	return NewStruct(fields, root.tags, nil)
}

func replaceTypesInSignature(root *Signature, typeMapping map[string]Type) *Signature {
	var newRecv *Var
	if root.recv != nil {
		newRecv := *root.recv
		newRecv.typ = replaceTypes(root.recv.typ, typeMapping)
	}

	var newParams *Tuple
	if root.params != nil && len(root.params.vars) > 0 {
		newParams = &Tuple{}
		for _, param := range root.params.vars {
			newParam := *param
			newParam.typ = replaceTypes(param.typ, typeMapping)
			newParams.vars = append(newParams.vars, &newParam)
		}
	}

	var newResults *Tuple
	if root.params != nil && len(root.params.vars) > 0 {
		newResults = &Tuple{}
		for _, result := range root.results.vars {
			newResult := *result
			newResult.typ = replaceTypes(result.typ, typeMapping)
			newResults.vars = append(newResults.vars, &newResult)
		}
	}

	return NewSignature(newRecv, newParams, newResults, root.variadic)
}
