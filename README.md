# Flight3: Enhanced Data Serving Platform

Flight3 is a modern data serving application that integrates **PocketBase**, **Rclone**, **Banquet**, and **SQLiter** to provide a powerful, flexible, and user-friendly interface for accessing and visualizing data from various cloud storage sources.

## Core Integration

Flight3 leverages four key technologies to deliver its functionality:

1.  **PocketBase as the Core Framework**:
    -   Serves as the backbone of the application, managing the HTTP server, routing, and database interactions.
    -   Stores configuration for remote data sources (`rclone_remotes`), conversion settings (`mksqlite_configs`), and data pipelines (`data_pipelines`) in its internal SQLite collections.
    -   Provides the Admin UI for managing these configurations dynamically without code changes.

2.  **Rclone for Universal File Access**:
    -   Provides a unified interface to access files from 40+ cloud storage providers (S3, GCS, R2, Drive, etc.).
    -   Uses VFS (Virtual File System) with full caching mode for efficient random access to large files.
    -   Credentials and configuration stored securely in PocketBase collections.
    -   Automatic connection pooling and reuse via VFS cache map.

3.  **Banquet for URL Parsing**:
    -   Handles complex, nested URLs to dynamically interpret data requests.
    -   Allows users to specify data sources, tables, columns, and filters directly in the URL (e.g., `/r2_main/data/sales.csv/Name,Amount?where=Amount>1000`).
    -   Parses these "Banquet URLs" into structured query parameters used by the backend.

4.  **SQLiter for Data Rendering**:
    -   Responsible for the presentation layer, rendering data into rich, interactive HTML tables.
    -   Uses embedded templates to generate consistent and aesthetically pleasing UI components.
    -   Supports streaming large datasets efficiently.

## Quick Start

```bash
# Build the application
go build ./cmd/flight

# Run Flight3
./flight serve

# Access the admin UI
open http://localhost:8090/_/

# Access data via Banquet URLs
open http://localhost:8090/your_remote/path/to/data.csv
```

## Configuration

Flight3 uses PocketBase collections for all configuration. See [Migration Guide](docs/MIGRATION_TO_POCKETBASE_RCLONE.md) for detailed setup instructions.

### Example: Configure Cloudflare R2

1. Access admin UI at `http://localhost:8090/_/`
2. Go to Collections → `rclone_remotes`
3. Create a new record with your R2 credentials
4. Access your data: `http://localhost:8090/r2_main/bucket/file.csv`

## Documentation

- **[Rclone + PocketBase Integration](docs/RCLONE_POCKETBASE.md)** - Architecture overview
- **[Implementation Plan](docs/RCLONE_POCKETBASE_IMPLEMENTATION_PLAN.md)** - Development roadmap
- **[Migration Guide](docs/MIGRATION_TO_POCKETBASE_RCLONE.md)** - Setup and configuration
- **[Rclone Integration](docs/RCLONE.md)** - VFS caching details
- **[PocketBase Features](docs/PB.md)** - PocketBase capabilities

## Directory Structure

The project is organized as follows:

-   **`bin/`**: Contains compiled executable binaries.
-   **`cmd/`**: Holds the main application entry points.
    -   `flight/`: The main Flight3 application.
    -   `simplepb/`: A minimal PocketBase server example.
-   **`docs/`**: Documentation files, including architecture notes and migration guides.
-   **`internal/`**: Contains the core application logic and private packages.
    -   `flight/`: The heart of the application, encompassing:
        -   `flight.go`: Main orchestration logic and initialization.
        -   `banquethandler.go`: Handling and serving of Banquet requests.
        -   `rclone_manager.go`: VFS management and file fetching.
        -   `managepocketbase.go`: Initializing PocketBase collections and schema.
        -   `cache.go`: Cache key generation and validation.
        -   `converter.go`: File-to-SQLite conversion via mksqlite.
        -   `server.go`: Query execution and data serving.
-   **`logs/`**: Stores application log files generated during runtime.
-   **`pb_data/`**: The default directory for PocketBase data storage.
    -   `cache/`: Cached SQLite databases.
    -   `temp/`: Temporary files during conversion.
-   **`pb_public/`**: Static assets served by PocketBase.
-   **`test_output/`**: Contains artifacts and logs generated from test executions.
-   **`tests/`**: Holds the project's test suites and integration tests.

## Features

- ✅ Multi-cloud data access (S3, GCS, R2, and 40+ providers)
- ✅ Dynamic configuration via PocketBase Admin UI
- ✅ Intelligent caching with TTL support
- ✅ SQL-like querying via URL parameters
- ✅ Automatic file format conversion (CSV, Excel, JSON → SQLite)
- ✅ Streaming large datasets efficiently
- ✅ VFS connection pooling and reuse
- 🚧 Editable tables (planned)
- 🚧 Real-time data updates (planned)

## Architecture

```
User Request (Banquet URL)
    ↓
HandleBanquet (parse URL)
    ↓
LookupRemote (PocketBase)
    ↓
GetVFS (rclone manager)
    ↓
Check Cache (validate TTL)
    ↓
[Cache Miss] → FetchFile (VFS) → Convert (mksqlite) → Cache
    ↓
[Cache Hit] → ServeFromCache (SQLite query) → Render (SQLiter)
    ↓
HTML Table Response
```

## Testing

```bash
# Run all tests
go test ./tests/...

# Run specific test
go test ./tests/ -run TestRclonePocketBaseIntegration

# Verbose output
go test -v ./tests/...
```

## Contributing

Flight3 is under active development. Key areas for contribution:
- Additional rclone backend support
- Enhanced caching strategies
- Performance optimizations
- Documentation improvements

## License

[Your License Here]
