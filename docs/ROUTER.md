# Routing Documentation

This document summarizes the URL routing decisions and handlers defined in the Flight3 project. All routing logic is centralized in `cmd/flight/main.go`.

## Route Definitions

The application uses the PocketBase router (`e.Router`) within the `OnServe` hook.

| Method | Path Pattern | Handler Logic | Defined In |
| :--- | :--- | :--- | :--- |
| **GET** | `/` | **Local Root Override**: Intercepts the root request, rewrites the internal request URI to `/https:/local@localhost/`, and serves the Banquet directory listing for the local `serve_folder`. | `cmd/flight/main.go` (approx line 212) |
| **GET** | `/*` | **Wildcard / Static UI**: Handles all other paths. <br> 1. Checks if path is root (`""` or `"/"`) and delegates to Local Root logic (above). <br> 2. Otherwise, attempts to serve static files from the embedded `ui` filesystem (PocketBase Admin UI). <br> 3. Falls back to `index.html` for SPA routing logic (if file not found). | `cmd/flight/main.go` (approx line 221) |
| **GET** | `/api/rclone/browse/{id}` | **Rclone Browser API**: JSON API for browsing remote filesystems via Rclone. Used by the frontend or API clients to list files in a bucket/remote. | `cmd/flight/main.go` (approx line 273) |
| **GET** | `/banquet/*` | **Banquet Handler (Legacy/Explicit)**: Handles requests specifically prefixed with `/banquet/`. Delegates to `handleBanquet`. | `cmd/flight/main.go` (approx line 278) |
| **GET** | `/http:/{any...}` | **Direct Http Banquet Link**: Handles "nested" URLs starting with `http:/`. Delegates to `handleBanquet`. | `cmd/flight/main.go` (approx line 283) |
| **GET** | `/https:/{any...}` | **Direct Https Banquet Link**: Handles "nested" URLs starting with `https:/`. Delegates to `handleBanquet`. | `cmd/flight/main.go` (approx line 286) |

## Middleware

| Path Pattern | Logic | Defined In |
| :--- | :--- | :--- |
| `/_/` (Admin UI) | **Dark Mode Injector**: Intercepts the PocketBase Admin Dashboard HTML response and injects a CSS `<style>` block to invert colors. | `cmd/flight/main.go` (approx line 173) |

## Handler Functions

- **`handleBanquet`**: The core logic for rendering data tables. It parses the URL (nested http/https scheme), resolves the `rclone_remote` or `data_pipeline`, fetches data (or lists directory via Rclone), converts it to SQLite (using `mksqlite`), and renders the result using `sqliter` and HTML templates.
- **`handleBrowse`**: A helper for the JSON API that uses Rclone to list files and returns a JSON response.
- **`subFS` Static File Serving**: Standard Go `http.ServeContent` logic used to serve the embedded Admin UI assets.
