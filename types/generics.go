package types

import (
	"fmt"

	"github.com/albrow/fo/ast"
)

// concreteType returns a new type with the concrete type parameters of e
// applied.
func (check *Checker) concreteType(e *ast.Ident, genericObj Object) Type {
	switch genericObj := genericObj.(type) {
	case *TypeName:
		if named, ok := genericObj.Type().(*Named); ok {
			switch underlying := named.Underlying().(type) {
			case *Struct:
				// TODO(albrow): cache concrete types in some sort of special scope so
				// we can avoid re-generating the concrete types on each usage.
				typeMap := check.createTypeMap(e.TypeParams, underlying.typeParams)
				newType := underlying.NewConcrete(typeMap)
				newTypeName := *genericObj
				newNamed := *named
				newNamed.underlying = newType
				newNamed.obj = &newTypeName
				newTypeName.typ = &newNamed
				// newTypeName.name = e.NameWithParams()
				return &newNamed
			}
		}
	}

	check.errorf(check.pos, "unexpected generic type for ident %s: %T", e.Name, genericObj)
	return nil
}

// TODO(albrow): catch case where the wrong number of type parameters has been
// given and test it.
func (check *Checker) createTypeMap(params *ast.ConcreteTypeParamList, genericParams []*TypeParam) map[string]Type {
	typeMap := map[string]Type{}
	for i, typ := range params.List {
		var x operand
		check.rawExpr(&x, typ, nil)
		if x.typ != nil {
			typeMap[genericParams[i].String()] = x.typ
		}
	}
	return typeMap
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
	case *Named:
		return replaceTypesInNamed(t, typeMapping)
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
	return NewStruct(fields, root.tags, root.typeParams)
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

	// TODO(albrow): Implement inherited type parameters here.

	return NewSignature(newRecv, newParams, newResults, root.variadic, root.typeParams)
}

func replaceTypesInNamed(root *Named, typeMapping map[string]Type) *Named {
	switch u := root.underlying.(type) {
	case *ConcreteStruct:
		newTypeMap := map[string]Type{}
		for key, given := range u.typeMap {
			if param, ok := given.(*TypeParam); ok {
				if inherited, found := typeMapping[param.String()]; found {
					newTypeMap[key] = inherited
					continue
				}
			}
			newTypeMap[key] = given
		}
		newU := *u
		newU.typeMap = newTypeMap
		newRoot := *root
		newRoot.underlying = &newU
		return &newRoot
	}
	return root
}
