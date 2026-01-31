# Flight3 Route Plan

This document defines the routing responsibilities between **Flight3**, **SQLiter**, and **PocketBase**.

## Routing Rules

| Pattern | Responsibility | Visibility | Description |
| :--- | :--- | :--- | :--- |
| `/_/*` | **PocketBase** | User (Admin) | PocketBase Dashboard and Admin UI. |
| `/api/*` | **PocketBase** | Internal | PocketBase REST API. |
| `/sqliter/*` | **SQLiter** | Internal | SQLiter Backend API (File sys, Headers, Rows). |
| `/*` (Scheme) | **Flight3** | User | "Scheme" URLs (e.g. `/s3/user/file.db`) trigger Flight3 to sync the file, then delegation to SQLiter. |
| `/*` (Other) | **SQLiter** | User | Main UI. Serves the React Client. |

## detailed Responsibilities

### 1. PocketBase (`/_/`, `/api/`)
These routes are reserved for the PocketBase framework.
-   **Action**: Unmodified passthrough to PocketBase `ServeEvent` default handling.

### 2. SQLiter API (`/sqliter/`)
These routes provide the data and functionality for the UI.
-   **Host**: `sqliter` Go Server.
-   **Configuration**: `BaseURL` set to `/sqliter/`.
-   **Purpose**: The React client (running at `/`) makes AJAX requests to these endpoints to list files, get table schemas, and query data.

### 3. Flight3 Logic (`scheme://` equivalent paths)
Routes that map to configured Rclone remotes (e.g., `/s3/...`, `/gdrive/...`).
-   **Action**:
    1.  **Intercept**: Flight3 middleware detects a request for a remote dataset.
    2.  **Sync**: Downloads/Caches the file from the remote source to the local `ServeFolder` (`pb_data/cache`).
    3.  **Delegate**: Once the file is local, the request falls through to the **SQLiter** handler to serve the UI for that file.

### 4. Main UI (`/`, `/*`)
The catch-all route serves the SQLiter frontend.
-   **Host**: `sqliter` Go Server (serving static assets + `index.html`).
-   **Action**: Serves the Single Page Application (SPA).
    -   Injects `window.SQLITER_CONFIG` with `basePath: "/sqliter/"` so the client knows where to find the API.
    -   The SPA handles client-side routing for `/path/to/db`.

## Implementation Strategy

All routing logic should be centralized in a single configuration function (e.g., `InstallRoutes` or `ConfigureRouting`) in `internal/flight/router.go`. This ensures the precedence and responsibility are clear at a glance.

```go
func ConfigureRouting(app *pocketbase.PocketBase, sqliter *sqliter.Server) {
    app.OnServe().BindFunc(func(e *core.ServeEvent) error {
        // 1. PocketBase (handled by exclusion/fallthrough or explicit checks)
        
        // 2. SQLiter API
        e.Router.Any("/sqliter/*", ...)

        // 3. Flight3 Scheme + SQLiter UI (Catch-all)
        e.Router.Any("/*", Flight3Middleware(sqliter))

        return e.Next()
    })
}
```
