package types

import (
	"fmt"
	"strings"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/token"
)

// TODO(albrow): document exported types here.

type GenericDecl struct {
	name       string
	typ        Type
	typeParams []*TypeParam
	usages     map[string]*GenericUsage
}

func (decl *GenericDecl) Name() string {
	return decl.name
}

func (decl *GenericDecl) TypeParams() []*TypeParam {
	return decl.typeParams
}

func (decl *GenericDecl) Usages() map[string]*GenericUsage {
	return decl.usages
}

type GenericUsage struct {
	typ      Type
	typeArgs []ast.Expr
	typeMap  map[string]Type
}

func (usg *GenericUsage) TypeArgs() []ast.Expr {
	return usg.typeArgs
}

func (usg *GenericUsage) TypeMap() map[string]Type {
	return usg.typeMap
}

func addGenericDecl(key string, obj Object, typeParams []*TypeParam) {
	pkg := obj.Pkg()
	name := obj.Name()
	if pkg.generics == nil {
		pkg.generics = map[string]*GenericDecl{}
	}
	pkg.generics[key] = &GenericDecl{
		name:       name,
		typ:        obj.Type(),
		typeParams: typeParams,
	}
}

func addGenericUsage(key string, genObj Object, typ Type, typeParams []ast.Expr, typeMap map[string]Type) {
	for _, typ := range typeMap {
		if _, ok := typ.(*TypeParam); ok {
			// If the type map includes a type parameter, it is not yet complete and
			// includes inherited type parameters. In this case, it is not a true
			// concrete usage, so we don't add it to the usage list.
			return
		}
	}
	pkg := genObj.Pkg()
	if pkg.generics == nil {
		pkg.generics = map[string]*GenericDecl{}
	}
	genDecl, found := pkg.generics[key]
	if !found {
		// TODO(albrow): can we avoid panicking here?
		panic(fmt.Errorf("declaration not found for generic object %s (%s)", key, genObj.Id()))
	}
	if genDecl.usages == nil {
		genDecl.usages = map[string]*GenericUsage{}
	}
	genDecl.usages[usageKey(typeMap, genDecl.typeParams)] = &GenericUsage{
		typ:      typ,
		typeArgs: typeParams,
		typeMap:  typeMap,
	}
}

// usageKey returns a unique key for a particular usage which is based on its
// type arguments. Another usage with the same type arguments will have the
// same key.
func usageKey(typeMap map[string]Type, typeParams []*TypeParam) string {
	stringParams := []string{}
	for _, param := range typeParams {
		stringParams = append(stringParams, typeMap[param.String()].String())
	}
	return strings.Join(stringParams, ",")
}

// concreteType returns a new type with the concrete type arguments of e
// applied.
//
// TODO(albrow): Cache concrete types in some sort of special scope so
// we can avoid re-generating the concrete types on each usage.
func (check *Checker) concreteType(expr *ast.TypeArgExpr, genType Type) Type {
	switch genType := genType.(type) {
	case *Named:
		typeMap := check.createTypeMap(expr.Types, genType.typeParams)
		if typeMap == nil {
			return genType
		}
		newNamed := replaceTypesInNamed(genType, expr.Types, typeMap)
		newNamed.typeParams = nil
		typeParams := make([]*TypeParam, len(genType.typeParams))
		copy(typeParams, genType.typeParams)
		newType := NewConcreteNamed(newNamed, typeParams, typeMap)
		newObj := *genType.obj
		newType.obj = &newObj
		newObj.typ = newType
		newType.methods = replaceTypesInMethods(genType.methods, expr.Types, typeMap)
		addGenericUsage(genType.obj.name, genType.obj, newType, expr.Types, typeMap)
		return newType
	case *Signature:
		typeMap := check.createTypeMap(expr.Types, genType.typeParams)
		if typeMap == nil {
			return genType
		}
		newSig := replaceTypesInSignature(genType, expr.Types, typeMap)
		newSig.typeParams = nil
		typeParams := make([]*TypeParam, len(genType.typeParams))
		copy(typeParams, genType.typeParams)
		newType := NewConcreteSignature(newSig, typeParams, typeMap)
		if genType.obj != nil {
			newObj := *genType.obj
			newObj.typ = newSig
			newSig.obj = &newObj
			addGenericUsage(newObj.name, &newObj, newType, expr.Types, typeMap)
		}
		return newType
	case *MethodPartial:
		typeMap := check.createTypeMap(expr.Types, genType.typeParams)
		if typeMap == nil {
			return genType
		}
		typeMap = mergeTypeMap(genType.recvTypeMap, typeMap)
		newSig := replaceTypesInSignature(genType.Signature, expr.Types, typeMap)
		newSig.typeParams = nil
		typeParams := make([]*TypeParam, len(genType.typeParams))
		copy(typeParams, genType.typeParams)
		newType := NewConcreteSignature(newSig, typeParams, typeMap)
		if genType.obj != nil {
			newObj := *genType.obj
			newObj.typ = newSig
			newSig.obj = &newObj
			key := genType.recvName + "." + newObj.name
			addGenericUsage(key, &newObj, newType, expr.Types, typeMap)
		}
		return newType
	}

	check.errorf(check.pos, "unexpected generic for %s: %T", expr.X, genType)
	return nil
}

