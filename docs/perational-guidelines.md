# Operational Guidelines for Project Waypoint - "Waypoint Archive" Scripts

**Date:** May 25, 2025
**Version:** 1.0

This document outlines the operational guidelines for the development of the "Waypoint Archive" scripts for Project Waypoint. Adherence to these guidelines is expected to ensure code quality, consistency, and maintainability.

## 1. Overall Language & Environment

1.1. **Primary Language:** Go (Golang), latest stable version (e.g., 1.22.x or newer).
1.2. **Fallback Language:** Python (latest stable 3.x), for consideration if Go development encounters significant, unforeseen roadblocks for the project's timeline.
1.3. **Execution Environment:**
    * Indexing scripts: User's laptop (OS to be confirmed by user, assumed to be a common platform like Windows, macOS, or Linux).
    * Archival & Data Extraction scripts: Synology NAS (Linux-based environment). Scripts should be developed with portability in mind.

## 2. Detailed Coding Standards (for Go)

2.1. **File Naming:**
    * Go source files should use `snake_case.go` (e.g., `page_navigator.go`, `topic_parser.go`).
    * Test files should be named `snake_case_test.go` (e.g., `page_navigator_test.go`).

2.2. **Package Naming:**
    * Package names should be short, concise, all lowercase, and typically a single word.
    * Avoid `under_scores` or `mixedCaps` in package names.
    * The package name is the base name of its source directory (e.g., a directory `indexer` contains files belonging to package `indexer`).

2.3. **Identifier Naming (Variables, Functions, Types, Constants):**
    * **Exported Identifiers (globally visible):** Must start with a capital letter (e.g., `MaxRetries`, `IndexSubForum`, `type PageInfo`). Use `PascalCase` or `MixedCaps`.
    * **Unexported (Package-Private) Identifiers:** Must start with a lowercase letter (e.g., `maxRetries`, `indexSubForumInternal`, `type pageInfoInternal`). Use `camelCase`.
    * Strive for short, descriptive names. Avoid overly long names but prefer clarity over extreme brevity.
    * Acronyms should be all uppercase if they are the entire identifier (e.g., `ServeHTTP`, `UserID`) or at the beginning of an unexported identifier (`userID`), but use standard casing if part of a mixed-case identifier (e.g., `parseUrl`, not `parseURL` for an unexported function).

2.4. **Code Formatting:**
    * **Mandatory Tool:** All Go code MUST be formatted using `gofmt` before committing. Using `goimports` (which also runs `gofmt` and manages imports) is highly recommended.
    * **Line Length:** While `gofmt` handles most things, aim for readable line lengths (typically under 100-120 characters where practical).

2.5. **Commenting Practices:**
    * All exported (public) packages, types, functions, and constants MUST have clear, concise Go `doc` comments explaining their purpose and usage.
    * Comment complex or non-obvious logic blocks inline.
    * Avoid redundant comments that merely restate what the code clearly does.
    * Use `// TODO:` or `// FIXME:` prefixes for items needing future attention.

2.6. **Module & Package Structure (Initial Proposal for Archival Scripts):**
    * A primary Go module will be created for the "Waypoint Archive" scripts (e.g., `waypoint_archive_scripts`).
    * Within this, distinct functionalities should be organized into packages. For example:
        * `main.go` (or `cmd/indexer/main.go`, `cmd/archiver/main.go` if creating separate executables) for the entry points.
        * `pkg/indexer` (or just `indexer` at the root if simple): For logic related to Epic 1 (sub-forum navigation, topic ID extraction, two-pass strategy, index storage, metrics).
        * `pkg/archiver` (or just `archiver`): For logic related to Epic 2 (topic page navigation, HTML download, file storage on NAS, resumability, metrics).
        * `pkg/parser` (or just `parser`): For logic related to Epic 3 (HTML content parsing, quote extraction, JSON structuring).
        * `pkg/common` (or `internal/common`): For any shared utilities, types, or constants used across multiple packages (e.g., politeness mechanisms, error handling helpers, data structures).
    * This structure can evolve, but the goal is logical separation of concerns.

