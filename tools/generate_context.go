package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var buf bytes.Buffer

	buf.WriteString("--- \n")
	buf.WriteString("context_type: kernel_source_dump \n")
	buf.WriteString("project: manglekit_v2 \n")
	buf.WriteString("language: go \n")
	buf.WriteString(fmt.Sprintf("last_updated: %s \n", time.Now().Format(time.RFC3339)))
	buf.WriteString("scan_mode: logic_focused \n")
	buf.WriteString("--- \n\n")

	buf.WriteString("# Manglekit v2 (Sovereign Logic Kernel) - Complete Context Dump\n\n")
	buf.WriteString("This file contains the canonical, auto-generated structural abstraction of the Manglekit v2 embedded kernel. It serves as the absolute single source of truth for the codebase.\n\n")

	buf.WriteString("## 1. THE COMPLETE FILE MAP\n\n")
	buf.WriteString("```text\n")

	// Print directory structure
	filepath.WalkDir("./", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == "." {
			return nil
		}
		if strings.HasPrefix(path, ".git") || strings.HasPrefix(path, "vendor") {
			return filepath.SkipDir
		}

		depth := strings.Count(path, string(os.PathSeparator))
		prefix := strings.Repeat("│   ", depth)

		if d.IsDir() {
			buf.WriteString(fmt.Sprintf("%s├── %s/\n", prefix, d.Name()))
		} else if strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), ".md") {
			buf.WriteString(fmt.Sprintf("%s├── %s\n", prefix, d.Name()))
		}
		return nil
	})
	buf.WriteString("```\n\n")

	buf.WriteString("## 2. COMPONENTS (The Hexagonal OODA Loop)\n\n")

	// Parse relevant .go files across the project
	fset := token.NewFileSet()

	targetDirs := []string{"core", "internal", "adapters", "providers", "sdk", "."}

	for _, dir := range targetDirs {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			// Don't dive deeply into dot dir itself, just top level
			if dir == "." && strings.Contains(path, "/") && path != "." {
				return nil
			}

			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return nil
			}

			pkgName := node.Name.Name

			// Ignore simple tools or test binaries
			if pkgName == "main" && !strings.HasSuffix(path, "manglekit.go") {
				return nil
			}

			buf.WriteString(fmt.Sprintf("### Component: `%s` (%s)\n", path, pkgName))

			var constants []string
			var types []string
			var funcs []string

			// Helper to get type string from ast.Expr
			var getTypeString func(expr ast.Expr) string
			getTypeString = func(expr ast.Expr) string {
				switch e := expr.(type) {
				case *ast.Ident:
					return e.Name
				case *ast.SelectorExpr:
					return getTypeString(e.X) + "." + e.Sel.Name
				case *ast.StarExpr:
					return "*" + getTypeString(e.X)
				case *ast.ArrayType:
					return "[]" + getTypeString(e.Elt)
				case *ast.MapType:
					return "map[" + getTypeString(e.Key) + "]" + getTypeString(e.Value)
				case *ast.InterfaceType:
					return "interface{}"
				case *ast.StructType:
					return "struct{...}"
				case *ast.FuncType:
					return "func(...)"
				case *ast.Ellipsis:
					return "..." + getTypeString(e.Elt)
				default:
					return fmt.Sprintf("%T", expr)
				}
			}

			// Helper to format field list (params or returns)
			formatFieldList := func(fl *ast.FieldList) string {
				if fl == nil {
					return ""
				}
				var parts []string
				for _, f := range fl.List {
					typ := getTypeString(f.Type)
					if len(f.Names) > 0 {
						var names []string
						for _, n := range f.Names {
							names = append(names, n.Name)
						}
						parts = append(parts, strings.Join(names, ", ")+" "+typ)
					} else {
						parts = append(parts, typ)
					}
				}
				return strings.Join(parts, ", ")
			}

			ast.Inspect(node, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.GenDecl:
					if x.Tok == token.CONST {
						for _, spec := range x.Specs {
							vs := spec.(*ast.ValueSpec)
							if len(vs.Names) > 0 && vs.Names[0].IsExported() {
								val := ""
								if len(vs.Values) > 0 {
									if lit, ok := vs.Values[0].(*ast.BasicLit); ok {
										val = " = " + lit.Value
									} else if ident, ok := vs.Values[0].(*ast.Ident); ok {
										val = " = " + ident.Name
									}
								}
								constants = append(constants, fmt.Sprintf("- `const %s%s`", vs.Names[0].Name, val))
							}
						}
					}
					if x.Tok == token.TYPE {
						for _, spec := range x.Specs {
							typeSpec := spec.(*ast.TypeSpec)
							typeName := typeSpec.Name.Name

							if !typeSpec.Name.IsExported() {
								continue
							}

							switch t := typeSpec.Type.(type) {
							case *ast.StructType:
								fields := []string{}
								if t.Fields != nil {
									for _, f := range t.Fields.List {
										if len(f.Names) == 0 {
											continue // embedded struct
										}
										if f.Names[0].IsExported() {
											fields = append(fields, f.Names[0].Name+" "+getTypeString(f.Type))
										}
									}
								}
								types = append(types, fmt.Sprintf("- **%s** (struct): [%s]", typeName, strings.Join(fields, ", ")))
							case *ast.InterfaceType:
								methods := []string{}
								if t.Methods != nil {
									for _, m := range t.Methods.List {
										if len(m.Names) == 0 {
											methods = append(methods, getTypeString(m.Type)) // embedded interface
											continue
										}
										if m.Names[0].IsExported() {
											if ft, ok := m.Type.(*ast.FuncType); ok {
												params := formatFieldList(ft.Params)
												returns := formatFieldList(ft.Results)
												if returns != "" {
													returns = " (" + returns + ")"
												}
												methods = append(methods, fmt.Sprintf("%s(%s)%s", m.Names[0].Name, params, returns))
											} else {
												methods = append(methods, m.Names[0].Name)
											}
										}
									}
								}
								types = append(types, fmt.Sprintf("- **%s** (interface):\n  - %s", typeName, strings.Join(methods, "\n  - ")))
							case *ast.Ident: // type aliases
								types = append(types, fmt.Sprintf("- **%s** (alias to %s)", typeName, t.Name))
							}
						}
					}
				case *ast.FuncDecl:
					if !x.Name.IsExported() {
						return true
					}

					recv := ""
					if x.Recv != nil && len(x.Recv.List) > 0 {
						switch rt := x.Recv.List[0].Type.(type) {
						case *ast.StarExpr:
							if ident, ok := rt.X.(*ast.Ident); ok {
								if len(x.Recv.List[0].Names) > 0 {
									recv = fmt.Sprintf("(%s *%s)", x.Recv.List[0].Names[0].Name, ident.Name)
								} else {
									recv = fmt.Sprintf("(*%s)", ident.Name)
								}
							}
						case *ast.Ident:
							if len(x.Recv.List[0].Names) > 0 {
								recv = fmt.Sprintf("(%s %s)", x.Recv.List[0].Names[0].Name, rt.Name)
							} else {
								recv = fmt.Sprintf("(%s)", rt.Name)
							}
						}
					}

					params := formatFieldList(x.Type.Params)
					returns := formatFieldList(x.Type.Results)
					if returns != "" {
						returns = " (" + returns + ")"
					}

					prefix := ""
					if recv != "" {
						prefix = fmt.Sprintf("func %s %s(%s)%s", recv, x.Name.Name, params, returns)
					} else {
						prefix = fmt.Sprintf("func %s(%s)%s", x.Name.Name, params, returns)
					}

					doc := ""
					if x.Doc != nil {
						doc = strings.TrimSpace(strings.ReplaceAll(x.Doc.Text(), "\n", " "))
						if len(doc) > 120 {
							doc = doc[:117] + "..."
						}
					}

					if doc != "" {
						funcs = append(funcs, fmt.Sprintf("- `%s` - %s", prefix, doc))
					} else {
						funcs = append(funcs, fmt.Sprintf("- `%s`", prefix))
					}
				}
				return true
			})

			if len(constants) > 0 {
				buf.WriteString("**Constants:**\n")
				for _, c := range constants {
					buf.WriteString(c + "\n")
				}
				buf.WriteString("\n")
			}

			if len(types) > 0 {
				buf.WriteString("**Core Types:**\n")
				for _, t := range types {
					buf.WriteString(t + "\n")
				}
				buf.WriteString("\n")
			}

			if len(funcs) > 0 {
				buf.WriteString("**Key Operations:**\n")
				for _, f := range funcs {
					buf.WriteString(f + "\n")
				}
				buf.WriteString("\n")
			}

			if len(constants) == 0 && len(types) == 0 && len(funcs) == 0 {
				buf.WriteString("*(No exported APIs)*\n\n")
			} else if len(funcs) == 0 && len(types) > 0 {
				// We already wrote the newline after Core Types
			} else {
				// buf.WriteString("\n")
			}

			return nil
		})
	}

	os.WriteFile("docs/CONTEXT.md", buf.Bytes(), 0644)
	fmt.Println("Regenerated docs/CONTEXT.md successfully.")
}
