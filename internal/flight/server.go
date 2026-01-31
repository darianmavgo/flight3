package flight

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"strings"

	"github.com/darianmavgo/banquet"
	"github.com/darianmavgo/banquet/sqlite"
	"github.com/darianmavgo/sqliter/sqliter"
	"github.com/pocketbase/pocketbase/core"
	_ "modernc.org/sqlite"
)

// ServeFromCache opens cached SQLite DB and serves query results
func ServeFromCache(cachePath string, b *banquet.Banquet, tw *sqliter.TableWriter, tpl *template.Template, e *core.RequestEvent) error {
	log.Printf("[SERVER] Serving from cache: %s", cachePath)

	// Open SQLite database
	db, err := sql.Open("sqlite", cachePath)
	if err != nil {
		return NewBanquetError(err, "Failed to open cache database", 500, b, "", cachePath)
	}
	defer db.Close()

	// Build SQL query from banquet fields
	// Infer table name if not provided
	if b.Table == "" {
		b.Table = sqlite.InferTable(b)
	}

	query := sqlite.Compose(b)
	log.Printf("[SERVER] Executing query: %s", query)

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return NewBanquetError(err, "Query execution failed", 400, b, query, cachePath)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return NewBanquetError(err, "Failed to get columns", 500, b, query, cachePath)
	}

	// Start HTML table with debug info
	title := b.DataSetPath
	if b.Table != "" {
		title = b.Table
	}

	// Create one-liner Banquet debug info
	banquetDebug := fmt.Sprintf("Banquet{Scheme:%q Host:%q Path:%q DataSetPath:%q Table:%q Select:%v Where:%q GroupBy:%q Having:%q OrderBy:%q SortDirection:%q Limit:%q Offset:%q} DB:%q",
		b.Scheme, b.Host, b.Path, b.DataSetPath, b.Table, b.Select, b.Where, b.GroupBy, b.Having, b.OrderBy, b.SortDirection, b.Limit, b.Offset, cachePath)

	tw.StartHTMLTableWithDebug(e.Response, columns, title, banquetDebug, query)

	// Determine if this looks like a directory listing or a table listing
	isDirListing := false
	isSqliteMaster := b.Table == "sqlite_master"
	nameIdx := -1
	pathIdx := -1
	typeIdx := -1
	for i, col := range columns {
		if col == "name" {
			nameIdx = i
		}
		if col == "path" {
			pathIdx = i
		}
		if col == "is_dir" {
			isDirListing = true
		}
		if col == "type" {
			typeIdx = i
		}
	}

	// Write rows
	rowIndex := 0
	for rows.Next() {
		// Create slice to hold column values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan row
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("[SERVER] Error scanning row: %v", err)
			continue
		}

		// Convert to strings
		cells := make([]string, len(columns))

		for i, val := range values {
			rawVal := ""
			if val != nil {
				rawVal = fmt.Sprintf("%v", val)
			}

			// Determine if this row should be a link
			shouldLink := false
			link := rawVal
			icon := ""

			if isDirListing && (i == nameIdx || i == pathIdx) && rawVal != "" && rawVal != "." && rawVal != ".." {
				shouldLink = true

				// Determine if it's a directory row
				isDirRow := false
				// Look for 'is_dir' column in current row values
				for j, col := range columns {
					if col == "is_dir" && values[j] != nil {
						isDirRow = fmt.Sprintf("%v", values[j]) == "1" || fmt.Sprintf("%v", values[j]) == "true"
						break
					}
				}

				if isDirRow {
					icon = "📁 "
				} else {
					icon = "📄 "
				}

				// Use the path column value as the link destination if available
				if pathIdx != -1 && i == nameIdx {
					if pVal := values[pathIdx]; pVal != nil {
						link = fmt.Sprintf("%v", pVal)
					}
				}
			} else if isSqliteMaster && i == nameIdx && rawVal != "" {
				// Check if it's a table (not an index or view)
				isTable := true
				if typeIdx != -1 && values[typeIdx] != nil {
					isTable = fmt.Sprintf("%v", values[typeIdx]) == "table"
				}
				if isTable {
					shouldLink = true
					icon = "📊 "
					// Construct link: /DataSetPath/TableName
					link = b.DataSetPath + "/" + rawVal
				}
			}

			if shouldLink {
				cells[i] = fmt.Sprintf("<a href=\"/%s\">%s%s</a>", strings.TrimPrefix(link, "/"), icon, rawVal)
			} else {
				cells[i] = rawVal
			}
		}

		// Write row
		if err := tw.WriteHTMLRow(e.Response, rowIndex, cells); err != nil {
			log.Printf("[SERVER] Error writing row: %v", err)
			break
		}
		rowIndex++
	}

	// End HTML table
	tw.EndHTMLTable(e.Response)

	log.Printf("[SERVER] Successfully served %d rows", rowIndex)
	return nil
}