2.7. **Error Handling:**
    * Adhere strictly to Go's idiomatic error handling: functions that can fail MUST return an `error` as their last return value.
    * Check for errors immediately after a function call: `if err != nil { return err // or handle appropriately }`.
    * Provide context to errors using `fmt.Errorf("myOperation: %w", err)` to wrap errors where appropriate, allowing for error unwrapping.
    * Define custom error types or use sentinel errors (e.g., `var ErrTopicNotFound = errors.New("topic not found")`) for specific, expected error conditions that callers might need to check for.

2.8. **Logging:**
    * For initial development, the standard `log` package is acceptable for simplicity.
    * Log messages should be clear and provide context (e.g., current operation, relevant IDs).
    * Use distinct log levels if employing a more advanced logging library later (DEBUG for detailed developer info, INFO for progress, ERROR for issues). For now, clear print statements via `log.Printf` are sufficient.

2.9. **Concurrency:**
    * Leverage goroutines and channels for concurrent operations, especially for network requests (indexing and archival) and potentially for parallel file processing in Phase 3.
    * Ensure proper synchronization mechanisms (e.g., `sync.WaitGroup`, mutexes) are used if shared data is accessed by multiple goroutines, although designing to avoid shared mutable state is preferred.

## 3. Comprehensive Testing Strategy (for Go)

3.1. **Philosophy:**
    * All new, non-trivial code implementing business logic (e.g., parsing, state management, core algorithmic steps) should be accompanied by automated tests.
    * Focus on creating tests that provide confidence in the correctness of the code and prevent regressions.
    * Tests should be reliable, repeatable, and easy to run.

3.2. **Types of Tests & Scope:**
    * **Unit Tests:**
        * **Primary Focus:** This will be the most common type of test. Unit tests should verify individual functions and methods in isolation.
        * **Scope:** Test specific logic for HTML parsing (e.g., extracting Topic IDs, parsing quote blocks), URL generation, data structuring, state management logic (e.g., resumability), and metric calculations.
        * **Dependencies:** External dependencies (network calls, file system access) MUST be mocked or stubbed out for unit tests to ensure they are fast and test only the unit of code in question. For example, when testing a function that parses an HTML string, provide the string directly rather than having the test make an HTTP call.
    * **Integration Tests (Simplified for this project):**
        * **Scope:** For the "Waypoint Archive" scripts, full-scale integration tests will largely be covered by the planned test runs on small, live sub-forums (as defined in User Stories like 1.7 and 2.9). These act as integration tests for the entire script's workflow (e.g., can it navigate, extract, and save data for a small sub-forum correctly?).
        * **Focus:** Verifying the interaction between the different modules/packages of your scripts (e.g., does the indexer correctly pass data to the component that saves it to a file?).

3.3. **Testing Frameworks & Libraries:**
    * **Primary Tool:** Utilize Go's built-in `testing` package for writing unit tests.
    * **Assertion Libraries:** While the standard library is often sufficient, consider using a minimal third-party assertion library (e.g., `stretchr/testify/assert`) if it significantly improves test readability and reduces boilerplate, but this is optional.
    * **Mocking:** For mocking interfaces (e.g., if you define an interface for HTTP fetching to allow for mock implementations in tests), you can use standard Go techniques or a library like `gomock` if complexity warrants, though for these scripts, simple interface satisfaction with mock structs might suffice.

3.4. **Guidelines for Writing Effective Tests:**
    * **Naming:** Test functions must be named `TestXxx` (where `Xxx` starts with a capital letter and describes the function being tested, e.g., `TestExtractTopicIDsFromPage`).
    * **Clarity:** Tests should be easy to read and understand. Each test should ideally verify one specific aspect or behavior.
    * **Table-Driven Tests:** For functions that need to be tested with multiple different inputs and expected outputs, use Go's table-driven test pattern to keep tests concise and comprehensive.
    * **AAA Pattern (Arrange, Act, Assert):** Structure your tests clearly:
        * **Arrange:** Set up the necessary preconditions and inputs.
        * **Act:** Execute the function or method being tested.
        * **Assert:** Verify that the outcome (return values, state changes) is as expected.
    * **Edge Cases:** Test for edge cases, error conditions, and boundary values (e.g., empty HTML, pages with no topics, malformed data where appropriate).

3.5. **Test Data Management:**
    * For unit tests involving HTML parsing, use small, self-contained sample HTML strings or embed sample HTML files directly within your test package as test data. Do not rely on live network requests in unit tests.
    * For the broader integration/test runs (Story 1.7, 2.9), a specific, small, live sub-forum will be used as test data.

