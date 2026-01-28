package flight

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"strings"

	"github.com/darianmavgo/banquet"
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
		return NewBanquetError(err, "Failed to open cache database", 500, b, "")
	}
	defer db.Close()

	// Build SQL query from banquet fields
	query := buildSQLQuery(b)
	log.Printf("[SERVER] Executing query: %s", query)

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return NewBanquetError(err, "Query execution failed", 400, b, query)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return NewBanquetError(err, "Failed to get columns", 500, b, query)
	}

	// Start HTML table with debug info
	title := b.DataSetPath
	if b.Table != "" {
		title = b.Table
	}

	// Create one-liner Banquet debug info
	banquetDebug := fmt.Sprintf("Banquet{Scheme:%q Host:%q Path:%q Table:%q Where:%q Limit:%q Offset:%q}",
		b.Scheme, b.Host, b.Path, b.Table, b.Where, b.Limit, b.Offset)

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
		}
		rowIndex++
	}

	// End HTML table
	tw.EndHTMLTable(e.Response)

	log.Printf("[SERVER] Successfully served %d rows", rowIndex)
	return nil
}

// buildSQLQuery constructs a SQL query from banquet fields
func buildSQLQuery(b *banquet.Banquet) string {
	var parts []string

	// SELECT clause
	selectClause := "*"
	if len(b.Select) > 0 && b.Select[0] != "*" {
		quotedSelects := make([]string, len(b.Select))
		for i, col := range b.Select {
			quotedSelects[i] = quoteIdentifier(col)
		}
		selectClause = strings.Join(quotedSelects, ", ")
	}
	parts = append(parts, "SELECT "+selectClause)

	// FROM clause
	table := b.Table
	if table == "" {
		table = "tb0" // Default table name
	}
	parts = append(parts, "FROM "+quoteIdentifier(table))

	// WHERE clause
	if b.Where != "" {
		parts = append(parts, "WHERE "+b.Where)
	}

	// GROUP BY clause
	if b.GroupBy != "" {
		parts = append(parts, "GROUP BY "+quoteIdentifier(b.GroupBy))
	}

	// HAVING clause
	if b.Having != "" {
		parts = append(parts, "HAVING "+b.Having)
	}

	// ORDER BY clause
	if b.OrderBy != "" {
		orderBy := quoteIdentifier(b.OrderBy)
		if b.SortDirection != "" {
			orderBy += " " + b.SortDirection
		}
		parts = append(parts, "ORDER BY "+orderBy)
	}

	// LIMIT clause
	if b.Limit != "" {
		parts = append(parts, "LIMIT "+b.Limit)
	}

	// OFFSET clause
	if b.Offset != "" {
		parts = append(parts, "OFFSET "+b.Offset)
	}

	return strings.Join(parts, " ")
}

// quoteIdentifier wraps a string in double quotes and escapes any double quotes within.
func quoteIdentifier(s string) string {
	if s == "" || s == "*" {
		return s
	}
	// If it already seems quoted or is a complex expression, skip for now?
	// But banquet usually returns clean names.
	return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
}
