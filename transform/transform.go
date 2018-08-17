package transform

import (
	"errors"
	"fmt"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/astclone"
	"github.com/albrow/fo/astutil"
	"github.com/albrow/fo/token"
	"github.com/albrow/fo/types"
)

// TODO(albrow): Implement transform.Package for operating on all files in a
// given package at once.

type Transformer struct {
	Fset *token.FileSet
	Pkg  *types.Package
	Info *types.Info
}

func (trans *Transformer) File(f *ast.File) (*ast.File, error) {
	withTypeConversions := astutil.Apply(f, trans.insertTypeConversions(), nil)
	result := astutil.Apply(withTypeConversions, trans.eraseGenerics(), nil)
	resultFile, ok := result.(*ast.File)
	if !ok {
		panic(fmt.Errorf("astutil.Apply returned a non-file type: %T", result))
	}

	return resultFile, nil
}

// eraseGenerics removes all type parameters and type arguments. If a type
// declaration or function signature contains type parameters, it replaces them
// with the empty interface.
func (trans *Transformer) eraseGenerics() func(c *astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		// TODO(albrow): Erase generics from function declarations
		// TODO(albrow): Figure out how to handle nested generic types
		case *ast.TypeArgExpr:
			// Remove type arguments.
			c.Replace(n.X)
			return false
		case *ast.IndexExpr:
			// We need to disambiguate to see if what the parser thinks is an
			// IndexExpr is actually a TypeArgExpr. If so, we need to remove the type
			// arguments.
			switch x := n.X.(type) {
			case *ast.Ident:
				if _, found := trans.Pkg.Generics()[x.Name]; found {
					c.Replace(n.X)
				}
			case *ast.SelectorExpr:
				selection, found := trans.Info.Selections[x]
				if !found {
					return true
				}
				var key string
				switch selection.Kind() {
				case types.FieldVal:
					key = selection.Obj().Name()
				case types.MethodVal:
					if named, ok := selection.Recv().(*types.ConcreteNamed); ok {
						key = named.Obj().Name() + "." + selection.Obj().Name()
					}
				}
				if key != "" {
					if _, found := trans.Pkg.Generics()[key]; found {
						c.Replace(n.X)
						return false
					}
				}
			}
		case *ast.TypeSpec:
			// We need to disambiguate to see if what the parser thinks is an
			// ArrayType is actually a parameterized type with a TypeParmDecl. If so,
			// we need to remove the type parameters.
			if arrayType, ok := n.Type.(*ast.ArrayType); ok {
				def, found := trans.Info.Defs[n.Name]
				if !found {
					panic(fmt.Errorf("could not find definition for type %s", n.Name))
				}
				if _, isGeneric := def.Type().(*types.GenericNamed); isGeneric {
					newTypeSpec := trans.eraseGenericsFromTypeSpec(n, arrayType.Elt)
					if newTypeSpec != nil {
						c.Replace(newTypeSpec)
					}
					return false
				}
			}
			newTypeSpec := trans.eraseGenericsFromTypeSpec(n, n.Type)
			if newTypeSpec != nil {
				c.Replace(newTypeSpec)
			}
			return false
		}
		return true
	}
}

func isTypeNestedGeneric(typ types.Type) bool {
	switch typ := typ.(type) {
	case *types.TypeParam:
		return true
	case *types.Slice:
		return isTypeNestedGeneric(typ.Elem())
	case *types.Array:
		return isTypeNestedGeneric(typ.Elem())
	case *types.Map:
		return isTypeNestedGeneric(typ.Key()) || isTypeNestedGeneric(typ.Elem())
	case *types.Pointer:
		return isTypeNestedGeneric(typ.Elem())
	case *types.Chan:
		return isTypeNestedGeneric(typ.Elem())
	case *types.Tuple:
		for i := 0; i < typ.Len(); i++ {
			if isVarNestedGeneric(typ.At(i)) {
				return true
			}
		}
	case *types.Signature:
		return isTypeNestedGeneric(typ.Params()) || isTypeNestedGeneric(typ.Results()) || isVarNestedGeneric(typ.Recv())
	}
	return false
}

