package transform

import (
	"fmt"

	"github.com/albrow/fo/ast"
)

// Returns an expression returning the zero value for the given type.
// T => reflect.ZeroValue(reflect.TypeOf(T)).Interface{}
func makeZeroValue(typ ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("reflect"),
					Sel: ast.NewIdent("Zero"),
				},
				Args: []ast.Expr{makeType(typ)},
			},
			Sel: ast.NewIdent("Interface"),
		},
	}
}

func makeType(typ ast.Expr) ast.Expr {
	switch typ := typ.(type) {
	case *ast.Ident:
		return makeTypeForIdent(typ)
	case *ast.ArrayType:
		if typ.Len == nil {
			return makeSliceType(typ)
		} else {
			return makeArrayType(typ.Len, typ)
		}
	}
	panic(fmt.Errorf("Could not make type for: %T: %s %+v", typ, typ, typ))
}

// T => reflect.TypeOf(T)
func makeTypeForIdent(n *ast.Ident) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("TypeOf"),
		},
		Args: []ast.Expr{
			n,
		},
	}
}

func makeSliceType(typ *ast.ArrayType) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("SliceOf"),
		},
		Args: []ast.Expr{makeType(typ.Elt)},
	}
}

func makeArrayType(count ast.Expr, typ *ast.ArrayType) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("ArrayOf"),
		},
		Args: []ast.Expr{count, makeType(typ.Elt)},
	}
}
