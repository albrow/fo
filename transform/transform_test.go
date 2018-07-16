package transform

import (
	"bytes"
	goast "go/ast"
	goparser "go/parser"
	gotoken "go/token"
	gotypes "go/types"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/albrow/fo/ast"
	"github.com/albrow/fo/format"
	"github.com/albrow/fo/importer"
	"github.com/albrow/fo/parser"
	"github.com/albrow/fo/token"
	"github.com/albrow/fo/types"
	"github.com/aryann/difflib"
)

var testCases = []struct {
	srcFilename      string
	expectedFilename string
}{
	{
		srcFilename:      "testdata/value_spec/main.src",
		expectedFilename: "testdata/value_spec/main.expected",
	},
	{
		srcFilename:      "testdata/struct_fields/main.src",
		expectedFilename: "testdata/struct_fields/main.expected",
	},
	{
		srcFilename:      "testdata/call_expr/main.src",
		expectedFilename: "testdata/call_expr/main.expected",
	},
	{
		srcFilename:      "testdata/binary_expr/main.src",
		expectedFilename: "testdata/binary_expr/main.expected",
	},
}

func TestTransform(t *testing.T) {
	for _, testCase := range testCases {
		testTransformFile(t, testCase.srcFilename, testCase.expectedFilename)
	}
}

func testTransformFile(t *testing.T, srcFilename string, expectedFilename string) {
	t.Helper()
	srcFile, err := os.Open(srcFilename)
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	orig, err := parser.ParseFile(fset, srcFile.Name(), srcFile, 0)
	if err != nil {
		t.Errorf("%s: Fo parser returned error: %s", srcFilename, err.Error())
		return
	}
	conf := types.Config{}
	conf.Importer = importer.Default()
	info := &types.Info{
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Uses:       map[*ast.Ident]types.Object{},
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
	}
	pkg, err := conf.Check(srcFile.Name(), fset, []*ast.File{orig}, info)
	if err != nil {
		t.Errorf("%s: Fo type-checker returned error: %s", srcFilename, err.Error())
		return
	}
	trans := &Transformer{
		Fset: fset,
		Pkg:  pkg,
		Info: info,
	}
	transformed, err := trans.File(orig)
	if err != nil {
		t.Errorf("%s: Fo transformer returned error: %s", srcFilename, err.Error())
		return
	}
	output := bytes.NewBuffer(nil)
	if err := format.Node(output, fset, transformed); err != nil {
		t.Errorf("%s: Fo formatter returned error: %s", srcFilename, err.Error())
		return
	}

	expectedFile, err := os.Open(expectedFilename)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := ioutil.ReadAll(expectedFile)
	if err != nil {
		t.Fatal(err)
	}

	if output.String() != string(expected) {
		diff := difflib.Diff(strings.Split(string(expected), "\n"), strings.Split(output.String(), "\n"))
		diffStrings := ""
		for _, d := range diff {
			diffStrings += d.String() + "\n"
		}
		t.Errorf(
			"%s: transformer output did not match expected\n\n%s",
			srcFilename,
			diffStrings,
		)
		return
	}

	typeCheckPureGo(t, expectedFilename, output)
}

func typeCheckPureGo(t *testing.T, filename string, src *bytes.Buffer) {
	fset := gotoken.NewFileSet()
	parsed, err := goparser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Errorf("%s: Go parser returned error: %s", filename, err.Error())
		return
	}
	conf := gotypes.Config{}
	info := &gotypes.Info{}
	_, err = conf.Check(filename, fset, []*goast.File{parsed}, info)
	if err != nil {
		t.Errorf("%s: Go type-checker returned error: %s", filename, err.Error())
		return
	}
}
