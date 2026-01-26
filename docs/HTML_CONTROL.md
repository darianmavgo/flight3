# HTML & CSS Control Documentation

This document summarizes the locations of HTML templates, CSS styles, and UI assets within the Flight3 project.

## 1. Banquet Data Views (Tables)
The "Banquet" feature renders SQL/CSV/Rclone data as HTML tables.

- **Location**: `templates/` directory in the project root.
- **Key Files**:
    - `templates/head.html`: Contains the HTML `<head>`, **CSS Styles** (`<style>...</style>`), and the opening `<table>` tag. **Edit this file to change the look and feel of the data tables.**
    - `templates/row.html`: Defines the rendering of a single table row (`<tr>...</tr>`). Includes logic for rendering clickable links (`<a href="...">`) for columns named "path" or "name".
    - `templates/foot.html`: contains the closing tags (`</table></body></html>`).
- **Logic**:
    - Loaded by `cmd/flight/main.go` via `template.ParseGlob("templates/*.html")`.
    - Rendered in `handleBanquet` function using `tpl.ExecuteTemplate(..., "row.html", ...)`.

## 2. Admin Dashboard (PocketBase UI)
The main application dashboard served at `/_/`.

- **Source**: `cmd/flight/ui/` directory.
    - `cmd/flight/ui/index.html`: The main entry point for the Single Page Application (SPA).
- **Styling**:
    - **Bundled CSS**: Contained within the assets linked in `index.html`.
    - **Injected CSS (Dark Mode)**: There is a middleware in `cmd/flight/main.go` (around line ~183) that intercepts requests to `/_/` and injects a `<style>` block to invert colors (Dark Mode effect):
      ```css
      html { filter: invert(1) hue-rotate(180deg); }
      ```
      **Modify `cmd/flight/main.go` to change or remove this global style override.**

## 3. Root Path (`/`)
The root path `http://localhost:8090/` is overlaid to serve the "Local" file listing.

- **Logic**: A specific handler in `cmd/flight/main.go` intercepts `GET /`.
- **Rendering**: It reuses the **Banquet Data Views** (see Section 1) to render the directory listing of the configured `serve_folder` (e.g., `sample_data/`).

## 4. Source & Backup Templates
- `cmd/flight/templates/`: Contains the original copy of the templates. If `templates/` in the root is deleted or needs resetting, these can be copied over.

## Summary of CSS Locations
| Component | Location | Notes |
| :--- | :--- | :--- |
| **Data Tables** | `templates/head.html` | `<style>` block defines table colors, dark mode background (`#0f172a`), and typography. |
| **Admin UI** | `cmd/flight/main.go` | Injected `<style>` block (filter inversion) in Go middleware. |
| **Admin UI (Base)** | `cmd/flight/ui/` | Bundled styling (minified). |
