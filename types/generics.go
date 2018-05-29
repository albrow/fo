package types

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/printer"
	"github.com/albrow/fo/token"
)

var enableCache = true

type typeCache map[GenericType]map[string]ConcreteType

func (tc typeCache) add(conType ConcreteType) {
	if !enableCache {
		return
	}
	genType := conType.GenericType()
	uk := usageKey(conType.TypeMap())
	entry, found := tc[genType]
	if !found {
		entry = map[string]ConcreteType{}
		tc[genType] = entry
	}
	entry[uk] = conType
}

func (tc typeCache) get(genType GenericType, typeMap map[string]Type) ConcreteType {
	// fmt.Printf("checking cache for %p %+v %+v\n", genType, genType, typeMap)
	if !enableCache {
		return nil
	}
	entry, found := tc[genType]
	if !found {
		return nil
	}
	uk := usageKey(typeMap)
	// fmt.Printf("found cached type for %s: %s %T\n", uk, entry[uk], entry[uk])
	return entry[uk]
}

var cache = typeCache{}

type typeArg struct {
	name string
	typ  Type
}

type GenericDecl struct {
	Name   string
	Type   GenericType
	Usages map[string]ConcreteType
}

func addGenericDecl(obj Object, typ GenericType) {
	pkg := obj.Pkg()
	if pkg.generics == nil {
		pkg.generics = map[string]*GenericDecl{}
	}
	dk := declKey(typ)
	pkg.generics[dk] = &GenericDecl{
		Name: obj.Name(),
		Type: typ,
	}
}

func addGenericUsage(genObj Object, typ ConcreteType) {
	pkg := genObj.Pkg()
	if pkg.generics == nil {
		pkg.generics = map[string]*GenericDecl{}
	}
	dk := declKey(genObj.Type().(GenericType))
	genDecl, found := pkg.generics[dk]
	if !found {
		// TODO(albrow): can we avoid panicking here?
		panic(fmt.Errorf("declaration not found for generic object %s (%s)", dk, genObj.Id()))
	}
	if genDecl.Usages == nil {
		genDecl.Usages = map[string]ConcreteType{}
	}
	genDecl.Usages[usageKey(typ.TypeMap())] = typ
}

func declKey(typ GenericType) string {
	key := ""
	if sig, ok := typ.(*GenericSignature); ok {
		if sig.recv != nil {
			key += sig.recv.object.name + "."
		}
	}
	key += typ.Object().Name()
	return key
}

// usageKey returns a unique key for a particular usage which is based on its
// type arguments. Another usage with the same type arguments will have the
// same key.
func usageKey(typeMap map[string]Type) string {
	typeArgs := []typeArg{}
	for name, typ := range typeMap {
		typeArgs = append(typeArgs, typeArg{
			name: name,
			typ:  typ,
		})
	}
	sort.Slice(typeArgs, func(i int, j int) bool {
		return typeArgs[i].name < typeArgs[j].name
	})
	stringParams := []string{}
	for _, arg := range typeArgs {
		stringParams = append(stringParams, arg.typ.String())
	}
	return strings.Join(stringParams, ";")
}

func checkIsPartial(typeMap map[string]Type) bool {
	for _, typ := range typeMap {
		if _, ok := typ.(*TypeParam); ok {
			return true
		}
	}
	return false
}

