package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/albrow/fo/format"
	"github.com/albrow/fo/parser"
	"github.com/albrow/fo/token"
	"github.com/albrow/fo/transform"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "Fo"
	app.Usage = "An experimental language which adds functional programming features to Go."
	app.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "run a single .fo file",
			Action: run,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	if !c.Args().Present() || len(c.Args().Tail()) != 0 {
		return errors.New("run expects exactly one argument: the name of a Fo file to run")
	}
	f, err := os.Open(c.Args().First())
	if err != nil {
		return fmt.Errorf("could not open file: %s", err)
	}
	if !strings.HasSuffix(f.Name(), ".fo") {
		return fmt.Errorf("%s is not a Fo file (expected '.fo' extension)", f.Name())
	}
	fset := token.NewFileSet()
	nodes, err := parser.ParseFile(fset, f.Name(), f, 0)
	if err != nil {
		return err
	}
	transformed, err := transform.File(fset, nodes)
	if err != nil {
		return err
	}
	outputName := strings.TrimSuffix(f.Name(), ".fo") + ".go"
	output, err := os.Create(outputName)
	if err != nil {
		return err
	}
	if err := format.Node(output, fset, transformed); err != nil {
		return err
	}
	cmd := exec.Command("go", "run", outputName)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
