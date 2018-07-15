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
		case *ast.FuncDecl:
			// Remove type parameters from function declarations.
			if n.TypeParams != nil {
				newFuncDecl := astclone.Clone(n).(*ast.FuncDecl)
				newBody := astutil.Apply(n.Body, trans.eraseGenerics(), nil).(*ast.BlockStmt)
				newReceiver := astutil.Apply(n.Recv, trans.eraseGenerics(), nil).(*ast.FieldList)
				newType := astutil.Apply(n.Type, trans.eraseGenerics(), nil).(*ast.FuncType)
				newFuncDecl.TypeParams = nil
				newFuncDecl.Body = newBody
				newFuncDecl.Recv = newReceiver
				newFuncDecl.Type = newType
				c.Replace(newFuncDecl)
				return false
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
					newTypeSpec := astclone.Clone(n).(*ast.TypeSpec)
					newType := astutil.Apply(arrayType.Elt, trans.eraseGenerics(), nil).(ast.Expr)
					newTypeSpec.TypeParams = nil
					newTypeSpec.Type = newType
					c.Replace(newTypeSpec)
					return false
				}
			}
			// If the typespec is an unambiguous generic type, we remove the type
			// parameters.
			newTypeSpec := astclone.Clone(n).(*ast.TypeSpec)
			newType := astutil.Apply(n.Type, trans.eraseGenerics(), nil).(ast.Expr)
			newTypeSpec.TypeParams = nil
			newTypeSpec.Type = newType
			c.Replace(newTypeSpec)
			return false
		case ast.Expr:
			if typAndValue, found := trans.Info.Types[n]; found {
				switch typAndValue.Type.(type) {
				case *types.TypeParam:
					// Change type parameters to the empty interface.
					emptyInterface := &ast.CompositeLit{
						Type: ast.NewIdent("interface"),
					}
					c.Replace(emptyInterface)
					return false
				}
			}
		}
		return true
	}
}

// insertTypeConversions inserts type casts and conversions so that any usage
// of generic types is made compatible with the empty interface version.
func (trans *Transformer) insertTypeConversions() func(c *astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		switch n := c.Node().(type) {
		case *ast.SelectorExpr:
			newNode := trans.createTypeConversionsForSelectorExpr(n)
			c.Replace(newNode)
			// TODO(albrow): Continue by checking each relevant ast.Node type and
			// inserting type conversions as needed.
		}
		return true
	}
}

func (trans *Transformer) createTypeConversionsForSelectorExpr(n *ast.SelectorExpr) ast.Node {
	xType := trans.Info.TypeOf(n.X)
	if xType == nil {
		panic(fmt.Errorf("could not find type for *ast.SelectorExpr: %+v", n))
	}
	switch xType := xType.(type) {
	case *types.ConcreteNamed:
		// We are accessing a field of a generic struct type.
		return trans.createTypeConversionsForStructFieldAccess(n, xType)
	}
	return n
}

func (trans *Transformer) createTypeConversionsForStructFieldAccess(n *ast.SelectorExpr, concreteNamed *types.ConcreteNamed) ast.Node {
	// Determine if the field we are accessing is type parameterized or a normal
	// field type. (e.g. is field x defined as in `struct{x T}` or
	// `struct{x string}`).
	genStructType, ok := concreteNamed.GenericType().Underlying().(*types.Struct)
	if !ok {
		panic(fmt.Errorf("selector used on unexpected *GenericNamed type: %T", concreteNamed.Underlying()))
	}
	fieldName := n.Sel.String()
	field := findStructField(genStructType, fieldName)
	if field == nil {
		panic(fmt.Errorf("could not find field named %q in struct type %s", fieldName, concreteNamed.Obj().Name()))
	}
	typeParam, fieldIsTypeParam := field.Type().(*types.TypeParam)
	if !fieldIsTypeParam {
		// If the type of the field is not parameterized, return the node
		// unaltered.
		return n
	}
	// If the type of the field is parameterized, insert a type assertion to
	// convert the field to the appropriate type.
	concreteFieldType := concreteNamed.TypeMap()[typeParam.String()]
	return wrapInTypeAssert(n, concreteFieldType)
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
func wrapInTypeAssert(n ast.Expr, typ types.Type) ast.Node {
	return &ast.TypeAssertExpr{
		X:    n,
		Type: typeToExpr(typ),
	}
}