// b overwrites a
func mergeTypeMap(a, b map[string]Type) map[string]Type {
	result := map[string]Type{}
	for name, typ := range a {
		result[name] = typ
	}
	for name, typ := range b {
		result[name] = typ
	}
	return result
}

// TODO(albrow): test case with wrong number of type arguments.
func (check *Checker) createTypeMap(typeArgs []ast.Expr, typeParams []*TypeParam) map[string]Type {
	if len(typeArgs) != len(typeParams) {
		check.errorf(check.pos, "wrong number of type arguments (expected %d but got %d)", len(typeParams), len(typeArgs))
		return nil
	}
	typeMap := map[string]Type{}
	for i, typ := range typeArgs {
		var x operand
		check.rawExpr(&x, typ, nil)
		if x.typ != nil {
			typeMap[typeParams[i].String()] = x.typ
		}
	}
	return typeMap
}

func createMethodTypeMap(recvType Type, typeMap map[string]Type) map[string]Type {
	if recvType, ok := recvType.(*ConcreteNamed); ok {
		if len(recvType.typeParams) == 0 {
			return typeMap
		}
		newTypeMap := map[string]Type{}
		// First copy all the values of the original type map.
		for name, typ := range typeMap {
			newTypeMap[name] = typ
		}
		// Then remap all the receiver type arguments to their appropriate type.
		for name, typ := range recvType.typeMap {
			if tp, ok := typ.(*TypeParam); ok {
				newTypeMap[tp.String()] = typeMap[name]
			}
		}
		return newTypeMap
	}

	return typeMap
}

// replaceTypes recursively replaces any type parameters starting at root with
// the corresponding concrete type by looking up in typeMap. typeMap is
// a map of type parameter identifier to concrete type. replaceTypes works with
// compound types such as maps, slices, and arrays whenever the type parameter
// is part of the type. For example, root can be a []T and replaceTypes will
// correctly replace T with the corresponding concrete type (assuming it is
// included in typeMap).
func replaceTypes(root Type, typeParams []ast.Expr, typeMap map[string]Type) Type {
	switch t := root.(type) {
	case *TypeParam:
		if newType, found := typeMap[t.String()]; found {
			// This part is important; if the concrete type is also a type parameter,
			// don't do the replacement. We assume that we're dealing with an
			// inherited type parameter and that the concrete form of the parent will
			// fill in this missing type parameter in the future. If it is not filled
			// in correctly in the future, we know how to generate an error.
			if _, newIsTypeParam := newType.(*TypeParam); newIsTypeParam {
				return t
			}
			return newType
		}
		return root
	case *Pointer:
		newPointer := *t
		newPointer.base = replaceTypes(t.base, typeParams, typeMap)
		return &newPointer
	case *Slice:
		newSlice := *t
		newSlice.elem = replaceTypes(t.elem, typeParams, typeMap)
		return &newSlice
	case *Map:
		newMap := *t
		newMap.key = replaceTypes(t.key, typeParams, typeMap)
		newMap.elem = replaceTypes(t.elem, typeParams, typeMap)
		return &newMap
	case *Array:
		newArray := *t
		newArray.elem = replaceTypes(t.elem, typeParams, typeMap)
		return &newArray
	case *Chan:
		newChan := *t
		newChan.elem = replaceTypes(t.elem, typeParams, typeMap)
		return &newChan
	case *Struct:
		return replaceTypesInStruct(t, typeParams, typeMap)
	case *Signature:
		return replaceTypesInSignature(t, typeParams, typeMap)
	case *Named:
		return replaceTypesInNamed(t, typeParams, typeMap)
	case *ConcreteNamed:
		return replaceTypesInConcreteNamed(t, typeParams, typeMap)
	}
	return root
}

func replaceTypesInStruct(root *Struct, typeParams []ast.Expr, typeMap map[string]Type) *Struct {
	var fields []*Var
	for _, field := range root.fields {
		newField := *field
		newField.typ = replaceTypes(field.Type(), typeParams, typeMap)
		fields = append(fields, &newField)
	}
	return NewStruct(fields, root.tags)
}

