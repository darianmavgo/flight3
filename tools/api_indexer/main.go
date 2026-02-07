package main

import (
	"database/sql"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := "flight_api.db"
	os.Remove(dbPath) // Start fresh

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTable := `
	CREATE TABLE public_functions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_package TEXT,
		package_name TEXT,
		function_name TEXT,
		signature TEXT,
		file_path TEXT,
		line_number INTEGER
	);`

	if _, err := db.Exec(createTable); err != nil {
		log.Fatal(err)
	}

	sources := map[string]string{
		"Flight3":    ".", // Current dir
		"Banquet":    "../banquet",
		"Mksqlite":   "../mksqlite",
		"Pocketbase": "/Users/darianhickman/go/pkg/mod/github.com/pocketbase/pocketbase@v0.36.1",
	}

	stmt, err := db.Prepare("INSERT INTO public_functions(source_package, package_name, function_name, signature, file_path, line_number) VALUES(?, ?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for sourceName, sourcePath := range sources {
		fmt.Printf("Scanning %s at %s...\n", sourceName, sourcePath)
		if err := scanSource(stmt, sourceName, sourcePath); err != nil {
			log.Printf("Error scanning %s: %v\n", sourceName, err)
		}
	}
	fmt.Println("Done.")
}

func scanSource(stmt *sql.Stmt, sourceName, rootPath string) error {
	return filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}

		// Skip hidden directories and vendor (optional but good practice)
		if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			return filepath.SkipDir
		}
		if d.Name() == "vendor" {
			return filepath.SkipDir
		}

		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, path, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if err != nil {
			// Don't fail the whole walk on one parse error
			log.Printf("Parse error in %s: %v", path, err)
			return nil
		}

		for _, pkg := range pkgs {
			for fileName, file := range pkg.Files {
				for _, decl := range file.Decls {
					if fn, ok := decl.(*ast.FuncDecl); ok {
						if fn.Name.IsExported() {
							funcName := fn.Name.Name
							// Handle methods
							if fn.Recv != nil && len(fn.Recv.List) > 0 {
								recvType := fn.Recv.List[0].Type
								typeName := ""
								switch t := recvType.(type) {
								case *ast.StarExpr:
									if ident, ok := t.X.(*ast.Ident); ok {
										typeName = ident.Name
									}
								case *ast.Ident:
									typeName = t.Name
								}
								if typeName != "" {
									// Only record methods on exported types if we want strictly "public" API
									// But usually if the method is exported, it counts.
									funcName = typeName + "." + funcName
								}
							}

							// Simple signature construction (can be improved)
							sig := "func " + funcName + "()" // Placeholder, constructing full sig is complex using AST

							// Extract simplistic signature from source if possible, or leave blank.
							// For now, let's just store the name.
							// Actually, let's look at the AST type to rebuild signature? Too complex for this quick script.
							// We will just store "func ..."

							// Get line number
							pos := fset.Position(fn.Pos())

							_, err := stmt.Exec(sourceName, pkg.Name, funcName, sig, fileName, pos.Line)
							if err != nil {
								log.Println("Insert error:", err)
							}
						}
					}
				}
			}
		}

		return nil
	})
}
