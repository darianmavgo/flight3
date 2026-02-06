# Flight3-Dart: Hyper-Local Client-Server Architecture

This plan establishes **Flight3** as the central "Hyper-Local Server" and **SQLiter-Dart** as the specialized "Client" interface. By leveraging Flight3's built-in PocketBase and Banquet capabilities, we create a robust ecosystem where the server manages data integrity and the client provides a rich, native user experience.

## Core Concept
Shift the architecture from two standalone apps to a **Client-Server** model.
-   **Server (Flight3)**: The "Brain". Manages the database, executes complex Banquet queries, handles Rclone caching, and authenticates users. It exposes the "Shared Database" (PocketBase) via a secure API.
-   **Client (SQLiter-Dart)**: The "Face". A lightweight, responsive Flutter interface that consumes data from Flight3. It uses Banquet URLs to request precise data views.

## The "Shared SQLite Database" Strategy
The user requirement "manage communication through a shared sqlite db" is fulfilled by **PocketBase** itself. PocketBase is essentially a networked SQLite database manager.

**Architecture:**
1.  **The Shared DB**: `flight3/pb_data/data.db` (The core PocketBase database).
2.  **Management**: Flight3 ensures schema consistency (`EnsureCollections`) and handles writes/background tasks.
3.  **Client Access**: SQLiter-Dart interacts with this DB via the PocketBase SDK (HTTP/Realtime), ensuring safe, concurrent access without file-locking issues.
4.  **Banquet's Role**: `Banquet-Links` are stored in this shared DB. The client can create a "Link" (request), and the Server (Flight3) processes it (e.g., pre-fetching data).

## Implementation Plan

### Phase 1: Server Readiness (Flight3)
Flight3 is already well-positioned. We need to verify it exposes the necessary endpoints.
-   [x] **PocketBase Integration**: Already fully integrated (`flight.go`, `managepocketbase.go`).
-   [x] **Banquet Endpoints**: Exposed via `/sqliter/` prefix.
-   [ ] **Realtime Events**: Verify `flight3` is broadcasting changes to collections like `banquet_links`.

### Phase 2: Client Upgrade (SQLiter-Dart)
Transform `sqliter-dart` from a local file viewer to a networked client.

**1. Add Dependencies**
Add the PocketBase SDK to `sqliter-dart`.
```yaml
dependencies:
  pocketbase: ^0.16.2 # Check latest version
  http: ^1.2.0
```

**2. Architecture Update**
Refactor `main.dart` or create a new `FlightClient` service.
-   **Local Mode**: Keep existing functionality (opening local `.sqlite` files).
-   **Flight Mode**: Connect to `http://localhost:8090` (or `flight` served port).

**3. "Shared DB" Communication Pattern**
Instead of just reading files, the client will "communicate" via the `banquet_links` collection.

*   **Scenario: Saving a Query**
    1.  User in Dart Client crafts a complex query: `data.sqlite;Table[0:100]`.
    2.  Client saves this to `banquet_links` collection via PocketBase SDK.
    3.  Server (Flight3) persists it.
    4.  Other clients (or the Web UI) see this new link instantly via Realtime subscription.

### Phase 3: The Banquet Data Protocol
How to fetch the actual table data?
-   **Direct**: Client requests `GET /sqliter/<BanquetURL>`.
-   **Response**: Flight3 parses the URL using its internal `banquet` library, fetches data (from local cache or Rclone), and returns JSON/Arrow.
-   **Client**: Renders the response in `PlutoGrid`.

This solves the "heavy lifting" problem—the Client doesn't need to parse complex CSVs or manage Rclone. It just asks Flight3.

## Workflow: Connecting Client to Server

1.  **Start Flight3**:
    ```bash
    ./flight serve --http=127.0.0.1:8090
    ```
2.  **Connect Client**:
    *   In SQLiter-Dart, add a "Connect to Flight" button.
    *   Input: `http://127.0.0.1:8090`.
    *   Auth: `admin@example.com` / `password123` (default superuser).
3.  **Browse**:
    *   Client lists `banquet_links` from the server.
    *   Clicking a link fetches data from the `/sqliter/` endpoint.

## Summary of Changes
-   **Flight3**: No major code changes needed immediately; it acts as the stable backend. Use `EnsureHelper` to add any new communication tables if needed.
-   **SQLiter-Dart**:
    -   Add `pocketbase` dependency.
    -   Implement `FlightService`.
    -   Add UI for "Remote Server" alongside "Local File".

## Future: "Inbox/Outbox" Pattern
If tighter coordination is needed (e.g. telling Server to "download dataset X"), we add a `jobs` collection to the shared DB.
1.  Client creates `job`: `{ type: "download", url: "s3://..." }`.
2.  Flight3 (watching `jobs`) triggers `rclone copy`.
3.  Flight3 updates `job`: `{ status: "done" }`.
4.  Client (watching `jobs`) notifies user.
