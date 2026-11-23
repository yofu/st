// Command commandcatalog extracts the public names and usage from ST's native dispatch.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func literal(e ast.Expr) string {
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		v, _ := strconv.Unquote(lit.Value)
		return v
	}
	return ""
}

func catalog(dir string) string {
	commands := map[string]string{}
	for _, src := range []struct{ file, function, key, prefix string }{
		{"st_exmode.go", "exCommand", "cname", ":"}, {"st_fig2mode.go", "Fig2Keyword", "key", "'"}, {"st_fig2_extra.go", "extraFig2Keyword", "key", "'"},
	} {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, src.file), nil, 0)
		if err != nil {
			panic(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != src.function {
				continue
			}
			for _, stmt := range fn.Body.List {
				sw, ok := stmt.(*ast.SwitchStmt)
				if !ok {
					continue
				}
				key, ok := sw.Tag.(*ast.Ident)
				if !ok || key.Name != src.key {
					continue
				}
				for _, branch := range sw.Body.List {
					c := branch.(*ast.CaseClause)
					for _, e := range c.List {
						name := literal(e)
						if name == "" {
							continue
						}
						name = src.prefix + name
						usage := commands[name]
						ast.Inspect(c, func(n ast.Node) bool {
							call, ok := n.(*ast.CallExpr)
							if !ok || len(call.Args) != 1 {
								return true
							}
							id, ok := call.Fun.(*ast.Ident)
							if ok && id.Name == "Usage" {
								s := literal(call.Args[0])
								if strings.HasPrefix(s, name+" ") {
									usage = strings.Split(s, "\n")[0]
								}
							}
							return true
						})
						commands[name] = usage
					}
				}
			}
		}
	}
	var rows []string
	for name, usage := range commands {
		rows = append(rows, strings.TrimSpace(name+"\t"+usage))
	}
	sort.Strings(rows)
	return strings.Join(rows, "\n") + "\n"
}
func main() {
	data := catalog(".")
	if err := os.WriteFile("native_commands.txt", []byte(data), 0644); err != nil {
		panic(err)
	}
	fmt.Printf("indexed %d native commands\n", strings.Count(data, "\n"))
}
