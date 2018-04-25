package astcmp

import (
	"fmt"
	"reflect"

	"github.com/albrow/fo/ast"
)

// A Mode value is a set of flags (or 0). They control how nodes are compared.
type Mode uint

const (
	// IgnorePos means that position information will be ignored and two nodes
	// will be considered equal even if they have different positions.
	IgnorePos Mode = 1 << iota
	// IgnoreUnresolved means that for any *ast.File, the Undeclared field will
	// be ignored.
	IgnoreUnresolved
)

// Equal returns true if the two nodes have equal values. Two nodes are
// considered equal if they have the same type and the same underlying values
// (regardless of whether they have the same pointer address), and if, by
// recursion, their children are also equal. Equal does not compare objects or
// scopes. If conf is nil, the Default config will be used.
func Equal(x, y ast.Node, mode Mode) bool {
	if reflect.TypeOf(x) != reflect.TypeOf(y) {
		return false
	}

	// Note: because an interface with an underlying value of nil is not
	// considered equal to a literal nil, we have to use reflection to compare the
	// underlying values of the interface. But before we do that, we also have to
	// check whether the values are valid because IsNil will panic if the value
	// was nil (as opposed to an interface or other type with an underlying value
	// of nil). See https://golang.org/doc/faq#nil_error for more information.
	valX := reflect.ValueOf(x)
	valY := reflect.ValueOf(y)
	if !valX.IsValid() && !valY.IsValid() {
		return true
	} else if valX.IsValid() && !valY.IsValid() {
		return false
	} else if !valX.IsValid() && valY.IsValid() {
		return false
	}
	if valX.IsNil() && valY.IsNil() {
		return true
	} else if valX.IsNil() && !valY.IsNil() {
		return false
	} else if !valX.IsNil() && valY.IsNil() {
		return false
	}

	if mode&IgnorePos == 0 {
		if x.Pos() != y.Pos() {
			return false
		} else if x.End() != y.End() {
			return false
		}
	}

	switch x := x.(type) {
	case *ast.Comment:
		y := y.(*ast.Comment)
		if x.Text != y.Text {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.Slash != y.Slash {
				return false
			}
		}

	case *ast.CommentGroup:
		y := y.(*ast.CommentGroup)
		if len(y.List) != len(x.List) {
			return false
		}
		for i, xe := range x.List {
			ye := y.List[i]
			if !Equal(xe, ye, mode) {
				return false
			}
		}

	case *ast.Field:
		y := y.(*ast.Field)
		if !Equal(x.Doc, y.Doc, mode) {
			return false
		}
		if !Equal(x.Type, y.Type, mode) {
			return false
		}
		if !Equal(x.Tag, y.Tag, mode) {
			return false
		}
		if !Equal(x.Comment, y.Comment, mode) {
			return false
		}
		if !compareIdents(x.Names, y.Names, mode) {
			return false
		}

	case *ast.FieldList:
		y := y.(*ast.FieldList)
		if mode&IgnorePos == 0 {
			if x.Opening != y.Opening {
				return false
			} else if x.Closing != y.Closing {
				return false
			}
		}
		if !compareFields(x.List, y.List, mode) {
			return false
		}

	case *ast.BadExpr:
		y := y.(*ast.BadExpr)
		if mode&IgnorePos == 0 {
			if x.From != y.From {
				return false
			} else if x.To != y.To {
				return false
			}
		}

	case *ast.Ident:
		y := y.(*ast.Ident)
		if x.Name != y.Name {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.NamePos != y.NamePos {
				return false
			}
		}
		// TODO(albrow): compare Obj?

	case *ast.BasicLit:
		y := y.(*ast.BasicLit)
		if x.Value != y.Value {
			return false
		}
		if x.Kind != y.Kind {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.ValuePos != y.ValuePos {
				return false
			}
		}

	case *ast.TypeParamDecl:
		y := y.(*ast.TypeParamDecl)
		if mode&IgnorePos == 0 {
			if x.Lbrack != y.Lbrack {
				return false
			} else if x.Rbrack != y.Rbrack {
				return false
			}
		}
		if !compareIdents(x.Names, y.Names, mode) {
			return false
		}

	case *ast.TypeParamExpr:
		y := y.(*ast.TypeParamExpr)
		if mode&IgnorePos == 0 {
			if x.Lbrack != y.Lbrack {
				return false
			} else if x.Rbrack != y.Rbrack {
				return false
			}
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}
		if !compareExprs(x.Params, y.Params, mode) {
			return false
		}

	case *ast.Ellipsis:
		y := y.(*ast.Ellipsis)
		if mode&IgnorePos == 0 {
			if x.Ellipsis != y.Ellipsis {
				return false
			}
		}
		if !Equal(x.Elt, y.Elt, mode) {
			return false
		}

	case *ast.FuncLit:
		y := y.(*ast.FuncLit)
		if !Equal(x.Body, y.Body, mode) {
			return false
		}
		if !Equal(x.Type, y.Type, mode) {
			return false
		}

	case *ast.CompositeLit:
		y := y.(*ast.CompositeLit)
		if mode&IgnorePos == 0 {
			if x.Lbrace != y.Lbrace {
				return false
			} else if x.Rbrace != y.Rbrace {
				return false
			}
		}
		if !Equal(x.Type, y.Type, mode) {
			return false
		}
		if !compareExprs(x.Elts, y.Elts, mode) {
			return false
		}

	case *ast.ParenExpr:
		y := y.(*ast.ParenExpr)
		if mode&IgnorePos == 0 {
			if x.Lparen != y.Lparen {
				return false
			} else if x.Rparen != y.Rparen {
				return false
			}
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}

	case *ast.SelectorExpr:
		y := y.(*ast.SelectorExpr)
		if !Equal(x.X, y.X, mode) {
			return false
		}
		if !Equal(x.Sel, y.Sel, mode) {
			return false
		}

	case *ast.IndexExpr:
		y := y.(*ast.IndexExpr)
		if mode&IgnorePos == 0 {
			if x.Lbrack != y.Lbrack {
				return false
			} else if x.Rbrack != y.Rbrack {
				return false
			}
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}
		if !Equal(x.Index, y.Index, mode) {
			return false
		}

	case *ast.SliceExpr:
		y := y.(*ast.SliceExpr)
		if x.Slice3 != y.Slice3 {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.Lbrack != y.Lbrack {
				return false
			} else if x.Lbrack != y.Lbrack {
				return false
			}
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}
		if !Equal(x.Low, y.Low, mode) {
			return false
		}
		if !Equal(x.High, y.High, mode) {
			return false
		}
		if !Equal(x.Max, y.Max, mode) {
			return false
		}

	case *ast.TypeAssertExpr:
		y := y.(*ast.TypeAssertExpr)
		if mode&IgnorePos == 0 {
			if x.Lparen != y.Lparen {
				return false
			} else if x.Rparen != y.Rparen {
				return false
			}
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}
		if !Equal(x.Type, y.Type, mode) {
			return false
		}

	case *ast.CallExpr:
		y := y.(*ast.CallExpr)
		if mode&IgnorePos == 0 {
			if x.Lparen != y.Lparen {
				return false
			} else if x.Rparen != y.Rparen {
				return false
			} else if x.Ellipsis != y.Ellipsis {
				return false
			}
		}
		if !Equal(x.Fun, y.Fun, mode) {
			return false
		}
		if !compareExprs(x.Args, y.Args, mode) {
			return false
		}

	case *ast.StarExpr:
		y := y.(*ast.StarExpr)
		if mode&IgnorePos == 0 {
			if x.Star != y.Star {
				return false
			}
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}

	case *ast.UnaryExpr:
		y := y.(*ast.UnaryExpr)
		if x.Op != y.Op {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.OpPos != y.OpPos {
				return false
			}
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}

	case *ast.BinaryExpr:
		y := y.(*ast.BinaryExpr)
		if x.Op != y.Op {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.OpPos != y.OpPos {
				return false
			}
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}
		if !Equal(x.Y, y.Y, mode) {
			return false
		}

	case *ast.KeyValueExpr:
		y := y.(*ast.KeyValueExpr)
		if mode&IgnorePos == 0 {
			if x.Colon != y.Colon {
				return false
			}
		}
		if !Equal(x.Key, y.Key, mode) {
			return false
		}
		if !Equal(x.Value, y.Value, mode) {
			return false
		}

	case *ast.ArrayType:
		y := y.(*ast.ArrayType)
		if mode&IgnorePos == 0 {
			if x.Lbrack != y.Lbrack {
				return false
			}
		}
		if !Equal(x.Len, y.Len, mode) {
			return false
		}
		if !Equal(x.Elt, y.Elt, mode) {
			return false
		}

	case *ast.StructType:
		y := y.(*ast.StructType)
		if x.Incomplete != y.Incomplete {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.Struct != y.Struct {
				return false
			}
		}
		if !Equal(x.Fields, y.Fields, mode) {
			return false
		}

	case *ast.FuncType:
		y := y.(*ast.FuncType)
		if mode&IgnorePos == 0 {
			if x.Func != y.Func {
				return false
			}
		}
		if !Equal(x.Params, y.Params, mode) {
			return false
		}
		if !Equal(x.Results, y.Results, mode) {
			return false
		}

	case *ast.InterfaceType:
		y := y.(*ast.InterfaceType)
		if x.Incomplete != y.Incomplete {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.Interface != y.Interface {
				return false
			}
		}
		if !Equal(x.Methods, y.Methods, mode) {
			return false
		}

	case *ast.MapType:
		y := y.(*ast.MapType)
		if mode&IgnorePos == 0 {
			if x.Map != y.Map {
				return false
			}
		}
		if !Equal(x.Key, y.Key, mode) {
			return false
		}
		if !Equal(x.Value, y.Value, mode) {
			return false
		}

	case *ast.ChanType:
		y := y.(*ast.ChanType)
		if x.Dir != y.Dir {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.Begin != y.Begin {
				return false
			} else if x.Arrow != y.Arrow {
				return false
			}
		}
		if !Equal(x.Value, y.Value, mode) {
			return false
		}

	case *ast.BadStmt:
		y := y.(*ast.BadStmt)
		if mode&IgnorePos == 0 {
			if x.From != y.From {
				return false
			} else if x.To != y.To {
				return false
			}
		}

	case *ast.DeclStmt:
		y := y.(*ast.DeclStmt)
		if !Equal(x.Decl, y.Decl, mode) {
			return false
		}

	case *ast.EmptyStmt:
		y := y.(*ast.EmptyStmt)
		if x.Implicit != y.Implicit {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.Semicolon != y.Semicolon {
				return false
			}
		}

	case *ast.LabeledStmt:
		y := y.(*ast.LabeledStmt)
		if mode&IgnorePos == 0 {
			if x.Colon != y.Colon {
				return false
			}
		}
		if !Equal(x.Label, y.Label, mode) {
			return false
		}
		if !Equal(x.Stmt, y.Stmt, mode) {
			return false
		}

	case *ast.ExprStmt:
		y := y.(*ast.ExprStmt)
		if !Equal(x.X, y.X, mode) {
			return false
		}

	case *ast.SendStmt:
		y := y.(*ast.SendStmt)
		if mode&IgnorePos == 0 {
			if x.Arrow != y.Arrow {
				return false
			}
		}
		if !Equal(x.Chan, y.Chan, mode) {
			return false
		}
		if !Equal(x.Value, y.Value, mode) {
			return false
		}

	case *ast.IncDecStmt:
		y := y.(*ast.IncDecStmt)
		if x.Tok != y.Tok {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.TokPos != y.TokPos {
				return false
			}
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}

	case *ast.AssignStmt:
		y := y.(*ast.AssignStmt)
		if x.Tok != y.Tok {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.TokPos != y.TokPos {
				return false
			}
		}
		if !compareExprs(x.Lhs, y.Lhs, mode) {
			return false
		}
		if !compareExprs(x.Rhs, y.Rhs, mode) {
			return false
		}

	case *ast.GoStmt:
		y := y.(*ast.GoStmt)
		if mode&IgnorePos == 0 {
			if x.Go != y.Go {
				return false
			}
		}
		if !Equal(x.Call, y.Call, mode) {
			return false
		}

	case *ast.DeferStmt:
		y := y.(*ast.DeferStmt)
		if mode&IgnorePos == 0 {
			if x.Defer != y.Defer {
				return false
			}
		}
		if !Equal(x.Call, y.Call, mode) {
			return false
		}

	case *ast.ReturnStmt:
		y := y.(*ast.ReturnStmt)
		if mode&IgnorePos == 0 {
			if x.Return != y.Return {
				return false
			}
		}
		if !compareExprs(x.Results, y.Results, mode) {
			return false
		}

	case *ast.BranchStmt:
		y := y.(*ast.BranchStmt)
		if x.Tok != y.Tok {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.TokPos != y.TokPos {
				return false
			}
		}
		if !Equal(x.Label, y.Label, mode) {
			return false
		}

	case *ast.BlockStmt:
		y := y.(*ast.BlockStmt)
		if mode&IgnorePos == 0 {
			if x.Lbrace != y.Lbrace {
				return false
			} else if x.Rbrace != y.Rbrace {
				return false
			}
		}
		if !compareStmts(x.List, y.List, mode) {
			return false
		}

	case *ast.IfStmt:
		y := y.(*ast.IfStmt)
		if mode&IgnorePos == 0 {
			if x.If != y.If {
				return false
			}
		}
		if !Equal(x.Init, y.Init, mode) {
			return false
		}
		if !Equal(x.Cond, y.Cond, mode) {
			return false
		}
		if !Equal(x.Body, y.Body, mode) {
			return false
		}
		if !Equal(x.Else, y.Else, mode) {
			return false
		}

	case *ast.CaseClause:
		y := y.(*ast.CaseClause)
		if mode&IgnorePos == 0 {
			if x.Case != y.Case {
				return false
			} else if x.Colon != y.Colon {
				return false
			}
		}
		if !compareExprs(x.List, y.List, mode) {
			return false
		}
		if !compareStmts(x.Body, y.Body, mode) {
			return false
		}

	case *ast.SwitchStmt:
		y := y.(*ast.SwitchStmt)
		if mode&IgnorePos == 0 {
			if x.Switch != y.Switch {
				return false
			}
		}
		if !Equal(x.Init, y.Init, mode) {
			return false
		}
		if !Equal(x.Tag, y.Tag, mode) {
			return false
		}
		if !Equal(x.Body, y.Body, mode) {
			return false
		}

	case *ast.TypeSwitchStmt:
		y := y.(*ast.TypeSwitchStmt)
		if mode&IgnorePos == 0 {
			if x.Switch != y.Switch {
				return false
			}
		}
		if !Equal(x.Init, y.Init, mode) {
			return false
		}
		if !Equal(x.Assign, y.Assign, mode) {
			return false
		}
		if !Equal(x.Body, y.Body, mode) {
			return false
		}

	case *ast.CommClause:
		y := y.(*ast.CommClause)
		if mode&IgnorePos == 0 {
			if x.Case != y.Case {
				return false
			} else if x.Colon != y.Colon {
				return false
			}
		}
		if !Equal(x.Comm, y.Comm, mode) {
			return false
		}
		if !compareStmts(x.Body, y.Body, mode) {
			return false
		}

	case *ast.SelectStmt:
		y := y.(*ast.SelectStmt)
		if mode&IgnorePos == 0 {
			if x.Select != y.Select {
				return false
			}
		}
		if !Equal(x.Body, y.Body, mode) {
			return false
		}

	case *ast.ForStmt:
		y := y.(*ast.ForStmt)
		if mode&IgnorePos == 0 {
			if x.For != y.For {
				return false
			}
		}
		if !Equal(x.Init, y.Init, mode) {
			return false
		}
		if !Equal(x.Cond, y.Cond, mode) {
			return false
		}
		if !Equal(x.Post, y.Post, mode) {
			return false
		}
		if !Equal(x.Body, y.Body, mode) {
			return false
		}

	case *ast.RangeStmt:
		y := y.(*ast.RangeStmt)
		if x.Tok != y.Tok {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.For != y.For {
				return false
			} else if x.TokPos != y.TokPos {
				return false
			}
		}
		if !Equal(x.Key, y.Key, mode) {
			return false
		}
		if !Equal(x.Value, y.Value, mode) {
			return false
		}
		if !Equal(x.X, y.X, mode) {
			return false
		}
		if !Equal(x.Body, y.Body, mode) {
			return false
		}

	case *ast.ImportSpec:
		y := y.(*ast.ImportSpec)
		if mode&IgnorePos == 0 {
			if x.EndPos != y.EndPos {
				return false
			}
		}
		if !Equal(x.Doc, y.Doc, mode) {
			return false
		}
		if !Equal(x.Name, y.Name, mode) {
			return false
		}
		if !Equal(x.Path, y.Path, mode) {
			return false
		}
		if !Equal(x.Comment, y.Comment, mode) {
			return false
		}

	case *ast.ValueSpec:
		y := y.(*ast.ValueSpec)
		if !Equal(x.Doc, y.Doc, mode) {
			return false
		}
		if !Equal(x.Type, y.Type, mode) {
			return false
		}
		if !Equal(x.Comment, y.Comment, mode) {
			return false
		}
		if !compareIdents(x.Names, y.Names, mode) {
			return false
		}
		if !compareExprs(x.Values, y.Values, mode) {
			return false
		}

	case *ast.TypeSpec:
		y := y.(*ast.TypeSpec)
		if mode&IgnorePos == 0 {
			if x.Assign != y.Assign {
				return false
			}
		}
		if !Equal(x.Doc, y.Doc, mode) {
			return false
		}
		if !Equal(x.Name, y.Name, mode) {
			return false
		}
		if !Equal(x.Type, y.Type, mode) {
			return false
		}
		if !Equal(x.Comment, y.Comment, mode) {
			return false
		}
		if !Equal(x.TypeParams, y.TypeParams, mode) {
			return false
		}

	case *ast.BadDecl:
		y := y.(*ast.BadDecl)
		if mode&IgnorePos == 0 {
			if x.From != y.From {
				return false
			} else if x.To != y.To {
				return false
			}
		}

	case *ast.GenDecl:
		y := y.(*ast.GenDecl)
		if x.Tok != y.Tok {
			return false
		}
		if mode&IgnorePos == 0 {
			if x.TokPos != y.TokPos {
				return false
			} else if x.Lparen != y.Lparen {
				return false
			} else if x.Rparen != y.Rparen {
				return false
			}
		}
		if !Equal(x.Doc, y.Doc, mode) {
			return false
		}
		if len(x.Specs) != len(y.Specs) {
			return false
		}
		for i, xe := range x.Specs {
			ye := y.Specs[i]
			if !Equal(xe, ye, mode) {
				return false
			}
		}

	case *ast.FuncDecl:
		y := y.(*ast.FuncDecl)
		if !Equal(x.Doc, y.Doc, mode) {
			return false
		}
		if !Equal(x.Recv, y.Recv, mode) {
			return false
		}
		if !Equal(x.Name, y.Name, mode) {
			return false
		}
		if !Equal(x.Type, y.Type, mode) {
			return false
		}
		if !Equal(x.Body, y.Body, mode) {
			return false
		}
		if !Equal(x.TypeParams, y.TypeParams, mode) {
			return false
		}

	case *ast.File:
		y := y.(*ast.File)
		if mode&IgnorePos == 0 {
			if x.Package != y.Package {
				return false
			}
		}
		if !Equal(x.Doc, y.Doc, mode) {
			return false
		}
		if !Equal(x.Name, y.Name, mode) {
			return false
		}
		if !Equal(x.Doc, y.Doc, mode) {
			return false
		}
		if mode&IgnoreUnresolved == 0 {
			if !compareIdents(x.Unresolved, y.Unresolved, mode) {
				return false
			}
		}
		if len(x.Decls) != len(y.Decls) {
			return false
		}
		for i, xe := range x.Decls {
			ye := y.Decls[i]
			if !Equal(xe, ye, mode) {
				return false
			}
		}
		if len(x.Imports) != len(y.Imports) {
			return false
		}
		for i, xe := range x.Imports {
			ye := y.Imports[i]
			if !Equal(xe, ye, mode) {
				return false
			}
		}
		if len(x.Comments) != len(y.Comments) {
			return false
		}
		for i, xe := range x.Comments {
			ye := y.Comments[i]
			if !Equal(xe, ye, mode) {
				return false
			}
		}
		// TODO(albrow): compare Scope?

	case *ast.Package:
		y := y.(*ast.Package)
		if x.Name != y.Name {
			return false
		}
		if len(x.Imports) != len(y.Imports) {
			return false
		}
		for name := range x.Imports {
			_, found := y.Imports[name]
			if !found {
				return false
			}
			// TODO(albrow): compare Object?
		}
		if len(x.Files) != len(y.Files) {
			return false
		}
		// The Truth is Out There. Doo doo doo doo doo dooooo.
		for name, xFile := range x.Files {
			yFile, found := y.Files[name]
			if !found {
				return false
			}
			if !Equal(xFile, yFile, mode) {
				return false
			}
		}
		// TODO(albrow): compare Scope?

	default:
		panic(fmt.Sprintf("astcmp.Equal: unexpected node type %T", x))
	}

	return true
}

func compareIdents(x, y []*ast.Ident, mode Mode) bool {
	if len(x) != len(y) {
		return false
	}
	for i, xi := range x {
		yi := y[i]
		if !Equal(xi, yi, mode) {
			return false
		}
	}

	return true
}

func compareFields(x, y []*ast.Field, mode Mode) bool {
	if len(x) != len(y) {
		return false
	}
	for i, xi := range x {
		yi := y[i]
		if !Equal(xi, yi, mode) {
			return false
		}
	}

	return true
}

func compareExprs(x, y []ast.Expr, mode Mode) bool {
	if len(x) != len(y) {
		return false
	}
	for i, xi := range x {
		yi := y[i]
		if !Equal(xi, yi, mode) {
			return false
		}
	}

	return true
}

func compareStmts(x, y []ast.Stmt, mode Mode) bool {
	if len(x) != len(y) {
		return false
	}
	for i, xi := range x {
		yi := y[i]
		if !Equal(xi, yi, mode) {
			return false
		}
	}

	return true
}
