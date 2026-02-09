# Flight3

Flight3 is a modern data serving platform that unifies cloud storage, local data, and configuration management into a single cohesive interface. It acts as the "glue" that binds **PocketBase** (Configuration), **Rclone** (Storage), **Banquet** (Parsing), and **SQLiter** (Rendering).

## Quick Start

### Start the Server
```bash
cd flight3
mage run
```
Server starts at `http://127.0.0.1:8090`

### Desktop Mode (Server + Client)
```bash
mage desktop
```
Automatically launches both:
- Flight3 server at `http://127.0.0.1:8090`
- SQLiter-Dart client (Flutter app)

### Access Admin UI
Open `http://127.0.0.1:8090/_/` in your browser to access the PocketBase admin dashboard.

---

## Features Supported

*   **Unified Cloud Storage Access**: Seamlessly connects to S3, GCS, R2, Google Drive, and 40+ other providers via embedded **Rclone**.
*   **Virtual File System (VFS)**: Implements smart caching, connection pooling, and partial reads to make remote cloud files feel local and fast.
*   **Dynamic Configuration**: Uses **PocketBase** as a backend for managing remotes, pipelines, and settings via a friendly Admin UI.
*   **On-the-Fly Conversion**: Automatically detects non-SQLite files (CSVs, Excel, directories) and uses `mksqlite` to convert them to queryable databases transparently.
*   **Banquet URL Support**: Fully supports the Banquet URL notation for complex, nested data queries.
*   **SQLiter API**: Internal REST API (`/sqliter/*`) for data querying compatible with SQLiter-Dart client.
*   **Desktop Mode**: Coordinated launch of server and Flutter client for seamless local development.
*   **Desktop Mode**: Coordinated launch of server and Flutter client for seamless local development.
*   **No Authentication**: API and data access are public by default in Desktop mode.

## Area of Responsibility

Flight3 is the **Orchestrator**. Its responsibility is **Resource Acquisition** (Scheme → DataSetPath).

*   Parse and authenticate Banquet URLs
*   Connect to remote storage via Rclone
*   Fetch and cache files
*   Convert non-SQLite files to SQLite databases
*   Provide PocketBase admin interface
*   Serve data via SQLiter API to clients

## Scope (What it explicitly doesn't do)

*   **No Native Storage Driver Implementation**: Flight3 does not write its own S3 or GCS clients. It relies entirely on **Rclone** for communicating with storage providers.
*   **No Query Execution**: It delegates SQL query building and execution to **SQLiter** (ColumnSetPath → Query).
*   **No UI Rendering**: It provides a REST API; UI rendering is handled by **SQLiter-Dart** client.
*   **No File Conversion Logic**: It delegates file format transcoding to **mksqlite**.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    SQLiter-Dart Client                      │
│                    (Flutter Desktop App)                    │
└─────────────────────────────────────────────────────────────┘
                             ↕ HTTP API
┌─────────────────────────────────────────────────────────────┐
│                      Flight3 Server                         │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ PocketBase  │  │ Rclone VFS   │  │  Mksqlite    │      │
│  │ (Config)    │  │ (Storage)    │  │ (Converter)  │      │
│  └─────────────┘  └──────────────┘  └──────────────┘      │
│  ┌─────────────┐  ┌──────────────┐                         │
│  │  Banquet    │  │  SQLiter API │                         │
│  │  (Parser)   │  │  (Querying)  │                         │
│  └─────────────┘  └──────────────┘                         │
└─────────────────────────────────────────────────────────────┘
                             ↕
                    Cloud Storage Providers
                    (S3, GCS, R2, Drive, etc.)
```

---

## Mage Build System

Flight3 uses [Mage](https://magefile.org/) for build automation. Common commands:

### Development
```bash
mage run         # Build and run server
mage dev         # Run with debug mode enabled
mage desktop     # Launch server + SQLiter-Dart client
mage launch      # Build, run, and open Chrome
```

### Building
```bash
mage build       # Build flight binary
mage clean       # Clean build artifacts (auto-backs up PocketBase)
mage deploy      # Build for multiple platforms
```

### Testing
```bash
mage test        # Run all tests
mage lint        # Run golangci-lint
mage fmt         # Format all Go code
mage all         # Run fmt, tidy, test, and build
```

### Installation & Service Management
```bash
mage install     # Install to /usr/local/bin (macOS)
mage uninstall   # Remove from system locations
mage service     # Install as macOS launchd service
mage unservice   # Remove launchd service
mage logs        # Tail service logs
```

### Utilities
```bash
mage backuppocketbase           # Create timestamped PocketBase backup
mage mergelogs server.log client.log  # Merge server/client logs chronologically
mage kill        # Terminate running flight processes
```

See [BUILD_TOOLS.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/BUILD_TOOLS.md) for detailed documentation.

---

## Client Integration

### SQLiter-Dart (Flutter Client)

Flight3 provides REST API endpoints for the SQLiter-Dart client:

**Connection:**
- Server: `http://127.0.0.1:8090`
**Connection:**
- Server: `http://127.0.0.1:8090`


**Data API:**
- File download: `GET /sqliter/file/{banquet-path}`
- Sync metadata: `GET /sqliter/sync/{banquet-path}`
- Query rows: `GET /sqliter/rows?path={banquet-path}&start={offset}&end={limit}`

**Desktop Mode:**
Run `mage desktop` to automatically launch both server and client in coordinated mode.

---

## PocketBase Integration

Flight3 uses PocketBase for configuration management:

**Admin UI:** `http://127.0.0.1:8090/_/`
- Default user: `admin@example.com` / `password123`
- Manage rclone remotes, app settings, banquet links

**Collections:**
- `rclone_remotes` - Cloud storage configurations
- `app_settings` - Application settings (serve_folder, etc.)
- `banquet_links` - Saved Banquet URL bookmarks

**Rclone Config UI:** `http://127.0.0.1:8090/_/rclone_config`
- Visual interface for adding/editing cloud remotes
- Provider templates for S3, GCS, R2, etc.
- Test connection functionality

---

## Testing Infrastructure

Flight3 has comprehensive test coverage:

**Unit Tests:**
- Banquet URL parsing
- Rclone integration
- Mksqlite conversion
- PocketBase operations

**Integration Tests:**
- Banquet listing and querying
- Remote file fetching (R2, S3)
- Directory indexing
- Cache validation

**Static Analysis:**
- Link validation in markdown docs
- HTML/CSS scanning
- Security checks

Run all tests: `mage test`

See [TESTING.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/TESTING.md) for details.

---

## Deployment Options

### Option 1: Run Locally
```bash
mage run
```

### Option 2: System Installation (macOS)
```bash
mage install
```
Installs to:
- Binary: `/usr/local/bin/flight`
- Data: `~/Library/Application Support/Flight3/`

### Option 3: Launch at Startup (macOS)
```bash
mage service
```
Installs as launchd service with auto-start on login.

Logs: `~/Library/Logs/Flight3/flight.log`

---

## Documentation

- [DEVELOPMENT.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/DEVELOPMENT.md) - Development workflow and hot reload
- [ARCHITECTURE.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/ARCHITECTURE.md) - System architecture and component responsibilities
- [Flight3Dart.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/Flight3Dart.md) - Client-server integration guide
- [BUILD_TOOLS.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/BUILD_TOOLS.md) - Mage build system reference
- [TESTING.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/TESTING.md) - Testing infrastructure and procedures
- [QUICK_REFERENCE.md](file:///Users/darianhickman/Documents/flight-buddies/flight3/docs/QUICK_REFERENCE.md) - Quick command reference

---

## License

See LICENSE file for details.
