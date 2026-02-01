# Flight3

Flight3 is a modern data serving platform that unifies cloud storage, local data, and configuration management into a single cohesive interface. It acts as the "glue" that binds **RocketBase** (Configuration), **Rclone** (Storage), **Banquet** (Parsing), and **SQLiter** (Rendering).

## Features Supported

*   **Unified cloud Storage Access**: Seamlessly connects to S3, GCS, R2, Google Drive, and 40+ other providers via embedded **Rclone**.
*   **Virtual File System (VFS)**: Implements smart caching, connection pooling, and partial reads to make remote cloud files feel local and fast.
*   **Dynamic Configuration**: Uses **PocketBase** as a backend for managing remotes, pipelines, and settings via a friendly Admin UI.
*   **On-the-Fly Conversion**: Automatically detects non-SQLite files (CSVs, Excel) and instigates `mksqlite` to convert them to queryable databases transparently.
*   **Banquet URL Support**: Fully supports the Banquet URL notation for complex, nested data queries.
*   **Integrated Rendering**: Embeds `sqliter` to provide instant visualization of any data source.

## Area of Responsibility

Flight3 is the **Orchestrator**. Its responsibility is **Integration and Delivery**.
*   It manages the lifecycle of requests.
*   It resolves abstract paths (Banquet URLs) to concrete physical resources (Cloud Storage objects).
*   It coordinates the "Fetch -> Convert -> Cache -> Serve" pipeline.
*   It provides the administrative interface for the entire system.

## Scope (What it explicitly doesn't do)

*   **No Native Storage Driver Implementation**: Flight3 does not write its own S3 or GCS clients. It relies entirely on **Rclone** for communicating with storage providers.
*   **No Query Parsing Logic**: It delegates all URL interpretation and query parsing to the **Banquet** library.
*   **No Rendering Logic**: It delegates the UI and table rendering to **SQLiter**.
*   **No File Conversion Logic**: It delegates file format transcoding to **mksqlite**.

## Architecture Overview

```
[User] -> [Flight3 Router]
              |
              +-> (URL?) -> [Banquet Parser]
              |
              +-> (Data?) -> [PocketBase Config] -> [Rclone VFS] -> [Cloud Storage]
              |                   |
              |              (Not SQLite?) -> [Mksqlite Converter]
              |                   |
              |              (Is SQLite?)
              |                   v
              +-> (View?) -> [SQLiter Renderer] -> [Browser]
```
