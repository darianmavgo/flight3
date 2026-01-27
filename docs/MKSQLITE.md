# MKSQLite Converters Reference

**mksqlite** is a robust library and CLI tool that converts various file formats into SQLite databases or SQL statements. It's used by Flight3 for converting data files on-the-fly.

## Overview

mksqlite automatically detects input types based on file extensions and converts them to queryable SQLite tables. All converters follow a consistent interface and can be used both as a library and via CLI.

## Installation & Usage

```bash
# CLI - Create SQLite database
mksqlite -i input.csv -o output.db

# CLI - Export SQL statements
mksqlite --sql -i input.csv > output.sql

# Library usage
import "github.com/darianmavgo/mksqlite/converters"
```

## Available Converters

### 1. CSV Converter

**Extensions**: `.csv`

**Input Requirements**:
- Plain text CSV file
- First row should contain column headers
- Supports various delimiters (auto-detected or configurable)

**Output**:
- Single table named `tb0` (default)
- Column names sanitized from CSV headers
- All data types stored as TEXT (SQLite's flexible typing handles this)

**Features**:
- Auto-detection of delimiter (comma, semicolon, tab, pipe)
- Handles quoted fields with embedded delimiters
- Streaming support for large files
- Concurrent processing for performance
- Handles inconsistent row lengths (pads or truncates)

**Configuration Options**:
```go
Config{
    Delimiter: ',',      // Custom delimiter
    HasHeaders: true,    // First row is headers
    SkipRows: 0,        // Skip N rows before headers
}
```

**Example**:
```bash
# Input: data.csv
Name,Age,City
Alice,30,NYC
Bob,25,LA

# Output: tb0 table with columns: Name, Age, City
```

---

### 2. Excel Converter

**Extensions**: `.xlsx`, `.xls`

**Input Requirements**:
- Microsoft Excel workbook
- First row of each sheet should contain headers
- Supports multiple sheets

**Output**:
- One table per sheet
- Table names derived from sheet names (sanitized for SQL)
- First row used as column headers

**Features**:
- Reads all sheets in workbook
- Handles merged cells
- Preserves data types where possible
- Supports both .xlsx (Office Open XML) and .xls (legacy)

**Limitations**:
- SQL export not yet supported (database creation only)
- Formulas are evaluated to their values

**Example**:
```bash
# Input: report.xlsx with sheets "Sales" and "Inventory"
# Output: Two tables: sales, inventory
```

---

### 3. HTML Converter

**Extensions**: `.html`, `.htm`

**Input Requirements**:
- HTML file containing `<table>` elements
- Tables can have optional `id` attribute for naming

**Output**:
- One table per `<table>` element found
- Table names from `id` attribute, or `table0`, `table1`, etc.
- First `<tr>` used as headers (or generated if missing)

**Features**:
- Extracts all tables from HTML document
- Handles `<thead>`, `<tbody>`, `<th>`, `<td>` properly
- Strips HTML tags from cell content
- Supports nested tables (outer table only)

**Example**:
```html
<!-- Input: report.html -->
<table id="users">
  <tr><th>Name</th><th>Email</th></tr>
  <tr><td>Alice</td><td>alice@example.com</td></tr>
</table>

<!-- Output: users table with columns: Name, Email -->
```

---

### 4. JSON Converter

**Extensions**: `.json`

**Input Requirements**:
- JSON file containing array of objects
- All objects should have consistent schema
- Nested objects are flattened with dot notation

**Output**:
- Single table named `tb0`
- Column names from object keys
- Nested objects flattened (e.g., `user.name`, `user.email`)

**Features**:
- Handles arrays of objects
- Flattens nested structures
- Converts arrays to JSON strings
- Type inference for columns

**Example**:
```json
// Input: data.json
[
  {"id": 1, "name": "Alice", "address": {"city": "NYC"}},
  {"id": 2, "name": "Bob", "address": {"city": "LA"}}
]

// Output: tb0 with columns: id, name, address_city
```

---

### 5. Markdown Converter

**Extensions**: `.md`, `.markdown`

**Input Requirements**:
- Markdown file containing tables (GitHub Flavored Markdown format)
- Tables must use pipe `|` delimiters

**Output**:
- One table per markdown table found
- Tables named `table0`, `table1`, etc.
- Headers from first row of markdown table

**Features**:
- Parses GitHub Flavored Markdown tables
- Handles alignment indicators (`:---`, `:---:`, `---:`)
- Strips markdown formatting from cells

**Example**:
```markdown
<!-- Input: README.md -->
| Name  | Age |
|-------|-----|
| Alice | 30  |
| Bob   | 25  |

<!-- Output: table0 with columns: Name, Age -->
```

---

### 6. TXT Converter

**Extensions**: `.txt`

**Input Requirements**:
- Plain text file
- Can be treated as single-column data or parsed with custom delimiter

**Output**:
- Single table named `tb0`
- Each line becomes a row
- Single column named `line` (or custom columns if delimiter specified)

**Features**:
- Line-by-line processing
- Optional delimiter-based parsing
- Handles large files efficiently

**Example**:
```bash
# Input: log.txt
Error: Connection failed
Warning: Retry attempt 1
Info: Connected successfully

# Output: tb0 with column: line (3 rows)
```

---

### 7. ZIP Converter

**Extensions**: `.zip`

**Input Requirements**:
- ZIP archive file
- Can contain any file types

**Output**:
- Single table named `file_list`
- Metadata about files in archive

**Columns**:
- `name` - File path within archive
- `size` - Uncompressed size in bytes
- `compressed_size` - Compressed size in bytes
- `crc` - CRC32 checksum
- `modified` - Modification timestamp
- `is_dir` - Boolean indicating if entry is directory

**Features**:
- Lists all files in archive
- Provides metadata without extracting
- Useful for archive inspection

**Limitations**:
- SQL export not yet supported
- Does not extract or convert file contents

**Example**:
```bash
# Input: data.zip containing multiple files
# Output: file_list table with metadata for each file
```

---

### 8. Filesystem Converter

**Extensions**: None (used for directories)

**Input Requirements**:
- Directory path
- Recursively scans all subdirectories

**Output**:
- Single table named `data`
- One row per file/directory found

**Columns**:
- `path` - Full relative path
- `name` - File/directory name
- `size` - Size in bytes (0 for directories)
- `extension` - File extension (empty for directories)
- `mod_time` - Last modification timestamp
- `is_dir` - Boolean indicating if entry is directory

**Features**:
- Recursive directory traversal
- Fast scanning with goroutines
- Respects .gitignore patterns (optional)
- Useful for file system indexing

**Example**:
```bash
# Input: ./project_dir/
# Output: data table listing all files and directories
```

---

## Common Features Across All Converters

### 1. Streaming Support
All converters support streaming to handle files larger than available memory.

### 2. SQL Export
Most converters can export to SQL statements instead of SQLite database:
```bash
mksqlite --sql -i data.csv > schema.sql
```

### 3. Table Naming
- Default table name: `tb0`
- Can be customized via configuration
- Multiple tables use descriptive names or `table0`, `table1`, etc.

### 4. Column Name Sanitization
All converters sanitize column names for SQL compatibility:
- Remove special characters
- Replace spaces with underscores
- Ensure names start with letter or underscore
- Handle reserved SQL keywords

### 5. Error Handling
- Graceful handling of malformed data
- Detailed error messages
- Partial conversion on errors (when possible)

## Configuration Options

### Global Options (common.ConversionConfig)

```go
type ConversionConfig struct {
    TableName    string  // Override default table name
    BatchSize    int     // Rows per transaction (default: 1000)
    Verbose      bool    // Enable detailed logging
    StrictMode   bool    // Fail on any error vs. best-effort
}
```

### Converter-Specific Options

Each converter may have additional options. Check the specific converter documentation or source code for details.

## Performance Considerations

### Large Files
- **CSV**: Streams rows, memory usage ~constant
- **Excel**: Loads sheets into memory, use streaming mode if available
- **JSON**: Parses entire file, memory usage proportional to file size
- **Filesystem**: Memory usage proportional to number of files

### Optimization Tips
1. Increase `BatchSize` for faster inserts (default: 1000)
2. Use streaming mode when available
3. For very large files, consider splitting before conversion
4. Use SQL export mode if you don't need SQLite database

## Integration with Flight3

Flight3 uses mksqlite automatically when serving data files:

```bash
# User requests CSV file
http://localhost:8090/data/sales.csv

# Flight3 automatically:
# 1. Detects .csv extension
# 2. Calls mksqlite converter
# 3. Creates cached SQLite database
# 4. Serves data via SQL queries
```

### Supported in Flight3
- ✅ CSV
- ✅ Excel (.xlsx, .xls)
- ✅ HTML tables
- ✅ JSON
- ✅ Markdown tables
- ✅ TXT files
- ✅ ZIP archives (metadata)
- ✅ Directories (filesystem listing)

## Error Messages

Common errors and solutions:

| Error | Cause | Solution |
|-------|-------|----------|
| `unsupported file type` | Unknown extension | Check file extension matches supported formats |
| `mksqlite binary not found` | mksqlite not in PATH | Install mksqlite or set correct PATH |
| `conversion failed` | Malformed input data | Check input file format and syntax |
| `failed to create destination directory` | Permission issue | Check write permissions on cache directory |

## CLI Reference

```bash
# Basic usage
mksqlite -i <input> -o <output.db>

# SQL export
mksqlite --sql -i <input> > output.sql

# With custom table name
mksqlite -i data.csv -o out.db --table my_data

# Verbose mode
mksqlite -v -i data.csv -o out.db

# Help
mksqlite --help
```

## Library API

```go
import "github.com/darianmavgo/mksqlite/converters"

// Auto-detect and convert
engine := converters.NewEngine()
err := engine.Convert("input.csv", "output.db")

// Use specific converter
csvConv := &csv.CSVConverter{}
err := csvConv.ConvertFile("input.csv", "output.db")

// Stream to SQL
file, _ := os.Open("data.csv")
csvConv.ConvertToSQL(os.Stdout)
```

## Version Information

- **Current Version**: Compatible with Go 1.25+
- **Dependencies**:
  - `github.com/mattn/go-sqlite3` - SQLite driver
  - `github.com/xuri/excelize/v2` - Excel support
  - `golang.org/x/net/html` - HTML parsing

## Future Enhancements

- [ ] Parquet support
- [ ] XML converter
- [ ] YAML converter
- [ ] SQL export for Excel and ZIP
- [ ] Streaming mode for all converters
- [ ] Custom type inference
- [ ] Schema validation

---

**For more information**: See the [mksqlite repository](https://github.com/darianmavgo/mksqlite)
