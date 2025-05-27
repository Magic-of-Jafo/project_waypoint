# Troubleshooting: Archiver Silent Exit and Logging Failure - 2025/05/27

## 1. Initial Problem Description

The `waypoint_archive_scripts/cmd/archiver/run_archiver.go` script was exhibiting a silent exit. When executed with `go run ./waypoint_archive_scripts/cmd/archiver/run_archiver.go ./config.json`, the script would terminate with an exit status 1 almost immediately after loading its `config.json` file. 

Critically, no log file was being generated at the configured path (`./test_archive_output/logs/test_run.log`), making initial diagnosis difficult. This behavior was reminiscent of a previously encountered issue.

## 2. Diagnostic Steps Undertaken

The primary challenge was the lack of log output. To overcome this, the following steps were taken:

1.  **Handoff Note Review**: Reviewed the predecessor's notes which suspected an issue in `main()` after `config.LoadConfig()` and within or before `initLogging()`.
2.  **Temporary Debug Prints**: Added `fmt.Println("DEBUG: Checkpoint X")` statements at strategic locations within the `main()` and `initLogging()` functions in `run_archiver.go`. This was crucial to trace the execution flow directly to the console, bypassing the standard `log` package which was failing to initialize file logging.
    *   Statements were placed before and after calls to `config.LoadConfig()` and `initLogging()`.
    *   Within `initLogging()`, prints were added before and after `filepath.Dir()`, `os.MkdirAll()`, `os.OpenFile()`, and `log.SetOutput()`. Values of key variables like `logDir`, `cfg.LogFilePath`, and errors from `os.OpenFile` were also printed.
3.  **Iterative Execution and Output Analysis**:
    *   The script was run with the debug prints. The console output from these prints showed that `initLogging()` was being called, `os.MkdirAll()` and `os.OpenFile()` were succeeding without error, and the `initLogging` function was running to completion.
    *   However, standard log messages (e.g., `log.Println("[DEBUG] main: initLogging completed.")`) were *not* appearing on the console *after* `initLogging` seemed to complete. This suggested that `log.SetOutput()` *was* successfully redirecting the `log` package's output away from the console.
4.  **Log File Inspection**: Based on the above, the log file (`./test_archive_output/logs/test_run.log`) was inspected. This revealed that logging *was* being written to the file. The file contained:
    *   Confirmation of successful redirection: `[INFO] initLogging: Successfully redirected log output to file...`
    *   Several subsequent log lines from `main()` showing progress.
    *   **The root cause**: A `[FATAL]` log message: `Failed to read sub-forum list ../test_data/subforum_list_test.csv: ReadSubForumListCSV: failed to open file ../test_data/subforum_list_test.csv: open ../test_data/subforum_list_test.csv: The system cannot find the path specified.`

## 3. Root Cause Identification

The `log.Fatalf()` call, triggered by the failure to open `cfg.SubForumListFile`, was causing the script to exit. The path specified in `config.json` for `SubForumListFile` was `../test_data/subforum_list_test.csv`.

The script was being executed from the project root (`C:\Users\magic\Synology\TeamFolder\Project Waypoint\code`). The Go runtime (specifically file operations in the `os` package) was interpreting this relative path in a way that didn't resolve to the intended file, despite the file existing at `code/test_data/subforum_list_test.csv`.

The issue was how the `../` was being resolved. While `go run` executes a temporary binary, file paths are generally relative to the directory where `go run` was invoked. The `../test_data/` from `code/` should correctly point to `C:\Users\magic\Synology\TeamFolder\Project Waypoint\test_data\`. The exact reason for the failure with `../` in this specific Go environment on Windows was not pinpointed but was circumvented.

## 4. Solution Implemented

The path for `SubForumListFile` in `config.json` was changed from:
`"SubForumListFile": "../test_data/subforum_list_test.csv"`
to:
`"SubForumListFile": "./test_data/subforum_list_test.csv"`

This path is relative to the project root directory (where `config.json` resides and `go run` is executed). This change allowed the Go program to correctly locate and open the `subforum_list_test.csv` file.

## 5. Verification of Fix

After modifying `config.json`:
1.  The script was re-run.
2.  The log file (`./test_archive_output/logs/test_run.log`) showed that the `ReadSubForumListCSV` function successfully read the sub-forum list using the new path.
3.  The script no longer terminated with a `FATAL` error at that point and proceeded with its main archival logic, processing subforums as configured.
4.  The temporary `fmt.Println` statements were removed from `run_archiver.go`.
5.  A final clean run confirmed the script executed successfully (exit code 0) and performed its tasks as per the log file.

## 6. Key Takeaways

*   **Silent Exits & Logging**: When a Go application exits silently and file logging isn't working, the `log` package itself (especially `log.Fatalf`) might be the cause *after* an attempt to redirect log output but *before* significant logging occurs post-redirection.
*   **`fmt.Println` for Low-Level Debugging**: Direct `fmt.Println` calls are invaluable for tracing execution when the standard `log` package's output is compromised or not yet established.
*   **Relative Paths in Go**: Pay close attention to how relative paths are resolved, especially when using `go run`. Paths relative to the invocation directory of `go run` (often the project root) are generally safer. Using `../` can sometimes lead to ambiguity or platform-dependent behavior if the Go program's understanding of its current working directory during file operations differs from expectations. Using `./` for paths from the project root where `config.json` and the `go run` command are executed provides a more explicit reference point.
*   **Check the Logs (Even if You Think They Failed)**: Even if console output suggests logging failed, always check the target log file if `os.OpenFile` appeared successful. The issue might be *after* redirection.

This incident highlights the importance of robust path handling and having fallback diagnostic techniques when standard logging is unavailable. 