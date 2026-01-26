# Rclone Integration in Flight2

This document summarizes how Flight2 integrates `rclone` to handle file fetching and caching, based on `internal/dataset_source/source.go`.

## Overview

Flight2 uses `rclone` as a universal filesystem abstraction layer, allowing it to treat local files, HTTP URLs, S3 buckets, and other cloud storage providers uniformly. It leverages `rclone`'s **VFS (Virtual File System)** feature to handle caching and file access.

## Key Components

### 1. Initialization and Configuration
The integration is initialized via `dataset_source.Init(cacheDir)`, which sets the global `rclone` cache directory.

### 2. VFS Caching Strategy
Flight2 implements a robust caching mechanism using `rclone`'s VFS layer:

-   **Global Cache Map**: A `vfsCache` map stores active VFS instances, keyed by a hash of the credentials and root path. This ensures reuse of connections and cache for the same source.
-   **Cache Configuration**:
    -   `CacheMode: vfscommon.CacheModeFull`: This is a critical setting. It means `rclone` will cache the entire file on disk. This allows for random access (seeking) which is essential for efficient reading of large files like SQLite databases or Excel sheets without downloading them repeatedly.
    -   `DirCacheTime`: 10 minutes.
    -   `CacheMaxAge`: 24 hours.
    -   `CachePollInterval`: 1 minute.
    -   `ChunkSize`: 128 MB.

### 3. File Access Workflow (`getVFS`)
When a file is requested:
1.  **Type Detection**: Determines the filesystem type (e.g., `local`, `http`, `s3`) based on credentials or URL scheme.
2.  **Root & Path Resolution**: Separates the "Remote Root" (bucket, domain, or system root) from the "Relative Path" to the file.
3.  **Hash Generation**: Creates a unique MD5 hash based on the credentials and file root to identify the VFS instance.
4.  **VFS Instantiation/Retrieval**: Checks the `vfsCache`. If a VFS for this source doesn't exist, it creates a new one using `regInfo.NewFs` and wraps it in `vfs.New`.

### 4. Operations
-   **Fetching Content (`GetFileStream`)**: Opens a file using the VFS `OpenFile` method. Because of `CacheModeFull`, access to this file will trigger `rclone` to download and cache it locally if it's not already present.
-   **Listing Directories (`ListEntries`)**: Uses `vfs.ReadDir` to list contents, abstracting away the differences between listing a local directory and an S3 bucket.

## Summary of Usage
-   **Fetching**: handled via `GetFileStream`.
-   **Caching**: handled transparently by `rclone` VFS with `CacheModeFull`.
-   **Protocols**: Supports `local`, `http`, `https`, and any cloud provider supported by `rclone` (via generic credential map passing).