// concreteType returns a new type with the concrete type arguments of e
// applied.
func (check *Checker) concreteType(expr *ast.TypeArgExpr, genType GenericType) Type {
	buf := &bytes.Buffer{}
	if err := printer.Fprint(buf, token.NewFileSet(), expr); err != nil {
		panic(err)
	}
	// fmt.Printf("concreteType(%s, %+v)\n", buf.String(), genType)
	typeMap := check.createTypeMap(expr, genType.TypeParams())
	if typeMap == nil {
		return Typ[Invalid]
	}
	if cachedType := cache.get(genType, typeMap); cachedType != nil {
		return cachedType
	}
	isPartial := checkIsPartial(typeMap)
	switch genType := genType.(type) {
	case *GenericNamed:
		if isPartial {
			return &PartialGenericNamed{
				Named:   genType.Named,
				genType: genType,
				typeMap: typeMap,
			}
		}
		newNamed := replaceTypesInNamed(genType.Named, typeMap)
		newType := &ConcreteNamed{
			Named:   newNamed,
			genType: genType,
			typeMap: typeMap,
		}
		newType.methods = replaceTypesInMethods(genType.methods, typeMap)
		cache.add(newType)
		addGenericUsage(genType.Object(), newType)
		return newType

	case *GenericSignature:
		if isPartial {
			return &PartialGenericSignature{
				Signature: genType.Signature,
				genType:   genType,
				typeMap:   typeMap,
			}
		}
		newSig := replaceTypesInSignature(genType.Signature, typeMap)
		newType := &ConcreteSignature{
			Signature: newSig,
			genType:   genType,
			typeMap:   typeMap,
		}
		cache.add(newType)
		addGenericUsage(genType.Object(), newType)
		return newType

	case *PartialGenericSignature:
		if cachedType := cache.get(genType.genType, typeMap); cachedType != nil {
			return cachedType
		}
		if isPartial {
			return &PartialGenericSignature{
				Signature: genType.Signature,
				genType:   genType.genType,
				typeMap:   typeMap,
			}
		}
		newTypeMap := mergeTypeMap(genType.typeMap, typeMap)
		newSig := replaceTypesInSignature(genType.Signature, newTypeMap)
		newType := &ConcreteSignature{
			Signature: newSig,
			genType:   genType.genType,
			typeMap:   newTypeMap,
		}
		cache.add(newType)
		addGenericUsage(genType.Object(), newType)
		return newType
	}

	panic(fmt.Errorf("unexpected generic for %s: %T", expr.X, genType))
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

func remapTypes(partial, incoming map[string]Type) map[string]Type {
	result := map[string]Type{}
	for key, typ := range partial {
		for inc, newTyp := range incoming {
			if typ.String() == inc {
				result[key] = newTyp
				break
			}
		}
		if _, found := result[key]; !found {
			result[key] = typ
		}
	}
	return result
}

// TODO(albrow): test case with wrong number of type arguments.
func (check *Checker) createTypeMap(typeArgExpr *ast.TypeArgExpr, typeParams []*TypeParam) map[string]Type {
	typeArgs := typeArgExpr.Types
	if len(typeArgs) != len(typeParams) {
		check.errorf(typeArgExpr.Pos(), "wrong number of type arguments (expected %d but got %d)", len(typeParams), len(typeArgs))
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
	recvType, _ = deref(recvType)
	if recvType, ok := recvType.(ConcreteType); ok {
		if len(recvType.GenericType().TypeParams()) == 0 {
			return typeMap
		}
		newTypeMap := map[string]Type{}
		// First copy all the values of the original type map.
		for name, typ := range typeMap {
			newTypeMap[name] = typ
		}
		// Then remap all the receiver type arguments to their appropriate type.
		for name, typ := range recvType.TypeMap() {
			if tp, ok := typ.(*TypeParam); ok {
				newTypeMap[tp.String()] = typeMap[name]
			}
		}
		// fmt.Printf("original typeMap: %v\n", typeMap)
		// fmt.Printf("new type map: %v\n", newTypeMap)
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
func replaceTypes(root Type, typeMap map[string]Type) Type {
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
		newPointer.base = replaceTypes(t.base, typeMap)
		return &newPointer
	case *Slice:
		newSlice := *t
		newSlice.elem = replaceTypes(t.elem, typeMap)
		return &newSlice
	case *Map:
		newMap := *t
		newMap.key = replaceTypes(t.key, typeMap)
		newMap.elem = replaceTypes(t.elem, typeMap)
		return &newMap
	case *Array:
		newArray := *t
		newArray.elem = replaceTypes(t.elem, typeMap)
		return &newArray
	case *Chan:
		newChan := *t
		newChan.elem = replaceTypes(t.elem, typeMap)
		return &newChan
	case *Struct:
		return replaceTypesInStruct(t, typeMap)
	case *Signature:
		return replaceTypesInSignature(t, typeMap)
	case *Named:
		return replaceTypesInNamed(t, typeMap)
	case *ConcreteNamed:
		panic(errors.New("case *ConcreteNamed not implemented"))
	case *ConcreteSignature:
		panic(errors.New("case *ConcreteSignature not implemened"))
	case *PartialGenericNamed:
		return replaceTypesInPartialGenericNamed(t, typeMap)
	case *PartialGenericSignature:
		panic(errors.New("case *PartialGenericSignature not implemened"))
	}
	return root
}

func replaceTypesInStruct(root *Struct, typeMap map[string]Type) *Struct {
	var fields []*Var
	for _, field := range root.fields {
		newField := *field
		newField.typ = replaceTypes(field.Type(), typeMap)
		fields = append(fields, &newField)
	}
	return NewStruct(fields, root.tags)
}

func replaceTypesInSignature(root *Signature, typeMap map[string]Type) *Signature {
	var newRecv *Var
	if root.recv != nil {
		recvType, _ := deref(root.recv.typ)
		if _, ok := recvType.(ConcreteType); ok {
			newRecv = root.recv
		} else {
			newRecv = new(Var)
			(*newRecv) = *root.recv
			newRecvType := replaceTypes(root.recv.typ, typeMap)
			newRecv.typ = newRecvType
		}
	}

	var newParams *Tuple
	if root.params != nil && len(root.params.vars) > 0 {
		newParams = &Tuple{}
		for _, param := range root.params.vars {
			newParam := *param
			newParam.typ = replaceTypes(param.typ, typeMap)
			newParams.vars = append(newParams.vars, &newParam)
		}
	}

	var newResults *Tuple
	if root.results != nil && len(root.results.vars) > 0 {
		newResults = &Tuple{}
		for _, result := range root.results.vars {
			newResult := *result
			newResult.typ = replaceTypes(result.typ, typeMap)
			newResults.vars = append(newResults.vars, &newResult)
		}
	}

	newSig := NewSignature(newRecv, newParams, newResults, root.variadic)
	return newSig
}

func replaceTypesInGenericSignature(root *GenericSignature, typeMap map[string]Type) Type {
	if cachedType := cache.get(root, typeMap); cachedType != nil {
		return cachedType
	}
	if checkIsPartial(typeMap) {
		return &PartialGenericSignature{
			Signature: root.Signature,
			genType:   root,
			typeMap:   typeMap,
		}
	}
	newSig := replaceTypesInSignature(root.Signature, typeMap)
	newType := &ConcreteSignature{
		Signature: newSig,
		genType:   root,
		typeMap:   typeMap,
	}
	cache.add(newType)
	addGenericUsage(root.obj, newType)
	return newType
}

func replaceTypesInNamed(root *Named, typeMap map[string]Type) *Named {
	newUnderlying := replaceTypes(root.underlying, typeMap)
	newNamed := *root
	newNamed.underlying = newUnderlying
	return &newNamed
}

func replaceTypesInPartialGenericNamed(root *PartialGenericNamed, typeMap map[string]Type) Type {
	if cachedType := cache.get(root.genType, typeMap); cachedType != nil {
		return cachedType
	}
	newTypeMap := remapTypes(root.typeMap, typeMap)
	if checkIsPartial(newTypeMap) {
		return &PartialGenericNamed{
			Named:   root.Named,
			genType: root.genType,
			typeMap: newTypeMap,
		}
	}
	newNamed := replaceTypesInNamed(root.Named, newTypeMap)
	newType := &ConcreteNamed{
		Named:   newNamed,
		genType: root.genType,
		typeMap: newTypeMap,
	}
	newType.methods = replaceTypesInMethods(root.methods, newTypeMap)
	cache.add(newType)
	addGenericUsage(root.obj, newType)
	return newType
}

func addPartialSigTypeParams(sig *GenericSignature, typeMap map[string]Type) map[string]Type {
	result := map[string]Type{}
	for key, typ := range typeMap {
		result[key] = typ
	}
	for _, tp := range sig.TypeParams() {
		result[tp.String()] = tp
	}
	return result
}

func replaceTypesInMethods(methods []*Func, typeMap map[string]Type) []*Func {
	newMethods := make([]*Func, len(methods))
	for i, m := range methods {
		switch meth := m.typ.(type) {
		case *Signature:
			newTypeMap := createMethodTypeMap(meth.recv.typ, typeMap)
			newSig := replaceTypesInSignature(meth, newTypeMap)
			newMethods[i] = replaceFuncType(m, newSig)
		case *GenericSignature:
			// Here we need to add both the implicit type args of the receiver type
			// and the type args of the signature itself.
			newTypeMap := addPartialSigTypeParams(
				meth,
				createMethodTypeMap(meth.recv.typ, typeMap),
			)
			newSig := replaceTypesInGenericSignature(meth, newTypeMap)
			newMethods[i] = replaceFuncType(m, newSig)
		default:
			panic(fmt.Errorf("unexpected meth.typ: %T", m.typ))
		}
	}
	return newMethods
}

func replaceFuncType(f *Func, newType Type) *Func {
	return &Func{
		object: object{
			parent:    f.parent,
			pos:       f.pos,
			pkg:       f.pkg,
			name:      f.name,
			typ:       newType,
			order_:    f.order_,
			scopePos_: f.scopePos_,
		},
	}
}

// typeArgsRequired reports an error if the typ is a generic type. It should be
// called in any context where a generic type is not valid (and a TypeArgExpr
// should be used instead).
//
// TODO(albrow): replace this with type argument inference.
func (check *Checker) typeArgsRequired(pos token.Pos, typ Type) {
	switch t := typ.(type) {
	case PartialGenericType:
		if len(t.TypeParams()) != len(t.TypeMap()) {
			check.errorf(
				pos,
				"wrong number of type arguments for type %s (expected %d but got %d, including implicit type arguments)",
				typ.String(),
				len(t.TypeParams()),
				len(t.TypeMap()),
			)
		}
	case GenericType:
		check.errorf(pos, "missing type arguments for type %s", typ.String())
	}
}