func replaceTypesInSignature(root *Signature, typeParams []ast.Expr, typeMap map[string]Type) *Signature {
	var newRecv *Var
	if root.recv != nil {
		newRecv = new(Var)
		(*newRecv) = *root.recv
		newRecvType := replaceTypes(root.recv.typ, typeParams, typeMap)
		newRecv.typ = newRecvType
	}

	var newParams *Tuple
	if root.params != nil && len(root.params.vars) > 0 {
		newParams = &Tuple{}
		for _, param := range root.params.vars {
			newParam := *param
			newParam.typ = replaceTypes(param.typ, typeParams, typeMap)
			newParams.vars = append(newParams.vars, &newParam)
		}
	}

	var newResults *Tuple
	if root.results != nil && len(root.results.vars) > 0 {
		newResults = &Tuple{}
		for _, result := range root.results.vars {
			newResult := *result
			newResult.typ = replaceTypes(result.typ, typeParams, typeMap)
			newResults.vars = append(newResults.vars, &newResult)
		}
	}

	newSig := NewSignature(newRecv, newParams, newResults, root.variadic, root.typeParams, root.recvTypeParams)

	if root.obj != nil {
		newObj := *root.obj
		newObj.typ = newSig
		newSig.obj = &newObj
	}
	return newSig
}

func replaceTypesInMethods(methods []*Func, typeParams []ast.Expr, typeMap map[string]Type) []*Func {
	newMethods := make([]*Func, len(methods))
	for i, meth := range methods {
		if sig, ok := meth.typ.(*Signature); ok {
			newTypeMap := createMethodTypeMap(sig.recv.typ, typeMap)
			newSig := replaceTypesInSignature(sig, typeParams, newTypeMap)
			newMethods[i] = &Func{
				object: object{
					parent:    meth.parent,
					pos:       meth.pos,
					pkg:       meth.pkg,
					name:      meth.name,
					typ:       newSig,
					order_:    meth.order_,
					scopePos_: meth.scopePos_,
				},
			}
		} else {
			panic(fmt.Errorf("unexpected meth.typ: %T", meth.typ))
		}
	}
	return newMethods
}

func replaceTypesInNamed(root *Named, typeParams []ast.Expr, typeMap map[string]Type) *Named {
	newUnderlying := replaceTypes(root.underlying, typeParams, typeMap)
	newNamed := *root
	newNamed.underlying = newUnderlying
	newObj := *root.obj
	newObj.typ = &newNamed
	newNamed.obj = &newObj
	return &newNamed
}

// TODO(albrow): optimize by doing nothing in the case where the new typeMap
// is equivalent to the old.
func replaceTypesInConcreteNamed(root *ConcreteNamed, typeParams []ast.Expr, typeMap map[string]Type) *ConcreteNamed {
	newTypeMap := map[string]Type{}
	for key, given := range root.typeMap {
		if param, givenIsTypeParam := given.(*TypeParam); givenIsTypeParam {
			if inherited, found := typeMap[param.String()]; found {
				if _, inheritedIsTypeParam := inherited.(*TypeParam); !inheritedIsTypeParam {
					newTypeMap[key] = inherited
					continue
				}
			}
		}
		newTypeMap[key] = given
	}
	newNamed := replaceTypesInNamed(root.Named, typeParams, newTypeMap)
	newType := NewConcreteNamed(newNamed, root.typeParams, newTypeMap)
	newNamed.typeParams = nil
	newObj := *root.obj
	newObj.typ = newType
	newType.obj = &newObj
	newType.methods = replaceTypesInMethods(root.methods, typeParams, newTypeMap)
	addGenericUsage(root.obj.name, root.obj, newType, typeParams, newTypeMap)
	return newType
}

// typeArgsRequired reports an error if the typ is a generic type. It should be
// called in any context where a generic type is not valid (and a TypeArgExpr
// should be used instead).
//
// TODO(albrow): replace this with type argument inference.
func (check *Checker) typeArgsRequired(pos token.Pos, typ Type) {
	switch typ := typ.(type) {
	case *Named:
		if len(typ.typeParams) > 0 {
			check.errorf(pos, "missing type arguments for type %s", typ.String())
		}
	case *Signature:
		if len(typ.typeParams) > 0 {
			check.errorf(pos, "missing type arguments for type %s", typ.String())
		}
	case *MethodPartial:
		if len(typ.typeParams) > 0 {
			check.errorf(pos, "missing type arguments for type %s", typ.String())
		}
	}
}