func isVarNestedGeneric(v *types.Var) bool {
	return isTypeNestedGeneric(v.Type())
}

func (trans *Transformer) eraseGenericsFromTypeSpec(n *ast.TypeSpec, typ ast.Expr) *ast.TypeSpec {
	newType := trans.eraseGenericsFromType(typ)
	if newType != nil {
		newTypeSpec := astclone.Clone(n).(*ast.TypeSpec)
		newTypeSpec.TypeParams = nil
		newTypeSpec.Type = newType
		return newTypeSpec
	}
	return nil
}

func (trans *Transformer) eraseGenericsFromType(typ ast.Expr) ast.Expr {
	// Structs are treated specially.
	if structType, ok := typ.(*ast.StructType); ok {
		return trans.eraseGenericsFromStructType(structType)
	}

	typeAndValue, found := trans.Info.Types[typ]
	if !found {
		return nil
	}
	if isTypeNestedGeneric(typeAndValue.Type) {
		return newEmptyInterface()
	}
	return nil
}

func (trans *Transformer) eraseGenericsFromStructType(typ *ast.StructType) ast.Expr {
	if typ.Fields == nil {
		return nil
	}
	needsReplacement := false
	newFields := make([]*ast.Field, len(typ.Fields.List))
	for i, field := range typ.Fields.List {
		newFieldType := trans.eraseGenericsFromType(field.Type)
		if newFieldType != nil {
			newField := astclone.Clone(field).(*ast.Field)
			newField.Type = newFieldType
			newFields[i] = newField
			needsReplacement = true
		} else {
			newFields[i] = field
		}
	}
	if needsReplacement {
		newStructType := astclone.Clone(typ).(*ast.StructType)
		newStructType.Fields = &ast.FieldList{
			List: newFields,
		}
		return newStructType
	}
	return nil
}

// insertTypeConversions inserts type casts and conversions so that any usage
// of generic types is made compatible with the empty interface version.
func (trans *Transformer) insertTypeConversions() func(c *astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		// TODO(albrow): Continue by checking each relevant ast.Node type and
		// inserting type conversions as needed.
		case *ast.ValueSpec:
			newNode := trans.createTypeConversionForValueSpec(n)
			if newNode != nil {
				c.Replace(newNode)
			}
		case *ast.IndexExpr:
			newNode := trans.createTypeConversionForIndexExpr(n)
			if newNode != nil {
				c.Replace(newNode)
			}
		case *ast.CallExpr:
			// newNode := trans.createTypeConversionForCallExpr(n)
			// if newNode != nil {
			// 	c.Replace(newNode)
			// }
		case *ast.BinaryExpr:
			// newNode := trans.createTypeConversionForBinaryExpr(n)
			// if newNode != nil {
			// 	c.Replace(newNode)
			// }

		case *ast.SliceExpr:
		case *ast.KeyValueExpr:
		case *ast.DeclStmt:
		case *ast.SendStmt:
		case *ast.IncDecStmt:
		case *ast.AssignStmt:
		case *ast.ReturnStmt:
		case *ast.IfStmt:
		case *ast.ForStmt:
		case *ast.RangeStmt:
		}
		return true
	}
}

func (trans *Transformer) createTypeConversionForValueSpec(n *ast.ValueSpec) ast.Node {
	var newValueSpec *ast.ValueSpec
	for i, value := range n.Values {
		// First look at the type on the left-hand side of the value spec.
		name := n.Names[i]
		def, found := trans.Info.Defs[name]
		if !found {
			panic(fmt.Errorf("could not find definition for expression: %s %T", name, name))
		}
		if _, ok := def.Type().(*types.ConcreteNamed); ok {
			// If the left-hand side is already a concrete named type, we don't need
			// to do any conversions.
			continue
		}
		// Next look at the value on the right-hand side of the value spec. We need
		// to insert a type assertion if value is a concrete named type.
		newValue := trans.createTypeConversionForExpr(value)
		if newValue != nil {
			if newValueSpec == nil {
				newValueSpec = astclone.Clone(n).(*ast.ValueSpec)
			}
			newValueSpec.Values[i] = newValue
		}
	}
	// This check is necessary because nil interface != nil value.
	if newValueSpec == nil {
		return nil
	}
	return newValueSpec
}

