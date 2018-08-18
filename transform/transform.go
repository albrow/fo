package transform

import (
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
		// TODO(albrow): Erase generics from function bodies
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
		case *ast.FuncDecl:
			newFuncDecl := trans.eraseGenericsFromFuncDecl(n)
			if newFuncDecl != nil {
				c.Replace(newFuncDecl)
			}
		case *ast.ValueSpec:
			newType := trans.eraseGenericsFromType(n.Type)
			if newType != nil {
				newValueSpec := astclone.Clone(n).(*ast.ValueSpec)
				newValueSpec.Type = newType
				c.Replace(newValueSpec)
			}
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
	if v == nil {
		return false
	}
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
	newFieldList := trans.eraseGenericsFromFieldList(typ.Fields)
	if newFieldList == nil {
		return nil
	}
	newStructType := astclone.Clone(typ).(*ast.StructType)
	newStructType.Fields = newFieldList
	return newStructType
}

func (trans *Transformer) eraseGenericsFromFuncDecl(funcDecl *ast.FuncDecl) ast.Node {
	// First check if the function is generic (either has type parameters or a
	// generic receiver type).
	def, found := trans.Info.Defs[funcDecl.Name]
	if !found {
		return nil
	}
	if _, ok := def.Type().(*types.GenericSignature); !ok {
		// If the function is non-generic, we don't need to change anything.
		return nil
	}
	newFuncDecl := astclone.Clone(funcDecl).(*ast.FuncDecl)
	newFuncDecl.TypeParams = nil
	if funcDecl.Recv != nil {
		newRecv := trans.eraseGenericsFromReceiver(funcDecl.Recv)
		if newRecv != nil {
			newFuncDecl.Recv = newRecv
		}
	}
	if funcDecl.Type.Params != nil {
		newParams := trans.eraseGenericsFromFieldList(funcDecl.Type.Params)
		if newParams != nil {
			newFuncDecl.Type.Params = newParams
		}
	}
	if funcDecl.Type.Results != nil {
		newResults := trans.eraseGenericsFromFieldList(funcDecl.Type.Results)
		if newResults != nil {
			newFuncDecl.Type.Results = newResults
		}
	}
	if funcDecl.Body != nil {
		newBody := astutil.Apply(funcDecl.Body, trans.eraseGenerics(), nil).(*ast.BlockStmt)
		newFuncDecl.Body = newBody
	}
	return newFuncDecl
}

func (trans *Transformer) eraseGenericsFromReceiver(recv *ast.FieldList) *ast.FieldList {
	if recv.List == nil || len(recv.List) == 0 {
		return nil
	}
	recvField := recv.List[0]
	typeArgExpr, ok := recvField.Type.(*ast.TypeArgExpr)
	if !ok {
		return nil
	}
	newRecvField := astclone.Clone(recvField).(*ast.Field)
	newRecvField.Type = typeArgExpr.X
	newRecv := astclone.Clone(recv).(*ast.FieldList)
	newRecv.List[0] = newRecvField
	return newRecv
}

func (trans *Transformer) eraseGenericsFromFieldList(params *ast.FieldList) *ast.FieldList {
	needsReplacement := false
	newFields := make([]*ast.Field, len(params.List))
	for i, field := range params.List {
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
		newParams := astclone.Clone(params).(*ast.FieldList)
		newParams.List = newFields
		return newParams
	}
	return nil
}

// insertTypeConversions inserts type casts and conversions so that any usage
// of generic types is made compatible with the empty interface version.
func (trans *Transformer) insertTypeConversions() func(c *astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.Ident:
			newNode := trans.createTypeConversionForIdent(n)
			if newNode != nil {
				c.Replace(newNode)
			}
		case *ast.SelectorExpr:
			newNode := trans.createTypeConversionForSelectorExpr(n)
			if newNode != nil {
				c.Replace(newNode)
				return false
			}
		}
		return true
	}
}

func (trans *Transformer) createTypeConversionForIdent(n *ast.Ident) ast.Expr {
	if n.Name == "true" || n.Name == "false" {
		return nil
	}
	typeAndValue, found := trans.Info.Types[n]
	if !found {
		return nil
	}
	concreteType, ok := typeAndValue.Type.(types.ConcreteType)
	if !ok {
		return nil
	}
	if _, ok := concreteType.Underlying().(*types.Struct); ok {
		// Don't convert struct types. They are handled separately.
		return nil
	}
	return wrapInTypeAssert(n, concreteType.Underlying())
}

func (trans *Transformer) createTypeConversionForSelectorExpr(n *ast.SelectorExpr) ast.Expr {
	selection, found := trans.Info.Selections[n]
	if !found {
		return nil
	}
	switch selection.Kind() {
	case types.FieldVal:
		return trans.createTypeConversionForFieldSelector(n, selection)
	case types.MethodVal:
		// panic(errors.New("MethodVal not supported"))
		return nil
	case types.MethodExpr:
		// panic(errors.New("MethodExpr not supported"))
		return nil
	}
	return nil
}

func (trans *Transformer) createTypeConversionForFieldSelector(n *ast.SelectorExpr, selection *types.Selection) ast.Expr {
	if _, ok := selection.Recv().(*types.ConcreteNamed); !ok {
		return nil
	}
	return wrapInTypeAssert(n, selection.Type())
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
