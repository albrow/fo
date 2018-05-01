package astclone

import (
	"fmt"

	"github.com/albrow/fo/ast"
)

func Clone(node ast.Node) ast.Node {

	switch n := node.(type) {
	case *ast.Comment:
		if n == nil {
			return nil
		}
		x := *n
		return &x

	case *ast.CommentGroup:
		return cloneCommentGroup(n)

	case *ast.Field:
		var tag *ast.BasicLit
		if n.Tag != nil {
			tag = Clone(n.Tag).(*ast.BasicLit)
		}
		return &ast.Field{
			Doc:     cloneCommentGroup(n.Doc),
			Names:   cloneIdentList(n.Names),
			Type:    cloneExpr(n.Type),
			Tag:     tag,
			Comment: cloneCommentGroup(n.Comment),
		}

	case *ast.FieldList:
		return cloneFieldList(n)

	case *ast.BadExpr:
		if n == nil {
			return nil
		}
		x := *n
		return &x

	case *ast.Ident:
		return cloneIdent(n)

	case *ast.BasicLit:
		if n == nil {
			return nil
		}
		x := *n
		return &x

	case *ast.Ellipsis:
		return &ast.Ellipsis{
			Ellipsis: n.Ellipsis,
			Elt:      cloneExpr(n.Elt),
		}

	case *ast.FuncLit:
		var typ *ast.FuncType
		if n.Type != nil {
			typ = Clone(n.Type).(*ast.FuncType)
		}
		return &ast.FuncLit{
			Type: typ,
			Body: cloneBlockStmt(n.Body),
		}

	case *ast.CompositeLit:
		return &ast.CompositeLit{
			Type:   cloneExpr(n.Type),
			Lbrace: n.Lbrace,
			Elts:   cloneExprList(n.Elts),
			Rbrace: n.Rbrace,
		}

	case *ast.ParenExpr:
		return &ast.ParenExpr{
			Lparen: n.Lparen,
			X:      cloneExpr(n.X),
			Rparen: n.Rparen,
		}

	case *ast.SelectorExpr:
		return &ast.SelectorExpr{
			X:   cloneExpr(n.X),
			Sel: cloneIdent(n.Sel),
		}

	case *ast.IndexExpr:
		return &ast.IndexExpr{
			X:      cloneExpr(n.X),
			Lbrack: n.Lbrack,
			Index:  cloneExpr(n.Index),
			Rbrack: n.Rbrack,
		}

	case *ast.SliceExpr:
		return &ast.SliceExpr{
			X:      cloneExpr(n.X),
			Lbrack: n.Lbrack,
			Low:    cloneExpr(n.Low),
			High:   cloneExpr(n.High),
			Max:    cloneExpr(n.Max),
			Slice3: n.Slice3,
			Rbrack: n.Rbrack,
		}

	case *ast.TypeParamDecl:
		return &ast.TypeParamDecl{
			Lbrack: n.Lbrack,
			Names:  cloneIdentList(n.Names),
			Rbrack: n.Rbrack,
		}

	case *ast.TypeArgExpr:
		return &ast.TypeArgExpr{
			X:      cloneExpr(n.X),
			Lbrack: n.Lbrack,
			Types:  cloneExprList(n.Types),
			Rbrack: n.Rbrack,
		}

	case *ast.TypeAssertExpr:
		return &ast.TypeAssertExpr{
			X:      cloneExpr(n.X),
			Lparen: n.Lparen,
			Type:   cloneExpr(n.Type),
			Rparen: n.Rparen,
		}

	case *ast.CallExpr:
		return &ast.CallExpr{
			Fun:      cloneExpr(n.Fun),
			Lparen:   n.Lparen,
			Args:     cloneExprList(n.Args),
			Ellipsis: n.Ellipsis,
			Rparen:   n.Rparen,
		}

	case *ast.StarExpr:
		return &ast.StarExpr{
			Star: n.Star,
			X:    cloneExpr(n.X),
		}

	case *ast.UnaryExpr:
		return &ast.UnaryExpr{
			OpPos: n.OpPos,
			Op:    n.Op,
			X:     cloneExpr(n.X),
		}

	case *ast.BinaryExpr:
		return &ast.BinaryExpr{
			X:     cloneExpr(n.X),
			OpPos: n.OpPos,
			Op:    n.Op,
			Y:     cloneExpr(n.Y),
		}

	case *ast.KeyValueExpr:
		return &ast.KeyValueExpr{
			Key:   cloneExpr(n.Key),
			Colon: n.Colon,
			Value: cloneExpr(n.Value),
		}

	case *ast.ArrayType:
		return &ast.ArrayType{
			Lbrack: n.Lbrack,
			Len:    cloneExpr(n.Len),
			Elt:    cloneExpr(n.Elt),
		}

	case *ast.StructType:
		return &ast.StructType{
			Struct:     n.Struct,
			Fields:     cloneFieldList(n.Fields),
			Incomplete: n.Incomplete,
		}

	case *ast.FuncType:
		return &ast.FuncType{
			Func:    n.Func,
			Params:  cloneFieldList(n.Params),
			Results: cloneFieldList(n.Results),
		}

	case *ast.InterfaceType:
		return &ast.InterfaceType{
			Interface:  n.Interface,
			Methods:    cloneFieldList(n.Methods),
			Incomplete: n.Incomplete,
		}

	case *ast.MapType:
		return &ast.MapType{
			Map:   n.Map,
			Key:   cloneExpr(n.Key),
			Value: cloneExpr(n.Value),
		}

	case *ast.ChanType:
		return &ast.ChanType{
			Begin: n.Begin,
			Arrow: n.Arrow,
			Dir:   n.Dir,
			Value: cloneExpr(n.Value),
		}

	case *ast.BadStmt:
		if n == nil {
			return nil
		}
		x := *n
		return &x

	case *ast.DeclStmt:
		return &ast.DeclStmt{
			Decl: cloneDecl(n.Decl),
		}

	case *ast.EmptyStmt:
		return &ast.EmptyStmt{
			Semicolon: n.Semicolon,
			Implicit:  n.Implicit,
		}

	case *ast.LabeledStmt:
		return &ast.LabeledStmt{
			Label: cloneIdent(n.Label),
			Colon: n.Colon,
			Stmt:  cloneStmt(n.Stmt),
		}

	case *ast.ExprStmt:
		return &ast.ExprStmt{
			X: cloneExpr(n.X),
		}

	case *ast.SendStmt:
		return &ast.SendStmt{
			Chan:  cloneExpr(n.Chan),
			Arrow: n.Arrow,
			Value: cloneExpr(n.Value),
		}

	case *ast.IncDecStmt:
		return &ast.IncDecStmt{
			X:      cloneExpr(n.X),
			TokPos: n.TokPos,
			Tok:    n.Tok,
		}

	case *ast.AssignStmt:
		return &ast.AssignStmt{
			Lhs:    cloneExprList(n.Lhs),
			TokPos: n.TokPos,
			Tok:    n.Tok,
			Rhs:    cloneExprList(n.Rhs),
		}

	case *ast.GoStmt:
		var call *ast.CallExpr
		if n.Call != nil {
			call = Clone(n.Call).(*ast.CallExpr)
		}
		return &ast.GoStmt{
			Go:   n.Go,
			Call: call,
		}

	case *ast.DeferStmt:
		var call *ast.CallExpr
		if n.Call != nil {
			call = Clone(n.Call).(*ast.CallExpr)
		}
		return &ast.DeferStmt{
			Defer: n.Defer,
			Call:  call,
		}

	case *ast.ReturnStmt:
		return &ast.ReturnStmt{
			Return:  n.Return,
			Results: cloneExprList(n.Results),
		}

	case *ast.BranchStmt:
		return &ast.BranchStmt{
			TokPos: n.TokPos,
			Tok:    n.Tok,
			Label:  cloneIdent(n.Label),
		}

	case *ast.BlockStmt:
		return cloneBlockStmt(n)

	case *ast.IfStmt:
		return &ast.IfStmt{
			If:   n.If,
			Init: cloneStmt(n.Init),
			Cond: cloneExpr(n.Cond),
			Body: cloneBlockStmt(n.Body),
			Else: cloneStmt(n.Else),
		}

	case *ast.CaseClause:
		return &ast.CaseClause{
			Case:  n.Case,
			List:  cloneExprList(n.List),
			Colon: n.Colon,
			Body:  cloneStmtList(n.Body),
		}

	case *ast.SwitchStmt:
		return &ast.SwitchStmt{
			Switch: n.Switch,
			Init:   cloneStmt(n.Init),
			Tag:    cloneExpr(n.Tag),
			Body:   cloneBlockStmt(n.Body),
		}

	case *ast.TypeSwitchStmt:
		return &ast.TypeSwitchStmt{
			Switch: n.Switch,
			Init:   cloneStmt(n.Init),
			Assign: cloneStmt(n.Assign),
			Body:   cloneBlockStmt(n.Body),
		}

	case *ast.CommClause:
		return &ast.CommClause{
			Case:  n.Case,
			Comm:  cloneStmt(n.Comm),
			Colon: n.Colon,
			Body:  cloneStmtList(n.Body),
		}

	case *ast.SelectStmt:
		return &ast.SelectStmt{
			Select: n.Select,
			Body:   cloneBlockStmt(n.Body),
		}

	case *ast.ForStmt:
		return &ast.ForStmt{
			For:  n.For,
			Init: cloneStmt(n.Init),
			Cond: cloneExpr(n.Cond),
			Post: cloneStmt(n.Post),
			Body: cloneBlockStmt(n.Body),
		}

	case *ast.RangeStmt:
		return &ast.RangeStmt{
			For:    n.For,
			Key:    cloneExpr(n.Key),
			Value:  cloneExpr(n.Value),
			TokPos: n.TokPos,
			Tok:    n.Tok,
			X:      cloneExpr(n.X),
			Body:   cloneBlockStmt(n.Body),
		}

	case *ast.ImportSpec:
		var path *ast.BasicLit
		if n.Path != nil {
			path = Clone(n.Path).(*ast.BasicLit)
		}
		return &ast.ImportSpec{
			Doc:     cloneCommentGroup(n.Doc),
			Name:    cloneIdent(n.Name),
			Path:    path,
			Comment: cloneCommentGroup(n.Comment),
			EndPos:  n.EndPos,
		}

	case *ast.ValueSpec:
		return &ast.ValueSpec{
			Doc:     cloneCommentGroup(n.Doc),
			Names:   cloneIdentList(n.Names),
			Type:    cloneExpr(n.Type),
			Values:  cloneExprList(n.Values),
			Comment: cloneCommentGroup(n.Comment),
		}

	case *ast.TypeSpec:
		return &ast.TypeSpec{
			Doc:     cloneCommentGroup(n.Doc),
			Name:    cloneIdent(n.Name),
			Assign:  n.Assign,
			Type:    cloneExpr(n.Type),
			Comment: cloneCommentGroup(n.Comment),
		}

	case *ast.BadDecl:
		if n == nil {
			return nil
		}
		x := *n
		return &x

	case *ast.GenDecl:
		var specs []ast.Spec
		if n.Specs != nil {
			specs = make([]ast.Spec, len(n.Specs))
			for i, spec := range n.Specs {
				if spec != nil {
					specs[i] = Clone(spec).(ast.Spec)
				}
			}
		}
		return &ast.GenDecl{
			Doc:    cloneCommentGroup(n.Doc),
			TokPos: n.TokPos,
			Tok:    n.Tok,
			Lparen: n.Lparen,
			Specs:  specs,
			Rparen: n.Rparen,
		}

	case *ast.FuncDecl:
		var typ *ast.FuncType
		if n.Type != nil {
			typ = Clone(n.Type).(*ast.FuncType)
		}
		return &ast.FuncDecl{
			Doc:  cloneCommentGroup(n.Doc),
			Recv: cloneFieldList(n.Recv),
			Name: cloneIdent(n.Name),
			Type: typ,
			Body: cloneBlockStmt(n.Body),
		}

	case *ast.File:
		var imports []*ast.ImportSpec
		if n.Imports != nil {
			imports = make([]*ast.ImportSpec, len(n.Imports))
			for i, imp := range n.Imports {
				if imp != nil {
					imports[i] = Clone(imp).(*ast.ImportSpec)
				}
			}
		}
		var comments []*ast.CommentGroup
		if n.Comments != nil {
			comments = make([]*ast.CommentGroup, len(n.Comments))
			for i, cg := range n.Comments {
				if cg != nil {
					comments[i] = Clone(cg).(*ast.CommentGroup)
				}
			}
		}
		return &ast.File{
			Doc:        cloneCommentGroup(n.Doc),
			Package:    n.Package,
			Name:       cloneIdent(n.Name),
			Decls:      cloneDeclList(n.Decls),
			Scope:      cloneScope(n.Scope),
			Imports:    imports,
			Unresolved: cloneIdentList(n.Unresolved),
			Comments:   comments,
		}

	case *ast.Package:
		var imports map[string]*ast.Object
		if n.Imports != nil {
			imports = map[string]*ast.Object{}
			for id, obj := range n.Imports {
				imports[id] = cloneObject(obj)
			}
		}
		var files map[string]*ast.File
		if n.Files != nil {
			files = map[string]*ast.File{}
			for name, file := range n.Files {
				if file != nil {
					files[name] = Clone(file).(*ast.File)
				} else {
					files[name] = nil
				}
			}
		}
		return &ast.Package{
			Name:    n.Name,
			Scope:   cloneScope(n.Scope),
			Imports: imports,
			Files:   files,
		}

	default:
		panic(fmt.Sprintf("astclone.Clone: unexpected node type %T", n))
	}
}

