package transform

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/format"
	"github.com/albrow/fo/token"
	"github.com/albrow/fo/types"
)

const maxSafeStringCounter = 1000

var safeSymbolMap = map[string]string{
	".": "_",
	"[": "_",
	"]": "_",
	"*": "_",
	"/": "_",
	"-": "_",
	" ": "_",
}

// unsafeToSafe is a mapping of unsafe type strings to safe type strings.
var unsafeToSafe map[string]string = map[string]string{}

// safeToUnsafe is a mapping of safe type strings to unsafe type strings.
var safeToUnsafe map[string]string = map[string]string{}

func typeToSafeString(typ types.Type) string {
	return exprToSafeString(typeToExpr(typ))
}

// TODO(albrow): This could be optimized.
func replaceUnsafeSymbols(unsafe string) string {
	unsafe = strings.TrimSpace(unsafe)
	if safe, found := unsafeToSafe[unsafe]; found {
		return safe
	}
	safe := unsafe
	for unsafeSymbol, safeSymbol := range safeSymbolMap {
		safe = strings.Replace(safe, unsafeSymbol, safeSymbol, -1)
	}
	if _, found := safeToUnsafe[safe]; found {
		// The safe string collides with another safe string that we have generated.
		// We need to append a counter to make it unique.
		safe = appendSafeStringCounter(safe)
	}
	unsafeToSafe[unsafe] = safe
	safeToUnsafe[safe] = unsafe
	return safe
}

// TODO(albrow): This could be optimized.
func appendSafeStringCounter(s string) string {
	for i := 0; i < 100; i++ {
		stringWithCounter := fmt.Sprintf("%s_%d", s, i)
		if _, found := safeToUnsafe[stringWithCounter]; !found {
			return stringWithCounter
		}
	}
	panic(fmt.Errorf("Could not find unique safe string for %s", s))
}

func exprToSafeString(expr ast.Expr) string {
	buf := bytes.Buffer{}
	format.Node(&buf, token.NewFileSet(), expr)
	return replaceUnsafeSymbols(buf.String())
}

func typeToExpr(typ types.Type) ast.Expr {
	switch typ := typ.(type) {
	case *types.Pointer:
		return pointerTypeToExpr(typ)
	case *types.Slice:
		return sliceTypeToExpr(typ)
	case *types.Array:
		return arrayTypeToExpr(typ)
	case *types.Map:
		return mapTypetoExpr(typ)
	case *types.Chan:
		return chanTypeToExpr(typ)
	case *types.Struct:
		return structTypeToExpr(typ)
	case *types.Signature:
		return signatureTypeToExpr(typ)
	case *types.Named:
		return namedTypeToExpr(typ)
	}
	return ast.NewIdent(typ.String())
}

func pointerTypeToExpr(ptr *types.Pointer) ast.Expr {
	return &ast.StarExpr{
		X: typeToExpr(ptr.Elem()),
	}
}

func sliceTypeToExpr(slice *types.Slice) ast.Expr {
	return &ast.ArrayType{
		Len: nil,
		Elt: typeToExpr(slice.Elem()),
	}
}

func arrayTypeToExpr(array *types.Array) ast.Expr {
	return &ast.ArrayType{
		Len: &ast.BasicLit{
			Kind:  token.INT,
			Value: strconv.Itoa(int(array.Len())),
		},
		Elt: typeToExpr(array.Elem()),
	}
}

func mapTypetoExpr(m *types.Map) ast.Expr {
	return &ast.MapType{
		Key:   typeToExpr(m.Key()),
		Value: typeToExpr(m.Elem()),
	}
}

func chanTypeToExpr(ch *types.Chan) ast.Expr {
	var chanDir ast.ChanDir
	switch ch.Dir() {
	case types.SendRecv:
		chanDir = ast.SEND & ast.RECV
	case types.SendOnly:
		chanDir = ast.SEND
	case types.RecvOnly:
		chanDir = ast.RECV
	}
	return &ast.ChanType{
		Dir:   chanDir,
		Value: typeToExpr(ch.Elem()),
	}
}

func structTypeToExpr(st *types.Struct) ast.Expr {
	fieldList := make([]*ast.Field, st.NumFields())
	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		fieldList[i] = &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(field.Name())},
			Type:  typeToExpr(field.Type()),
		}
	}
	return &ast.StructType{
		Fields: &ast.FieldList{
			List: fieldList,
		},
	}
}

func signatureTypeToExpr(sig *types.Signature) ast.Expr {
	return &ast.FuncType{
		Params:  tupleToFieldList(sig.Params()),
		Results: tupleToFieldList(sig.Results()),
	}
}

func namedTypeToExpr(named *types.Named) ast.Expr {
	if named.Obj() == nil || named.Obj().Pkg() == nil {
		return ast.NewIdent(named.String())
	}
	if named.Obj().Pkg().Name() == "main" {
		return ast.NewIdent(named.Obj().Id())
	}
	return &ast.SelectorExpr{
		X:   ast.NewIdent(named.Obj().Pkg().Name()),
		Sel: ast.NewIdent(named.Obj().Id()),
	}
}

func tupleToFieldList(tuple *types.Tuple) *ast.FieldList {
	fieldList := make([]*ast.Field, tuple.Len())
	for i := 0; i < tuple.Len(); i++ {
		field := tuple.At(i)
		fieldList[i] = &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(field.Name())},
			Type:  typeToExpr(field.Type()),
		}
	}
	return &ast.FieldList{
		List: fieldList,
	}
}
