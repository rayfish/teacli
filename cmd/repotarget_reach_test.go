package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// resolvers are the functions that turn arguments into a repository. A command
// that takes a repository has to reach one of them.
var resolvers = map[string]bool{"repoTarget": true, "repoTargetByArg": true}

// callGraph maps each function in the package to the functions it calls.
func callGraph(t *testing.T) map[string][]string {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatalf("parsing the package: %v", err)
	}

	graph := map[string][]string{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if ident, ok := call.Fun.(*ast.Ident); ok {
						graph[fn.Name.Name] = append(graph[fn.Name.Name], ident.Name)
					}
					return true
				})
			}
		}
	}
	return graph
}

// reaches reports whether from can call any of the resolvers, directly or
// through other functions in the package.
func reaches(graph map[string][]string, from string) bool {
	seen := map[string]bool{}
	var walk func(string) bool
	walk = func(name string) bool {
		if resolvers[name] {
			return true
		}
		if seen[name] {
			return false
		}
		seen[name] = true
		for _, callee := range graph[name] {
			if walk(callee) {
				return true
			}
		}
		return false
	}
	return walk(from)
}

// runEName reports the name of the function behind a command's RunE.
func runEName(cmd *cobra.Command) string {
	if cmd.RunE == nil {
		return ""
	}
	full := runtime.FuncForPC(reflect.ValueOf(cmd.RunE).Pointer()).Name()
	return full[strings.LastIndex(full, ".")+1:]
}

func TestEveryRepositoryCommandResolvesItsRepository(t *testing.T) {
	graph := callGraph(t)

	walk(RootCmd, func(cmd *cobra.Command) {
		tokens := positionals(cmd.Use)
		if len(tokens) == 0 || !strings.Contains(tokens[0], "<owner>/") {
			return
		}
		path := cmd.CommandPath()
		if _, skip := noInference[path]; skip {
			return
		}

		name := runEName(cmd)
		if name == "" {
			t.Errorf("%s: no RunE", path)
			return
		}
		if !reaches(graph, name) {
			t.Errorf("%s: %s never reaches repoTarget, so its repository argument "+
				"is optional in the help text but not in the code", path, name)
		}
	})
}