// Helper functions for common nodes
func cloneIdentList(list []*ast.Ident) []*ast.Ident {
	if list == nil {
		return nil
	}
	newList := make([]*ast.Ident, len(list))
	for i, x := range list {
		newList[i] = cloneIdent(x)
	}
	return newList
}

func cloneExprList(list []ast.Expr) []ast.Expr {
	if list == nil {
		return nil
	}
	newList := make([]ast.Expr, len(list))
	for i, x := range list {
		newList[i] = cloneExpr(x)
	}
	return newList
}

func cloneIdent(n *ast.Ident) *ast.Ident {
	if n == nil {
		return nil
	}
	return &ast.Ident{
		NamePos: n.NamePos,
		Name:    n.Name,
		Obj:     cloneObject(n.Obj),
	}
}

func cloneExpr(n ast.Expr) ast.Expr {
	if n != nil {
		return Clone(n).(ast.Expr)
	}
	return nil
}

func cloneFieldList(n *ast.FieldList) *ast.FieldList {
	if n == nil {
		return nil
	}
	list := make([]*ast.Field, len(n.List))
	for i, c := range n.List {
		if c != nil {
			list[i] = Clone(c).(*ast.Field)
		}
	}
	return &ast.FieldList{
		List: list,
	}
}

func cloneDecl(n ast.Decl) ast.Decl {
	if n == nil {
		return nil
	}
	return Clone(n).(ast.Decl)
}

