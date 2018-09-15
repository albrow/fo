package transform

import (
	"fmt"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/token"
)

// Returns an expression returning the zero value for the given type.
// T => reflect.ZeroValue(reflect.TypeOf(T)).Interface{}
func makeZeroValue(typ ast.Expr) ast.Expr {
	return reflectValToInterface(
		&ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("reflect"),
				Sel: ast.NewIdent("Zero"),
			},
			Args: []ast.Expr{makeType(typ)},
		},
	)
}

func reflectValToInterface(val ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: val,
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
			return makeSliceType(typ.Elt)
		} else {
			return makeArrayType(typ.Len, typ.Elt)
		}
	case *ast.MapType:
		return makeMapType(typ.Key, typ.Value)
	case *ast.StarExpr:
		return makePtrType(typ.X)
	case *ast.ChanType:
		return makeChanType(typ.Dir, typ.Value)
	case *ast.FuncType:
		return makeFuncType(typ)
	}
	panic(fmt.Errorf("Could not make type for: %T: %s %+v", typ, typ, typ))
}

// T => reflect.TypeOf(T)
func makeTypeForIdent(n *ast.Ident) ast.Expr {
	if typeIsPrimitive(n) {
		return makePrimitiveType(n)
	}
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

func makeSliceType(elt ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("SliceOf"),
		},
		Args: []ast.Expr{makeType(elt)},
	}
}

func makeArrayType(length ast.Expr, elt ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("ArrayOf"),
		},
		Args: []ast.Expr{length, makeType(elt)},
	}
}

func makeMapType(key ast.Expr, val ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("MapOf"),
		},
		Args: []ast.Expr{makeType(key), makeType(val)},
	}
}

func makePtrType(typ ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("PtrTo"),
		},
		Args: []ast.Expr{makeType(typ)},
	}
}

func makeChanType(dir ast.ChanDir, typ ast.Expr) ast.Expr {
	dirExpr := getChanDir(dir)
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("ChanOf"),
		},
		Args: []ast.Expr{dirExpr, makeType(typ)},
	}
}

func getChanDir(dir ast.ChanDir) ast.Expr {
	switch dir {
	case ast.SEND:
		return &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("SendDir"),
		}
	case ast.RECV:
		return &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("RecvDir"),
		}
	default:
		return &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("BothDir"),
		}
	}
}

func makeFuncType(funcType *ast.FuncType) ast.Expr {
	in := makeTypesFromFieldList(funcType.Params)
	out := makeTypesFromFieldList(funcType.Results)
	var variadicExpr ast.Expr
	if funcIsVariadic(funcType.Params) {
		variadicExpr = ast.NewIdent("true")
	} else {
		variadicExpr = ast.NewIdent("false")
	}
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("FuncOf"),
		},
		Args: []ast.Expr{in, out, variadicExpr},
	}
}

func makeTypesFromFieldList(fieldList *ast.FieldList) ast.Expr {
	if fieldList == nil || fieldList.List == nil || len(fieldList.List) == 0 {
		return makeSliceLiteral(&ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("Type"),
		}, []ast.Expr{})
	}
	types := make([]ast.Expr, len(fieldList.List))
	for i, field := range fieldList.List {
		types[i] = makeType(field.Type)
	}
	return makeSliceLiteral(&ast.SelectorExpr{
		X:   ast.NewIdent("reflect"),
		Sel: ast.NewIdent("Type"),
	}, types)
}

func funcIsVariadic(params *ast.FieldList) bool {
	return false
}

func makeSliceLiteral(elType ast.Expr, values []ast.Expr) ast.Expr {
	return &ast.CompositeLit{
		Type: &ast.ArrayType{
			Elt: elType,
		},
		Elts: values,
	}
}

func typeIsPrimitive(typ ast.Expr) bool {
	ident, ok := typ.(*ast.Ident)
	if !ok {
		return false
	}
	switch ident.String() {
	case "string", "bool", "byte", "rune", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "complex64", "complex128",
		"uintptr":
		return true
	}
	return false
}

func makePrimitiveZeroVal(typ *ast.Ident) ast.Expr {
	switch typ.String() {
	case "string":
		return &ast.BasicLit{
			Kind:  token.STRING,
			Value: `""`,
		}
	case "bool":
		return ast.NewIdent("false")
	case "byte", "rune", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64",
		"complex64", "complex128", "uintptr":
		return &ast.CallExpr{
			Fun: ast.NewIdent(typ.String()),
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.INT,
					Value: "0",
				},
			},
		}
	}
	panic(fmt.Errorf("unknown primitive type: %T %s", typ, typ))
}

func makePrimitiveType(typ *ast.Ident) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("TypeOf"),
		},
		Args: []ast.Expr{
			makePrimitiveZeroVal(typ),
		},
	}
}

// n -> reflect.Len(reflect.ValueOf(n)).Interface()
func makeLenExpr(n ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   makeValueOf(n),
			Sel: ast.NewIdent("Len"),
		},
	}
}

func makeValueOf(n ast.Expr) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("reflect"),
			Sel: ast.NewIdent("ValueOf"),
		},
		Args: []ast.Expr{
			n,
		},
	}
}
