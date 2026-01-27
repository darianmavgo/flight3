package flight

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"strings"

	"github.com/darianmavgo/banquet"
	"github.com/darianmavgo/sqliter/sqliter"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pocketbase/pocketbase/core"
)

// ServeFromCache opens cached SQLite DB and serves query results
func ServeFromCache(cachePath string, b *banquet.Banquet, tw *sqliter.TableWriter, tpl *template.Template, e *core.RequestEvent) error {
	log.Printf("[SERVER] Serving from cache: %s", cachePath)

	// Open SQLite database
	db, err := sql.Open("sqlite3", cachePath)
	if err != nil {
		return fmt.Errorf("failed to open cache database: %w", err)
	}
	defer db.Close()

	// Build SQL query from banquet fields
	query := buildSQLQuery(b)
	log.Printf("[SERVER] Executing query: %s", query)

	// Execute query
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Start HTML table
	title := b.DataSetPath
	if b.Table != "" {
		title = b.Table
	}
	tw.StartHTMLTable(e.Response, columns, title)

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
			if val == nil {
				cells[i] = ""
			} else {
				cells[i] = fmt.Sprintf("%v", val)
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
		selectClause = strings.Join(b.Select, ", ")
	}
	parts = append(parts, "SELECT "+selectClause)

	// FROM clause
	table := b.Table
	if table == "" {
		table = "tb0" // Default table name
	}
	parts = append(parts, "FROM "+table)

	// WHERE clause
	if b.Where != "" {
		parts = append(parts, "WHERE "+b.Where)
	}

	// GROUP BY clause
	if b.GroupBy != "" {
		parts = append(parts, "GROUP BY "+b.GroupBy)
	}

	// HAVING clause
	if b.Having != "" {
		parts = append(parts, "HAVING "+b.Having)
	}

	// ORDER BY clause
	if b.OrderBy != "" {
		orderBy := b.OrderBy
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