func cloneDeclList(list []ast.Decl) []ast.Decl {
	if list == nil {
		return nil
	}
	newList := make([]ast.Decl, len(list))
	for i, x := range list {
		newList[i] = cloneDecl(x)
	}
	return newList
}

func cloneStmt(n ast.Stmt) ast.Stmt {
	if n == nil {
		return nil
	}
	return Clone(n).(ast.Stmt)
}

func cloneStmtList(list []ast.Stmt) []ast.Stmt {
	if list == nil {
		return nil
	}
	newList := make([]ast.Stmt, len(list))
	for i, x := range list {
		newList[i] = cloneStmt(x)
	}
	return newList
}

func cloneBlockStmt(n *ast.BlockStmt) *ast.BlockStmt {
	if n == nil {
		return nil
	}
	return &ast.BlockStmt{
		Lbrace: n.Lbrace,
		List:   cloneStmtList(n.List),
		Rbrace: n.Rbrace,
	}
}

func cloneCommentGroup(n *ast.CommentGroup) *ast.CommentGroup {
	if n == nil {
		return nil
	}
	list := make([]*ast.Comment, len(n.List))
	for i, c := range n.List {
		if c != nil {
			list[i] = Clone(c).(*ast.Comment)
		}
	}
	return &ast.CommentGroup{
		List: list,
	}
}

