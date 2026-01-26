# Flight3: Enhanced Data Serving Platform

Flight3 is a modern data serving application that integrates **PocketBase**, **Banquet**, and **SQLiter** to provide a powerful, flexible, and user-friendly interface for accessing and visualizing data from various sources.

## Core Integration

Flight3 leverages three key technologies to deliver its functionality:

1.  **PocketBase as the Core Framework**:
    -   Serves as the backbone of the application, managing the HTTP server, routing, and database interactions.
    -   Stores configuration for remote data sources (`rclone_remotes`) and data pipelines (`data_pipelines`) in its internal SQLite collections.
    -   Provides the Admin UI for managing these configurations.

2.  **Banquet for URL Parsing**:
    -   Handles complex, nested URLs to dynamically interpret data requests.
    -   Allows users to specify data sources, tables, columns, and filters directly in the URL (e.g., `/<alias>@<source>/<path>?where=id>10`).
    -   Parses these "Banquet URLs" into structured query parameters used by the backend.

3.  **SQLiter for Data Rendering**:
    -   Responsible for the presentation layer, rendering data into rich, interactive HTML tables.
    -   Uses embedded templates to generate consistent and aesthetically pleasing UI components.
    -   (Planned) Will support editable tables for in-browser CRUD operations.

## Directory Structure

The project is organized as follows:

-   **`bin/`**: Contains compiled executable binaries.
-   **`cmd/`**: Holds the main application entry points.
    -   `flight/`: The main Flight3 application.
    -   `simplepb/`: A minimal PocketBase server example.
-   **`docs/`**: Documentation files, including architecture notes and upgrade guides.
-   **`internal/`**: Contains the core application logic and private packages.
    -   `flight/`: The heart of the application, encompassing:
        -   `flight.go`: Main orchestration logic.
        -   `banquethandler.go`: Handling and serving of Banquet requests.
        -   `managepocketbase.go`: Initializing PocketBase collections and schema.
        -   `cache.go`: Data source caching and key generation.
        -   `history.go`: Tracking of recent requests.
-   **`logs/`**: Stores application log files generated during runtime.
-   **`pb_data/`**: The default directory for PocketBase data storage (SQLite files).
-   **`pb_public/`**: Static assets served by PocketBase, such as sample data and potentially frontend artifacts.
-   **`test_output/`**: Contains artifacts and logs generated from test executions.
-   **`tests/`**: Holds the project's test suites and integration tests.