3.6. **Test File Location:**
    * Test files MUST be located in the same package (directory) as the code they are testing.
    * Test files MUST be named with the `_test.go` suffix (e.g., `parser_test.go` would test code in `parser.go`).

3.7. **Code Coverage (Guideline):**
    * While a specific percentage is not mandated, aim for high code coverage (e.g., >80%) for critical logic, especially for parsing functions and state management.
    * Use Go's built-in code coverage tools (`go test -cover`) to assess coverage and identify untested areas.
    * The focus should be on meaningful tests that provide confidence, not just achieving a coverage number.

## 4. Error Handling and Logging Protocols (for Go)

4.1. **Error Handling Philosophy:**
    * Errors are expected and should be handled gracefully to ensure script robustness and data integrity.
    * Follow Go's idiomatic error handling: functions that can encounter errors MUST return an `error` type as their last return value.
    * Nil `error` values indicate success; non-nil values indicate failure.

4.2. **Checking and Propagating Errors:**
    * Errors returned from function calls MUST be checked immediately.
        ```go
        data, err := someFunction()
        if err != nil {
            // Handle error or return it, possibly wrapped
            return fmt.Errorf("failed during my current operation because %s failed: %w", "someFunction", err)
        }
        // Proceed with data
        ```
    * When returning an error from a function, wrap it with `fmt.Errorf("my_function_context: %w", err)` to provide a stack-like trace of context. This helps in pinpointing where an error originated.
    * For errors that are expected and actionable by the caller (e.g., "item not found," "rate limit exceeded"), consider defining specific sentinel errors (e.g., `var ErrRateLimited = errors.New("rate limit exceeded")`) or custom error types that callers can check against using `errors.Is()` or `errors.As()`.

4.3. **Retry Logic:**
    * For transient errors (e.g., temporary network issues, some HTTP 5xx server errors), a simple retry mechanism MAY be implemented.
    * Retries should include a delay (e.g., exponential backoff) and a maximum number of attempts to avoid indefinite loops.
    * This is distinct from the overall script resumability (which handles longer interruptions) but can improve robustness for short-lived issues.

4.4. **Logging Protocols:**
    * **Logging Tool:** For simplicity and consistency, the standard Go `log` package (`log.Printf`, `log.Fatalf`, etc.) is the baseline. If more structured logging is desired later, a library like `zerolog` or `zap` could be considered, but `log` is sufficient for the initial "Waypoint Archive" scripts.
    * **Log Levels (Conceptual, even with standard `log`):**
        * **`INFO` Level (using `log.Printf`):** For significant progress milestones, script start/stop events, completion of major tasks (e.g., "INFO: Started indexing sub-forum ID 54", "INFO: Successfully archived 1000 pages for topic X", "INFO: ETC for current sub-forum: 2h 15m"). The metrics and ETCs from Story 1.5 and 2.7 fall into this category.
        * **`WARNING` Level (using `log.Printf` prefixed with `[WARNING]`):** For recoverable issues, unexpected data that can be skipped, or situations that don't stop the script but should be noted (e.g., "WARNING: Could not parse quote block attribution on page X, skipping quote metadata but continuing with post text", "WARNING: HTTP 404 for topic Y, page Z - marking as not found").
        * **`ERROR` Level (using `log.Printf` prefixed with `[ERROR]`):** For errors related to a specific item that prevent its successful processing (e.g., "ERROR: Failed to download page X after 3 retries: connection timed out") or more significant issues that might require attention. The script might continue with the next item if the error is localized.
        * **`FATAL` Level (using `log.Fatalf`):** For critical errors that prevent the script from continuing at all (e.g., invalid essential configuration, inability to write to output directory). `log.Fatalf` will print the message and then call `os.Exit(1)`.
    * **Log Format & Content:**
        * All log messages produced by the standard `log` package will automatically include a timestamp.
        * Messages should be clear, concise, and provide relevant context (e.g., current sub-forum ID, topic ID, page URL, function where the event occurred).
        * Example: `log.Printf("[INFO] Indexer: Processing page %d for sub-forum %s", currentPage, subForumID)`
    * **Log Output:**
        * Logs should be written to `os.Stdout` (for `Printf`) and `os.Stderr` (implicitly for `Fatalf`) by default, allowing for easy monitoring when running manually and for capture by schedulers like `cron`.
        * Consider adding a command-line flag to optionally redirect log output to a file on the Synology NAS for persistent records of long runs.
    * **Sensitive Information:** No sensitive information (like passwords, API keys, if any were ever to be used by these specific scripts) should be written to logs. (Currently not applicable, but a general good practice).