// Functions for cloning things which do not implement ast.Node.

func cloneObject(o *ast.Object) *ast.Object {
	// TODO(albrow): *ast.Objects tend to be recursive data structures, so a simple
	// recursive approach here will result in a stack overflow. Can we
	// intelligently clone *ast.Objects while avoiding a stack overflow?
	return o
	// if o == nil {
	// 	return nil
	// }
	// var decl interface{}
	// if o.Decl != nil {
	// 	switch d := o.Decl.(type) {
	// 	case ast.Node:
	// 		decl = Clone(d)
	// 	default:
	// 		panic(fmt.Sprintf("astclone.cloneObject: unexpected Decl type %T", o.Decl))
	// 	}
	// }
	// var data interface{}
	// if o.Data != nil {
	// 	switch d := o.Data.(type) {
	// 	case *ast.Scope:
	// 		data = cloneScope(d)
	// 	default:
	// 		panic(fmt.Sprintf("astclone.cloneObject: unexpected Data type %T", o.Data))
	// 	}
	// }
	// var typ interface{}
	// if o.Type != nil {
	// 	switch t := o.Type.(type) {
	// 	case *ast.Scope:
	// 		typ = cloneScope(t)
	// 	default:
	// 		panic(fmt.Sprintf("astclone.cloneObject: unexpected Type type %T", o.Data))
	// 	}
	// }
	// return &ast.Object{
	// 	Kind: o.Kind,
	// 	Name: o.Name,
	// 	Decl: decl,
	// 	Data: data,
	// 	Type: typ,
	// }
}

func cloneScope(s *ast.Scope) *ast.Scope {
	return s
	// if s == nil {
	// 	return nil
	// }
	// var objects map[string]*ast.Object
	// if s.Objects != nil {
	// 	objects = map[string]*ast.Object{}
	// 	for name, obj := range s.Objects {
	// 		objects[name] = cloneObject(obj)
	// 	}
	// }
	// return &ast.Scope{
	// 	Outer:   cloneScope(s.Outer),
	// 	Objects: objects,
	// }
}
