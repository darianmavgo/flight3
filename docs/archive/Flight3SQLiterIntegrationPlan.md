# Flight3 and SQLiter 1.5.0 Integration Plan

## Goal
Integrate Flight3 with the local `sqliter` 1.5.0 library to share SQLite files and ensure seamless UI hosting under a subpath.

## User Review Required
> [!IMPORTANT]
> **BaseURL Configuration**: We will configure `sqliter` with a `BaseURL` of `/sqliter/`. This requires `sqliter` to correctly handle this prefix when serving the React app and static assets. We may need to modify `sqliter`'s `ServeHTTP` to strip this prefix for static file resolution.

> [!NOTE]
> **Shared Folder**: We assume the "shared sqlite files" are located in `Flight3`'s `pb_data/cache` directory, which is currently configured. If a different shared location is intended, please update the `ServeFolder`.

## Proposed Changes

### Flight3 (`github.com/darianmavgo/flight3`)

#### [NEW] [router.go](file:///Users/darianhickman/Documents/flight3/internal/flight/router.go)
-   Create a new file to centralize routing logic.
-   Implement `ConfigureRouting(e *core.ServeEvent, sqliter *sqliter.Server) error`.
-   Define the router middleware chain:
    1.  PocketBase Checks (`/_/`, `/api`).
    2.  SQLiter API (`/sqliter/*`) -> Delegate to `sqliterServer`.
    3.  Flight3 Middleware -> Check for Scheme/Remote URLs, trigger sync/download.
    4.  SQLiter UI (`/*`) -> Delegate to `sqliterServer`.

#### [MODIFY] [flight.go](file:///Users/darianhickman/Documents/flight3/internal/flight/flight.go)
-   Update `sqliterConfig` to include `BaseURL: "/sqliter/"`.
-   Verify and ensure `ServeFolder` points to the desired shared directory (keeping `pb_data/cache` or changing if needed).
-   Replace the inline routing logic in `OnServe` with a call to `ConfigureRouting` (from `router.go`).

### SQLiter (`github.com/darianmavgo/sqliter`)

*No changes required.* We will rely on `sqliter` 1.5.0's existing `BaseURL` support.

## Verification Plan

### Automated Tests
-   Run `Flight3` logic: `go run cmd/flight/main.go serve --http=127.0.0.1:8090`.
-   Verify routing with `curl`:
    -   `curl -L http://127.0.0.1:8090/sqliter` (should redirect to `/sqliter/` and return HTML).
    -   `curl http://127.0.0.1:8090/sqliter/` (should return HTML with `window.SQLITER_CONFIG = { basePath: "/sqliter/" }`).
    -   `curl http://127.0.0.1:8090/sqliter/sqliter/fs` (should return JSON file list).
    -   `curl http://127.0.0.1:8090/` (should return HTML for the main UI).

### Manual Verification
1.  **Start Flight3**: `go run cmd/flight/main.go serve`
2.  **Access UI**: Open `http://localhost:8090/` in browser.
3.  **Check Experience**:
    -   Ensure the UI loads.
    -   Navigate to a "Scheme URL" (if testable) or ensure standard local files work.
    -   Verify PocketBase admin still works at `http://localhost:8090/_/`.