func (trans *Transformer) createTypeConversionForIndexExpr(n *ast.IndexExpr) ast.Expr {
	underlyingType := trans.getUnderlyingTypeForExpr(n.X)
	if underlyingType == nil {
		return nil
	}
	newIndexExpr := astclone.Clone(n).(*ast.IndexExpr)
	newIndexExpr.X = wrapInTypeAssert(newIndexExpr.X, underlyingType)
	return newIndexExpr
}

func (trans *Transformer) createTypeConversionForExpr(n ast.Expr) ast.Expr {
	underlyingType := trans.getUnderlyingTypeForExpr(n)
	if underlyingType == nil {
		return nil
	}
	return wrapInTypeAssert(n, underlyingType)
}

func (trans *Transformer) getUnderlyingTypeForExpr(n ast.Expr) types.Type {
	switch n := n.(type) {
	case *ast.Ident:
		return trans.getUnderlyingTypeForIdent(n)
	case *ast.SelectorExpr:
		return trans.getUnderlyingTypeForSelectorExpr(n)
	}
	return nil
}

func (trans *Transformer) getUnderlyingTypeForIdent(n *ast.Ident) types.Type {
	typeAndValue, found := trans.Info.Types[n]
	if !found {
		return nil
	}
	concreteNamed, ok := typeAndValue.Type.(*types.ConcreteNamed)
	if !ok {
		return nil
	}
	return concreteNamed.Underlying()
}

func (trans *Transformer) getUnderlyingTypeForSelectorExpr(n *ast.SelectorExpr) types.Type {
	selection, found := trans.Info.Selections[n]
	if !found {
		return nil
	}
	switch selection.Kind() {
	case types.FieldVal:
		return trans.getUnderlyingTypeForFieldSelection(selection)
	case types.MethodVal:
		panic(errors.New("MethodVal selection not yet supported"))
	case types.MethodExpr:
		panic(errors.New("MethodExpr selection not yet supported"))
	}
	return nil
}

func (trans *Transformer) getUnderlyingTypeForFieldSelection(selection *types.Selection) types.Type {
	_, ok := selection.Recv().(*types.ConcreteNamed)
	if !ok {
		return nil
	}
	return selection.Type()
}

// If typ is a *types.TypeParam, finds the corresponding actual type by looking
// in concreteType.TypeMap and returns a new *ast.TypeAssertion which converts n
// to that type.
func wrapIfTypeParam(n ast.Expr, concreteType types.ConcreteType, typ types.Type) ast.Expr {
	typeParam, isTypeParam := typ.(*types.TypeParam)
	if !isTypeParam {
		return nil
	}
	// If the type is a TypeParameter, insert a type assertion to convert the
	// expression to the concrete type.
	actualType := concreteType.TypeMap()[typeParam.String()]
	return wrapInTypeAssert(n, actualType)
}

func findStructField(structType *types.Struct, fieldName string) *types.Var {
	for i := 0; i < structType.NumFields(); i++ {
		field := structType.Field(i)
		if field.Name() == fieldName {
			return field
		}
	}
	return nil
}

// wrapInTypeAssert returns n wrapped in a type assert expression (e.g., x
// becomes x.(type)).
func wrapInTypeAssert(n ast.Expr, typ types.Type) ast.Expr {
	return &ast.TypeAssertExpr{
		X:    n,
		Type: typeToExpr(typ),
	}
}

func newEmptyInterface() ast.Expr {
	return &ast.CompositeLit{
		Type: ast.NewIdent("interface"),
	}
}
