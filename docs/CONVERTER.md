# MkSQLite Usage in Flight2

This document summarizes how **Flight2** utilizes the `mksqlite` library for data conversion, based on codebase analysis of `flight2/internal/dataset/manager.go`.

## Import Strategy

Flight2 imports the necessary packages from `mksqlite` as follows:

```go
import (
    "github.com/darianmavgo/mksqlite/converters"
    "github.com/darianmavgo/mksqlite/converters/common"
    _ "github.com/darianmavgo/mksqlite/converters/all" // Registers all drivers side-effect
)
```

## Conversion Workflow

The core logic resides in `Manager.GetSQLiteDB`.

1.  **Driver Resolution**:
    *   Flight2 maintains an explicit `extensionMap` (e.g., `.csv` -> `csv`, `.xlsx` -> `excel`).
    *   If no match is found in the map, it attempts to use the extension itself (minus dot) as the driver name.

2.  **Source Preparation**:
    *   It streams the remote file (via `dataset_source`) to a temporary local file (`tmpSource`).

3.  **Conversion Process**:
    *   **Passthrough**: If the driver is determined to be `sqlite`, it simply copies the source file to the destination.
    *   **Conversion**: For other formats, it utilizes `mksqlite`:
        ```go
        // Open the converter for the specific driver
        conv, err := converters.Open(driver, srcFile, &common.ConversionConfig{Verbose: m.verbose})
        
        // Execute import to the destination SQLite file
        err = converters.ImportToSQLite(conv, tmpOut, &converters.ImportOptions{Verbose: m.verbose})
        ```

4.  **Filesystem Support**:
    *   If the source is a local directory, it uses the special `filesystem` driver to create a directory listing database.

## Caching

Flight2 implements a two-tier cache (Memory via BigCache + Disk) for the converted databases, identified by a hash of `alias:sourcePath`.
