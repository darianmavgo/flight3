# Verbose Mode Logging Strategy

## Flight2 Strategy
Flight2 implements a centralized "Verbose Mode" managed via its HCL configuration (`config.hcl`).

1.  **Configuration**: The `Config` struct has a `Verbose bool` field loaded from HCL.
2.  **Global Logging**:
    *   In `main.go`, it configures `log.SetOutput` to write to both `os.Stderr` and a file `logs/app.log` using `io.MultiWriter`.
3.  **Propagation**:
    *   **Banquet Lib**: Calls `banquet.SetVerbose(true)` to enable debug logs in the URL parser.
    *   **Data Manager**: Passes `cfg.Verbose` to `dataset.NewManager`.
    *   **MkSQLite**: The Manager passes `Verbose: m.verbose` into `common.ConversionConfig` and `converters.ImportOptions`. This enables detailed logging within the converter libraries (e.g. CSV line counts, SQL statements).
4.  **Runtime**:
    *   Logs "Verbose mode enabled across repositories" on startup.

## Flight3 Implementation Status

Flight3 currently does **NOT** fully implement this centralized strategy.

### Implemented
*   [x] **Partially**: `banquethandler.go` has hardcoded verbose behaviors (e.g. `Verbose: true` for converters).
*   [x] **Partially**: `main.go` has conditional logging for some events but mostly relies on default `log` which goes to stdout/stderr (and captured by `restart.sh` into a file).

### Missing / Recommended Implementation
To match Flight2's capabilities, Flight3 should:

1.  **Centralize Control**:
    *   Use PocketBase's `--debug` flag (`app.IsDebug()`) OR a `verbose` key in the `app_settings` collection to toggle this mode globally.
    
2.  **Propagate to Converters**:
    *   Update `banquethandler.HandleBanquet` to accept a `verbose` boolean (or read from App/Context).
    *   **Change**: In `banquethandler.go`, replace hardcoded `Verbose: true` with a variable.
    
    ```go
    // banninghandler.go
    verbose := app.IsDebug() // or passed in argument
    convCfg := &common.ConversionConfig{
        Verbose: verbose,
    }
    // ...
    err = converters.ImportToSQLite(conv, outFile, &converters.ImportOptions{Verbose: verbose})
    ```

3.  **Propagate to Banquet Lib**:
    *   Call `banquet.SetVerbose(verbose)` in `main.go` startup or `OnServe` based on the setting.

4.  **Unified Log Output**:
    *   Configure `log.SetOutput` in `main.go` (similar to Flight2) to ensure logs persist to `logs/flight_output.txt` even if not run via `restart.sh`, or better yet, rely on the system service/process manager (which `restart.sh` currently mimics).
