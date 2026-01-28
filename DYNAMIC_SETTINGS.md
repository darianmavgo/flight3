# Dynamic Settings Configuration

Flight3 supports changing certain configuration settings at runtime without requiring a server restart. This is achieved using the `app_settings` collection in PocketBase.

## How It Works

The application checks the `app_settings` collection on every request for specific keys. If a key is found, its value is used immediately.

## Supported Settings

### `serve_folder` (Dynamic Root Directory)

This setting controls the local directory served by Flight3 when listing local files or banquet datasets.

*   **Key**: `serve_folder`
*   **Value**: The path to the directory you want to serve.
    *   **Absolute Path**: e.g., `/Users/me/Documents/MyData`
    *   **Relative Path**: e.g., `../different_folder` (Relative to the application root, typically the parent of `pb_data`).
*   **Default**: `pb_public` (if no setting is found).

## usage Instructions

1.  **Open Admin UI**: Go to `/_/` and log in.
2.  **Open `app_settings`**: Navigate to the `app_settings` collection.
3.  **Create/Edit Record**:
    *   Create a new record (or edit existing).
    *   Set **key** to `serve_folder`.
    *   Set **value** to your desired path.
4.  **Save**.
5.  **Test**: Navigate to the Flight3 homepage `/`. It should immediately show the contents of the new folder.

## Troubleshooting

*   **Recursion**: Avoid setting `serve_folder` to a parent directory that contains `pb_data` to prevent infinite loops if scanning recursively.
*   **Permissions**: Ensure the application (or user running it) has read permissions for the target directory.