## 5. Security Best Practices (for Go Scripts)

5.1. **Input Sanitization/Validation (for Script Parameters):**
    * Any parameters passed to the scripts (e.g., sub-forum IDs, target URLs, directory paths from configuration files or command line) should be reasonably validated to prevent unexpected behavior or errors. For example, ensure numeric IDs are indeed numeric, paths are sensible.
    * This is less about preventing web vulnerabilities (as these are not web servers) and more about robust script operation.

5.2. **Secrets Management (Minimal for these scripts):**
    * The "Waypoint Archive" scripts are not expected to handle sensitive secrets like passwords or API keys for external services (beyond what might be needed for The Magic Cafe if it had such protections, which it doesn't seem to for read access).
    * The custom User-Agent string, if it includes an email, should be configurable and not hardcoded if the script were ever to be shared. For your personal use, this is less critical but good to keep in mind.
    * Avoid hardcoding any file paths that might be specific to one machine if the scripts are intended to be portable even between your own systems (e.g., laptop to NAS); use configurable paths instead.

5.3. **Safe File System Operations:**
    * When creating directories and writing files (raw HTML, structured JSON, index files, logs), ensure that file paths are constructed safely.
    * All file operations should be contained within the designated root archive directory on your Synology NAS or laptop to prevent accidental writes to unintended locations. Validate and sanitize any path components derived from external data (like topic titles, if used in file names, though IDs are safer).

5.4. **Dependency Management & Security:**
    * Use Go modules (`go.mod`, `go.sum`) to manage dependencies.
    * Periodically, you can run `go list -m -u all` to check for outdated dependencies and consider updating them, especially if security advisories are released for any libraries used. Tools like `govulncheck` can be used to check for known vulnerabilities in your dependencies.
    * For this project, keep the number of third-party dependencies minimal and use well-known, reputable libraries where necessary (e.g., `PuerkitoBio/goquery` if chosen for HTML parsing).

5.5. **Polite Scraping & Web Interaction:**
    * Adherence to the "Polite Scraping" strategy (rate limiting, off-peak hours, User-Agent) is also a security consideration. Aggressive scraping can lead to your IP being blocked by the forum, hindering data collection. It also respects the target server's resources.

5.6. **Data Integrity (of the Archive):**
    * While full checksumming of every downloaded file might be overkill, the scripts should handle HTTP errors correctly to ensure that partially downloaded or corrupted HTML pages are either retried or clearly logged as problematic, rather than being saved as if they were complete.
    * Ensure file write operations check for errors to confirm data is saved successfully to disk.

5.7. **Principle of Least Privilege (for Script Execution):**
    * The scripts should only require the permissions necessary to perform their tasks: network access to `themagiccafe.com` and read/write access to the designated local directories for the archive, index files, and logs.

## 6. Version Control (Git) Workflow

6.1. **Repository Setup:**
    * A single Git repository SHOULD be used for all code related to the "Waypoint Archive" scripts (indexer, archiver, data extractor, and any utility/shared modules). This aligns with the PRD's technical assumption.
    * Initialize the repository at the root of your `waypoint_archive_scripts` project directory.

6.2. **Branching Strategy (Recommendation for Solo Developer):**
    * **`main` Branch:** This branch should represent the stable, working version of your scripts.
    * **Feature Branches:** For developing each new User Story (e.g., Story 1.1, Story 1.2), it is highly recommended to create a new branch from `main` (e.g., `feature/story-1.1-nav-logic`, `fix/parser-bug`).
        * Do your development and commits on this feature branch.
        * Once the story is complete and tested (as per its Acceptance Criteria), merge the feature branch back into `main`.
        * This keeps `main` cleaner and makes it easier to manage ongoing work.
    * While working directly on `main` is possible for a solo project, using feature branches is a good habit for better organization and isolation of changes.

6.3. **Commit Messages:**
    * Commits MUST have clear, concise, and descriptive messages.
    * **Format:** Start with a subject line (e.g., 50 characters max, imperative mood like "Add: Sub-forum page navigation logic" or "Fix: Handle missing pagination").
    * **Body (Optional):** If more detail is needed, add a blank line after the subject and provide a more detailed explanation.
    * **Reference Story ID (Recommended):** If a commit relates to a specific User Story, consider referencing it in the commit message (e.g., "feat(indexer): Implement page navigation for sub-forums (Story 1.1)").

6.4. **Commit Frequency:**
    * Commit small, logical, atomic units of work.
    * Avoid making very large commits with many unrelated changes.
    * Commit frequently as you complete distinct parts of a feature or fix.

6.5. **`.gitignore` File:**
    * A `.gitignore` file MUST be used at the root of the repository.
    * It should ignore:
        * Compiled binaries (e.g., if you compile Go executables locally before moving to NAS).
        * Operating system-specific files (e.g., `.DS_Store` for macOS, `Thumbs.db` for Windows).
        * IDE configuration files/directories (e.g., `.vscode/`, `.idea/`).
        * Any sensitive files (though none are anticipated for these scripts).
        * Large log files or test output files that don't need to be versioned.
        * Potentially the `progress.json` or similar state files if they are machine-specific and not intended to be shared or versioned.

6.6. **Remote Repository (Recommended):**
    * While not strictly required for local execution, it is highly recommended to use a private remote Git repository (e.g., on GitHub, GitLab, Bitbucket) for your project.
    * This provides a secure backup of your codebase.
    * It facilitates easier transfer of code between your laptop and NAS (if you choose to clone the repo there).
    * It's essential if you ever decide to collaborate with someone else.
    * Push your `main` branch and feature branches to the remote repository regularly.

## 7. Code Documentation Standards (for Go)

7.1. **Philosophy:**
    * Code should be documented to the extent necessary for another developer (or your future self) to understand its purpose, usage, and any non-obvious design decisions.
    * Good naming and clear code structure can reduce the need for excessive comments, but documentation is still vital for context and APIs.

7.2. **Go Doc Comments (Mandatory for Exported Identifiers):**
    * All exported (public) packages, functions, types (structs, interfaces), and global constants/variables MUST have Go `doc` comments immediately preceding their declaration.
    * These comments should start with the name of the identifier being documented (e.g., `// IndexSubForum performs...` or `// PageInfo holds...`).
    * Their purpose is to explain what the identifier represents or what the function does, its parameters, and its return values. This allows `go doc` and IDEs to generate helpful documentation.
    * Example:
        ```go
        // Package indexer handles the discovery and collection of topic IDs from the forum.
        package indexer

        // MaxRetries defines the default maximum number of retries for a network request.
        const MaxRetries = 3

        // TopicInfo holds basic information about a discovered forum topic.
        type TopicInfo struct {
            ID    string
            Title string
            URL   string
        }

        // FetchAndParsePage fetches the given URL, parses it, and returns PageInfo.
        // It handles retries internally based on MaxRetries.
        func FetchAndParsePage(pageURL string) (*PageInfo, error) {
            // ... implementation ...
        }
        ```

7.3. **Inline Comments:**
    * Use inline comments (`//`) within function bodies to explain complex or non-obvious sections of code, algorithms, or important decisions.
    * Avoid commenting on code that is self-explanatory. Focus on the *why* not just the *what* if the *what* is clear from the code.

7.4. **Package-Level Comments:**
    * Each package should have a package comment (a `doc` comment preceding the `package` clause in one of the `.go` files, often `doc.go` or the main file of the package) explaining the overall purpose and responsibility of the package.

7.5. **Module `README.md` (Project Level):**
    * It is highly recommended to have a `README.md` file at the root of your `waypoint_archive_scripts` Go module/project.
    * This `README.md` should briefly describe:
        * The purpose of the scripts (Waypoint Archive indexing, archival, etc.).
        * How to build/compile the scripts (if necessary).
        * How to configure the scripts (e.g., command-line arguments, expected environment variables, configuration files).
        * How to run the different scripts (indexer, archiver).
        * Any prerequisites or dependencies needed to run the scripts.

7.6. **Consistency:**
    * Strive for a consistent style and level of detail in your comments and documentation.