# Rclone Configuration in PocketBase

This document details how `rclone` settings, configuration, and cache keys are stored and retrieved using PocketBase collections in the Flight3 application.

## Overview

PocketBase serves as the central configuration engine for data orchestration. Instead of static configuration files, Flight3 queries PocketBase collections to dynamically determine how to connect to various remotes (S3, GCS, R2, etc.) and how to process the data they contains.

## 1. PocketBase Collections

The following collections in PocketBase manage the `rclone` and data pipeline lifecycle:

### `rclone_remotes`
This collection stores the base connection details for any cloud storage provider.
- **`name`** (Text, Required): A unique alias for the remote (e.g., `gs`, `r2_main`, `marketing_bucket`). This typically matches the `Hostname` used in a Banquet URI.
- **`type`** (Text, Required): The `rclone` backend type (e.g., `s3`, `google cloud storage`, `drive`).
- **`config`** (JSON): A dictionary of `rclone`-specific configuration parameters.
    - Example for R2: `{"provider": "Cloudflare", "access_key_id": "...", "secret_access_key": "...", "endpoint": "..."}`
    - Example for GCS: `{"service_account_file": "...", "project_number": "..."}`

### `mksqlite_configs`
Defines how raw files fetched via `rclone` should be converted into SQLite databases.
- **`name`** (Text, Required): Name of the configuration.
- **`driver`** (Text): The `mksqlite` converter to use (e.g., `csv`, `excel`, `json`).
- **`args`** (JSON): Driver-specific arguments (e.g., `{"delimiter": ",", "header": true}`).

### `data_pipelines`
Orchestrates the link between a remote source and its processing logic.
- **`name`** (Text, Required): Human-readable name for the pipeline.
- **`rclone_remote`** (Relation): Link to a record in `rclone_remotes`.
- **`rclone_path`** (Text): The path within the remote where the dataset resides (e.g., `path/to/data.csv`).
- **`mksqlite_config`** (Relation): Link to a record in `mksqlite_configs`.
- **`cache_ttl`** (Number): Time-to-live for the cached SQLite database in minutes.

---

## 2. Retrieval Logic

When a Banquet request arrives:

1. **URI Parsing**: The `HandleBanquet` handler parses the request into a `banquet.Banquet` object.
2. **Identification**: The `Hostname()` from the banquet object is used to query the `rclone_remotes` collection by the `name` field.
3. **Configuration Hydration**: The `type` and `config` JSON are extracted to create an `rclone` `Fs` object.
4. **VFS Attachment**: A Virtual File System (VFS) is instantiated for that specific remote/config combination.

---

## 3. Cache Key Generation (`GenCacheKey`)

Cache keys are critical for ensuring that data is reused across requests while maintaining security and isolation.

The `GenCacheKey` function generates a unique string by joining:
- **`UserInfo()`**: Captures any authentication or custom user scoping present in the URL.
- **`Hostname()`**: Identifies the specific remote alias (`rclone_remotes.name`).
- **`DatasetPath()`**: Identifies the specific file or directory within that remote.

**Format**: `[UserInfo]-[Hostname]-[DatasetPath]`

This key is used to name the local SQLite cache file (e.g., `pb_data/cache/user-gs-path_to_data.db`).

---

## 4. VFS Settings Storage

Standard VFS settings (like `CacheModeFull`, `ChunkSize`, and `ReadAhead`) are currently initialized with defaults in the Go code. However, the architecture allows for these to be overridden per remote by adding a `vfs_settings` JSON field to the `rclone_remotes` collection in the future.

### Default VFS Tuning:
- **Cache Mode**: `CacheModeFull` (Necessary for random-access on SQLite/Excel).
- **Read Chunk Size**: `128M`.
- **Directory Cache Time**: `10m`.
