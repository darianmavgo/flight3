# Integrated Rclone Browser App

This application provides a web interface to browse files on Rclone-supported remote storage keys, managed via PocketBase.

## Setup

1.  **Dependencies**: Ensure you have dependencies installed.
    ```bash
    go mod tidy
    ```

2.  **Run**:
    ```bash
    go run ./internal/pocket_rclone
    ```

3.  **Admin Setup**:
    *   Open [http://localhost:8090/_/](http://localhost:8090/_/)
    *   Create an Admin account.
    *   The `rclone_remotes` collection will be automatically created.
    *   Add a test record to `rclone_remotes`:
        *   **Name**: `public_test`
        *   **Type**: `http`
        *   **Config**: `{"url": "https://pub.rclone.org"}`

4.  **Usage**:
    *   Open [http://localhost:8090/](http://localhost:8090/)
    *   You should see the "public_test" remote (after you create it).
    *   Click to browse files.

## Architecture

*   **PocketBase**: Manages data (`rclone_remotes`) and serves the HTTP API.
*   **Rclone**: Used as a library to connect to remotes dynamically.
*   **UI**: Single HTML file embedded in the binary, served at root.
