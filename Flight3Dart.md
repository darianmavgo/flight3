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

### Phase 1: Server Readiness (Flight3) ✅ COMPLETE
Flight3 is production-ready with all necessary endpoints.
-   [x] **PocketBase Integration**: Fully integrated (`flight.go`, `managepocketbase.go`).
-   [x] **Banquet Endpoints**: Exposed via `/sqliter/` prefix.
-   [x] **Auto-Login**: Implemented at `/api/auto_login` for desktop mode.
-   [x] **Desktop Mode**: Coordinated launch via `mage desktop` command.

### Phase 2: Client Integration (SQLiter-Dart) ✅ IMPLEMENTED
`sqliter-dart` is now a fully functional networked client.

**Current Implementation:**

1. **Desktop Mode**
   - Launch via `mage desktop` from flight3 directory
   - Automatically builds and starts both server and client
   - Server URL passed via `--dart-define=FLIGHT_URL=http://127.0.0.1:8090`
   - Auto-login enabled for seamless desktop experience

2. **API Integration**
   - File download: `GET /sqliter/file/{banquet-path}`
   - Sync metadata: `GET /sqliter/sync/{banquet-path}`
   - Query rows: `GET /sqliter/rows?path={path}&start={offset}&end={limit}`

3. **Home Page**
   - Displays `banquet_links` collection from server
   - Users can select links to navigate to datasets
   - PocketBase realtime updates (if subscribed)

**Features:**
-   [x] **Local Mode**: Opens local `.sqlite` files directly
-   [x] **Flight Mode**: Connects to Flight3 server
-   [x] **Auto-Discovery**: Server URL configured via build-time dart-define
-   [x] **PlutoGrid Rendering**: Rich table display with sorting/filtering

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
