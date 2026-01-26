package main

import (
	"fmt"

	"github.com/darianmavgo/banquet"
)

func main() {
	// Parse a dummy url to get a populated struct to avoid nil panic on String()
	b, _ := banquet.ParseNested("/https:/host/path")
	fmt.Printf("DataSetPath: %s\n", b.DataSetPath)
	// Check possible names for columnset
	fmt.Printf("ColumnPath: %s\n", b.ColumnPath) // Uncomment if suspected
}
