# Potential Upgrade: Feature Gap Analysis

This document outlines features available in external libraries and the Flight2 predecessor that are not yet implemented in Flight3.

## 1. Flight3 Dependencies Features (Unused)

Flight3 imports several powerful libraries but utilizes only a subset of their capabilities.

### `pocketbase/pocketbase` (v0.36.1)
*   **Used**: Core App, `OnServe` hook, Collections initialization, Record CRUD.
*   **Not Used**:
    *   **Admin UI Customization**: While some routes hook into custom controllers, there is no custom Admin UI plugin or profound integration beyond standard startup.
    *   **Realtime Subscriptions**: No usage of `app.OnRecord...` hooks for realtime updates or `Realtime` API.
    *   **Background Jobs**: No use of PocketBase cron scheduler (`app.Cron()`).
    *   **Email Sending**: No configured mailer usage (defaults are likely used but no custom email templates or logic).
    *   **Migrations**: Only default migrations import. No custom migration logic file structure.

### `rclone/rclone` (v1.72.1)
*   **Used**: `rclone/fs` for filesystem abstraction and `backend/all` for registration. `NewFs`, `List`, `Open` (Copy).
*   **Not Used**:
    *   **Sync/Moves**: No usage of `sync` package for directory synchronization or server-side moves.
    *   **VFS (Virtual File System)**: Flight3 (and Flight2) implement their own caching on top of basic Rclone primitives ("download-then-serve"). Rclone has a sophisticated VFS layer that handles mounting, caching, and streaming which is largely re-invented here.
    *   **Filters**: No usage of inclusion/exclusion rules during listings/transfers.

### `darianmavgo/mksqlite` (v1.0.5)
*   **Used**: `converters.Open`, `ImportToSQLite`.
*   **Not Used**:
    *   **Multi-Table imports**: Flight3 logic seems to force single-table (`tb0`) logic in `banquethandler.go` (line 467ish hardcodes `tb0` unless overridden by args, but usage flow implies single source -> single table). `mksqlite` (especially Excel/FS drivers) can produce multi-table DBs.
    *   **Streaming Transformation**: The interface supports hooking into the stream, but Flight3 treats it as a black box converter.

### `darianmavgo/sqliter` (v1.1.7)
*   **Used**: `TableWriter`, `NewTableWriter` with embedded templates.
*   **Not Used**:
    *   **Editable Mode**: Flight2 has extensive logic (`handleCRUD`, `EnableEditable`) to support IN-BROWSER editing of the SQLite database. Flight3 currently only **Reads** data (`SELECT`) in `HandleBanquet`.
    *   **Configurable Pagination**: Basic offsets used, but full pagination UI helper features of `sqliter` might be underutilized.

### `darianmavgo/banquet` (v1.0.6)
*   **Used**: `ParseNested`, `ParseBanquet`, `SetVerbose`.
*   **Not Used**:
    *   **Full URI Construction**: Logic to reconstruct URLs or manipulate them programmatically beyond basic patching seems limited.
    *   **Validation**: Advanced validation of `Where` / `Select` clauses is implicit.

---

## 2. Flight2 Feature Gap

Scanning `flight2` reveals several major features missing in `flight3`.

### A. Editable Data (`internal/server/server.go`)
*   **Feature**: **In-Browser CRUD**.
*   **Flight2**: Implements `handleCRUD` which processes JSON payloads for `create`, `update`, `delete` actions and executes SQL against the SQLite DB. The `TableWriter` is configured with `EnableEditable(true)`.
*   **Flight3**: Completely missing. `HandleBanquet` is read-only.

### B. User-Defined Secrets Management (`internal/secrets/`)
*   **Feature**: **Secrets Database**.
*   **Flight2**: Uses a standalone SQLite DB (`user_secrets.db`) to store encrypted credentials (`credentials` table) using AES-256-GCM. It includes endpoints (`/app/credentials/...`) to add/edit/delete these secrets via a custom UI.
*   **Flight3**: Relies on PocketBase collections (`rclone_remotes`). This is actually an **improvement/migration**, but the **UI** for managing them is currently just the PocketBase Admin UI (implied). Flight2 had a custom user-facing page (`/app/credentials/manage`) for this.

### C. Directory Browsing UI (`internal/server/server.go`)
*   **Feature**: **File Browser**.
*   **Flight2**: `handleBrowse` renders a file listing.
*   **Flight3**: `handleBrowse` exists in `main.go` but returns raw JSON (`e.JSON(...)`). It does not appear to have the HTML frontend integration Flight2 presumably had (or Flight2 returned HTML? `handleBrowse` in Flight2 logic isn't fully visible in snippets but usually paired with UI).
    *   *Correction*: Flight3 has `handleBrowse` returning JSON. Flight2 likely used this for a UI. The **Frontend** consuming this JSON (HTML templates/JS) might be missing or different.

### D. Request History (`internal/server/server.go`)
*   **Feature**: **Recent Requests**.
*   **Flight2**: Maintains an in-memory `RequestHistory` (last 20 URLs) and displays them on error pages to help users navigate back.
*   **Flight3**: No request history tracking found.

### E. Custom Error Pages (`internal/server/server.go`)
*   **Feature**: **Rich Error UI**.
*   **Flight2**: `handleBanquet` includes elaborate HTML generation for errors (lines 359-416), offering suggestions and history.
*   **Flight3**: Returns simple JSON errors (`e.JSON(http.StatusInternalServerError, ...)`).

### F. Local DB Auto-Discovery (`registerLocalDBRoutes`)
*   **Feature**: **Serve Local DBs**.
*   **Flight2**: Scans the CWD for `.db/.sqlite` files and automatically registers routes for them.
*   **Flight3**: Does not appear to have auto-discovery of local DB files outside of the configured `serve_folder` logic via Banquet. Use of `controller` might hide this, but `main` logic doesn't show it.

### G. CSS/JS Serving
*   **Feature**: **Custom Assets**.
*   **Flight2**: Explicitly serves `cssjs/` directory.
*   **Flight3**: Not explicitly serving static assets in `main.go`, though PocketBase might serve `pb_public`. Flight2 had custom styles (`default.css` refs).

## Summary Recommendation

To reach parity with Flight2, Flight3 should prioritize:
1.  **Editable Tables**: Port `handleCRUD` logic from Flight2.
2.  **Rich UI/Errors**: Replace JSON error responses with HTML templates.
3.  **Frontend/Assets**: Ensure CSS/JS for the TableWriter (and error pages) are served.
