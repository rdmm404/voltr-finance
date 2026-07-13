package architecture

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestApplicationPackagesHaveInwardDependencies(t *testing.T) {
	root := projectRoot(t)
	forbidden := []string{
		"rdmm404/voltr-finance/internal/api",
		"rdmm404/voltr-finance/internal/cli",
		"rdmm404/voltr-finance/internal/database",
		"rdmm404/voltr-finance/internal/httpapi",
		"rdmm404/voltr-finance/internal/postgres",
		"rdmm404/voltr-finance/internal/restclient",
		"rdmm404/voltr-finance/internal/server",
		"github.com/jackc/pgx",
	}
	err := filepath.WalkDir(filepath.Join(root, "internal", "app"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, declaration := range file.Decls {
			importDeclaration, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, specification := range importDeclaration.Specs {
				importPath, _ := strconv.Unquote(specification.(*ast.ImportSpec).Path.Value)
				for _, prefix := range forbidden {
					if strings.HasPrefix(importPath, prefix) {
						t.Errorf("%s imports forbidden package %s", file.Name.Name, importPath)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCLIProductionGraphHasNoServerOrPostgresPackages(t *testing.T) {
	root := projectRoot(t)
	command := exec.Command("go", "list", "-deps", "./cmd/cli")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{
		"rdmm404/voltr-finance/internal/app",
		"rdmm404/voltr-finance/internal/database",
		"rdmm404/voltr-finance/internal/httpapi",
		"rdmm404/voltr-finance/internal/postgres",
		"rdmm404/voltr-finance/internal/server",
		"github.com/jackc/pgx",
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		dependency := scanner.Text()
		for _, prefix := range forbidden {
			if strings.HasPrefix(dependency, prefix) {
				t.Errorf("CLI imports forbidden production dependency %s", dependency)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve architecture test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
